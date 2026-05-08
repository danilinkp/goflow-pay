-- +goose Up
SELECT 'up SQL query';

CREATE TABLE IF NOT EXISTS outbox
(
    id             UUID PRIMARY KEY,
    aggregate_id   UUID         NOT NULL,
    aggregate_type VARCHAR      NOT NULL,
    event_type     VARCHAR(100) NOT NULL,
    topic          VARCHAR      NOT NULL,
    payload        JSONB        NOT NULL,
    failed_reason  TEXT,
    attempts       INT                   DEFAULT 0,
    published_at   TIMESTAMP,
    created_at     TIMESTAMP    NOT NULL DEFAULT NOW()
);

-- +goose Down
SELECT 'down SQL query';
DROP TABLE IF EXISTS outbox;