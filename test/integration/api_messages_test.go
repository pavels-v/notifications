//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"notifications/internal/domain"
)

func TestSendMessageStoresAndReturnsTheResponse(t *testing.T) {
	s := newServer(t)
	seedCustomer(t, s.db)

	rec := postJSON(t, s.srv, "/v1/customers/3000000001/messages", map[string]string{"template_type": domain.TemplateTypeReminder})
	require.Equal(t, http.StatusCreated, rec.Code, "body = %s", rec.Body)

	var got domain.Message
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))

	require.Equal(t, domain.StatusSentToProvider, got.Status,
		"an accepted send is recorded as reaching the provider, not as delivered")
	require.NotEmpty(t, got.ProviderMessageID, "the provider identifier must be returned to the caller")
	require.Empty(t, got.Error)
	require.Contains(t, got.Body, "John Doe", "the name must be substituted")
	require.Contains(t, got.Body, "1250.00 EUR", "the amount must be substituted and formatted")
	require.Contains(t, got.Body, "2026-09-25", "the due date must be substituted as ISO 8601")

	sent := s.provider.Requests()
	require.Len(t, sent, 1, "exactly one message must reach the provider")
	require.Equal(t, []string{"+999000000001"}, sent[0].Recipients, "the customer's stored number is the recipient")
	require.Equal(t, "ACME", sent[0].Originator, "the configured originator must be used")
	require.Equal(t, got.Body, sent[0].Body, "what was stored must be what was sent")

	history, err := s.messages.List(context.Background(), "3000000001")
	require.NoError(t, err)
	require.Len(t, history, 1, "the send must be recorded against the loan")

	var raw struct {
		ID         string `json:"id"`
		Recipients struct {
			Items []struct {
				Status string `json:"status"`
			} `json:"items"`
		} `json:"recipients"`
	}
	require.NoError(t, json.Unmarshal([]byte(history[0].ResponseRaw), &raw), "response_raw must hold the provider's JSON")
	require.Equal(t, got.ProviderMessageID, raw.ID, "the stored response must be the one the identifier came from")
	require.Len(t, raw.Recipients.Items, 1)
	require.Equal(t, "sent", raw.Recipients.Items[0].Status)
}

func TestSendMessageRejections(t *testing.T) {
	cases := []struct {
		name       string
		arrange    func(t *testing.T, s stand) // optional, in addition to the standard customer
		path       string
		body       any
		wantStatus int
	}{
		{
			name:       "credit number that does not exist",
			path:       "/v1/customers/9000000001/messages",
			body:       map[string]string{"template_type": domain.TemplateTypeReminder},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "template type that does not exist",
			path:       "/v1/customers/3000000001/messages",
			body:       map[string]string{"template_type": "nope"},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "template field missing",
			path:       "/v1/customers/3000000001/messages",
			body:       map[string]string{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "template field empty",
			path:       "/v1/customers/3000000001/messages",
			body:       map[string]string{"template_type": ""},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "template contains a placeholder that cannot be resolved",
			arrange: func(t *testing.T, s stand) {
				_, err := s.templates.Upsert(context.Background(), domain.TemplateTypeReminder, "Dear {full_name}, your {iban} is due")
				require.NoError(t, err)
			},
			path:       "/v1/customers/3000000001/messages",
			body:       map[string]string{"template_type": domain.TemplateTypeReminder},
			wantStatus: http.StatusUnprocessableEntity,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := newServer(t)
			seedCustomer(t, s.db)
			if c.arrange != nil {
				c.arrange(t, s)
			}

			rec := postJSON(t, s.srv, c.path, c.body)

			require.Equal(t, c.wantStatus, rec.Code, "body = %s", rec.Body)
			require.Empty(t, s.provider.Requests(), "a refused request must never reach the provider — an SMS cannot be recalled")
		})
	}
}

func TestSendMessageRecordsProviderRefusal(t *testing.T) {
	s := newServer(t)
	seedCustomer(t, s.db)
	s.provider.FailWith(http.StatusServiceUnavailable, 9, "Service temporarily unavailable")

	rec := postJSON(t, s.srv, "/v1/customers/3000000001/messages", map[string]string{"template_type": domain.TemplateTypeReminder})
	require.Equal(t, http.StatusBadGateway, rec.Code, "body = %s", rec.Body)

	history, err := s.messages.List(context.Background(), "3000000001")
	require.NoError(t, err)
	require.Len(t, history, 1, "a refused attempt must still be recorded, not silently dropped")

	require.Equal(t, domain.StatusRejectedByProvider, history[0].Status,
		"an answer from the provider is proof that nothing was sent")
	require.Empty(t, history[0].ProviderMessageID, "a refused message never got an identifier")
	require.Contains(t, history[0].ResponseRaw, "Service temporarily unavailable",
		"the operator must find the provider's own words in the table")
	require.Contains(t, history[0].Error, "503", "and the status it refused with")
}

func TestSendMessageLeavesTheOutcomeUnknownWhenTheProviderNeverAnswers(t *testing.T) {
	s := newServer(t)
	seedCustomer(t, s.db)
	s.provider.DropConnections()

	rec := postJSON(t, s.srv, "/v1/customers/3000000001/messages", map[string]string{"template_type": domain.TemplateTypeReminder})
	require.Equal(t, http.StatusBadGateway, rec.Code, "body = %s", rec.Body)

	require.Len(t, s.provider.Requests(), 1, "the message did reach the provider, which is what makes it ambiguous")

	history, err := s.messages.List(context.Background(), "3000000001")
	require.NoError(t, err)
	require.Len(t, history, 1, "the row written before the call is what survives an unanswered send")

	require.Equal(t, domain.StatusPending, history[0].Status,
		"an unanswered send may still have gone out, so it must not be recorded as refused")
	require.Empty(t, history[0].ResponseRaw, "there was no response to keep")
	require.NotEmpty(t, history[0].Error, "the operator needs the reason the outcome is unknown")
}

func TestMessageHistoryIsNewestFirst(t *testing.T) {
	s := newServer(t)
	seedCustomer(t, s.db)

	for _, tpl := range []string{domain.TemplateTypeReminder, domain.TemplateTypeDunning} {
		rec := postJSON(t, s.srv, "/v1/customers/3000000001/messages", map[string]string{"template_type": tpl})
		require.Equal(t, http.StatusCreated, rec.Code, "body = %s", rec.Body)
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/customers/3000000001/messages", http.NoBody)
	rec := httptest.NewRecorder()
	s.srv.Handler().ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var history []domain.Message
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &history))

	require.Len(t, history, 2)
	require.Equal(t, domain.TemplateTypeDunning, history[0].TemplateType, "the newest message comes first")
	require.Equal(t, domain.TemplateTypeReminder, history[1].TemplateType)
}
