#!/usr/bin/env bash
# HTTP-проверки маршрутизации NGINX (ЛР4 по Web).
# Использование: ADMIN_USER=admin ADMIN_PASSWORD=... ./scripts/check_routes.sh [base_url]
set -uo pipefail

BASE="${1:-http://127.0.0.1:${NGINX_PORT:-80}}"
ADMIN_USER="${ADMIN_USER:-admin}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-}"

pass=0
fail=0

ok()  { echo "  OK   $1"; pass=$((pass + 1)); }
bad() { echo "  FAIL $1"; fail=$((fail + 1)); }

# check <описание> <ожидаемый код> <ожидаемая подстрока в теле или ""> <curl args...>
check() {
    local name="$1" want_code="$2" want_body="$3"
    shift 3
    local body code
    body=$(curl -s -o - -w '\n%{http_code}' "$@")
    code="${body##*$'\n'}"
    body="${body%$'\n'*}"

    if [[ "$code" != "$want_code" ]]; then
        bad "$name: код $code, ожидался $want_code"
        return
    fi
    if [[ -n "$want_body" && "$body" != *"$want_body"* ]]; then
        bad "$name: в ответе нет '$want_body'"
        return
    fi
    ok "$name ($code)"
}

# check_header <описание> <имя заголовка> <подстрока> <curl args...>
check_header() {
    local name="$1" header="$2" want="$3"
    shift 3
    local value
    value=$(curl -s -o /dev/null -D - "$@" | grep -i "^$header:" | tr -d '\r')
    if [[ "$value" != *"$want"* ]]; then
        bad "$name: заголовок '$value', ожидалось '$want'"
        return
    fi
    ok "$name ($value)"
}

echo "Проверка стенда $BASE"

echo "[статика]"
check "/ отдаёт приветственную страницу" 200 "GoFlow Pay" "$BASE/"
check "/img/dashboard.svg загружается" 200 "<svg" "$BASE/img/dashboard.svg"
check_header "/img/*.svg имеет тип SVG" "Content-Type" "image/svg+xml" "$BASE/img/dashboard.svg"
check "/reserved/ отдаёт ту же страницу" 200 "Эскизы будущего приложения" "$BASE/reserved/"
if [[ "$(curl -s "$BASE/")" == "$(curl -s "$BASE/reserved/")" ]]; then
    ok "/ и /reserved/ совпадают побайтно"
else
    bad "/ и /reserved/ отличаются"
fi
check "/reserved без слэша редиректит" 301 "" "$BASE/reserved"
check "/management содержит ссылки" 200 'href="/admin/"' "$BASE/management"
check "/documentation отдаёт README.md" 200 "# GoFlow Pay" "$BASE/documentation"
check_header "/documentation в UTF-8" "Content-Type" "charset=utf-8" "$BASE/documentation"
check "несуществующий файл -> 404" 404 "" "$BASE/no-such-file.html"

echo "[API: префикс сохраняется и доходит до gateway]"
check "GET /api/v1/companies/accounts без токена" 401 "" "$BASE/api/v1/companies/accounts"
check "GET /api/v2/accounts без токена" 401 "" "$BASE/api/v2/accounts"
check "POST /api/v1/auth/login доходит до auth-сервиса" 401 "invalid credentials" -X POST -H 'Content-Type: application/json' -d '{}' "$BASE/api/v1/auth/login"
check "POST /api/v2/auth/login доходит до auth-сервиса" 401 "invalid credentials" -X POST -H 'Content-Type: application/json' -d '{}' "$BASE/api/v2/auth/login"
check "GET /api/v3/unknown -> 404 от gateway" 404 "" "$BASE/api/v3/unknown"

echo "[Swagger UI не перекрывает API]"
check "/api/v2/docs отдаёт Swagger UI" 200 "swagger-ui" -L "$BASE/api/v2/docs"
check "/api/v2/docs/openapi.yaml отдаёт спецификацию" 200 "openapi:" "$BASE/api/v2/docs/openapi.yaml"
check "/api/v2/accounts после docs всё ещё API" 401 "" "$BASE/api/v2/accounts"

echo "[legacy SPA]"
check "/legacy редиректит на /legacy/" 301 "" "$BASE/legacy"
check "/legacy/ отдаёт SPA" 200 'id="root"' "$BASE/legacy/"
asset=$(curl -s "$BASE/legacy/" | grep -o '/legacy/assets/[^"]*\.js' | head -n1)
if [[ -n "$asset" ]]; then
    check "ассет SPA $asset" 200 "" "$BASE$asset"
else
    bad "в /legacy/ нет ссылки на /legacy/assets/*.js"
fi
check "deep link /legacy/dashboard отдаёт SPA" 200 'id="root"' "$BASE/legacy/dashboard"

echo "[служебные]"
check "/status (stub_status)" 200 "Active connections" "$BASE/status"
check "/health (liveness)" 200 '"status":"ok"' "$BASE/health"
check "/ready (readiness)" 200 '"status":"ready"' "$BASE/ready"

echo "[ограничение доступа к /admin]"
check "/admin без учётных данных" 401 "" "$BASE/admin/"
check "/admin с неверным паролем" 401 "" -u "$ADMIN_USER:wrong-password" "$BASE/admin/"
if [[ -n "$ADMIN_PASSWORD" ]]; then
    check "/admin с Basic Auth открывает Adminer" 200 "Adminer" -u "$ADMIN_USER:$ADMIN_PASSWORD" "$BASE/admin/"
else
    echo "  SKIP /admin с Basic Auth: ADMIN_PASSWORD не задан"
fi

echo
echo "Итого: $pass OK, $fail FAIL"
[[ "$fail" -eq 0 ]]
