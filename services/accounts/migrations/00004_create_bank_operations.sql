-- +goose Up
SELECT 'up SQL query';
CREATE TABLE IF NOT EXISTS bank_operations
(
    bank_operation_id UUID PRIMARY KEY,
    account_id        UUID             NOT NULL REFERENCES accounts (account_id),
    bank_account_id   UUID             NOT NULL REFERENCES bank_accounts (bank_account_id),
    initiator_id      UUID             NOT NULL,
    bank_name         VARCHAR(255)     NOT NULL,
    operation_type    operation_type   NOT NULL,
    operation_status  operation_status NOT NULL DEFAULT 'pending',
    amount            BIGINT           NOT NULL,
    balance_after     BIGINT           NOT NULL,
    idempotency_key   VARCHAR(255)     NOT NULL UNIQUE,
    external_id       VARCHAR(255)     NOT NULL DEFAULT '',
    updated_at        TIMESTAMP        NOT NULL DEFAULT NOW(),
    created_at        TIMESTAMP        NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_bank_operations_account_id ON bank_operations (account_id);
CREATE INDEX idx_bank_operations_idempotency_key ON bank_operations (idempotency_key);

-- +goose Down
SELECT 'down SQL query';
DROP TABLE IF EXISTS bank_operations;
