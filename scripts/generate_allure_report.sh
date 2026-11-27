#!/usr/bin/env bash
set -uo pipefail

REPORTS_DIR="$(pwd)/reports"
ALLURE_RESULTS="$REPORTS_DIR/allure-results"
ALLURE_REPORT="$REPORTS_DIR/allure-report"

# 1. Start test infrastructure (Postgres + Redis)
echo "=== 1. Подготовка тестовой инфраструктуры для отчёта ==="
docker compose -f docker-compose.test.yml up -d --wait postgres-test redis-test > /dev/null 2>&1 || true
./scripts/test_init.sh > /dev/null 2>&1 || true

rm -rf "$ALLURE_RESULTS" "$ALLURE_REPORT"
mkdir -p "$ALLURE_RESULTS"

# 2. Loop to build history trend (4 cycles)
ITERATIONS=4
for i in $(seq 1 $ITERATIONS); do
    echo "=== Прогон $i из $ITERATIONS (Пирамида тестирования) ==="
    rm -f "$ALLURE_RESULTS"/*-result.json "$ALLURE_RESULTS"/*-container.json
    [ -d "$ALLURE_REPORT/history" ] && cp -r "$ALLURE_REPORT/history" "$ALLURE_RESULTS/"
    
    printf '{"name":"Local Test Runner","type":"standalone","buildOrder":%d,"buildName":"Запуск #%d","reportName":"GoFlow Allure Report"}\n' "$i" "$i" > "$ALLURE_RESULTS/executor.json"
    
    UNIT_PASSED=true
    INTEG_PASSED=true

    # --------------------------------------------------------------------------
    # Этап 1: Unit-тесты (Лондонская и Классическая школы)
    # --------------------------------------------------------------------------
    if ! ALLURE_RESULTS_DIR="$ALLURE_RESULTS" go test -count=1 ./services/transactions/internal/service/... > /dev/null 2>&1 || \
       ! ALLURE_RESULTS_DIR="$ALLURE_RESULTS" go test -count=1 -run TestAccountService_Classic ./services/accounts/internal/services/... > /dev/null 2>&1; then
        echo "  [FAIL] Этап 1 (Unit-тесты) завершился с ошибкой!"
        UNIT_PASSED=false
    else
        echo "  [PASS] Этап 1 (Unit-тесты) успешно пройден."
    fi

    # --------------------------------------------------------------------------
    # Этап 2: Интеграционные тесты (Real PostgreSQL + Outbox)
    # Требование №8: Если предыдущий этап упал, последующие не запускаются,
    # а помечаются как skipped.
    # --------------------------------------------------------------------------
    if [ "$UNIT_PASSED" = false ]; then
        echo "  [SKIP] Этап 2 (Интеграционные тесты) пропущен из-за сбоя Unit-тестов."
        SKIP_INTEGRATION_TESTS=true ALLURE_RESULTS_DIR="$ALLURE_RESULTS" go test -count=1 -tags=integration ./services/accounts/internal/services/... -run "TestAccountService_.*_Integration" > /dev/null 2>&1 || true
        INTEG_PASSED=false
    else
        if ! ALLURE_RESULTS_DIR="$ALLURE_RESULTS" go test -count=1 -tags=integration ./services/accounts/internal/services/... -run "TestAccountService_.*_Integration" > /dev/null 2>&1; then
            echo "  [FAIL] Этап 2 (Интеграционные тесты) завершился с ошибкой!"
            INTEG_PASSED=false
        else
            echo "  [PASS] Этап 2 (Интеграционные тесты) успешно пройден."
        fi
    fi

    # --------------------------------------------------------------------------
    # Этап 3: Сквозные E2E-тесты (Gateway HTTP API)
    # --------------------------------------------------------------------------
    if [ "$UNIT_PASSED" = false ] || [ "$INTEG_PASSED" = false ]; then
        echo "  [SKIP] Этап 3 (E2E-тесты) пропущен из-за сбоя на предыдущем этапе."
        SKIP_E2E_TESTS=true ALLURE_RESULTS_DIR="$ALLURE_RESULTS" ./scripts/run_e2e_client.sh > /dev/null 2>&1 || true
    else
        ./scripts/test_stand_up.sh > /dev/null 2>&1 || true
        if ! ALLURE_RESULTS_DIR="$ALLURE_RESULTS" ./scripts/run_e2e_client.sh > /dev/null 2>&1; then
            echo "  [FAIL] Этап 3 (E2E-тесты) завершился с ошибкой!"
        else
            echo "  [PASS] Этап 3 (E2E-тесты) успешно пройден."
        fi
        ./scripts/test_stand_down.sh > /dev/null 2>&1 || true
    fi

    # --------------------------------------------------------------------------
    # Генерация Allure-отчёта (выполняется ВСЕГДА, даже если этапы падали)
    # Требование №8: Не запускавшиеся тесты помечаются как skipped
    # --------------------------------------------------------------------------
    python3 scripts/mark_skipped_tests.py "$ALLURE_RESULTS" > /dev/null 2>&1 || true
    npx -y allure-commandline generate "$ALLURE_RESULTS" -o "$ALLURE_REPORT" --clean > /dev/null 2>&1
done

# Остановка тестовых контейнеров
docker compose -f docker-compose.test.yml down -v > /dev/null 2>&1 || true

echo "=========================================================================="
echo " [УСПЕХ] Автоматический Allure-отчет сформирован в: $ALLURE_REPORT"
echo " Запустите 'make test-report-open' для просмотра в браузере."
echo "=========================================================================="
