package service

import (
	"context"

	"notifications/internal/domain"
	"notifications/internal/sms"
)

type fakeCustomerRepo struct {
	customers map[string]domain.Customer
	created   []domain.Customer
	updated   []domain.Customer
	err       error
}

func newFakeCustomerRepo(seed ...domain.Customer) *fakeCustomerRepo {
	repo := &fakeCustomerRepo{customers: map[string]domain.Customer{}}
	for _, c := range seed {
		repo.customers[c.CreditNumber] = c
	}
	return repo
}

func (f *fakeCustomerRepo) Get(_ context.Context, creditNumber string) (*domain.Customer, error) {
	if f.err != nil {
		return nil, f.err
	}
	c, ok := f.customers[creditNumber]
	if !ok {
		return nil, domain.ErrCustomerNotFound
	}
	return &c, nil
}

func (f *fakeCustomerRepo) List(context.Context) ([]*domain.Customer, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := []*domain.Customer{}
	for _, c := range f.customers {
		out = append(out, &c)
	}
	return out, nil
}

func (f *fakeCustomerRepo) Create(_ context.Context, c domain.Customer) (*domain.Customer, error) {
	if f.err != nil {
		return nil, f.err
	}
	if _, ok := f.customers[c.CreditNumber]; ok {
		return nil, domain.ErrCustomerAlreadyExists
	}
	f.created = append(f.created, c)
	f.customers[c.CreditNumber] = c
	return &c, nil
}

func (f *fakeCustomerRepo) Update(_ context.Context, c domain.Customer) (*domain.Customer, error) {
	if f.err != nil {
		return nil, f.err
	}
	if _, ok := f.customers[c.CreditNumber]; !ok {
		return nil, domain.ErrCustomerNotFound
	}
	f.updated = append(f.updated, c)
	f.customers[c.CreditNumber] = c
	return &c, nil
}

type fakeTemplateRepo struct {
	templates map[string]domain.Template
	upserted  []domain.Template
	err       error
}

func newFakeTemplateRepo(seed ...domain.Template) *fakeTemplateRepo {
	repo := &fakeTemplateRepo{templates: map[string]domain.Template{}}
	for _, t := range seed {
		repo.templates[t.Type] = t
	}
	return repo
}

func (f *fakeTemplateRepo) Get(_ context.Context, typ string) (*domain.Template, error) {
	if f.err != nil {
		return nil, f.err
	}
	t, ok := f.templates[typ]
	if !ok {
		return nil, domain.ErrTemplateNotFound
	}
	return &t, nil
}

func (f *fakeTemplateRepo) List(context.Context) ([]*domain.Template, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := []*domain.Template{}
	for _, t := range f.templates {
		out = append(out, &t)
	}
	return out, nil
}

func (f *fakeTemplateRepo) Upsert(_ context.Context, typ, body string) (*domain.Template, error) {
	if f.err != nil {
		return nil, f.err
	}
	t := domain.Template{Type: typ, Body: body}
	f.upserted = append(f.upserted, t)
	f.templates[typ] = t
	return &t, nil
}

type fakeMessageRepo struct {
	rows       []domain.Message
	nextID     int64
	insertErr  error
	outcomeErr error
}

func newFakeMessageRepo() *fakeMessageRepo {
	return &fakeMessageRepo{}
}

func (f *fakeMessageRepo) Insert(_ context.Context, m domain.Message) (*domain.Message, error) {
	if f.insertErr != nil {
		return nil, f.insertErr
	}
	f.nextID++
	m.ID = f.nextID
	m.Status = domain.StatusPending
	f.rows = append(f.rows, m)
	return &m, nil
}

func (f *fakeMessageRepo) UpdateOutcome(_ context.Context, id int64, o domain.MessageOutcome) (*domain.Message, error) {
	if f.outcomeErr != nil {
		return nil, f.outcomeErr
	}
	for i := range f.rows {
		if f.rows[i].ID != id {
			continue
		}
		f.rows[i].Status = o.Status
		f.rows[i].ProviderMessageID = o.ProviderMessageID
		f.rows[i].ResponseRaw = o.ResponseRaw
		f.rows[i].Error = o.Error
		row := f.rows[i]
		return &row, nil
	}
	return nil, domain.ErrNotFound
}

func (f *fakeMessageRepo) List(_ context.Context, creditNumber string) ([]*domain.Message, error) {
	out := []*domain.Message{}
	for i := range f.rows {
		if f.rows[i].CreditNumber == creditNumber {
			row := f.rows[i]
			out = append(out, &row)
		}
	}
	return out, nil
}

type fakeSender struct {
	sent         []sms.Message
	statusAtCall []domain.MessageStatus
	repo         *fakeMessageRepo
	resp         *sms.Response
	err          error
}

func (f *fakeSender) Send(_ context.Context, m sms.Message) (*sms.Response, error) {
	f.sent = append(f.sent, m)
	if f.repo != nil {
		for _, row := range f.repo.rows {
			f.statusAtCall = append(f.statusAtCall, row.Status)
		}
	}
	return f.resp, f.err
}
