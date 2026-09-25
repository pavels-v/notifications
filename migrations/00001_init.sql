-- +goose Up
CREATE TABLE customers (
    credit_number TEXT        PRIMARY KEY,
    phone_number  TEXT        NOT NULL,
    full_name     TEXT        NOT NULL,
    amount_minor  BIGINT      NOT NULL CHECK (amount_minor >= 0),
    currency      CHAR(3)     NOT NULL,
    due_date      DATE        NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- A message copies a value out of this table, it does not belong to a row in it.
CREATE TABLE templates (
    type       TEXT        PRIMARY KEY CHECK (type IN ('reminder', 'dunning', 'termination')),
    body       TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- RESTRICT because credit_number is the only thing tying a message to whom it concerns.
CREATE TABLE messages (
    id                  BIGSERIAL   PRIMARY KEY,
    credit_number       TEXT        NOT NULL REFERENCES customers (credit_number) ON DELETE RESTRICT,
    template_type       TEXT        NOT NULL,
    body                TEXT        NOT NULL,
    provider_message_id TEXT        NOT NULL DEFAULT '',
    status              TEXT        NOT NULL DEFAULT 'pending'
                                    CHECK (status IN ('pending', 'sent_to_provider', 'rejected_by_provider')),
    response_raw        TEXT        NOT NULL DEFAULT '',
    error               TEXT        NOT NULL DEFAULT '',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX messages_credit_number_created_at_idx
    ON messages (credit_number, created_at DESC);

-- +goose Down
DROP TABLE messages;
DROP TABLE templates;
DROP TABLE customers;
