#!/usr/bin/env bash
set -euo pipefail

COMPOSE="docker compose -f docker-compose.test.yml"

# Images are built only when missing, so the stand starts without network access;
# run `make test-stand-build` after changing service code or stand scripts
echo "--> Starting test stand containers (Auth, Accounts, Transactions, Gateway)..."
if ! $COMPOSE up -d --wait gateway; then
    echo "--> Test stand failed to start, service logs:"
    $COMPOSE logs --no-color stand-init auth accounts transactions gateway || true
    exit 1
fi

echo "--> Gateway is ready on http://127.0.0.1:18080"
