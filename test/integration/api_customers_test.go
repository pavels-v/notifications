//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"notifications/internal/domain"
)

func validCustomer() map[string]any {
	return map[string]any{
		"credit_number": "4000000001",
		"phone_number":  "+999000000001",
		"full_name":     "Jane Doe",
		"amount_minor":  90000,
		"currency":      "EUR",
		"due_date":      "2026-10-01",
	}
}

func TestCreateAndReadCustomer(t *testing.T) {
	s := newServer(t)

	rec := doJSON(t, s.srv, http.MethodPost, "/v1/customers", validCustomer())
	require.Equal(t, http.StatusCreated, rec.Code, "body = %s", rec.Body)

	rec = doJSON(t, s.srv, http.MethodGet, "/v1/customers/4000000001", nil)
	require.Equal(t, http.StatusOK, rec.Code)

	var got domain.Customer
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))

	require.Equal(t, "4000000001", got.CreditNumber)
	require.Equal(t, "Jane Doe", got.FullName)
	require.Equal(t, int64(90000), got.AmountMinor)
	require.Equal(t, "EUR", got.Currency)
}

func TestCreateCustomerValidation(t *testing.T) {
	cases := []struct {
		name  string
		field string
		value any // nil means: remove the field entirely
	}{
		{name: "credit number missing", field: "credit_number", value: nil},
		{name: "credit number empty", field: "credit_number", value: ""},
		{name: "full name missing", field: "full_name", value: nil},
		{name: "full name empty", field: "full_name", value: ""},
		{name: "phone is not E.164", field: "phone_number", value: "0170 1234567"},
		{name: "phone missing the plus", field: "phone_number", value: "491701234567"},
		{name: "currency too long", field: "currency", value: "EURO"},
		{name: "currency missing", field: "currency", value: nil},
		{name: "due date in dotted day-first format", field: "due_date", value: "01.10.2026"},
		{name: "due date missing", field: "due_date", value: nil},
		{name: "amount negative", field: "amount_minor", value: -1},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := newServer(t)

			body := validCustomer()
			if c.value == nil {
				delete(body, c.field)
			} else {
				body[c.field] = c.value
			}

			rec := doJSON(t, s.srv, http.MethodPost, "/v1/customers", body)
			require.Equal(t, http.StatusBadRequest, rec.Code, "body = %s", rec.Body)
		})
	}
}

func TestListCustomersIncludesTheCreatedOne(t *testing.T) {
	s := newServer(t)

	rec := doJSON(t, s.srv, http.MethodPost, "/v1/customers", validCustomer())
	require.Equal(t, http.StatusCreated, rec.Code, "body = %s", rec.Body)

	rec = doJSON(t, s.srv, http.MethodGet, "/v1/customers", nil)
	require.Equal(t, http.StatusOK, rec.Code)

	var all []domain.Customer
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &all))

	numbers := make([]string, 0, len(all))
	for _, c := range all {
		numbers = append(numbers, c.CreditNumber)
	}
	require.Contains(t, numbers, "4000000001")
}

func TestAPIUpdateCustomer(t *testing.T) {
	s := newServer(t)
	seedCustomer(t, s.db)

	rec := doJSON(t, s.srv, http.MethodPut, "/v1/customers/3000000001", map[string]any{
		"phone_number": "+999000000099",
		"full_name":    "John Doe",
		"amount_minor": 1000,
		"currency":     "EUR",
		"due_date":     "2026-11-01",
	})
	require.Equal(t, http.StatusOK, rec.Code, "body = %s", rec.Body)

	var got domain.Customer
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))

	require.Equal(t, "+999000000099", got.PhoneNumber)
	require.Equal(t, int64(1000), got.AmountMinor)
}

func TestCustomerNotFoundMapping(t *testing.T) {
	cases := []struct {
		name   string
		method string
		body   any
	}{
		{name: "read", method: http.MethodGet, body: nil},
		{name: "update", method: http.MethodPut, body: validCustomer()},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := newServer(t)

			rec := doJSON(t, s.srv, c.method, "/v1/customers/9000000001", c.body)
			require.Equal(t, http.StatusNotFound, rec.Code, "body = %s", rec.Body)
		})
	}
}
