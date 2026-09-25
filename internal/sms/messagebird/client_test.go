package messagebird

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"notifications/internal/config"
	"notifications/internal/sms"
)

func clientFor(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()

	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	return New(config.MessageBird{
		AccessKey:  "test-key",
		Originator: "ACME",
		BaseURL:    srv.URL,
		Timeout:    2 * time.Second,
	}, slog.New(slog.DiscardHandler))
}

func TestSendBuildsTheProviderRequest(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
		gotAuth   string
		gotBody   struct {
			Originator string   `json:"originator"`
			Recipients []string `json:"recipients"`
			Body       string   `json:"body"`
		}
	)

	c := clientFor(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"mb-123","recipients":{"items":[{"status":"sent"}]}}`))
	})

	_, err := c.Send(context.Background(), sms.Message{To: "+999000000001", Body: "hello"})
	require.NoError(t, err)

	require.Equal(t, http.MethodPost, gotMethod)
	require.Equal(t, "/messages", gotPath)
	require.Equal(t, "AccessKey test-key", gotAuth, "the access key must travel in the Authorization header")
	require.Equal(t, "ACME", gotBody.Originator, "the configured originator must be used")
	require.Equal(t, []string{"+999000000001"}, gotBody.Recipients)
	require.Equal(t, "hello", gotBody.Body)
}

func TestSendResponseHandling(t *testing.T) {
	cases := []struct {
		name         string
		status       int
		payload      string
		wantErr      bool
		wantRejected bool
		wantID       string
		wantRawBody  string
	}{
		{
			name:    "accepted",
			status:  http.StatusCreated,
			payload: `{"id":"mb-123","recipients":{"items":[{"status":"sent"}]}}`,
			wantID:  "mb-123",
		},
		{
			name:    "accepted but the recipient list is empty",
			status:  http.StatusCreated,
			payload: `{"id":"mb-124","recipients":{"items":[]}}`,
			wantID:  "mb-124",
		},
		{
			name:         "bad access key",
			status:       http.StatusUnauthorized,
			payload:      `{"errors":[{"code":2,"description":"Request not allowed (incorrect access_key)"}]}`,
			wantErr:      true,
			wantRejected: true,
			wantRawBody:  `{"errors":[{"code":2,"description":"Request not allowed (incorrect access_key)"}]}`,
		},
		{
			name:         "provider is down",
			status:       http.StatusInternalServerError,
			payload:      `upstream unavailable`,
			wantErr:      true,
			wantRejected: true,
			wantRawBody:  "upstream unavailable",
		},
		{
			name:    "success status with a body that is not JSON",
			status:  http.StatusCreated,
			payload: `<html>gateway</html>`,
			wantErr: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			client := clientFor(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(c.status)
				_, _ = w.Write([]byte(c.payload))
			})

			resp, err := client.Send(context.Background(), sms.Message{To: "+999000000001", Body: "hello"})

			if !c.wantErr {
				require.NoError(t, err)
				require.Equal(t, c.wantID, resp.ProviderMessageID)
				require.Equal(t, c.payload, resp.RawBody, "the untouched payload must be kept for storage")
				return
			}

			require.Error(t, err)
			if !c.wantRejected {
				var rejected *sms.RejectedError
				require.NotErrorIs(t, err, sms.ErrRejected,
					"an error with no reply from the provider must not read as a refusal")
				require.NotErrorAs(t, err, &rejected)
				return
			}

			var rejected *sms.RejectedError
			require.ErrorAs(t, err, &rejected, "a non-2xx reply must surface as *sms.RejectedError")
			require.ErrorIs(t, err, sms.ErrRejected, "the api layer switches on this sentinel")
			require.Equal(t, c.status, rejected.StatusCode)
			require.Equal(t, c.wantRawBody, rejected.RawBody,
				"the provider's own words must reach the operator")
		})
	}
}

func TestSendRespectsContextCancellation(t *testing.T) {
	release := make(chan struct{})

	c := clientFor(t, func(_ http.ResponseWriter, _ *http.Request) {
		<-release
	})

	t.Cleanup(func() { close(release) })

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := c.Send(ctx, sms.Message{To: "+999000000001", Body: "hello"})
	require.Error(t, err, "a hung provider must not hang the request")
}

var _ sms.Sender = (*Client)(nil)
