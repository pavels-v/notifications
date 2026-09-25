package service

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"notifications/internal/domain"
	"notifications/internal/sms"
)

func discardLog() *slog.Logger { return slog.New(slog.DiscardHandler) }

func aTemplate() domain.Template {
	return domain.Template{Type: domain.TemplateTypeReminder, Body: "Dear {full_name}, please repay {amount} by {due_date}."}
}

func newMessages(t *testing.T, sender *fakeSender) (*Messages, *fakeMessageRepo) {
	t.Helper()

	messages := newFakeMessageRepo()
	sender.repo = messages

	return NewMessages(
		newFakeCustomerRepo(aCustomer()),
		newFakeTemplateRepo(aTemplate()),
		messages,
		sender,
		discardLog(),
	), messages
}

func TestSendRecordsTheIntentBeforeCallingTheProvider(t *testing.T) {
	t.Parallel()

	sender := &fakeSender{resp: &sms.Response{ProviderMessageID: "prov-1", RawBody: `{"id":"prov-1"}`}}
	svc, repo := newMessages(t, sender)

	got, err := svc.Send(context.Background(), "4000000001", domain.TemplateTypeReminder)
	require.NoError(t, err)

	require.Equal(t, []domain.MessageStatus{domain.StatusPending}, sender.statusAtCall,
		"the row must already exist, and say pending, at the moment the provider is called")
	require.Len(t, repo.rows, 1)
	require.Equal(t, domain.StatusSentToProvider, got.Status)
	require.Equal(t, "prov-1", got.ProviderMessageID)
	require.Contains(t, got.Body, "Jane Doe", "the rendered text is what is stored and what is sent")
	require.Equal(t, got.Body, sender.sent[0].Body)
	require.Equal(t, "+999000000001", sender.sent[0].To)
}

func TestSendReturnsTheRowAndAnErrorWhenTheProviderRefuses(t *testing.T) {
	t.Parallel()

	sender := &fakeSender{err: &sms.RejectedError{StatusCode: 503, RawBody: "Service temporarily unavailable"}}
	svc, repo := newMessages(t, sender)

	got, err := svc.Send(context.Background(), "4000000001", domain.TemplateTypeReminder)

	require.ErrorIs(t, err, ErrSendFailed)
	require.NotNil(t, got, "the attempt is returned even though the send failed")
	require.Equal(t, domain.StatusRejectedByProvider, got.Status)
	require.Contains(t, got.Error, "503", "the provider's own status is what is recorded")
	require.Len(t, repo.rows, 1, "the refusal must not remove the row")
}

func TestSendLeavesTheOutcomeUnknownWhenTheProviderNeverAnswers(t *testing.T) {
	t.Parallel()

	sender := &fakeSender{err: errors.New("context deadline exceeded")}
	svc, _ := newMessages(t, sender)

	got, err := svc.Send(context.Background(), "4000000001", domain.TemplateTypeReminder)

	require.ErrorIs(t, err, ErrSendFailed)
	require.NotNil(t, got)
	require.Equal(t, domain.StatusPending, got.Status,
		"an unanswered send may well have gone out, so it must not be recorded as refused")
}

func TestSendRefusesBeforeReachingTheProvider(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		creditNumber string
		templateType string
		wantErr      error
	}{
		{"unknown credit number", "9000000001", domain.TemplateTypeReminder, domain.ErrCustomerNotFound},
		{"unknown template type", "4000000001", "nope", ErrTemplateUnusable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sender := &fakeSender{resp: &sms.Response{ProviderMessageID: "prov-1"}}
			svc, repo := newMessages(t, sender)

			_, err := svc.Send(context.Background(), tt.creditNumber, tt.templateType)

			require.ErrorIs(t, err, tt.wantErr)
			require.NotErrorIs(t, err, domain.ErrTemplateNotFound,
				"a missing dependency must not reach the API as the sentinel that endpoint answers 404 on")
			require.Empty(t, sender.sent, "a refused request must never reach the provider — an SMS cannot be recalled")
			require.Empty(t, repo.rows, "and it must not leave a row claiming an attempt was made")
		})
	}
}

func TestSendPassesStorageFailuresThrough(t *testing.T) {
	t.Parallel()

	errStorage := errors.New("connection refused")

	tests := []struct {
		name      string
		customers *fakeCustomerRepo
		templates *fakeTemplateRepo
		notErr    error
	}{
		{"customer lookup fails", &fakeCustomerRepo{err: errStorage}, newFakeTemplateRepo(aTemplate()), domain.ErrCustomerNotFound},
		{"template lookup fails", newFakeCustomerRepo(aCustomer()), &fakeTemplateRepo{err: errStorage}, ErrTemplateUnusable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sender := &fakeSender{}
			messages := newFakeMessageRepo()
			svc := NewMessages(tt.customers, tt.templates, messages, sender, discardLog())

			_, err := svc.Send(context.Background(), "4000000001", domain.TemplateTypeReminder)

			require.ErrorIs(t, err, errStorage)
			require.NotErrorIs(t, err, tt.notErr, "a storage failure must answer 500, not the status of a missing record")
			require.Empty(t, sender.sent)
			require.Empty(t, messages.rows)
		})
	}
}

func TestSendRefusesATemplateItCannotRender(t *testing.T) {
	t.Parallel()

	sender := &fakeSender{resp: &sms.Response{ProviderMessageID: "prov-1"}}
	messages := newFakeMessageRepo()
	sender.repo = messages

	svc := NewMessages(
		newFakeCustomerRepo(aCustomer()),
		newFakeTemplateRepo(domain.Template{Type: domain.TemplateTypeReminder, Body: "Dear {full_name}, your {iban} is due"}),
		messages,
		sender,
		discardLog(),
	)

	_, err := svc.Send(context.Background(), "4000000001", domain.TemplateTypeReminder)

	require.ErrorIs(t, err, domain.ErrUnknownPlaceholder)
	require.Contains(t, err.Error(), "iban", "the text names the placeholder; it is written into the 422 body")
	require.Empty(t, sender.sent)
	require.Empty(t, messages.rows)
}

func TestSendReportsStorageFailureWhenTheIntentCannotBeRecorded(t *testing.T) {
	t.Parallel()

	sender := &fakeSender{resp: &sms.Response{ProviderMessageID: "prov-1"}}
	svc, repo := newMessages(t, sender)
	repo.insertErr = domain.ErrNotFound

	got, err := svc.Send(context.Background(), "4000000001", domain.TemplateTypeReminder)

	require.Nil(t, got, "no row exists yet, so there is nothing to hand back")
	require.ErrorIs(t, err, ErrAttemptCannotBeRecorded, "a failure to record the intent must be tagged as a storage failure")
	require.NotErrorIs(t, err, domain.ErrNotFound,
		"the fake returns domain.ErrNotFound to prove the wrap hides it: a storage failure must never read as the message not existing")
	require.Empty(t, sender.sent, "the provider must not be called before the intent is recorded")
}

func TestSendReportsStorageFailureWhenTheOutcomeCannotBeRecorded(t *testing.T) {
	t.Parallel()

	sender := &fakeSender{resp: &sms.Response{ProviderMessageID: "prov-1"}}
	svc, repo := newMessages(t, sender)
	repo.outcomeErr = domain.ErrNotFound

	got, err := svc.Send(context.Background(), "4000000001", domain.TemplateTypeReminder)

	require.Nil(t, got, "the outcome write failed, so there is no row to hand back")
	require.ErrorIs(t, err, ErrAttemptCannotBeRecorded, "a failure to record the outcome must be tagged as a storage failure")
	require.NotErrorIs(t, err, domain.ErrNotFound,
		"the fake returns domain.ErrNotFound to prove the wrap hides it: a storage failure must never read as the message not existing")
	require.NotEmpty(t, sender.sent,
		"the provider was already called before this write failed — this is why the failure must not be misread as the message never having existed")
}

func TestSendLogsTheProviderAnswerWhenTheOutcomeCannotBeRecorded(t *testing.T) {
	t.Parallel()

	var logs bytes.Buffer
	sender := &fakeSender{err: &sms.RejectedError{StatusCode: 422, RawBody: "no (correct) recipients found"}}
	messages := newFakeMessageRepo()
	messages.outcomeErr = domain.ErrNotFound
	sender.repo = messages
	svc := NewMessages(
		newFakeCustomerRepo(aCustomer()),
		newFakeTemplateRepo(aTemplate()),
		messages,
		sender,
		slog.New(slog.NewJSONHandler(&logs, nil)),
	)

	_, err := svc.Send(context.Background(), "4000000001", domain.TemplateTypeReminder)

	require.ErrorIs(t, err, ErrAttemptCannotBeRecorded)
	require.Contains(t, logs.String(), "no (correct) recipients found",
		"once the outcome write fails, the log is the only place the provider's refusal survives")
}

func TestListRefusesAnUnknownCreditNumber(t *testing.T) {
	t.Parallel()

	svc, _ := newMessages(t, &fakeSender{})

	_, err := svc.List(context.Background(), "9000000001")

	require.ErrorIs(t, err, domain.ErrCustomerNotFound,
		"an unknown loan answers not found rather than an empty history")
}
