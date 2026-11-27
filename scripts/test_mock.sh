#!/usr/bin/env bash
set -e

PORT=4010
SPEC_FILE="docs/openapi/openapi.yaml"

echo "============================================================"
echo " 1. Автоматическая валидация схем и примеров линтером Redocly"
echo "============================================================"
npx -y @redocly/cli lint "$SPEC_FILE"

echo ""
echo "============================================================"
echo " 2. Запуск Prism Mock Server на порту $PORT..."
echo "============================================================"

npx -y -p @faker-js/faker -p @stoplight/prism-cli prism mock "$SPEC_FILE" -p $PORT > /dev/null 2>&1 &
PRISM_PID=$!

cleanup() {
    echo ""
    echo "Остановка mock-сервера (PID: $PRISM_PID)..."
    kill "$PRISM_PID" 2>/dev/null || true
}
trap cleanup EXIT

echo -n "Ожидание запуска сервера"
for i in {1..20}; do
    if curl -s "http://127.0.0.1:$PORT/api/v2/accounts" -H "Authorization: Bearer test" >/dev/null 2>&1; then
        echo " -> Готов!"
        break
    fi
    echo -n "."
    sleep 1
done

echo ""
echo "============================================================"
echo " 3. Воспроизводимое тестирование сценариев на Mock-сервере"
echo "============================================================"

pass_count=0

test_req() {
    local desc="$1"
    local expected_code="$2"
    local method="$3"
    local path="$4"
    shift 4
    local extra_args=("$@")

    echo -n "Тест: $desc... "
    local http_code
    http_code=$(curl -s -o /dev/null -w "%{http_code}" -X "$method" "http://127.0.0.1:$PORT$path" "${extra_args[@]}")

    if [ "$http_code" -eq "$expected_code" ]; then
        echo -e "\033[32m[PASS]\033[0m (HTTP $http_code)"
        pass_count=$((pass_count + 1))
    else
        echo -e "\033[31m[FAIL]\033[0m (Ожидался $expected_code, получен $http_code)"
    fi
}

# 1. Аутентификация (Login)
test_req "POST /api/v2/auth/login (Вход в систему)" 200 POST "/api/v2/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"email":"finance@acme-corp.com","password":"SecureP@ssw0rd2026!"}'

# 2. Получение списка счетов (Read List)
test_req "GET /api/v2/accounts (Список счетов с токеном)" 200 GET "/api/v2/accounts" \
    -H "Authorization: Bearer mock-jwt-token"

# 3. Частичное обновление счета (PATCH / Partial Update)
test_req "PATCH /api/v2/accounts/{id} (Частичное обновление)" 200 PATCH "/api/v2/accounts/3fa85f64-5717-4562-b3fc-2c963f66afa6" \
    -H "Authorization: Bearer mock-jwt-token" \
    -H "Content-Type: application/merge-patch+json" \
    -d '{"status":"INACTIVE"}'

# 4. Создание платежа (Create Transaction с Idempotency-Key)
test_req "POST /api/v2/transactions (Создание перевода)" 201 POST "/api/v2/transactions" \
    -H "Authorization: Bearer mock-jwt-token" \
    -H "Content-Type: application/json" \
    -H "Idempotency-Key: 9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d" \
    -d '{"from_account_id":"3fa85f64-5717-4562-b3fc-2c963f66afa6","to_account_id":"4fa85f64-5717-4562-b3fc-2c963f66afa7","amount":2500000,"currency":"RUB"}'

# 5. Проверка правил безопасности (Security validation без токена)
test_req "GET /api/v2/transactions (Отказ без токена)" 401 GET "/api/v2/transactions"

# 6. Отвязка / удаление банковского счета (DELETE)
test_req "DELETE /api/v2/bank-accounts/{id} (Отвязка счета)" 204 DELETE "/api/v2/bank-accounts/550e8400-e29b-41d4-a716-446655440000" \
    -H "Authorization: Bearer mock-jwt-token"

# 7. Исключение (увольнение) сотрудника из компании (DELETE)
test_req "DELETE /api/v2/companies/{companyId}/users/{userId} (Увольнение)" 204 DELETE "/api/v2/companies/a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11/users/e3a12345-6789-4bcd-8ef0-123456789abc" \
    -H "Authorization: Bearer mock-jwt-token"

# 8. Реестр всех компаний (GET для администратора платформы)
test_req "GET /api/v2/companies (Реестр всех компаний)" 200 GET "/api/v2/companies" \
    -H "Authorization: Bearer mock-jwt-token"

echo ""
echo "============================================================"
echo -e "Итог: \033[32m$pass_count из 8 тестов успешно пройдено!\033[0m"
echo "============================================================"
