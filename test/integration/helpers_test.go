//go:build integration

package integration

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"

	"notifications/internal/api"
	v1 "notifications/internal/api/v1"
	"notifications/internal/config"
	"notifications/internal/domain"
	"notifications/internal/service"
	"notifications/internal/sms/messagebird"
	"notifications/internal/store"
	"notifications/migrations/seed"
	fake "notifications/test/fake/messagebird"
)

const defaultTestDatabaseURL = "postgres://notifications:notifications@localhost:5432/notifications?sslmode=disable"

func testDatabaseURL() string {
	if url := os.Getenv("TEST_DATABASE_URL"); url != "" {
		return url
	}
	return defaultTestDatabaseURL
}

func testDatabaseConfig(t *testing.T) config.Database {
	t.Helper()

	conn, err := pgx.ParseConfig(testDatabaseURL())
	require.NoError(t, err, "TEST_DATABASE_URL is not a valid connection string")

	sslMode := "disable"
	if conn.TLSConfig != nil {
		sslMode = "require"
	}

	return config.Database{
		Host:           conn.Host,
		Port:           conn.Port,
		User:           conn.User,
		Password:       conn.Password,
		Name:           conn.Database,
		SSLMode:        sslMode,
		ConnectTimeout: conn.ConnectTimeout,
	}
}

func newStore(t *testing.T) *store.DB {
	t.Helper()

	url := testDatabaseURL()
	ctx := context.Background()

	st, err := store.New(ctx, testDatabaseConfig(t))
	require.NoError(t, err, "could not open a pool against %s", url)
	t.Cleanup(st.Close)

	require.NoError(t, st.Migrate(ctx),
		"could not migrate the test database at %s; start it with `make db-up` or point TEST_DATABASE_URL at another one", url)
	reset(t, url)

	return st
}

func reset(t *testing.T, url string) {
	t.Helper()

	db, err := sql.Open("pgx", url)
	require.NoError(t, err, "could not open a connection for the reset")
	defer func() { _ = db.Close() }()

	ctx := context.Background()

	_, err = db.ExecContext(ctx, `TRUNCATE messages, customers, templates RESTART IDENTITY CASCADE`)
	require.NoError(t, err, "could not empty the test database")

	_, err = db.ExecContext(ctx, seed.SQL)
	require.NoError(t, err, "could not reload the seed")
}

const testAccessKey = "test-access-key"

type stand struct {
	srv       *api.Server
	provider  *fake.Fake
	db        *store.DB
	templates *store.Templates
	messages  *store.Messages
}

func newServer(t *testing.T) stand {
	t.Helper()

	st := newStore(t)

	provider := fake.New(testAccessKey)
	providerSrv := httptest.NewServer(provider.Handler())
	t.Cleanup(providerSrv.Close)

	log := slog.New(slog.DiscardHandler)

	sender := messagebird.New(config.MessageBird{
		AccessKey:  testAccessKey,
		Originator: "ACME",
		BaseURL:    providerSrv.URL,
		Timeout:    5 * time.Second,
	}, log)

	customerRepo := store.NewCustomers(st.DB())
	templateRepo := store.NewTemplates(st.DB())
	messageRepo := store.NewMessages(st.DB())

	customers := service.NewCustomers(customerRepo)
	templates := service.NewTemplates(templateRepo)
	messages := service.NewMessages(customerRepo, templateRepo, messageRepo, sender, log)

	return stand{
		srv:       api.NewServer(v1.NewServer(customers, templates, messages, log), log),
		provider:  provider,
		db:        st,
		templates: templateRepo,
		messages:  messageRepo,
	}
}

func mustDate(t *testing.T, s string) time.Time {
	t.Helper()

	d, err := time.Parse("2006-01-02", s)
	require.NoError(t, err, "bad date literal in the test itself: %q", s)
	return d
}

var sampleDueDate = time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)

func sampleCustomer() domain.Customer {
	return domain.Customer{
		CreditNumber: "2000000001",
		PhoneNumber:  "+999000000011",
		FullName:     "John Doe",
		AmountMinor:  125000,
		Currency:     "EUR",
		DueDate:      sampleDueDate,
	}
}

func seedCustomer(t *testing.T, st *store.DB) {
	t.Helper()

	_, err := store.NewCustomers(st.DB()).Create(context.Background(), domain.Customer{
		CreditNumber: "3000000001",
		PhoneNumber:  "+999000000001",
		FullName:     "John Doe",
		AmountMinor:  125000,
		Currency:     "EUR",
		DueDate:      mustDate(t, "2026-09-25"),
	})
	require.NoError(t, err)
}

func postJSON(t *testing.T, srv *api.Server, path string, body any) *httptest.ResponseRecorder {
	t.Helper()

	payload, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	return rec
}

func doJSON(t *testing.T, srv *api.Server, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		payload, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(payload)
	}

	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	return rec
}
