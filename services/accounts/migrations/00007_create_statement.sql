-- +goose Up
SELECT 'up SQL query';

CREATE TABLE IF NOT EXISTS statements
(
    statement_id    UUID PRIMARY KEY,
    account_id      UUID      NOT NULL REFERENCES accounts (account_id),
    company_id      UUID      NOT NULL,
    initiator_id    UUID      NOT NULL,
    period_from     TIMESTAMP NOT NULL,
    period_to       TIMESTAMP NOT NULL,
    opening_balance BIGINT    NOT NULL DEFAULT 0,
    closing_balance BIGINT    NOT NULL DEFAULT 0,
    total_debit     BIGINT    NOT NULL DEFAULT 0,
    total_credit    BIGINT    NOT NULL DEFAULT 0,
    currency        currency  NOT NULL,
    entries         JSONB     NOT NULL,
    updated_at      TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at      TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_by_date ON statements (account_id, period_from, period_to);

-- +goose Down
SELECT 'down SQL query';
DROP TABLE IF EXISTS statements;