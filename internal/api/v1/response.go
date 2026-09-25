package v1

import (
	"encoding/json"
	"errors"
	"net/http"

	"notifications/internal/domain"
	"notifications/internal/service"
)

const (
	contentTypeJSON = "application/json"

	codeInternal              = "internal"
	codeInvalidJSON           = "invalid_json"
	codeValidationFailed      = "validation_failed"
	codeMissingTemplate       = "missing_template"
	codeMissingBody           = "missing_body"
	codeInvalidType           = "invalid_type"
	codeNotFound              = "not_found"
	codeCustomerNotFound      = "customer_not_found"
	codeTemplateNotFound      = "template_not_found"
	codeAlreadyExists         = "already_exists"
	codeCustomerAlreadyExists = "customer_already_exists"
	codeTemplateUnusable      = "template_unusable"
	codeUnknownPlaceholder    = "unknown_placeholder"
)

type errorBody struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", contentTypeJSON)
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(v); err != nil {
		s.logger.Error("failed to encode response", "error", err)
	}
}

func (s *Server) writeError(w http.ResponseWriter, status int, code, message string) {
	s.writeJSON(w, status, errorBody{Error: code, Message: message})
}

func (s *Server) statusFor(r *http.Request, err error, internal string) (int, errorBody) {
	switch {
	case errors.Is(err, service.ErrAttemptCannotBeRecorded):
		s.logger.Error("failed to serve request", "method", r.Method, "path", r.URL.Path, "error", err)
		return http.StatusInternalServerError, errorBody{Error: codeInternal, Message: internal}
	case errors.Is(err, domain.ErrCustomerNotFound):
		return http.StatusNotFound, errorBody{Error: codeCustomerNotFound, Message: "no loan with this credit number"}
	case errors.Is(err, domain.ErrTemplateNotFound):
		return http.StatusNotFound, errorBody{Error: codeTemplateNotFound, Message: "no template of this type"}
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound, errorBody{Error: codeNotFound, Message: "no record of this kind"}
	case errors.Is(err, domain.ErrCustomerAlreadyExists):
		return http.StatusConflict, errorBody{Error: codeCustomerAlreadyExists, Message: "a loan with this credit number already exists"}
	case errors.Is(err, domain.ErrAlreadyExists):
		return http.StatusConflict, errorBody{Error: codeAlreadyExists, Message: "a record of this kind already exists"}
	case errors.Is(err, service.ErrTemplateUnusable):
		return http.StatusUnprocessableEntity, errorBody{Error: codeTemplateUnusable, Message: "the template named by this message cannot be used"}
	case errors.Is(err, domain.ErrUnknownPlaceholder):
		return http.StatusUnprocessableEntity, errorBody{Error: codeUnknownPlaceholder, Message: err.Error()}
	case errors.Is(err, domain.ErrInvalidTemplateType):
		return http.StatusBadRequest, errorBody{Error: codeInvalidType, Message: "template type must be reminder, dunning or termination"}
	case errors.Is(err, domain.ErrValidation):
		return http.StatusBadRequest, errorBody{Error: codeValidationFailed, Message: err.Error()}
	default:
		s.logger.Error("failed to serve request", "method", r.Method, "path", r.URL.Path, "error", err)
		return http.StatusInternalServerError, errorBody{Error: codeInternal, Message: internal}
	}
}
