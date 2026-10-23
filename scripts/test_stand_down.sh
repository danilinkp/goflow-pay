#!/usr/bin/env bash
set -e

COMPOSE="docker compose -f docker-compose.test.yml"

echo "--> Stopping test stand containers..."
$COMPOSE rm -sf gateway transactions accounts auth > /dev/null 2>&1 || true

# Teardown runs inside a container so the host needs neither psql nor redis-cli
$COMPOSE run --rm --no-deps stand-init ./scripts/test_teardown.sh > /dev/null 2>&1 || true
$COMPOSE rm -sf stand-init > /dev/null 2>&1 || true
