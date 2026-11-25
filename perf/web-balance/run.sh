#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
cd "${REPO_ROOT}"

SCENARIO="${1:-steady}"
K6_IMAGE="${K6_IMAGE:-grafana/k6:0.52.0}"
INSTANCES=(gateway gateway-ro-1 gateway-ro-2 auth auth-ro-1 auth-ro-2 accounts accounts-ro-1 accounts-ro-2 nginx postgres-auth postgres-accounts)

container_id() {
    docker compose ps -q "$1"
}

container_ip() {
    docker inspect -f '{{range $name, $net := .NetworkSettings.Networks}}{{if ne $name "goflow_loadtest"}}{{$net.IPAddress}}{{end}}{{end}}' "$(container_id "$1")"
}

cpu_usec() {
    local file="/sys/fs/cgroup/system.slice/docker-$(docker inspect -f '{{.Id}}' "$(container_id "$1")").scope/cpu.stat"
    [[ -r "$file" ]] || { echo 0; return; }
    awk '$1 == "usage_usec" { print $2 }' "$file"
}

[[ -n "$(docker compose ps --status running -q nginx)" ]] || { echo "стенд не запущен: make web-up" >&2; exit 1; }

GATEWAY_ADDR="$(container_ip gateway):8080"
GATEWAY_RO_1_ADDR="$(container_ip gateway-ro-1):8080"
GATEWAY_RO_2_ADDR="$(container_ip gateway-ro-2):8080"

NETWORK=goflow_loadtest
BASE_URL=http://nginx
if [[ "${SCENARIO}" == "ratelimit" ]]; then
    NETWORK="$(docker inspect -f '{{range $name, $net := .NetworkSettings.Networks}}{{if ne $name "goflow_loadtest"}}{{$name}}{{end}}{{end}}' "$(container_id nginx)")"
fi

TESTID="$(date +%Y%m%d-%H%M%S)-${SCENARIO}"
OUT="${REPO_ROOT}/perf/results/web-balance/${TESTID}"
mkdir -p "${OUT}"
chmod 777 "${OUT}"

{
    echo "testid=${TESTID}"
    echo "scenario=${SCENARIO}"
    echo "network=${NETWORK}"
    echo "gateway=${GATEWAY_ADDR}"
    echo "gateway-ro-1=${GATEWAY_RO_1_ADDR}"
    echo "gateway-ro-2=${GATEWAY_RO_2_ADDR}"
    env | awk -F= '/^(READ_RPS|WRITE_RPS|DURATION|RAMP_PEAK|COMPANIES)=/'
} > "${OUT}/env.txt"

declare -A CPU_BEFORE
for svc in "${INSTANCES[@]}"; do
    CPU_BEFORE[$svc]="$(cpu_usec "$svc")"
done

START="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
STARTED_AT="$(date +%s)"

set +e
docker run --rm --network "${NETWORK}" \
    --label com.docker.compose.service=k6 \
    -v "${SCRIPT_DIR}/k6:/scripts:ro" \
    -v "${OUT}:/results" \
    -e BASE_URL="${BASE_URL}" \
    -e SCENARIO="${SCENARIO}" \
    -e GATEWAY_ADDR="${GATEWAY_ADDR}" \
    -e GATEWAY_RO_1_ADDR="${GATEWAY_RO_1_ADDR}" \
    -e GATEWAY_RO_2_ADDR="${GATEWAY_RO_2_ADDR}" \
    -e READ_RPS -e WRITE_RPS -e DURATION -e RAMP_PEAK -e COMPANIES \
    -e K6_PROMETHEUS_RW_SERVER_URL=http://prometheus:9090/api/v1/write \
    -e K6_PROMETHEUS_RW_PUSH_INTERVAL=5s \
    -e K6_PROMETHEUS_RW_TREND_AS_NATIVE_HISTOGRAM=true \
    "${K6_IMAGE}" run --quiet \
    --tag testid="${TESTID}" \
    --system-tags=status,method,name,scenario,expected_response \
    --out experimental-prometheus-rw \
    --out csv=/results/k6.csv.gz \
    --summary-export=/results/summary.json /scripts/balance.js | tee "${OUT}/summary.txt"
K6_EXIT=${PIPESTATUS[0]}
set -e

ELAPSED=$(( $(date +%s) - STARTED_AT ))

{
    printf "%-16s %12s %10s\n" instance cpu_seconds cpu_cores
    for svc in "${INSTANCES[@]}"; do
        after="$(cpu_usec "$svc")"
        awk -v s="$svc" -v b="${CPU_BEFORE[$svc]}" -v a="$after" -v t="$ELAPSED" \
            'BEGIN { d = (a - b) / 1e6; printf "%-16s %12.1f %10.2f\n", s, d, d / t }'
    done
} > "${OUT}/cpu.txt"

docker compose logs --no-log-prefix --since "${START}" nginx 2>/dev/null \
    | awk -v g="${GATEWAY_ADDR}" -v r1="${GATEWAY_RO_1_ADDR}" -v r2="${GATEWAY_RO_2_ADDR}" '
        / upstream=/ {
            split($0, q, "\""); split(q[2], req, " "); method = req[1]
            match($0, /upstream=[^ ]+/); addr = substr($0, RSTART + 9, RLENGTH - 9)
            name = addr == g ? "gateway" : addr == r1 ? "gateway-ro-1" : addr == r2 ? "gateway-ro-2" : addr == "-" ? "none" : "other"
            n[method " " name]++; total[method]++
        }
        END {
            for (k in n) { split(k, p, " "); printf "%-6s %-14s %8d %6.1f%%\n", p[1], p[2], n[k], 100 * n[k] / total[p[1]] }
        }' | sort > "${OUT}/nginx-split.body"
{ printf "%-6s %-14s %8s %7s\n" method instance requests share; cat "${OUT}/nginx-split.body"; } > "${OUT}/nginx-split.txt"
rm -f "${OUT}/nginx-split.body"

echo
echo "nginx access log, распределение по экземплярам:"
cat "${OUT}/nginx-split.txt"
python3 "${SCRIPT_DIR}/stages.py" "${OUT}/k6.csv.gz" > "${OUT}/stages.txt" || true

echo
echo "p95 чтений по 30-секундным окнам, мс:"
cat "${OUT}/stages.txt"
echo
echo "CPU за ${ELAPSED} с:"
cat "${OUT}/cpu.txt"
FINISHED_AT="$(date +%s)"
if [[ -n "${GRAFANA_ADMIN_PASSWORD:-}" ]]; then
    params="$(awk -F= '/^(READ_RPS|WRITE_RPS|DURATION|RAMP_PEAK|COMPANIES)=/ { printf "%s ", $0 }' "${OUT}/env.txt")"
    curl -s -o /dev/null -u "${GRAFANA_ADMIN_USER:-admin}:${GRAFANA_ADMIN_PASSWORD}" \
        -H 'Content-Type: application/json' \
        -d "{\"time\": ${STARTED_AT}000, \"timeEnd\": ${FINISHED_AT}000, \"tags\": [\"loadtest\", \"${SCENARIO}\"], \"text\": \"${TESTID} ${params}k6 exit ${K6_EXIT}\"}" \
        "http://127.0.0.1:${NGINX_PORT:-80}/monitoring/api/annotations" || true
fi

echo
echo "testid: ${TESTID}"
echo "результаты: ${OUT}"
exit "${K6_EXIT}"
