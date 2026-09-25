package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"notifications/internal/domain"
)

type Messages struct {
	db *sqlx.DB
}

func NewMessages(db *sqlx.DB) *Messages {
	return &Messages{db: db}
}

func (m *Messages) Insert(ctx context.Context, msg domain.Message) (*domain.Message, error) {
	var out domain.Message

	err := m.db.GetContext(ctx, &out,
		`INSERT INTO messages (credit_number, template_type, body)`+
			` VALUES ($1, $2, $3)`+
			` RETURNING id, credit_number, template_type, body, provider_message_id, status, response_raw, error, created_at, updated_at`,
		msg.CreditNumber, // 1
		msg.TemplateType, // 2
		msg.Body,         // 3
	)
	if err != nil {
		return nil, fmt.Errorf("insert message: %w", err)
	}

	return &out, nil
}

func (m *Messages) UpdateOutcome(ctx context.Context, id int64, o domain.MessageOutcome) (*domain.Message, error) {
	var out domain.Message

	err := m.db.GetContext(ctx, &out,
		`UPDATE messages SET status = $2, provider_message_id = $3, response_raw = $4, error = $5, updated_at = now()`+
			` WHERE id = $1`+
			` RETURNING id, credit_number, template_type, body, provider_message_id, status, response_raw, error, created_at, updated_at`,
		id,                  // 1
		o.Status,            // 2
		o.ProviderMessageID, // 3
		o.ResponseRaw,       // 4
		o.Error,             // 5
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("update message outcome: %w", err)
	}

	return &out, nil
}

func (m *Messages) List(ctx context.Context, creditNumber string) ([]*domain.Message, error) {
	out := []*domain.Message{}

	err := m.db.SelectContext(ctx, &out,
		`SELECT id, credit_number, template_type, body, provider_message_id, status, response_raw, error, created_at, updated_at`+
			` FROM messages WHERE credit_number = $1 ORDER BY created_at DESC, id DESC`,
		creditNumber)
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}

	return out, nil
}
