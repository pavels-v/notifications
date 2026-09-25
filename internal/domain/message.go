package domain

import "time"

type MessageStatus string

const (
	StatusPending            MessageStatus = "pending"
	StatusSentToProvider     MessageStatus = "sent_to_provider"
	StatusRejectedByProvider MessageStatus = "rejected_by_provider"
)

type Message struct {
	ID                int64         `json:"id" db:"id"`
	CreditNumber      string        `json:"credit_number" db:"credit_number"`
	TemplateType      string        `json:"template" db:"template_type"`
	Body              string        `json:"body" db:"body"`
	ProviderMessageID string        `json:"provider_message_id,omitempty" db:"provider_message_id"`
	Status            MessageStatus `json:"status" db:"status"`
	ResponseRaw       string        `json:"-" db:"response_raw"`
	Error             string        `json:"error,omitempty" db:"error"`
	CreatedAt         time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at" db:"updated_at"`
}

type MessageOutcome struct {
	Status            MessageStatus
	ProviderMessageID string
	ResponseRaw       string
	Error             string
}
