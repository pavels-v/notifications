package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func validCustomer() Customer {
	return Customer{
		CreditNumber: "4000000001",
		PhoneNumber:  "+999000000001",
		FullName:     "Jane Doe",
		AmountMinor:  90000,
		Currency:     "EUR",
		DueDate:      time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
	}
}

func TestCustomerValidateAcceptsAWholeCustomer(t *testing.T) {
	t.Parallel()

	require.NoError(t, validCustomer().Validate())
}

func TestCustomerValidateRejects(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		corrupt func(c *Customer)
		field   string
	}{
		{"credit number empty", func(c *Customer) { c.CreditNumber = "" }, "credit_number"},
		{"full name empty", func(c *Customer) { c.FullName = "" }, "full_name"},
		{"phone is not E.164", func(c *Customer) { c.PhoneNumber = "0170 1234567" }, "phone_number"},
		{"phone missing the plus", func(c *Customer) { c.PhoneNumber = "491701234567" }, "phone_number"},
		{"phone starting with zero", func(c *Customer) { c.PhoneNumber = "+049170123456" }, "phone_number"},
		{"phone too short", func(c *Customer) { c.PhoneNumber = "+1234567" }, "phone_number"},
		{"currency too long", func(c *Customer) { c.Currency = "EURO" }, "currency"},
		{"currency empty", func(c *Customer) { c.Currency = "" }, "currency"},
		{"amount negative", func(c *Customer) { c.AmountMinor = -1 }, "amount_minor"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := validCustomer()
			tt.corrupt(&c)

			err := c.Validate()
			require.Error(t, err)
			require.ErrorIs(t, err, ErrValidation, "the API maps this sentinel to 400")
			require.Contains(t, err.Error(), tt.field, "the message must name the field it refuses")
			require.NotContains(t, err.Error(), "validation",
				"the sentinel must not leak into the text: it is written verbatim into the response body")
		})
	}
}

func TestValidateTemplateType(t *testing.T) {
	t.Parallel()

	for _, typ := range []string{TemplateTypeReminder, TemplateTypeDunning, TemplateTypeTermination} {
		require.NoError(t, ValidateTemplateType(typ), "type %q is one of the three", typ)
	}

	for _, typ := range []string{"", "Reminder", "reminders", "nope"} {
		err := ValidateTemplateType(typ)
		require.ErrorIs(t, err, ErrInvalidTemplateType, "type %q must be refused", typ)
		require.ErrorIs(t, err, ErrValidation, "and it must still read as a validation failure")
	}
}

func TestNotFoundSentinelsNest(t *testing.T) {
	t.Parallel()

	require.ErrorIs(t, ErrCustomerNotFound, ErrNotFound)
	require.ErrorIs(t, ErrTemplateNotFound, ErrNotFound)
	require.NotErrorIs(t, ErrCustomerNotFound, ErrTemplateNotFound,
		"the two must stay distinguishable, or a send cannot tell which thing was missing")
}
