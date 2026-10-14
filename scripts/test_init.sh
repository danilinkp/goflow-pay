#!/usr/bin/env bash
set -euo pipefail

# Determine host/port based on environment (Docker container vs Local host)
if [ "${TEST_ENV:-}" = "docker" ]; then
    PG_HOST="${PG_HOST:-postgres-test}"
    PG_PORT="${PG_PORT:-5432}"
    REDIS_HOST="${REDIS_HOST:-redis-test}"
    REDIS_PORT="${REDIS_PORT:-6379}"
else
    PG_HOST="${PG_HOST:-localhost}"
    PG_PORT="${PG_PORT:-5439}"
    REDIS_HOST="${REDIS_HOST:-localhost}"
    REDIS_PORT="${REDIS_PORT:-6389}"
fi

PG_USER="${PG_USER:-postgres}"
PG_PASSWORD="${PG_PASSWORD:-postgres}"
ACCOUNTS_DB="${ACCOUNTS_DB_NAME:-account_test}"
TRANSACTIONS_DB="${TRANSACTIONS_DB_NAME:-transaction_test}"

export PGPASSWORD="$PG_PASSWORD"

echo "=========================================================="
echo " [TEST-INIT] Initializing test stand infrastructure..."
echo " Target PostgreSQL: $PG_HOST:$PG_PORT ($PG_USER)"
echo " Target Redis:      $REDIS_HOST:$REDIS_PORT"
echo "=========================================================="

# 1. Wait for PostgreSQL to be ready
echo "--> Waiting for PostgreSQL at $PG_HOST:$PG_PORT..."
MAX_RETRIES=30
COUNT=0
until pg_isready -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" > /dev/null 2>&1; do
    COUNT=$((COUNT + 1))
    if [ $COUNT -ge $MAX_RETRIES ]; then
        echo "Error: Timed out waiting for PostgreSQL at $PG_HOST:$PG_PORT"
        exit 1
    fi
    sleep 1
done
echo "--> PostgreSQL is ready!"

# 2. Wait for Redis to be ready (if redis-cli available)
if command -v redis-cli > /dev/null 2>&1; then
    echo "--> Waiting for Redis at $REDIS_HOST:$REDIS_PORT..."
    COUNT=0
    until redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" ping 2>&1 | grep -q PONG; do
        COUNT=$((COUNT + 1))
        if [ $COUNT -ge $MAX_RETRIES ]; then
            echo "Warning: Timed out waiting for Redis at $REDIS_HOST:$REDIS_PORT"
            break
        fi
        sleep 1
    done
    echo "--> Redis is ready!"
fi

# 3. Ensure test databases exist
ensure_db() {
    local db_name="$1"
    local exists
    exists=$(psql -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname='$db_name'")
    if [ "$exists" != "1" ]; then
        echo "--> Creating database '$db_name'..."
        psql -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d postgres -c "CREATE DATABASE $db_name;"
    else
        echo "--> Database '$db_name' already exists."
    fi
}

ensure_db "$ACCOUNTS_DB"
ensure_db "$TRANSACTIONS_DB"

# 4. Run Goose migrations
echo "--> Applying Goose migrations for Accounts ($ACCOUNTS_DB)..."
ACCOUNTS_DSN="postgres://$PG_USER:$PG_PASSWORD@$PG_HOST:$PG_PORT/$ACCOUNTS_DB?sslmode=disable"
goose -dir services/accounts/migrations postgres "$ACCOUNTS_DSN" up

echo "--> Applying Goose migrations for Transactions ($TRANSACTIONS_DB)..."
TX_DSN="postgres://$PG_USER:$PG_PASSWORD@$PG_HOST:$PG_PORT/$TRANSACTIONS_DB?sslmode=disable"
goose -dir services/transactions/migrations postgres "$TX_DSN" up

echo "=========================================================="
echo " [TEST-INIT] Test stand successfully initialized!"
echo "=========================================================="
