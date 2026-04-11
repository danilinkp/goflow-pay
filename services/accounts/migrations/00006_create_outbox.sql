-- +goose Up
SELECT 'up SQL query';

CREATE TABLE IF NOT EXISTS outbox
(
    id         UUID PRIMARY KEY,
    event_type VARCHAR(100) NOT NULL,
    payload    JSONB        NOT NULL,
    status     VARCHAR(20)  NOT NULL DEFAULT 'pending',
    error      TEXT,
    created_at TIMESTAMP    NOT NULL DEFAULT NOW(),
    sent_at    TIMESTAMP
);

CREATE INDEX idx_outbox_status ON outbox (status) WHERE status = 'pending';

-- +goose Down
SELECT 'down SQL query';
DROP TABLE IF EXISTS outbox;