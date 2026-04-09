-- +goose Up
SELECT 'up SQL query';

CREATE TABLE IF NOT EXISTS notifications
(
    notification_id UUID PRIMARY KEY,
    user_id         UUID      NOT NULL,
    title           TEXT,
    message         TEXT,
    source_id       uuid      NOT NULL,
    updated_at      TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at      TIMESTAMP NOT NULL DEFAULT NOW()
);


-- +goose Down
SELECT 'down SQL query';
DROP TABLE IF EXISTS notifications;