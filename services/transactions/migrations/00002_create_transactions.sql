-- +goose Up
SELECT 'up SQL query';

CREATE TABLE IF NOT EXISTS transactions
(
    transaction_id  uuid PRIMARY KEY,
    initiator_id    uuid               NOT NULL,
    from_account_id uuid               NOT NULL,
    to_account_id   uuid               NOT NULL,
    amount          BIGINT             NOT NULL CHECK ( amount >= 0 ),
    currency        currency           NOT NULL,
    idempotency_key VARCHAR(255)       NOT NULL,
    status          transaction_status NOT NULL DEFAULT 'pending',
    updated_at      TIMESTAMP          NOT NULL DEFAULT NOW(),
    created_at      TIMESTAMP          NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_transactions_idempotency_key ON transactions (idempotency_key);
CREATE INDEX IF NOT EXISTS idx_transactions_from_created ON transactions (from_account_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_transactions_to_created ON transactions (to_account_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_transactions_pending ON transactions (created_at) WHERE status = 'pending';


-- +goose Down
SELECT 'down SQL query';

DROP TABLE IF EXISTS transactions;