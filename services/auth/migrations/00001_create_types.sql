-- +goose Up
SELECT 'up SQL query';

CREATE TYPE user_role AS ENUM (
    'admin',
    'company_admin',
    'employee',
    'guest'
    );

-- +goose Down
SELECT 'down SQL query';
DROP TYPE IF EXISTS user_role;