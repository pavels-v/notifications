//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"notifications/internal/domain"
)

type errorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func sendBody(t *testing.T, s stand, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var payload []byte
	switch b := body.(type) {
	case nil:
	case string:
		payload = []byte(b)
	default:
		marshalled, err := json.Marshal(b)
		require.NoError(t, err, "bad literal in the test itself")
		payload = marshalled
	}

	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.srv.Handler().ServeHTTP(rec, req)
	return rec
}

func TestErrorContract(t *testing.T) {
	brokenReminder := func(t *testing.T, s stand) {
		t.Helper()
		seedCustomer(t, s.db)
		_, err := s.templates.Upsert(context.Background(), domain.TemplateTypeReminder, "Dear {full_name}, your {iban} is due")
		require.NoError(t, err)
	}

	takeTheCreditNumber := func(t *testing.T, s stand) {
		t.Helper()
		rec := sendBody(t, s, http.MethodPost, "/v1/customers", validCustomer())
		require.Equal(t, http.StatusCreated, rec.Code, "the arrange step itself must succeed, body = %s", rec.Body)
	}

	withCustomer := func(t *testing.T, s stand) {
		t.Helper()
		seedCustomer(t, s.db)
	}

	reminder := map[string]string{"template_type": domain.TemplateTypeReminder}

	cases := []struct {
		name      string
		arrange   func(t *testing.T, s stand)
		method    string
		path      string
		body      any
		status    int
		errorCode string
	}{
		{
			name:      "body is not json",
			method:    http.MethodPost,
			path:      "/v1/customers",
			body:      "{",
			status:    http.StatusBadRequest,
			errorCode: "invalid_json",
		},
		{
			name:      "loan breaks a validation rule",
			method:    http.MethodPost,
			path:      "/v1/customers",
			body:      map[string]any{},
			status:    http.StatusBadRequest,
			errorCode: "validation_failed",
		},
		{
			name:      "credit number is already taken",
			arrange:   takeTheCreditNumber,
			method:    http.MethodPost,
			path:      "/v1/customers",
			body:      validCustomer(),
			status:    http.StatusConflict,
			errorCode: "customer_already_exists",
		},
		{
			name:      "reading a loan that does not exist",
			method:    http.MethodGet,
			path:      "/v1/customers/9000000001",
			status:    http.StatusNotFound,
			errorCode: "customer_not_found",
		},
		{
			name:      "replacing a loan that does not exist",
			method:    http.MethodPut,
			path:      "/v1/customers/9000000001",
			body:      validCustomer(),
			status:    http.StatusNotFound,
			errorCode: "customer_not_found",
		},
		{
			name:      "replacing a loan with a body that breaks a rule",
			arrange:   withCustomer,
			method:    http.MethodPut,
			path:      "/v1/customers/3000000001",
			body:      map[string]any{},
			status:    http.StatusBadRequest,
			errorCode: "validation_failed",
		},
		{
			name:      "reading a template type that does not exist",
			method:    http.MethodGet,
			path:      "/v1/templates/invoice",
			status:    http.StatusNotFound,
			errorCode: "template_not_found",
		},
		{
			name:      "writing a template type outside the three",
			method:    http.MethodPut,
			path:      "/v1/templates/invoice",
			body:      map[string]string{"body": "text"},
			status:    http.StatusBadRequest,
			errorCode: "invalid_type",
		},
		{
			name:      "writing a template with no body field",
			method:    http.MethodPut,
			path:      "/v1/templates/reminder",
			body:      map[string]any{},
			status:    http.StatusBadRequest,
			errorCode: "missing_body",
		},
		{
			name:      "writing a template with a body that is not json",
			method:    http.MethodPut,
			path:      "/v1/templates/reminder",
			body:      "{",
			status:    http.StatusBadRequest,
			errorCode: "invalid_json",
		},
		{
			name:      "sending without naming a template type",
			arrange:   withCustomer,
			method:    http.MethodPost,
			path:      "/v1/customers/3000000001/messages",
			body:      map[string]any{},
			status:    http.StatusBadRequest,
			errorCode: "missing_template",
		},
		{
			name:      "sending to a loan that does not exist",
			method:    http.MethodPost,
			path:      "/v1/customers/9000000001/messages",
			body:      reminder,
			status:    http.StatusNotFound,
			errorCode: "customer_not_found",
		},
		{
			name:      "sending a template type that does not exist",
			arrange:   withCustomer,
			method:    http.MethodPost,
			path:      "/v1/customers/3000000001/messages",
			body:      map[string]string{"template_type": "nope"},
			status:    http.StatusUnprocessableEntity,
			errorCode: "template_unusable",
		},
		{
			name:      "sending a template that names a field that does not exist",
			arrange:   brokenReminder,
			method:    http.MethodPost,
			path:      "/v1/customers/3000000001/messages",
			body:      reminder,
			status:    http.StatusUnprocessableEntity,
			errorCode: "unknown_placeholder",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := newServer(t)
			if c.arrange != nil {
				c.arrange(t, s)
			}

			rec := sendBody(t, s, c.method, c.path, c.body)

			require.Equal(t, c.status, rec.Code, "body = %s", rec.Body)
			require.Equal(t, "application/json", rec.Header().Get("Content-Type"))

			var got errorResponse
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got), "a refusal must be our error object")
			require.Equal(t, c.errorCode, got.Error, "this is the code a caller branches on")
			require.NotEmpty(t, got.Message, "a code with no sentence beside it tells the caller nothing")
		})
	}
}
