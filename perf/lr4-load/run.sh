#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

SCENARIO="${1:-smoke}"
ERROR_RATE="${2:-0}"

# shellcheck source=/dev/null
source "${SCRIPT_DIR}/profiles.env"

echo "--> Подготовка сертификатов и мониторинга"
if [[ ! -f "${REPO_ROOT}/certs/private.pem" || ! -f "${REPO_ROOT}/certs/public.pem" ]]; then
    mkdir -p "${REPO_ROOT}/certs"
    openssl genpkey -algorithm RSA -out "${REPO_ROOT}/certs/private.pem" -pkeyopt rsa_keygen_bits:2048
    openssl rsa -pubout -in "${REPO_ROOT}/certs/private.pem" -out "${REPO_ROOT}/certs/public.pem"
fi

docker network inspect goflow-load >/dev/null 2>&1 || docker network create goflow-load
docker compose -p goflow-mon -f "${REPO_ROOT}/perf/monitoring/compose.yml" up -d --wait

echo "--> Сборка и запуск стенда SUT"
docker compose -p goflow-load -f "${SCRIPT_DIR}/compose.sut.yml" down -v
docker compose -p goflow-load -f "${SCRIPT_DIR}/compose.sut.yml" up -d --build --wait

echo "--> Ожидание готовности Gateway"
READY=0
for _ in $(seq 1 60); do
    if curl -s -o /dev/null -w "%{http_code}" http://127.0.0.1:8080/swagger | grep -q "200"; then
        READY=1
        break
    fi
    sleep 1
done

if [[ "$READY" -ne 1 ]]; then
    echo "Gateway failed to respond with 200 within 60s" >&2
    docker compose -p goflow-load -f "${SCRIPT_DIR}/compose.sut.yml" logs >&2
    exit 1
fi

TIMESTAMP="$(date +%Y%m%d-%H%M%S)"
OUT_DIR_NAME="${SCENARIO}-err${ERROR_RATE}-${TIMESTAMP}"
OUT="${REPO_ROOT}/perf/results/lr4/${OUT_DIR_NAME}"
mkdir -p "${OUT}"
chmod 777 "${OUT}"

echo "--> Сбор 15 с фона до нагрузки"
sleep 15

START_TIME="$(date +%s)"

echo "--> Запуск генератора нагрузки k6 (сценарий: ${SCENARIO}, доля ошибок: ${ERROR_RATE})"
set +e
docker run --rm --network goflow-load --cpuset-cpus "${CPUS_LOADGEN:-4,5}" --memory 1g \
  --label com.docker.compose.service=k6 \
  -v "${SCRIPT_DIR}/k6:/scripts:ro" \
  -v "${OUT}:/results" \
  -e K6_PROMETHEUS_RW_SERVER_URL=http://prometheus:9090/api/v1/write \
  -e K6_PROMETHEUS_RW_TREND_STATS="p(50),p(75),p(90),p(95),p(99),min,max,avg" \
  -e K6_PROMETHEUS_RW_PUSH_INTERVAL=5s \
  -e K6_CSV_TIME_FORMAT=unix_milli \
  -e BASE_URL=http://gateway:8080 \
  -e SCENARIO="${SCENARIO}" \
  -e ERROR_RATE="${ERROR_RATE}" \
  -e SEED_COMPANIES="${SEED_COMPANIES}" \
  -e THINK_MIN="${THINK_MIN}" \
  -e THINK_MAX="${THINK_MAX}" \
  -e ACTIONS_PER_SESSION="${ACTIONS_PER_SESSION}" \
  -e RAMP_START_VUS="${RAMP_START_VUS}" \
  -e RAMP_STEP_VUS="${RAMP_STEP_VUS}" \
  -e RAMP_STEPS="${RAMP_STEPS}" \
  -e RAMP_STEP_DURATION="${RAMP_STEP_DURATION}" \
  -e NORMAL_VUS="${NORMAL_VUS}" \
  -e STRESS_VUS="${STRESS_VUS}" \
  -e STRESS_BASELINE="${STRESS_BASELINE}" \
  -e STRESS_SPIKE_RAMP="${STRESS_SPIKE_RAMP}" \
  -e STRESS_HOLD="${STRESS_HOLD}" \
  -e STRESS_RECOVERY="${STRESS_RECOVERY}" \
  -e SOAK_VUS="${SOAK_VUS}" \
  -e SOAK_DURATION="${SOAK_DURATION}" \
  -e SESSION_PAUSE_MIN="${SESSION_PAUSE_MIN}" \
  -e SESSION_PAUSE_MAX="${SESSION_PAUSE_MAX}" \
  grafana/k6:0.52.0 run --tag testid="${OUT_DIR_NAME}" \
    --out experimental-prometheus-rw \
    --out "csv=/results/k6.csv.gz" \
    /scripts/main.js
K6_EXIT=$?
set -e

END_TIME="$(date +%s)"
echo "--> Сбор 30 с метрик восстановления после нагрузки (k6 exit: ${K6_EXIT})"
sleep 30

GIT_COMMIT="$(git -C "${REPO_ROOT}" rev-parse HEAD 2>/dev/null || echo "unknown")"
python3 -c "
import json
data = {
    'scenario': '${SCENARIO}',
    'error_rate': float('${ERROR_RATE}'),
    'start': int('${START_TIME}'),
    'end': int('${END_TIME}'),
    'profile': {
        'SEED_COMPANIES': '${SEED_COMPANIES}',
        'THINK_MIN': '${THINK_MIN}',
        'THINK_MAX': '${THINK_MAX}',
        'ACTIONS_PER_SESSION': '${ACTIONS_PER_SESSION}',
        'RAMP_START_VUS': '${RAMP_START_VUS}',
        'RAMP_STEP_VUS': '${RAMP_STEP_VUS}',
        'RAMP_STEPS': '${RAMP_STEPS}',
        'RAMP_STEP_DURATION': '${RAMP_STEP_DURATION}',
        'NORMAL_VUS': '${NORMAL_VUS}',
        'STRESS_VUS': '${STRESS_VUS}',
        'STRESS_BASELINE': '${STRESS_BASELINE}',
        'STRESS_SPIKE_RAMP': '${STRESS_SPIKE_RAMP}',
        'STRESS_HOLD': '${STRESS_HOLD}',
        'STRESS_RECOVERY': '${STRESS_RECOVERY}',
        'SOAK_VUS': '${SOAK_VUS}',
        'SOAK_DURATION': '${SOAK_DURATION}',
        'SESSION_PAUSE_MIN': '${SESSION_PAUSE_MIN}',
        'SESSION_PAUSE_MAX': '${SESSION_PAUSE_MAX}',
    },
    'git_commit': '${GIT_COMMIT}'
}
with open('${OUT}/run.json', 'w') as f:
    json.dump(data, f, indent=2)
"

echo "--> Остановка стенда и выгрузка логов"
docker compose -p goflow-load -f "${SCRIPT_DIR}/compose.sut.yml" logs > "${OUT}/sut.log"
docker compose -p goflow-load -f "${SCRIPT_DIR}/compose.sut.yml" down -v

echo "--> Загрузка результатов ЛР4 и сбор ресурсов"
"${SCRIPT_DIR}/load.sh" "${OUT}"
python3 "${SCRIPT_DIR}/resources.py" "${OUT}"

FROM_MS=$(((START_TIME - 15) * 1000))
TO_MS=$(((END_TIME + 30) * 1000))
echo "http://localhost:3000/d/lr4-load?var-testid=${OUT_DIR_NAME}&from=${FROM_MS}&to=${TO_MS}"
