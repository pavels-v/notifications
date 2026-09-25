//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"notifications/internal/domain"
)

func TestListTemplatesReturnsTheSeededThree(t *testing.T) {
	s := newServer(t)

	rec := doJSON(t, s.srv, http.MethodGet, "/v1/templates", nil)
	require.Equal(t, http.StatusOK, rec.Code)

	var got []domain.Template
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))

	require.Len(t, got, 3)

	types := make([]string, 0, len(got))
	for _, tpl := range got {
		types = append(types, tpl.Type)
		require.NotEmpty(t, tpl.Body, "template %q has an empty body", tpl.Type)
	}
	require.ElementsMatch(t, []string{domain.TemplateTypeReminder, domain.TemplateTypeDunning, domain.TemplateTypeTermination}, types)
}

func TestUpsertTemplate(t *testing.T) {
	s := newServer(t)

	const body = "Dear {full_name}, {amount} is due on {due_date}."

	rec := doJSON(t, s.srv, http.MethodPut, "/v1/templates/reminder", map[string]string{"body": body})
	require.Equal(t, http.StatusOK, rec.Code, "body = %s", rec.Body)

	var got domain.Template
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Equal(t, body, got.Body)

	rec = doJSON(t, s.srv, http.MethodGet, "/v1/templates/reminder", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Equal(t, body, got.Body, "the change must be persisted, not just echoed")
}

func TestTemplateRequestValidation(t *testing.T) {
	cases := []struct {
		name       string
		method     string
		path       string
		body       any
		wantStatus int
	}{
		{
			name:       "a type outside the three the brief names",
			method:     http.MethodPut,
			path:       "/v1/templates/invoice",
			body:       map[string]string{"body": "text"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "an empty body",
			method:     http.MethodPut,
			path:       "/v1/templates/reminder",
			body:       map[string]string{"body": ""},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "the body field missing entirely",
			method:     http.MethodPut,
			path:       "/v1/templates/reminder",
			body:       map[string]string{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "reading a type that does not exist",
			method:     http.MethodGet,
			path:       "/v1/templates/invoice",
			body:       nil,
			wantStatus: http.StatusNotFound,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := newServer(t)

			rec := doJSON(t, s.srv, c.method, c.path, c.body)
			require.Equal(t, c.wantStatus, rec.Code, "body = %s", rec.Body)
		})
	}
}

func TestEditingATemplateLeavesHistoryIntact(t *testing.T) {
	s := newServer(t)
	seedCustomer(t, s.db)

	rec := postJSON(t, s.srv, "/v1/customers/3000000001/messages", map[string]string{"template_type": domain.TemplateTypeReminder})
	require.Equal(t, http.StatusCreated, rec.Code, "body = %s", rec.Body)

	rec = doJSON(t, s.srv, http.MethodPut, "/v1/templates/reminder",
		map[string]string{"body": "Completely different wording."})
	require.Equal(t, http.StatusOK, rec.Code, "body = %s", rec.Body)

	req := httptest.NewRequest(http.MethodGet, "/v1/customers/3000000001/messages", http.NoBody)
	recorder := httptest.NewRecorder()
	s.srv.Handler().ServeHTTP(recorder, req)
	require.Equal(t, http.StatusOK, recorder.Code)

	var history []domain.Message
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &history))
	require.Len(t, history, 1)
	require.Contains(t, history[0].Body, "John Doe", "the sent text must survive the template it came from")
	require.NotContains(t, history[0].Body, "Completely different wording",
		"history must not follow the template it was rendered from")
}
