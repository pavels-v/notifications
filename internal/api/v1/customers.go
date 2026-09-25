package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"notifications/internal/domain"
)

const dateLayout = "2006-01-02"

type CustomerService interface {
	Get(ctx context.Context, creditNumber string) (*domain.Customer, error)
	List(ctx context.Context) ([]*domain.Customer, error)
	Create(ctx context.Context, c domain.Customer) (*domain.Customer, error)
	Update(ctx context.Context, c domain.Customer) (*domain.Customer, error)
}

type customerRequest struct {
	CreditNumber string `json:"credit_number"`
	PhoneNumber  string `json:"phone_number"`
	FullName     string `json:"full_name"`
	AmountMinor  int64  `json:"amount_minor"`
	Currency     string `json:"currency"`
	DueDate      string `json:"due_date"`
}

func (r customerRequest) toCustomer(creditNumber string) (domain.Customer, error) {
	due, err := time.Parse(dateLayout, r.DueDate)
	if err != nil {
		return domain.Customer{}, fmt.Errorf(`"due_date" must be ISO 8601, for example 2026-09-25, got %q`, r.DueDate)
	}

	return domain.Customer{
		CreditNumber: creditNumber,
		PhoneNumber:  r.PhoneNumber,
		FullName:     r.FullName,
		AmountMinor:  r.AmountMinor,
		Currency:     r.Currency,
		DueDate:      due,
	}, nil
}

// @Summary List loans
// @Tags customers
// @Produce json
// @Success 200 {array} domain.Customer
// @Failure 500 {object} errorBody
// @Router /customers [get]
func (s *Server) listCustomers(w http.ResponseWriter, r *http.Request) {
	customers, err := s.customers.List(r.Context())
	if err != nil {
		status, body := s.statusFor(r, err, "could not load the loans")
		s.writeJSON(w, status, body)
		return
	}

	s.writeJSON(w, http.StatusOK, customers)
}

// @Summary Get a loan
// @Tags customers
// @Produce json
// @Param credit_number path string true "Credit number"
// @Success 200 {object} domain.Customer
// @Failure 404 {object} errorBody
// @Failure 500 {object} errorBody
// @Router /customers/{credit_number} [get]
func (s *Server) getCustomer(w http.ResponseWriter, r *http.Request) {
	customer, err := s.customers.Get(r.Context(), chi.URLParam(r, paramCreditNumber))
	if err != nil {
		status, body := s.statusFor(r, err, "could not load the loan")
		s.writeJSON(w, status, body)
		return
	}

	s.writeJSON(w, http.StatusOK, customer)
}

// @Summary Create a loan
// @Tags customers
// @Accept json
// @Produce json
// @Param customer body customerRequest true "Loan"
// @Success 201 {object} domain.Customer
// @Failure 400 {object} errorBody
// @Failure 409 {object} errorBody
// @Failure 500 {object} errorBody
// @Router /customers [post]
func (s *Server) createCustomer(w http.ResponseWriter, r *http.Request) {
	var req customerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, codeInvalidJSON, err.Error())
		return
	}

	customer, err := req.toCustomer(req.CreditNumber)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	created, err := s.customers.Create(r.Context(), customer)
	if err != nil {
		status, body := s.statusFor(r, err, "could not create the loan")
		s.writeJSON(w, status, body)
		return
	}

	s.writeJSON(w, http.StatusCreated, created)
}

// @Summary Update a loan
// @Tags customers
// @Accept json
// @Produce json
// @Param credit_number path string true "Credit number"
// @Param customer body customerRequest true "Loan"
// @Success 200 {object} domain.Customer
// @Failure 400 {object} errorBody
// @Failure 404 {object} errorBody
// @Failure 500 {object} errorBody
// @Router /customers/{credit_number} [put]
func (s *Server) updateCustomer(w http.ResponseWriter, r *http.Request) {
	var req customerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, codeInvalidJSON, err.Error())
		return
	}

	customer, err := req.toCustomer(chi.URLParam(r, paramCreditNumber))
	if err != nil {
		s.writeError(w, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	updated, err := s.customers.Update(r.Context(), customer)
	if err != nil {
		status, body := s.statusFor(r, err, "could not update the loan")
		s.writeJSON(w, status, body)
		return
	}

	s.writeJSON(w, http.StatusOK, updated)
}
