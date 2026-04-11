-- +goose Up
SELECT 'up SQL query';

CREATE TABLE IF NOT EXISTS accounts
(
    account_id UUID PRIMARY KEY,
    company_id UUID           NOT NULL,
    balance    BIGINT         NOT NULL DEFAULT 0,
    currency   currency       NOT NULL,
    status     account_status NOT NULL DEFAULT 'active',
    updated_at TIMESTAMP      NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP      NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_accounts_company_id ON accounts (company_id);

-- +goose Down
SELECT 'down SQL query';

DROP TABLE IF EXISTS accounts CASCADE;