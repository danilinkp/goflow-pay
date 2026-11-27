#!/usr/bin/env bash
set -euo pipefail

# Runs inside the stand-init container: services read configs and JWT keys
# from the shared named volume, because bind mounts break under the CI runner.
STAND_DIR="${STAND_DIR:-/stand}"
PG_HOST="${PG_HOST:-postgres-test}"
PG_PORT="${PG_PORT:-5432}"
REDIS_HOST="${REDIS_HOST:-redis-test}"
REDIS_PORT="${REDIS_PORT:-6379}"

mkdir -p "$STAND_DIR/config" "$STAND_DIR/certs"

if [ ! -s "$STAND_DIR/certs/private.pem" ] || [ ! -s "$STAND_DIR/certs/public.pem" ]; then
    echo "--> Generating test JWT RSA keys..."
    openssl genpkey -algorithm RSA -out "$STAND_DIR/certs/private.pem" -pkeyopt rsa_keygen_bits:2048 2>/dev/null
    openssl rsa -pubout -in "$STAND_DIR/certs/private.pem" -out "$STAND_DIR/certs/public.pem" 2>/dev/null
fi
chmod 644 "$STAND_DIR/certs/"*.pem

# gRPC targets use passthrough:/// because the default dns resolver also looks up
# _grpc_config TXT records, which Docker DNS forwards upstream; the first call then
# stalls past the gateway's 4s HTTP timeout
echo "--> Generating service configs in $STAND_DIR/config..."
cat <<EOF > "$STAND_DIR/config/auth.yaml"
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

cat <<EOF > "$STAND_DIR/config/accounts.yaml"
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
  port: 50051
  timeout: 5s
kafka:
  brokers: "localhost:9092"
outbox:
  interval: 60s
  batch_size: 10
bank_gateway:
  failure_rate: 0.0
EOF

cat <<EOF > "$STAND_DIR/config/transactions.yaml"
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
  port: 50051
  timeout: 5s
grpc_client:
  addr: "passthrough:///accounts:50051"
kafka:
  brokers: "localhost:9092"
outbox:
  interval: 60s
  batch_size: 10
recover:
  interval: 60s
  stale_after: 30s
EOF

cat <<EOF > "$STAND_DIR/config/gateway.yaml"
env: "test"
http_server:
  address: "0.0.0.0:8080"
  timeout: 4s
  idle_timeout: 60s
grpc_clients:
  auth_address: "passthrough:///auth:50051"
  account_address: "passthrough:///accounts:50051"
  transaction_address: "passthrough:///transactions:50051"
EOF

./scripts/test_init.sh
