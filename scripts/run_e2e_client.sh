#!/bin/sh
# Runs the E2E client in its own container on a network shared only with gateway,
# so the services and databases under test are unreachable from it.
# POSIX sh on purpose: the CI job image (docker:26) has no bash.
set -u

COMPOSE="docker compose -f docker-compose.test.yml"
NAME="e2e-client-$$-$(date +%s)"
RESULTS_DIR="${ALLURE_RESULTS_DIR:-}"

$COMPOSE run --no-deps --name "$NAME" \
    -e ALLURE_RESULTS_DIR=/app/reports/allure-results \
    -e SKIP_E2E_TESTS="${SKIP_E2E_TESTS:-}" \
    e2e-client
STATUS=$?

if [ -n "$RESULTS_DIR" ]; then
    mkdir -p "$RESULTS_DIR"
    docker cp "$NAME:/app/reports/allure-results/." "$RESULTS_DIR/" > /dev/null 2>&1 || true
fi
docker rm -f "$NAME" > /dev/null 2>&1 || true

exit $STATUS
