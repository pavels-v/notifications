package service

import (
	"context"
	"errors"
	"fmt"

	"notifications/internal/domain"
	"notifications/internal/pkg/logger"
	"notifications/internal/sms"
	"notifications/internal/tmpl"
)

type MessageRepo interface {
	Insert(ctx context.Context, m domain.Message) (*domain.Message, error)
	UpdateOutcome(ctx context.Context, id int64, o domain.MessageOutcome) (*domain.Message, error)
	List(ctx context.Context, creditNumber string) ([]*domain.Message, error)
}

type Messages struct {
	customers CustomerRepo
	templates TemplateRepo
	messages  MessageRepo
	sender    sms.Sender
	log       logger.Logger
}

func NewMessages(customers CustomerRepo, templates TemplateRepo, messages MessageRepo,
	sender sms.Sender, log logger.Logger) *Messages {
	return &Messages{customers: customers, templates: templates, messages: messages, sender: sender, log: log}
}

func (s *Messages) Send(ctx context.Context, creditNumber, templateType string) (*domain.Message, error) {
	customer, err := s.customers.Get(ctx, creditNumber)
	if err != nil {
		return nil, err
	}

	template, err := s.templates.Get(ctx, templateType)
	if err != nil {
		if errors.Is(err, domain.ErrTemplateNotFound) {
			return nil, fmt.Errorf("%w: %v", ErrTemplateUnusable, err)
		}
		return nil, err
	}

	body, err := tmpl.Render(template.Body, *customer)
	if err != nil {
		return nil, err
	}

	record, err := s.messages.Insert(ctx, domain.Message{
		CreditNumber: customer.CreditNumber,
		TemplateType: template.Type,
		Body:         body,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: record the intent to send: %v", ErrAttemptCannotBeRecorded, err)
	}

	resp, sendErr := s.sender.Send(ctx, sms.Message{To: customer.PhoneNumber, Body: body})
	outcome := outcomeFor(resp, sendErr)

	saved, err := s.messages.UpdateOutcome(ctx, record.ID, outcome)
	if err != nil {
		s.log.Error("failed to update message outcome", "message_id", record.ID, "credit_number", creditNumber,
			"status", outcome.Status, "provider_message_id", outcome.ProviderMessageID, "send_error", outcome.Error, "error", err)
		return nil, fmt.Errorf("%w: record the send outcome: %v", ErrAttemptCannotBeRecorded, err)
	}

	if sendErr != nil {
		s.log.Error("failed to send sms", "credit_number", creditNumber, "template", template.Type,
			"message_id", saved.ID, "status", saved.Status, "error", sendErr)
		return saved, fmt.Errorf("%w: %w", ErrSendFailed, sendErr)
	}

	return saved, nil
}

func (s *Messages) List(ctx context.Context, creditNumber string) ([]*domain.Message, error) {
	if _, err := s.customers.Get(ctx, creditNumber); err != nil {
		return nil, err
	}

	return s.messages.List(ctx, creditNumber)
}

func outcomeFor(resp *sms.Response, err error) domain.MessageOutcome {
	if err != nil {
		var rejected *sms.RejectedError
		if errors.As(err, &rejected) {
			return domain.MessageOutcome{
				Status:      domain.StatusRejectedByProvider,
				ResponseRaw: rejected.RawBody,
				Error:       rejected.Error(),
			}
		}

		return domain.MessageOutcome{Status: domain.StatusPending, Error: err.Error()}
	}

	return domain.MessageOutcome{
		Status:            domain.StatusSentToProvider,
		ProviderMessageID: resp.ProviderMessageID,
		ResponseRaw:       resp.RawBody,
	}
}
