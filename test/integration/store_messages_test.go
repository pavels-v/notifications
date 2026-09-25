//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"notifications/internal/domain"
	"notifications/internal/store"
)

func insertIntent(t *testing.T, s *store.DB, templateType, body string) *domain.Message {
	t.Helper()

	m, err := store.NewMessages(s.DB()).Insert(context.Background(), domain.Message{
		CreditNumber: "2000000001",
		TemplateType: templateType,
		Body:         body,
	})
	require.NoError(t, err)
	return m
}

func TestInsertMessageRecordsIntentAsPending(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()

	_, err := store.NewCustomers(s.DB()).Create(ctx, sampleCustomer())
	require.NoError(t, err)

	saved := insertIntent(t, s, domain.TemplateTypeReminder, "first text")

	require.NotZero(t, saved.ID, "the database must generate an identifier")
	require.False(t, saved.CreatedAt.IsZero(), "CreatedAt must come back populated")
	require.Equal(t, domain.StatusPending, saved.Status,
		"a row written before the provider is called cannot claim any outcome")
	require.Empty(t, saved.ProviderMessageID)
	require.Empty(t, saved.ResponseRaw)
	require.Empty(t, saved.Error)

	history, err := store.NewMessages(s.DB()).List(ctx, "2000000001")
	require.NoError(t, err)
	require.Len(t, history, 1, "the intent must be durable on its own, before any outcome is known")
}

func TestUpdateMessageOutcome(t *testing.T) {
	cases := []struct {
		name    string
		outcome domain.MessageOutcome
	}{
		{
			name: "the provider accepted it",
			outcome: domain.MessageOutcome{
				Status:            domain.StatusSentToProvider,
				ProviderMessageID: "mb-1",
				ResponseRaw:       `{"id":"mb-1","recipients":{"items":[{"status":"sent"}]}}`,
			},
		},
		{
			name: "the provider refused it",
			outcome: domain.MessageOutcome{
				Status:      domain.StatusRejectedByProvider,
				ResponseRaw: `{"errors":[{"code":9,"description":"no (correct) recipients found"}]}`,
				Error:       "sms: provider rejected the message: http 422",
			},
		},
		{
			name: "the provider never answered, so the outcome stays unknown",
			outcome: domain.MessageOutcome{
				Status: domain.StatusPending,
				Error:  "call messagebird: context deadline exceeded",
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := newStore(t)
			ctx := context.Background()

			_, err := store.NewCustomers(s.DB()).Create(ctx, sampleCustomer())
			require.NoError(t, err)

			intent := insertIntent(t, s, domain.TemplateTypeReminder, "first text")

			saved, err := store.NewMessages(s.DB()).UpdateOutcome(ctx, intent.ID, c.outcome)
			require.NoError(t, err)

			require.Equal(t, intent.ID, saved.ID, "the outcome must land on the row written beforehand")
			require.Equal(t, c.outcome.Status, saved.Status)
			require.Equal(t, c.outcome.ProviderMessageID, saved.ProviderMessageID)
			require.Equal(t, c.outcome.ResponseRaw, saved.ResponseRaw,
				"the raw response must come back byte for byte")
			require.Equal(t, c.outcome.Error, saved.Error)
			require.Equal(t, "first text", saved.Body, "the outcome must not disturb what was sent")
			require.True(t, saved.UpdatedAt.After(intent.UpdatedAt),
				"UpdatedAt must record when the outcome was learnt")

			history, err := store.NewMessages(s.DB()).List(ctx, "2000000001")
			require.NoError(t, err)
			require.Len(t, history, 1, "recording an outcome must update the row, not add one")
		})
	}
}

func TestUpdateMessageOutcomeOnAnUnknownRow(t *testing.T) {
	s := newStore(t)

	_, err := store.NewMessages(s.DB()).UpdateOutcome(context.Background(), 999999, domain.MessageOutcome{
		Status: domain.StatusSentToProvider,
	})
	require.ErrorIs(t, err, domain.ErrNotFound, "the API layer switches on this sentinel")
}

func TestMessageStatusIsConstrainedBySchema(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()

	_, err := store.NewCustomers(s.DB()).Create(ctx, sampleCustomer())
	require.NoError(t, err)

	intent := insertIntent(t, s, domain.TemplateTypeReminder, "first text")

	_, err = store.NewMessages(s.DB()).UpdateOutcome(ctx, intent.ID, domain.MessageOutcome{Status: "delivered"})
	require.Error(t, err, "the schema must refuse a status the service does not define")
	require.NotErrorIs(t, err, domain.ErrNotFound, "a refused status is not a missing row")
}

func TestListMessagesIsNewestFirst(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()

	_, err := store.NewCustomers(s.DB()).Create(ctx, sampleCustomer())
	require.NoError(t, err)

	history, err := store.NewMessages(s.DB()).List(ctx, "2000000001")
	require.NoError(t, err)
	require.Empty(t, history, "a loan with no sends has no history")

	first := insertIntent(t, s, domain.TemplateTypeReminder, "first")
	_, err = store.NewMessages(s.DB()).UpdateOutcome(ctx, first.ID, domain.MessageOutcome{
		Status:            domain.StatusSentToProvider,
		ProviderMessageID: "mb-1",
	})
	require.NoError(t, err)

	insertIntent(t, s, domain.TemplateTypeDunning, "second")

	history, err = store.NewMessages(s.DB()).List(ctx, "2000000001")
	require.NoError(t, err)
	require.Len(t, history, 2)

	require.Equal(t, domain.TemplateTypeDunning, history[0].TemplateType, "the newest message comes first")
	require.Equal(t, "mb-1", history[1].ProviderMessageID, "the older message keeps its provider identifier")
}
