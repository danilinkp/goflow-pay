-- +goose Up
SELECT 'up SQL query';

CREATE TYPE transaction_status AS ENUM (
    'pending',
    'success',
    'failed',
    'processing'
    );

CREATE TYPE currency AS ENUM (
    'EUR',
    'USD',
    'RUB',
    'CNY'
    );

-- +goose Down
SELECT 'down SQL query';

DROP TYPE IF EXISTS transaction_status;
DROP TYPE IF EXISTS currency;