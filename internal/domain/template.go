package domain

import "time"

const (
	TemplateTypeReminder    = "reminder"
	TemplateTypeDunning     = "dunning"
	TemplateTypeTermination = "termination"
)

type Template struct {
	Type      string    `json:"type" db:"type"`
	Body      string    `json:"body" db:"body"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

//nolint:gochecknoglobals // read-only lookup table
var templateTypes = map[string]bool{
	TemplateTypeReminder:    true,
	TemplateTypeDunning:     true,
	TemplateTypeTermination: true,
}

func ValidateTemplateType(typ string) error {
	if !templateTypes[typ] {
		return ErrInvalidTemplateType
	}
	return nil
}
