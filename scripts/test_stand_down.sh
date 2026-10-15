#!/usr/bin/env bash
set -e

if [ -f /tmp/goflow_test_stand.pids ]; then
    echo "--> Stopping test stand microservices..."
    PIDS=$(cat /tmp/goflow_test_stand.pids)
    kill $PIDS 2>/dev/null || true
    rm -f /tmp/goflow_test_stand.pids
fi

# Cleanup database state
./scripts/test_teardown.sh > /dev/null 2>&1 || true
