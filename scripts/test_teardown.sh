#!/usr/bin/env bash
set -euo pipefail

# Determine host/port based on environment
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
AUTH_DB="${AUTH_DB_NAME:-auth_test}"

export PGPASSWORD="$PG_PASSWORD"

echo "=========================================================="
echo " [TEST-TEARDOWN] Purging and resetting test state..."
echo " Target PostgreSQL: $PG_HOST:$PG_PORT"
echo " Target Redis:      $REDIS_HOST:$REDIS_PORT"
echo "=========================================================="

truncate_db_tables() {
    local db_name="$1"
    echo "--> Truncating tables in database '$db_name'..."

    # Check if DB exists
    local db_exists
    db_exists=$(psql -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname='$db_name'" 2>/dev/null || echo "0")
    if [ "$db_exists" != "1" ]; then
        echo "Database '$db_name' does not exist, skipping."
        return 0
    fi

    # Query all public tables except migration metadata
    local tables
    tables=$(psql -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d "$db_name" -tAc "
        SELECT string_agg(quote_ident(table_name), ', ')
        FROM information_schema.tables
        WHERE table_schema = 'public'
          AND table_type = 'BASE TABLE'
          AND table_name != 'goose_db_version';
    " 2>/dev/null || echo "")

    if [ -n "$tables" ] && [ "$tables" != "" ]; then
        echo "    Executing: TRUNCATE TABLE $tables RESTART IDENTITY CASCADE;"
        psql -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d "$db_name" -c "TRUNCATE TABLE $tables RESTART IDENTITY CASCADE;"
    else
        echo "    No user tables found in '$db_name'."
    fi
}

# 1. Truncate PostgreSQL databases
truncate_db_tables "$ACCOUNTS_DB"
truncate_db_tables "$TRANSACTIONS_DB"
truncate_db_tables "$AUTH_DB"

# 2. Reset Redis cache / sessions
echo "--> Flushing Redis cache and user sessions..."
if command -v redis-cli > /dev/null 2>&1; then
    redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" FLUSHALL || true
    echo "    Redis flushed via local redis-cli."
elif command -v docker > /dev/null 2>&1 && docker ps -q -f name=redis-test | grep -q .; then
    REDIS_CID=$(docker ps -q -f name=redis-test | head -n 1)
    docker exec "$REDIS_CID" redis-cli FLUSHALL || true
    echo "    Redis flushed via docker exec ($REDIS_CID)."
else
    echo "    redis-cli / docker not available on host, skipping redis flush."
fi

echo "=========================================================="
echo " [TEST-TEARDOWN] All test state successfully cleaned!"
echo "=========================================================="
