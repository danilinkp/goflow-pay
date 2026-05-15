
DB_DRIVER=postgres
DB_DSN="postgres://postgres:postgres@localhost:5433/account_service?sslmode=disable"
MIG_DIR="./migrations"

migrate-up:
	goose -dir $(MIG_DIR) $(DB_DRIVER) $(DB_DSN) up

migrate-down:
	goose -dir $(MIG_DIR) $(DB_DRIVER) $(DB_DSN) down

migrate-reset:
	goose -dir $(MIG_DIR) $(DB_DRIVER) $(DB_DSN) reset

migrate-status:
	goose -dir $(MIG_DIR) $(DB_DRIVER) $(DB_DSN) status