//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"notifications/internal/domain"
	"notifications/internal/store"
)

func TestGetSeededTemplates(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()

	cases := []struct {
		typ      string
		wantFind bool
	}{
		{typ: domain.TemplateTypeReminder, wantFind: true},
		{typ: domain.TemplateTypeDunning, wantFind: true},
		{typ: domain.TemplateTypeTermination, wantFind: true},
		{typ: "nope", wantFind: false},
		{typ: "", wantFind: false},
	}

	for _, c := range cases {
		t.Run("type="+c.typ, func(t *testing.T) {
			tpl, err := store.NewTemplates(s.DB()).Get(ctx, c.typ)

			if !c.wantFind {
				require.ErrorIs(t, err, domain.ErrNotFound, "an unknown type must map to the sentinel")
				return
			}
			require.NoError(t, err)
			require.Equal(t, c.typ, tpl.Type)
			require.NotEmpty(t, tpl.Body, "the seeded body must not be empty")
		})
	}
}

func TestUpsertTemplateReplacesRatherThanDuplicates(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()

	updated, err := store.NewTemplates(s.DB()).Upsert(ctx, domain.TemplateTypeReminder, "new body {full_name}")
	require.NoError(t, err)
	require.Equal(t, "new body {full_name}", updated.Body)

	all, err := store.NewTemplates(s.DB()).List(ctx)
	require.NoError(t, err)
	require.Len(t, all, 3, "upsert on an existing type must not insert a second row")
}
