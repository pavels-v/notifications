package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	v1 "notifications/internal/api/v1"
	"notifications/internal/pkg/logger"
)

type Server struct {
	v1     *v1.Server
	logger logger.Logger
}

func NewServer(
	v1 *v1.Server,
	logger logger.Logger,
) *Server {
	return &Server{
		v1:     v1,
		logger: logger,
	}
}

func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(s.logRequests)
	r.Use(s.recoverPanics)

	r.NotFound(s.notFound)
	r.MethodNotAllowed(s.methodNotAllowed)

	r.Mount("/v1", s.v1.Handler())

	return r
}

func (s *Server) notFound(w http.ResponseWriter, r *http.Request) {
	s.writeError(w, http.StatusNotFound, codeNotFound, "no endpoint at this path")
}

func (s *Server) methodNotAllowed(w http.ResponseWriter, r *http.Request) {
	s.writeError(w, http.StatusMethodNotAllowed, codeMethodNotAllowed, "this method is not allowed at this path")
}
