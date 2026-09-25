package tmpl

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"notifications/internal/domain"
)

func customer() domain.Customer {
	return domain.Customer{
		CreditNumber: "1000000001",
		FullName:     "John Doe",
		AmountMinor:  125000,
		Currency:     "EUR",
		DueDate:      time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC),
	}
}

func TestRender(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{
			name: "the seeded reminder template",
			body: "{credit_number}\n\nDear {full_name},\n\nplease repay the amount of {amount} by {due_date}.",
			want: "1000000001\n\nDear John Doe,\n\nplease repay the amount of 1250.00 EUR by 2026-09-25.",
		},
		{name: "credit number alone", body: "{credit_number}", want: "1000000001"},
		{name: "name alone", body: "{full_name}", want: "John Doe"},
		{name: "amount is formatted, not raw minor units", body: "{amount}", want: "1250.00 EUR"},
		{name: "due date is ISO 8601", body: "{due_date}", want: "2026-09-25"},
		{name: "the same placeholder twice", body: "{full_name} and {full_name}", want: "John Doe and John Doe"},
		{name: "no placeholders at all", body: "plain text", want: "plain text"},
		{name: "an empty body", body: "", want: ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := Render(c.body, customer())
			require.NoError(t, err)
			require.Equal(t, c.want, got)
		})
	}
}

func TestRenderRejectsUnmatchedNames(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		wantName string // the placeholder the error must name
	}{
		{name: "capitalised", body: "{Amount}", wantName: "Amount"},
		{name: "shouted", body: "{AMOUNT}", wantName: "AMOUNT"},
		{name: "camel case", body: "{creditNumber}", wantName: "creditNumber"},
		{name: "the spelling used in the brief", body: "{Creditnumber}", wantName: "Creditnumber"},
		{name: "camel case date from the brief", body: "{dueDate}", wantName: "dueDate"},
		{name: "mixed case with underscore", body: "{Due_Date}", wantName: "Due_Date"},
		{name: "a field that does not exist", body: "Dear {full_name}, your {iban} is overdue", wantName: "iban"},
		{name: "the storage column rather than the rendered field", body: "{amount_minor}", wantName: "amount_minor"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := Render(c.body, customer())

			require.ErrorIs(t, err, domain.ErrUnknownPlaceholder)
			require.Contains(t, err.Error(), c.wantName,
				"the error must name the offending placeholder so the template can be fixed")
			require.Empty(t, got, "nothing may be returned for a body that failed to render")
		})
	}
}

func TestRenderReadsTheCustomerItIsGiven(t *testing.T) {
	t.Parallel()

	got, err := Render("{credit_number} {full_name} {amount} {due_date}", domain.Customer{
		CreditNumber: "1000000001",
		FullName:     "John Doe",
		AmountMinor:  125000,
		Currency:     "EUR",
		DueDate:      time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC),
	})

	require.NoError(t, err)
	require.Equal(t, "1000000001 John Doe 1250.00 EUR 2026-09-25", got)
}

func TestFormatAmount(t *testing.T) {
	cases := []struct {
		name     string
		minor    int64
		currency string
		want     string
	}{
		{name: "whole amount", minor: 125000, currency: "EUR", want: "1250.00 EUR"},
		{name: "with cents", minor: 48050, currency: "EUR", want: "480.50 EUR"},
		{name: "under one unit", minor: 5, currency: "EUR", want: "0.05 EUR"},
		{name: "zero", minor: 0, currency: "EUR", want: "0.00 EUR"},
		{name: "another currency", minor: 100, currency: "USD", want: "1.00 USD"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.want, FormatAmount(c.minor, c.currency))
		})
	}
}
