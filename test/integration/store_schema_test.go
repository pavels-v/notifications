//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"notifications/internal/store"
)

func TestMigrateAndSeedProduceAKnownState(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()

	customers, err := store.NewCustomers(s.DB()).List(ctx)
	require.NoError(t, err)
	require.Len(t, customers, 2, "the seed's two demo loans")

	templates, err := store.NewTemplates(s.DB()).List(ctx)
	require.NoError(t, err)
	require.Len(t, templates, 3, "reminder, dunning and termination")

	messages, err := store.NewMessages(s.DB()).List(ctx, customers[0].CreditNumber)
	require.NoError(t, err)
	require.Empty(t, messages, "a freshly seeded database has no sends")
}

func TestMigrateIsIdempotent(t *testing.T) {
	s := newStore(t)

	require.NoError(t, s.Migrate(context.Background()),
		"migrating an already-migrated database must be a no-op")
}
