#!/usr/bin/env bash
set -uo pipefail

BASE="${1:-http://127.0.0.1:${NGINX_PORT:-80}}"
ADMIN_USER="${ADMIN_USER:-admin}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-}"

pass=0
fail=0

ok()  { echo "  OK   $1"; pass=$((pass + 1)); }
bad() { echo "  FAIL $1"; fail=$((fail + 1)); }

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

echo "[лимит запросов к API]"
limited=$(docker compose exec -T nginx sh -c 'for i in $(seq 1 40); do curl -s -o /dev/null -w "%{http_code}\n" http://127.0.0.1/api/v2/accounts; done' | grep -c '^429$')
if [[ "$limited" -gt 0 ]]; then
    ok "40 запросов подряд с одного адреса: $limited получили 429"
else
    bad "limit_req не сработал на 40 запросах подряд"
fi
sleep 3

echo "[имя сервера]"
check_header "Server на статике" "Server" "GoFlow Pay" "$BASE/"
check_header "Server на API" "Server" "GoFlow Pay" "$BASE/api/v2/accounts"
check_header "Server на ошибке nginx" "Server" "GoFlow Pay" "$BASE/no-such-file.html"

echo "[кэширование]"
check_header "HTML ревалидируется" "Cache-Control" "no-cache" "$BASE/"
check_header "картинки кэшируются на сутки" "Cache-Control" "max-age=86400" "$BASE/img/dashboard.svg"
if [[ -n "$asset" ]]; then
    check_header "хэшированные ассеты SPA immutable" "Cache-Control" "immutable" "$BASE$asset"
    curl -s -o /dev/null "$BASE$asset"
    check_header "ассет SPA отдаётся из кэша nginx" "X-Cache-Status" "HIT" "$BASE$asset"
fi
etag=$(curl -s -o /dev/null -D - "$BASE/" | grep -i '^etag:' | cut -d' ' -f2- | tr -d '\r')
check "повторный запрос с ETag -> 304" 304 "" -H "If-None-Match: $etag" "$BASE/"
check_header "API не кэшируется" "Cache-Control" "no-store" "$BASE/api/v2/accounts"
check_header "/management не кэшируется" "Cache-Control" "no-store" "$BASE/management"
check_header "/status не кэшируется" "Cache-Control" "no-store" "$BASE/status"
check_header "/health не кэшируется" "Cache-Control" "no-store" "$BASE/health"
check_header "/admin не кэшируется" "Cache-Control" "no-store" "$BASE/admin/"
check_header "/monitoring не кэшируется" "Cache-Control" "no-store" "$BASE/monitoring/"

echo "[gzip]"
check_header "HTML сжимается" "Content-Encoding" "gzip" -H 'Accept-Encoding: gzip' "$BASE/"
check_header "SVG сжимается" "Content-Encoding" "gzip" -H 'Accept-Encoding: gzip' "$BASE/img/dashboard.svg"
check_header "OpenAPI-спецификация сжимается" "Content-Encoding" "gzip" -H 'Accept-Encoding: gzip' "$BASE/api/v2/docs/openapi.yaml"
check_header "Vary: Accept-Encoding" "Vary" "Accept-Encoding" -H 'Accept-Encoding: gzip' "$BASE/"
if curl -s -o /dev/null -D - "$BASE/" | grep -qi '^content-encoding:'; then
    bad "без Accept-Encoding ответ сжат"
else
    ok "без Accept-Encoding не сжимается"
fi

echo "[мониторинг]"
check "/monitoring редиректит на /monitoring/" 301 "" "$BASE/monitoring"
check "/monitoring/ отдаёт Grafana" 200 "Grafana" "$BASE/monitoring/"
check "Grafana видит базу" 200 '"database": "ok"' "$BASE/monitoring/api/health"
check "дашборд GoFlow Pay подключен" 200 '"uid":"goflow-pay"' "$BASE/monitoring/api/search?query=GoFlow"
check "источник Prometheus" 200 '"status":"success"' "$BASE/monitoring/api/datasources/proxy/uid/prometheus/api/v1/query?query=up"
check "источник Loki" 200 '"status":"success"' "$BASE/monitoring/api/datasources/proxy/uid/loki/loki/api/v1/labels"

echo
echo "Итого: $pass OK, $fail FAIL"
[[ "$fail" -eq 0 ]]
