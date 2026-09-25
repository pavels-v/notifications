package api

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	v1 "notifications/internal/api/v1"
)

func testHandler(t *testing.T) http.Handler {
	t.Helper()

	log := slog.New(slog.DiscardHandler)

	return NewServer(v1.NewServer(nil, nil, nil, log), log).Handler()
}

func TestRouterServesEveryVersionUnderItsPrefix(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	testHandler(t).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/swagger/doc.json", http.NoBody))

	require.Equal(t, http.StatusOK, rec.Code, "the v1 router must answer under /v1")

	var spec struct {
		BasePath string `json:"basePath"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&spec), "the body must be the generated spec")
	require.Equal(t, "/v1", spec.BasePath, "the spec must describe the paths the version is served at")
}

func TestRouterRefusesInJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		method string
		path   string
		status int
		code   string
	}{
		{"unknown path", http.MethodGet, "/nope", http.StatusNotFound, "not_found"},
		{"unversioned path", http.MethodGet, "/customers", http.StatusNotFound, "not_found"},
		{"unknown version", http.MethodGet, "/v2/customers", http.StatusNotFound, "not_found"},
		{"unknown nested path", http.MethodGet, "/v1/customers/3000000001/invoices", http.StatusNotFound, "not_found"},
		{"wrong method", http.MethodDelete, "/v1/customers", http.StatusMethodNotAllowed, "method_not_allowed"},
		{"wrong method on a sub-route", http.MethodDelete, "/v1/customers/3000000001/messages", http.StatusMethodNotAllowed, "method_not_allowed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rec := httptest.NewRecorder()
			testHandler(t).ServeHTTP(rec, httptest.NewRequest(tt.method, tt.path, http.NoBody))

			require.Equal(t, tt.status, rec.Code)
			require.Equal(t, "application/json", rec.Header().Get("Content-Type"))

			var body errorBody
			require.NoError(t, json.NewDecoder(rec.Body).Decode(&body), "the body must be our error object")
			require.Equal(t, tt.code, body.Error)
			require.NotEmpty(t, body.Message)
		})
	}
}

func testHandlerLogging(t *testing.T) (http.Handler, *bytes.Buffer) {
	t.Helper()

	var logs bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&logs, nil))

	return NewServer(v1.NewServer(nil, nil, nil, log), log).Handler(), &logs
}

func TestRouterRecoversFromAPanic(t *testing.T) {
	t.Parallel()

	h, logs := testHandlerLogging(t)
	rec := httptest.NewRecorder()

	require.NotPanics(t, func() {
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/customers", http.NoBody))
	})

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var body errorBody
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body), "a panic must still answer in our error shape")
	require.Equal(t, "internal", body.Error)

	require.Contains(t, logs.String(), `"msg":"failed to serve request after panic"`, "the panic must reach slog, not stderr")
	require.Contains(t, logs.String(), `"level":"ERROR"`)
	require.Contains(t, logs.String(), `"status":500`, "the access line must record the recovered status")
}

func TestRouterLogsAndEchoesEveryRequest(t *testing.T) {
	t.Parallel()

	h, logs := testHandlerLogging(t)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/nope", http.NoBody))

	reqID := rec.Header().Get("X-Request-Id")
	require.NotEmpty(t, reqID, "the request id must come back to the caller")

	line := logs.String()
	require.Contains(t, line, `"msg":"request"`)
	require.Contains(t, line, `"method":"GET"`)
	require.Contains(t, line, `"path":"/nope"`)
	require.Contains(t, line, `"status":404`)
	require.Contains(t, line, reqID, "the access line must carry the id the caller was given")
}
