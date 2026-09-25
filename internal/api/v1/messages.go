package v1

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"notifications/internal/domain"
	"notifications/internal/service"
)

type MessageService interface {
	Send(ctx context.Context, creditNumber, templateType string) (*domain.Message, error)
	List(ctx context.Context, creditNumber string) ([]*domain.Message, error)
}

type sendRequest struct {
	TemplateType string `json:"template_type"`
}

// @Summary Send a message to a loan
// @Tags messages
// @Accept json
// @Produce json
// @Param credit_number path string true "Credit number"
// @Param message body sendRequest true "Template to render"
// @Success 201 {object} domain.Message
// @Failure 400 {object} errorBody
// @Failure 404 {object} errorBody
// @Failure 422 {object} errorBody
// @Failure 500 {object} errorBody
// @Failure 502 {object} domain.Message
// @Router /customers/{credit_number}/messages [post]
func (s *Server) sendMessage(w http.ResponseWriter, r *http.Request) {
	var req sendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, codeInvalidJSON, err.Error())
		return
	}

	if req.TemplateType == "" {
		s.writeError(w, http.StatusBadRequest, codeMissingTemplate, `field "template_type" is required`)
		return
	}

	creditNumber := chi.URLParam(r, paramCreditNumber)

	message, err := s.messages.Send(r.Context(), creditNumber, req.TemplateType)
	if err != nil {
		if errors.Is(err, service.ErrSendFailed) {
			s.writeJSON(w, http.StatusBadGateway, message)
			return
		}

		status, body := s.statusFor(r, err, "the message could not be sent")
		s.writeJSON(w, status, body)
		return
	}

	s.writeJSON(w, http.StatusCreated, message)
}

// @Summary List the messages of a loan
// @Tags messages
// @Produce json
// @Param credit_number path string true "Credit number"
// @Success 200 {array} domain.Message
// @Failure 404 {object} errorBody
// @Failure 500 {object} errorBody
// @Router /customers/{credit_number}/messages [get]
func (s *Server) listMessages(w http.ResponseWriter, r *http.Request) {
	history, err := s.messages.List(r.Context(), chi.URLParam(r, paramCreditNumber))
	if err != nil {
		status, body := s.statusFor(r, err, "could not load the history")
		s.writeJSON(w, status, body)
		return
	}

	s.writeJSON(w, http.StatusOK, history)
}
