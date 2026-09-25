package v1

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	_ "notifications/api/v1"
	"notifications/internal/pkg/logger"
)

const (
	paramCreditNumber = "credit_number"
	paramType         = "type"
)

type Server struct {
	customers CustomerService
	templates TemplateService
	messages  MessageService
	logger    logger.Logger
}

func NewServer(
	customers CustomerService,
	templates TemplateService,
	messages MessageService,
	logger logger.Logger,
) *Server {
	return &Server{
		customers: customers,
		templates: templates,
		messages:  messages,
		logger:    logger,
	}
}

func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()

	r.Get("/swagger/*", httpSwagger.Handler(httpSwagger.InstanceName("v1"), httpSwagger.URL("/v1/swagger/doc.json")))

	r.Get("/customers", s.listCustomers)
	r.Post("/customers", s.createCustomer)
	r.Route("/customers/{credit_number}", func(r chi.Router) {
		r.Get("/", s.getCustomer)
		r.Put("/", s.updateCustomer)
		r.Get("/messages", s.listMessages)
		r.Post("/messages", s.sendMessage)
	})

	r.Get("/templates", s.listTemplates)
	r.Get("/templates/{type}", s.getTemplate)
	r.Put("/templates/{type}", s.upsertTemplate)

	return r
}
