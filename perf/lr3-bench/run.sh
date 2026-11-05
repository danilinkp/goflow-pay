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

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
OUT_REL="${OUT:-perf/results/lr3/$(date +%Y%m%d-%H%M%S)}"
OUT_DIR="${REPO_ROOT}/${OUT_REL}"
mkdir -p "${OUT_DIR}"

echo "--> Подготовка мониторинга"
docker network inspect goflow-load >/dev/null 2>&1 || docker network create goflow-load
docker compose -p goflow-mon -f "${REPO_ROOT}/perf/monitoring/compose.yml" up -d --wait

echo "--> Сборка образов для ЛР3"
docker build -t goflow-lr3-bench:go1.23 --build-arg GO_IMAGE=golang:1.23.4-alpine "${REPO_ROOT}/perf/lr3-bench"
docker build -t goflow-lr3-bench:go1.25 --build-arg GO_IMAGE=golang:1.25.4-alpine "${REPO_ROOT}/perf/lr3-bench"

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
        docker run --rm \
          --cpuset-cpus "$BENCH_CPU" \
          --memory 512m \
          --network none \
          --user "$(id -u):$(id -g)" \
          -e GOMAXPROCS=1 \
          -v "${OUT_DIR}:/out" \
          "goflow-lr3-bench:${version}" \
          -scenario "$scenario" \
          -size "$size" \
          -run "$run" \
          -warmup "$WARMUP" \
          -duration "$DURATION" \
          -teardown "$TEARDOWN" \
          -out "/out/${out_subpath}"
      done
    done
  done
done

"${REPO_ROOT}/perf/lr3-bench/load.sh" "${OUT_DIR}"
