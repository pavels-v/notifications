package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"notifications/internal/domain"
)

type Customers struct {
	db *sqlx.DB
}

func NewCustomers(db *sqlx.DB) *Customers {
	return &Customers{db: db}
}

func (c *Customers) Get(ctx context.Context, creditNumber string) (*domain.Customer, error) {
	var out domain.Customer

	err := c.db.GetContext(ctx, &out,
		`SELECT credit_number, phone_number, full_name, amount_minor, currency, due_date, created_at, updated_at`+
			` FROM customers WHERE credit_number = $1`, creditNumber,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrCustomerNotFound
		}
		return nil, fmt.Errorf("get customer: %w", err)
	}

	return &out, nil
}

func (c *Customers) List(ctx context.Context) ([]*domain.Customer, error) {
	out := []*domain.Customer{}

	err := c.db.SelectContext(ctx, &out,
		`SELECT credit_number, phone_number, full_name, amount_minor, currency, due_date, created_at, updated_at`+
			` FROM customers ORDER BY credit_number`)
	if err != nil {
		return nil, fmt.Errorf("list customers: %w", err)
	}

	return out, nil
}

func (c *Customers) Create(ctx context.Context, customer domain.Customer) (*domain.Customer, error) {
	var out domain.Customer

	err := c.db.GetContext(ctx, &out,
		`INSERT INTO customers (credit_number, phone_number, full_name, amount_minor, currency, due_date)`+
			` VALUES ($1, $2, $3, $4, $5, $6)`+
			` RETURNING credit_number, phone_number, full_name, amount_minor, currency, due_date, created_at, updated_at`,
		customer.CreditNumber, // 1
		customer.PhoneNumber,  // 2
		customer.FullName,     // 3
		customer.AmountMinor,  // 4
		customer.Currency,     // 5
		customer.DueDate,      // 6
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, domain.ErrCustomerAlreadyExists
		}
		return nil, fmt.Errorf("create customer: %w", err)
	}

	return &out, nil
}

func (c *Customers) Update(ctx context.Context, customer domain.Customer) (*domain.Customer, error) {
	var out domain.Customer

	err := c.db.GetContext(ctx, &out,
		`UPDATE customers SET phone_number = $2, full_name = $3, amount_minor = $4, currency = $5, due_date = $6, updated_at = now()`+
			` WHERE credit_number = $1`+
			` RETURNING credit_number, phone_number, full_name, amount_minor, currency, due_date, created_at, updated_at`,
		customer.CreditNumber, // 1
		customer.PhoneNumber,  // 2
		customer.FullName,     // 3
		customer.AmountMinor,  // 4
		customer.Currency,     // 5
		customer.DueDate,      // 6
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrCustomerNotFound
		}
		return nil, fmt.Errorf("update customer: %w", err)
	}

	return &out, nil
}
