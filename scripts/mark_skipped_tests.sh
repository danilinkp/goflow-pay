#!/bin/sh
set -eu

RESULTS_DIR="${1:-reports/allure-results}"
mkdir -p "$RESULTS_DIR"

mark_if_missing() {
    TEST_NAME="$1"
    SUITE_NAME="$2"
    PARENT_SUITE="$3"
    PACKAGE_NAME="$4"
    REASON="$5"

    # Check if a result file containing this test name already exists
    if grep -r "\"name\": \"$TEST_NAME\"" "$RESULTS_DIR" >/dev/null 2>&1 || grep -r "\"name\":\"$TEST_NAME\"" "$RESULTS_DIR" >/dev/null 2>&1; then
        return 0
    fi

    if [ -f /proc/sys/kernel/random/uuid ]; then
        UUID="$(cat /proc/sys/kernel/random/uuid)"
    elif command -v uuidgen >/dev/null 2>&1; then
        UUID="$(uuidgen)"
    else
        UUID="skip-$(date +%s)-$$"
    fi

    cat <<EOF > "$RESULTS_DIR/${UUID}-result.json"
{
  "uuid": "$UUID",
  "historyId": "$TEST_NAME",
  "name": "$TEST_NAME",
  "fullName": "$PACKAGE_NAME.$TEST_NAME",
  "status": "skipped",
  "statusDetails": {
    "message": "$REASON"
  },
  "stage": "finished",
  "labels": [
    {"name": "parentSuite", "value": "$PARENT_SUITE"},
    {"name": "suite", "value": "$SUITE_NAME"},
    {"name": "package", "value": "$PACKAGE_NAME"}
  ]
}
EOF
    echo "[Requirement 8] Помечен как SKIPPED: $TEST_NAME"
}

mark_if_missing "TestAccountService_ReserveWithdraw_Integration" "Интеграционное тестирование счетов (Real PostgreSQL)" "Интеграционные тесты" "accounts/internal/services" "Этап integration-tests пропущен: сбой на предыдущем этапе (unit-tests)"
mark_if_missing "TestAccountService_RollbackAndCompensation_Integration" "Интеграционное тестирование счетов (Real PostgreSQL)" "Интеграционные тесты" "accounts/internal/services" "Этап integration-tests пропущен: сбой на предыдущем этапе (unit-tests)"
mark_if_missing "TestAccountService_GenerateStatement_Integration" "Интеграционное тестирование счетов (Real PostgreSQL)" "Интеграционные тесты" "accounts/internal/services" "Этап integration-tests пропущен: сбой на предыдущем этапе (unit-tests)"
mark_if_missing "TestE2E_CompanyTransferAndStatementFlow" "GoFlow Pay Platform E2E" "E2E Тестирование платформы" "tests/e2e" "Этап e2e-tests пропущен: сбой на предыдущем этапе пирамиды тестирования"
