-- +goose Up
SELECT 'up SQL query';

CREATE TABLE IF NOT EXISTS account_operations
(
    operation_id     UUID PRIMARY KEY,
    account_id       UUID             NOT NULL REFERENCES accounts (account_id),
    transaction_id   UUID             NOT NULL,
    operation_type   operation_type   NOT NULL,
    operation_status operation_status NOT NULL DEFAULT 'pending',
    amount           BIGINT           NOT NULL,
    updated_at       TIMESTAMP        NOT NULL DEFAULT NOW(),
    created_at       TIMESTAMP        NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_account_operations_account_id ON account_operations (account_id);
CREATE INDEX idx_account_operations_transaction_id ON account_operations (transaction_id);

-- +goose Down
SELECT 'down SQL query';
DROP TABLE IF EXISTS account_operations;