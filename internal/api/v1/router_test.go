package v1

import (
	"io"
	"log/slog"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

func testHandler(t *testing.T) http.Handler {
	t.Helper()

	return NewServer(nil, nil, nil, slog.New(slog.NewTextHandler(io.Discard, nil))).Handler()
}

func TestRouterMatchesEveryEndpoint(t *testing.T) {
	t.Parallel()

	routes, ok := testHandler(t).(chi.Routes)
	require.True(t, ok, "Handler() must expose the chi routing table")

	tests := []struct {
		method string
		path   string
		params map[string]string
	}{
		{http.MethodGet, "/customers", nil},
		{http.MethodPost, "/customers", nil},
		{http.MethodGet, "/customers/3000000001", map[string]string{"credit_number": "3000000001"}},
		{http.MethodPut, "/customers/3000000001", map[string]string{"credit_number": "3000000001"}},
		{http.MethodGet, "/customers/3000000001/messages", map[string]string{"credit_number": "3000000001"}},
		{http.MethodPost, "/customers/3000000001/messages", map[string]string{"credit_number": "3000000001"}},
		{http.MethodGet, "/templates", nil},
		{http.MethodGet, "/templates/reminder", map[string]string{"type": "reminder"}},
		{http.MethodPut, "/templates/reminder", map[string]string{"type": "reminder"}},
		{http.MethodGet, "/swagger/index.html", nil},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			t.Parallel()

			rctx := chi.NewRouteContext()
			require.True(t, routes.Match(rctx, tt.method, tt.path), "no route for this request")

			for name, want := range tt.params {
				require.Equal(t, want, rctx.URLParam(name), "path parameter %q", name)
			}
		})
	}
}
