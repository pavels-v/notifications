package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"notifications/internal/domain"
)

func aCustomer() domain.Customer {
	return domain.Customer{
		CreditNumber: "4000000001",
		PhoneNumber:  "+999000000001",
		FullName:     "Jane Doe",
		AmountMinor:  90000,
		Currency:     "EUR",
		DueDate:      time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
	}
}

func TestCustomersCreateWrites(t *testing.T) {
	t.Parallel()

	repo := newFakeCustomerRepo()
	created, err := NewCustomers(repo).Create(context.Background(), aCustomer())

	require.NoError(t, err)
	require.Equal(t, "4000000001", created.CreditNumber)
	require.Len(t, repo.created, 1, "the customer must reach the repository")
}

func TestCustomersCreateRefusesBeforeWriting(t *testing.T) {
	t.Parallel()

	c := aCustomer()
	c.PhoneNumber = "0170 1234567"

	repo := newFakeCustomerRepo()
	_, err := NewCustomers(repo).Create(context.Background(), c)

	require.ErrorIs(t, err, domain.ErrValidation)
	require.Empty(t, repo.created, "an invalid customer must never reach the repository")
}

func TestCustomersCreateSurfacesTheConflict(t *testing.T) {
	t.Parallel()

	repo := newFakeCustomerRepo(aCustomer())
	_, err := NewCustomers(repo).Create(context.Background(), aCustomer())

	require.ErrorIs(t, err, domain.ErrCustomerAlreadyExists)
	require.ErrorIs(t, err, domain.ErrAlreadyExists, "the API answers 409 on the general sentinel")
	require.Empty(t, repo.created, "a duplicate must not be recorded as written")
}

func TestCustomersUpdateRefusesBeforeWriting(t *testing.T) {
	t.Parallel()

	c := aCustomer()
	c.Currency = "EURO"

	repo := newFakeCustomerRepo(aCustomer())
	_, err := NewCustomers(repo).Update(context.Background(), c)

	require.ErrorIs(t, err, domain.ErrValidation)
	require.Empty(t, repo.updated, "an invalid customer must never reach the repository")
}

func TestCustomersGetPassesTheSentinelThrough(t *testing.T) {
	t.Parallel()

	_, err := NewCustomers(newFakeCustomerRepo()).Get(context.Background(), "9000000001")

	require.ErrorIs(t, err, domain.ErrCustomerNotFound)
	require.ErrorIs(t, err, domain.ErrNotFound, "the API answers 404 on the general sentinel")
}
