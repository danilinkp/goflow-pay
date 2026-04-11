-- +goose Up
SELECT 'up SQL query';

CREATE TYPE account_status AS ENUM (
    'active',
    'inactive'
    );

CREATE TYPE operation_status AS ENUM (
    'pending',
    'success',
    'failed'
    );


CREATE TYPE operation_type AS ENUM (
    'deposit',
    'withdrawal'
    );

CREATE TYPE currency AS ENUM (
    'EUR',
    'USD',
    'RUB',
    'CNY'
    );

-- +goose Down
SELECT 'down SQL query';

DROP TYPE IF EXISTS account_status CASCADE;
DROP TYPE IF EXISTS operation_status CASCADE;
DROP TYPE IF EXISTS operation_type CASCADE;
DROP TYPE IF EXISTS currency CASCADE;
