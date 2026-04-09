-- +goose Up
SELECT 'up SQL query';
CREATE TABLE IF NOT EXISTS bank_accounts
(
    bank_account_id    UUID PRIMARY KEY,
    company_id         UUID         NOT NULL,
    name               VARCHAR(255) NOT NULL,
    bic                VARCHAR(20)  NOT NULL,
    settlement_account VARCHAR(50)  NOT NULL,
    currency           currency     NOT NULL,
    created_at         TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMP    NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_bank_accounts_company_id ON bank_accounts (company_id);

-- +goose Down
SELECT 'down SQL query';
DROP TABLE IF EXISTS bank_accounts;