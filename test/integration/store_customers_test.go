//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"notifications/internal/domain"
	"notifications/internal/store"
)

func TestStoreCreateAndGetCustomer(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()

	created, err := store.NewCustomers(s.DB()).Create(ctx, sampleCustomer())
	require.NoError(t, err)
	require.False(t, created.CreatedAt.IsZero(), "CreatedAt must come back from the database default")

	got, err := store.NewCustomers(s.DB()).Get(ctx, "2000000001")
	require.NoError(t, err)

	require.Equal(t, "John Doe", got.FullName)
	require.Equal(t, int64(125000), got.AmountMinor, "the amount must survive as minor units")
	require.Equal(t, "EUR", got.Currency)
	require.True(t, got.DueDate.Equal(sampleDueDate), "DueDate = %v, want %v", got.DueDate, sampleDueDate)
}

func TestCustomerErrorMapping(t *testing.T) {
	cases := []struct {
		name    string
		arrange func(t *testing.T, s *store.DB) // optional setup
		act     func(ctx context.Context, s *store.DB) error
		wantErr error
	}{
		{
			name: "get a credit number that does not exist",
			act: func(ctx context.Context, s *store.DB) error {
				_, err := store.NewCustomers(s.DB()).Get(ctx, "9000000001")
				return err
			},
			wantErr: domain.ErrNotFound,
		},
		{
			name: "create a credit number that already exists",
			arrange: func(t *testing.T, s *store.DB) {
				_, err := store.NewCustomers(s.DB()).Create(context.Background(), sampleCustomer())
				require.NoError(t, err)
			},
			act: func(ctx context.Context, s *store.DB) error {
				_, err := store.NewCustomers(s.DB()).Create(ctx, sampleCustomer())
				return err
			},
			wantErr: domain.ErrAlreadyExists,
		},
		{
			name: "update a credit number that does not exist",
			act: func(ctx context.Context, s *store.DB) error {
				c := sampleCustomer()
				c.CreditNumber = "9000000001"
				_, err := store.NewCustomers(s.DB()).Update(ctx, c)
				return err
			},
			wantErr: domain.ErrNotFound,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := newStore(t)
			ctx := context.Background()

			if c.arrange != nil {
				c.arrange(t, s)
			}

			err := c.act(ctx, s)
			require.ErrorIs(t, err, c.wantErr, "the API layer switches on this sentinel")
		})
	}
}

func TestStoreUpdateCustomer(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()

	_, err := store.NewCustomers(s.DB()).Create(ctx, sampleCustomer())
	require.NoError(t, err)

	c := sampleCustomer()
	c.PhoneNumber = "+999000000022"
	c.AmountMinor = 50000

	updated, err := store.NewCustomers(s.DB()).Update(ctx, c)
	require.NoError(t, err)

	require.Equal(t, "+999000000022", updated.PhoneNumber)
	require.Equal(t, int64(50000), updated.AmountMinor)
}
