#!/usr/bin/env bash
set -euo pipefail

RUNS="${RUNS:-3}"
WARMUP="${WARMUP:-2s}"
DURATION="${DURATION:-5s}"
TEARDOWN="${TEARDOWN:-1s}"
BENCH_CPU="${BENCH_CPU:-2}"
VERSIONS="${VERSIONS:-go1.23 go1.25}"
SCENARIOS="${SCENARIOS:-lookup_hit lookup_miss update churn mixed grow}"
SIZES="${SIZES:-1000 100000 1000000}"
LIVE="${LIVE:-0}"
LIVE_INTERVAL="${LIVE_INTERVAL:-250ms}"
CPUS_MON="${CPUS_MON:-6,7}"

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
OUT_REL="${OUT:-perf/results/lr3/$(date +%Y%m%d-%H%M%S)}"
OUT_DIR="${REPO_ROOT}/${OUT_REL}"
mkdir -p "${OUT_DIR}"
BENCH_ID="$(basename "${OUT_DIR}")"
MON_COMPOSE="${REPO_ROOT}/perf/monitoring/compose.yml"

bench_run() {
  docker run --rm \
    --cpuset-cpus "$BENCH_CPU" \
    --memory 512m \
    --network none \
    --user "$(id -u):$(id -g)" \
    -e GOMAXPROCS=1 \
    -v "${OUT_DIR}:/out" \
    "$@"
}

live_sink() {
  taskset -c "$CPUS_MON" awk -F, -v bench="$BENCH_ID" -v version="$1" -v scenario="$2" -v size="$3" -v run="$4" '
    function emit() {
      if (rows != "") {
        print "INSERT INTO lr3_live VALUES " rows ";"
        fflush()
      }
      rows = ""
    }
    BEGIN { print "SET synchronous_commit = off;"; fflush() }
    $1 != ts { emit(); ts = $1 }
    {
      rows = rows (rows == "" ? "" : ",") sprintf("(to_timestamp(%s / 1000.0), '"'"'%s'"'"', '"'"'%s'"'"', '"'"'%s'"'"', %d, %d, '"'"'%s'"'"', '"'"'%s'"'"', %d, %d, %d, %d)", $1, bench, version, scenario, size, run, $2, $3, $4, $5, $6, $7)
    }
    END { emit() }
  ' | docker compose -p goflow-mon -f "$MON_COMPOSE" exec -T results-db psql -q -U results -d results -v ON_ERROR_STOP=1 >/dev/null
}

echo "--> Подготовка мониторинга"
docker network inspect goflow-load >/dev/null 2>&1 || docker network create goflow-load
docker compose -p goflow-mon -f "${MON_COMPOSE}" up -d --wait
docker compose -p goflow-mon -f "${MON_COMPOSE}" exec -T results-db psql -q -U results -d results -c "SET client_min_messages = warning;" -f - < "${REPO_ROOT}/perf/monitoring/results-db/schema.sql"

echo "--> Сборка образов для ЛР3"
docker build -t goflow-lr3-bench:go1.23 --build-arg GO_IMAGE=golang:1.23.4-alpine "${REPO_ROOT}/perf/lr3-bench"
docker build -t goflow-lr3-bench:go1.25 --build-arg GO_IMAGE=golang:1.25.4-alpine "${REPO_ROOT}/perf/lr3-bench"

if [ "$LIVE" = "1" ]; then
  echo "--> Дашборд в реальном времени: http://localhost:3000/d/lr3-live"
fi

echo "--> Запуск микробенчмарков ЛР3"
for run in $(seq 1 "$RUNS"); do
  for scenario in $SCENARIOS; do
    if [ "$scenario" = "grow" ]; then
      curr_sizes="1000"
    else
      curr_sizes="$SIZES"
    fi
    for size in $curr_sizes; do
      for version in $VERSIONS; do
        out_subpath="${version}/${scenario}/${size}/run-${run}"
        echo "--> Прогон ${run}/${RUNS}: ${scenario} ${size} ${version}"
        args=(
          "goflow-lr3-bench:${version}"
          -scenario "$scenario"
          -size "$size"
          -run "$run"
          -warmup "$WARMUP"
          -duration "$DURATION"
          -teardown "$TEARDOWN"
          -out "/out/${out_subpath}"
        )
        if [ "$LIVE" = "1" ]; then
          bench_run "${args[@]}" -live "$LIVE_INTERVAL" | live_sink "$version" "$scenario" "$size" "$run"
        else
          bench_run "${args[@]}"
        fi
      done
    done
  done
done

"${REPO_ROOT}/perf/lr3-bench/load.sh" "${OUT_DIR}"
