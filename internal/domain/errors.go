package domain

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound         = errors.New("not found")
	ErrCustomerNotFound = fmt.Errorf("customer %w", ErrNotFound)
	ErrTemplateNotFound = fmt.Errorf("template %w", ErrNotFound)

	ErrValidation          = errors.New("validation")
	ErrInvalidTemplateType = fmt.Errorf("%w: template type must be reminder, dunning or termination", ErrValidation)

	ErrAlreadyExists         = errors.New("already exists")
	ErrCustomerAlreadyExists = fmt.Errorf("customer %w", ErrAlreadyExists)

	ErrUnknownPlaceholder = errors.New("unknown placeholder")
)

type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

func (e *ValidationError) Unwrap() error { return ErrValidation }

func invalid(format string, args ...any) error {
	return &ValidationError{Message: fmt.Sprintf(format, args...)}
}
