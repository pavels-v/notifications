package domain

import (
	"regexp"
	"time"
)

var e164 = regexp.MustCompile(`^\+[1-9]\d{7,14}$`)

type Customer struct {
	CreditNumber string    `json:"credit_number" db:"credit_number"`
	PhoneNumber  string    `json:"phone_number" db:"phone_number"`
	FullName     string    `json:"full_name" db:"full_name"`
	AmountMinor  int64     `json:"amount_minor" db:"amount_minor"`
	Currency     string    `json:"currency" db:"currency"`
	DueDate      time.Time `json:"due_date" db:"due_date"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

func (c Customer) Validate() error {
	if c.CreditNumber == "" {
		return invalid(`"credit_number" is required`)
	}

	if c.FullName == "" {
		return invalid(`"full_name" is required`)
	}

	if !e164.MatchString(c.PhoneNumber) {
		return invalid(`"phone_number" must be E.164: a plus sign followed by 8 to 15 digits, got %q`, c.PhoneNumber)
	}

	if len(c.Currency) != 3 {
		return invalid(`"currency" must be a 3-letter code, got %q`, c.Currency)
	}

	if c.AmountMinor < 0 {
		return invalid(`"amount_minor" must not be negative`)
	}

	return nil
}
