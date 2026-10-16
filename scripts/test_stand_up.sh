#!/usr/bin/env bash
set -euo pipefail

# 1. Determine environment
if [ "${TEST_ENV:-}" = "docker" ]; then
    PG_HOST="${PG_HOST:-postgres-test}"
    PG_PORT="${PG_PORT:-5432}"
    REDIS_HOST="${REDIS_HOST:-redis-test}"
    REDIS_PORT="${REDIS_PORT:-6379}"
else
    PG_HOST="${PG_HOST:-127.0.0.1}"
    PG_PORT="${PG_PORT:-5439}"
    REDIS_HOST="${REDIS_HOST:-127.0.0.1}"
    REDIS_PORT="${REDIS_PORT:-6389}"
fi

export DATABASE_USER="${PG_USER:-postgres}"
export DATABASE_PASSWORD="${PG_PASSWORD:-postgres}"
export JWT_PRIVATE_KEY_PATH="$(pwd)/certs/private.pem"
export JWT_PUBLIC_KEY_PATH="$(pwd)/certs/public.pem"
export BOOTSTRAP_TOKEN="test-bootstrap-token"

LOG_DIR="/tmp/goflow_logs"
mkdir -p config/test bin "$LOG_DIR"

# 2. Ensure test JWT RSA keys exist
if [ ! -s "$JWT_PRIVATE_KEY_PATH" ] || [ ! -s "$JWT_PUBLIC_KEY_PATH" ]; then
    echo "--> Generating test JWT RSA keys in certs/..."
    mkdir -p "$(dirname "$JWT_PRIVATE_KEY_PATH")"
    openssl genpkey -algorithm RSA -out "$JWT_PRIVATE_KEY_PATH" -pkeyopt rsa_keygen_bits:2048 2>/dev/null
    openssl rsa -pubout -in "$JWT_PRIVATE_KEY_PATH" -out "$JWT_PUBLIC_KEY_PATH" 2>/dev/null
fi

# 3. Ensure test databases and migrations
echo "--> Initializing test databases and Goose migrations..."
./scripts/test_init.sh

# 4. Compile binaries if needed
echo "--> Ensuring service binaries are compiled in ./bin..."
go build -buildvcs=false -o ./bin/auth ./services/auth/cmd/auth_server
go build -buildvcs=false -o ./bin/accounts ./services/accounts/cmd/account_server
go build -buildvcs=false -o ./bin/transactions ./services/transactions/cmd/transaction_server
go build -buildvcs=false -o ./bin/gateway ./services/gateway/cmd/gateway

# 3. Generate test configs
cat <<EOF > config/test/auth.yaml
env: "test"
storage:
  type: postgres
  postgres:
    host: "$PG_HOST"
    port: "$PG_PORT"
    name: "auth_test"
    ssl_mode: "disable"
    connect_timeout: 5s
    max_retries_time: 2s
redis:
  host: "$REDIS_HOST"
  port: $REDIS_PORT
  password: ""
  db: 0
grpc:
  port: 50051
  timeout: 5s
jwt:
  ttl: 24h
hash_cost: 10
EOF

cat <<EOF > config/test/accounts.yaml
env: "test"
storage:
  type: "postgres"
  postgres:
    host: "$PG_HOST"
    port: "$PG_PORT"
    name: "account_test"
    ssl_mode: "disable"
    connect_timeout: 5s
    max_retries_time: 2s
grpc:
  port: 50052
  timeout: 5s
kafka:
  brokers: "localhost:9092"
outbox:
  interval: 60s
  batch_size: 10
bank_gateway:
  failure_rate: 0.0
EOF

cat <<EOF > config/test/transactions.yaml
env: "test"
storage:
  type: "postgres"
  postgres:
    host: "$PG_HOST"
    port: "$PG_PORT"
    name: "transaction_test"
    ssl_mode: "disable"
    connect_timeout: 5s
    max_retries_time: 2s
grpc:
  port: 50053
  timeout: 5s
grpc_client:
  addr: "127.0.0.1:50052"
kafka:
  brokers: "localhost:9092"
outbox:
  interval: 60s
  batch_size: 10
recover:
  interval: 60s
  stale_after: 30s
EOF

cat <<EOF > config/test/gateway.yaml
env: "test"
http_server:
  address: "0.0.0.0:8080"
  timeout: 4s
  idle_timeout: 60s
grpc_clients:
  auth_address: "127.0.0.1:50051"
  account_address: "127.0.0.1:50052"
  transaction_address: "127.0.0.1:50053"
EOF

# 4. Stop any previously running test stand instances
./scripts/test_stand_down.sh > /dev/null 2>&1 || true

echo "--> Starting microservices (Auth, Accounts, Transactions, Gateway)..."
nohup env CONFIG_PATH=config/test/auth.yaml \
    DATABASE_USER="$DATABASE_USER" \
    DATABASE_PASSWORD="$DATABASE_PASSWORD" \
    JWT_PRIVATE_KEY_PATH="$JWT_PRIVATE_KEY_PATH" \
    ./bin/auth < /dev/null > "$LOG_DIR/auth_test.log" 2>&1 &
PID_AUTH=$!

nohup env CONFIG_PATH=config/test/accounts.yaml \
    DATABASE_USER="$DATABASE_USER" \
    DATABASE_PASSWORD="$DATABASE_PASSWORD" \
    ./bin/accounts < /dev/null > "$LOG_DIR/accounts_test.log" 2>&1 &
PID_ACC=$!

nohup env CONFIG_PATH=config/test/transactions.yaml \
    DATABASE_USER="$DATABASE_USER" \
    DATABASE_PASSWORD="$DATABASE_PASSWORD" \
    ./bin/transactions < /dev/null > "$LOG_DIR/transactions_test.log" 2>&1 &
PID_TX=$!

sleep 1

nohup env CONFIG_PATH=config/test/gateway.yaml \
    JWT_PUBLIC_KEY_PATH="$JWT_PUBLIC_KEY_PATH" \
    BOOTSTRAP_TOKEN="$BOOTSTRAP_TOKEN" \
    ./bin/gateway < /dev/null > "$LOG_DIR/gateway_test.log" 2>&1 &
PID_GW=$!

disown -a 2>/dev/null || true
echo "$PID_AUTH $PID_ACC $PID_TX $PID_GW" > /tmp/goflow_test_stand.pids

# 5. Wait for Gateway readiness
echo -n "--> Waiting for Gateway on http://127.0.0.1:8080/swagger"
for i in {1..30}; do
    if curl -s -o /dev/null -w "%{http_code}" http://127.0.0.1:8080/swagger | grep -q "200"; then
        echo " [READY!]"
        exit 0
    fi
    echo -n "."
    sleep 0.5
done

echo " [FAILED]"
echo "Gateway logs:"
cat "$LOG_DIR/gateway_test.log" || true
./scripts/test_stand_down.sh
exit 1
