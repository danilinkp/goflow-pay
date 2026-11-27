#!/usr/bin/env bash
set -euo pipefail

BENCH_DIR="${1:-}"
if [[ -z "${BENCH_DIR}" || ! -d "${BENCH_DIR}" ]]; then
    echo "Usage: $0 <bench_dir>" >&2
    exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
MON_COMPOSE="${REPO_ROOT}/perf/monitoring/compose.yml"
SCHEMA_SQL="${REPO_ROOT}/perf/monitoring/results-db/schema.sql"
AGG_SQL="${REPO_ROOT}/perf/monitoring/results-db/lr3_agg.sql"
BENCH_ID="$(basename "${BENCH_DIR}")"

docker compose -p goflow-mon -f "${MON_COMPOSE}" exec -T results-db pg_isready -U results -d results >/dev/null 2>&1 || {
    echo "results-db is not ready" >&2
    exit 1
}

echo "--> Загрузка результатов ЛР3 (${BENCH_ID}) в results-db"

RUN_DIRS=()
while IFS= read -r meta_file; do
    RUN_DIRS+=("$(dirname "${meta_file}")")
done < <(find "${BENCH_DIR}" -name "meta.json" | sort)

if [[ ${#RUN_DIRS[@]} -eq 0 ]]; then
    echo "No runs found in ${BENCH_DIR}" >&2
    exit 1
fi

{
    echo "SET client_min_messages = warning;"
    cat "${SCHEMA_SQL}"
    echo "BEGIN;"
    echo "DELETE FROM lr3_runs WHERE bench_id = '${BENCH_ID}';"
    echo "CREATE TEMP TABLE staging_samples (start_ns bigint, latency_ns bigint, op text, phase text) ON COMMIT DROP;"

    for r_dir in "${RUN_DIRS[@]}"; do
        meta_file="${r_dir}/meta.json"
        samples_file="${r_dir}/samples.csv"
        if [[ ! -f "${samples_file}" ]]; then
            continue
        fi

        run_name="$(basename "${r_dir}")"
        size_name="$(basename "$(dirname "${r_dir}")")"
        scenario_name="$(basename "$(dirname "$(dirname "${r_dir}")")")"
        version_name="$(basename "$(dirname "$(dirname "$(dirname "${r_dir}")")")")"
        run_num="${run_name#run-}"

        meta_json="$(sed "s/'/''/g" "${meta_file}")"

        echo "INSERT INTO lr3_runs (bench_id, version, scenario, size, run, meta) VALUES ('${BENCH_ID}', '${version_name}', '${scenario_name}', ${size_name}, ${run_num}, '${meta_json}'::jsonb);"
        echo "\copy staging_samples FROM stdin WITH (FORMAT csv, HEADER true);"
        cat "${samples_file}"
        echo "\."
        echo "INSERT INTO lr3_samples (run_id, t_ms, latency_ns, op, phase) SELECT currval('lr3_runs_id_seq'), start_ns / 1000000.0, latency_ns, op, phase FROM staging_samples;"
        echo "TRUNCATE staging_samples;"
    done

    echo "INSERT INTO lr3_stats (run_id, bench_id, version, scenario, size, run, op, p50_ns, p99_ns, ops_count)"
    echo "SELECT r.id, r.bench_id, r.version, r.scenario, r.size, r.run, s.op,"
    echo "       percentile_cont(0.5) WITHIN GROUP (ORDER BY s.latency_ns),"
    echo "       percentile_cont(0.99) WITHIN GROUP (ORDER BY s.latency_ns),"
    echo "       count(*)"
    echo "FROM lr3_runs r"
    echo "JOIN lr3_samples s ON s.run_id = r.id"
    echo "WHERE r.bench_id = '${BENCH_ID}' AND s.phase = 'measure'"
    echo "GROUP BY r.id, r.bench_id, r.version, r.scenario, r.size, r.run, s.op;"

    echo "\\set bench '${BENCH_ID}'"
    cat "${AGG_SQL}"

    echo "COMMIT;"
} | docker compose -p goflow-mon -f "${MON_COMPOSE}" exec -T results-db psql -U results -d results -v ON_ERROR_STOP=1 >/dev/null

echo "--> Ссылка на дашборд ЛР3:"
echo "http://localhost:3000/d/lr3-maps?var-bench=${BENCH_ID}"
