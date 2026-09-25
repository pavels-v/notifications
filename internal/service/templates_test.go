package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"notifications/internal/domain"
)

func TestTemplatesUpsertWrites(t *testing.T) {
	t.Parallel()

	repo := newFakeTemplateRepo()
	got, err := NewTemplates(repo).Upsert(context.Background(), domain.TemplateTypeReminder, "Dear {full_name}")

	require.NoError(t, err)
	require.Equal(t, domain.TemplateTypeReminder, got.Type)
	require.Len(t, repo.upserted, 1)
}

func TestTemplatesUpsertRefusesAnUnknownType(t *testing.T) {
	t.Parallel()

	repo := newFakeTemplateRepo()
	_, err := NewTemplates(repo).Upsert(context.Background(), "invoice", "Dear {full_name}")

	require.ErrorIs(t, err, domain.ErrInvalidTemplateType, "the API answers this one with invalid_type")
	require.Empty(t, repo.upserted, "an unknown type must never reach the repository")
}

func TestTemplatesGetPassesTheSentinelThrough(t *testing.T) {
	t.Parallel()

	_, err := NewTemplates(newFakeTemplateRepo()).Get(context.Background(), domain.TemplateTypeReminder)

	require.ErrorIs(t, err, domain.ErrTemplateNotFound)
	require.ErrorIs(t, err, domain.ErrNotFound)
}
