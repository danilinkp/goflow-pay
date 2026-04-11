-- +goose Up
SELECT 'up SQL query';

CREATE TABLE IF NOT EXISTS users
(
    user_id       uuid PRIMARY KEY,
    company_id    uuid REFERENCES companies (company_id) NOT NULL,
    login         VARCHAR(70)                            NOT NULL,
    email         VARCHAR(255)                           NOT NULL UNIQUE,
    password_hash VARCHAR(255)                           NOT NULL,
    role          user_role                              NOT NULL,
    updated_at    TIMESTAMP                              NOT NULL DEFAULT NOW(),
    created_at    TIMESTAMP                              NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_company_id ON users (company_id);
CREATE INDEX IF NOT EXISTS idx_email ON users (email);

-- +goose Down
SELECT 'down SQL query';
DROP TABLE IF EXISTS users;