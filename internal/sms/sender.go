package sms

import (
	"context"
	"errors"
	"fmt"
)

var ErrRejected = errors.New("sms: provider rejected the message")

type Message struct {
	To   string
	Body string
}

type Response struct {
	ProviderMessageID string
	RawBody           string
}

type RejectedError struct {
	StatusCode int
	RawBody    string
}

func (e *RejectedError) Error() string {
	return fmt.Sprintf("sms: provider rejected the message: http %d: %s", e.StatusCode, e.RawBody)
}

func (e *RejectedError) Unwrap() error { return ErrRejected }

type Sender interface {
	Send(ctx context.Context, m Message) (*Response, error)
}
