package v1

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"notifications/internal/domain"
)

type TemplateService interface {
	Get(ctx context.Context, typ string) (*domain.Template, error)
	List(ctx context.Context) ([]*domain.Template, error)
	Upsert(ctx context.Context, typ, body string) (*domain.Template, error)
}

type templateRequest struct {
	Body string `json:"body"`
}

// @Summary List templates
// @Tags templates
// @Produce json
// @Success 200 {array} domain.Template
// @Failure 500 {object} errorBody
// @Router /templates [get]
func (s *Server) listTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := s.templates.List(r.Context())
	if err != nil {
		status, body := s.statusFor(r, err, "could not load the templates")
		s.writeJSON(w, status, body)
		return
	}

	s.writeJSON(w, http.StatusOK, templates)
}

// @Summary Get a template
// @Tags templates
// @Produce json
// @Param type path string true "Template type" Enums(reminder, dunning, termination)
// @Success 200 {object} domain.Template
// @Failure 400 {object} errorBody
// @Failure 404 {object} errorBody
// @Failure 500 {object} errorBody
// @Router /templates/{type} [get]
func (s *Server) getTemplate(w http.ResponseWriter, r *http.Request) {
	template, err := s.templates.Get(r.Context(), chi.URLParam(r, paramType))
	if err != nil {
		status, body := s.statusFor(r, err, "could not load the template")
		s.writeJSON(w, status, body)
		return
	}

	s.writeJSON(w, http.StatusOK, template)
}

// @Summary Create or replace a template
// @Tags templates
// @Accept json
// @Produce json
// @Param type path string true "Template type" Enums(reminder, dunning, termination)
// @Param template body templateRequest true "Template body"
// @Success 200 {object} domain.Template
// @Failure 400 {object} errorBody
// @Failure 500 {object} errorBody
// @Router /templates/{type} [put]
func (s *Server) upsertTemplate(w http.ResponseWriter, r *http.Request) {
	var req templateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, codeInvalidJSON, err.Error())
		return
	}

	if req.Body == "" {
		s.writeError(w, http.StatusBadRequest, codeMissingBody, `field "body" is required`)
		return
	}

	template, err := s.templates.Upsert(r.Context(), chi.URLParam(r, paramType), req.Body)
	if err != nil {
		status, body := s.statusFor(r, err, "could not save the template")
		s.writeJSON(w, status, body)
		return
	}

	s.writeJSON(w, http.StatusOK, template)
}
