-- +goose Up
SELECT 'up SQL query';

CREATE TABLE IF NOT EXISTS companies
(
    company_id  uuid PRIMARY KEY,
    name        VARCHAR(55)  NOT NULL,
    invite_code VARCHAR(255) NOT NULL UNIQUE,
    updated_at  TIMESTAMP    NOT NULL DEFAULT NOW(),
    created_at  TIMESTAMP    NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_invite_code ON companies (invite_code);

-- +goose Down
SELECT 'down SQL query';
DROP TABLE IF EXISTS companies;