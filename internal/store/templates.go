package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"notifications/internal/domain"
)

type Templates struct {
	db *sqlx.DB
}

func NewTemplates(db *sqlx.DB) *Templates {
	return &Templates{db: db}
}

func (t *Templates) Get(ctx context.Context, typ string) (*domain.Template, error) {
	var out domain.Template

	err := t.db.GetContext(ctx, &out, `SELECT type, body, created_at, updated_at FROM templates WHERE type = $1`, typ)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrTemplateNotFound
		}
		return nil, fmt.Errorf("get template: %w", err)
	}

	return &out, nil
}

func (t *Templates) List(ctx context.Context) ([]*domain.Template, error) {
	out := []*domain.Template{}

	err := t.db.SelectContext(ctx, &out, `SELECT type, body, created_at, updated_at FROM templates ORDER BY type`)
	if err != nil {
		return nil, fmt.Errorf("list templates: %w", err)
	}

	return out, nil
}

func (t *Templates) Upsert(ctx context.Context, typ, body string) (*domain.Template, error) {
	var out domain.Template

	err := t.db.GetContext(ctx, &out,
		`INSERT INTO templates (type, body) VALUES ($1, $2)`+
			` ON CONFLICT (type) DO UPDATE SET body = EXCLUDED.body, updated_at = now()`+
			` RETURNING type, body, created_at, updated_at`,
		typ,  // 1
		body, // 2
	)
	if err != nil {
		return nil, fmt.Errorf("upsert template: %w", err)
	}

	return &out, nil
}
