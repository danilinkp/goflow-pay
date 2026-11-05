#!/usr/bin/env bash
set -euo pipefail

RUN_DIR="${1:-}"
if [[ -z "${RUN_DIR}" || ! -d "${RUN_DIR}" ]]; then
    echo "Usage: $0 <run_dir>" >&2
    exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
MON_COMPOSE="${REPO_ROOT}/perf/monitoring/compose.yml"
TESTID="$(basename "${RUN_DIR}")"
RUN_JSON="${RUN_DIR}/run.json"
CSV_GZ="${RUN_DIR}/k6.csv.gz"

if [[ ! -f "${RUN_JSON}" || ! -f "${CSV_GZ}" ]]; then
    echo "Required files (run.json, k6.csv.gz) not found in ${RUN_DIR}" >&2
    exit 1
fi

docker compose -p goflow-mon -f "${MON_COMPOSE}" exec -T results-db pg_isready -U results -d results >/dev/null 2>&1 || {
    echo "results-db is not ready" >&2
    exit 1
}

echo "--> Загрузка результатов ЛР4 (${TESTID}) в results-db"

read -r SCENARIO ERROR_RATE START_TS END_TS BASE_DUR SPIKE_RAMP_DUR HOLD_DUR REC_DUR RAMP_STEP_DUR RAMP_START_VUS RAMP_STEP_VUS < <(python3 -c "
import json, re, sys

def parse_dur(s):
    if not s:
        return 0.0
    s = str(s).strip()
    m = re.match(r'^(\d+(?:\.\d+)?)([smhd])$', s)
    if not m:
        return float(s)
    v, u = float(m.group(1)), m.group(2)
    return v * {'s': 1, 'm': 60, 'h': 3600, 'd': 86400}[u]

with open('${RUN_JSON}') as f:
    d = json.load(f)
p = d.get('profile', {})
print(d.get('scenario', 'smoke'),
      d.get('error_rate', 0),
      int(d.get('start', 0)),
      int(d.get('end', 0)),
      parse_dur(p.get('STRESS_BASELINE', '5m')),
      parse_dur(p.get('STRESS_SPIKE_RAMP', '30s')),
      parse_dur(p.get('STRESS_HOLD', '3m')),
      parse_dur(p.get('STRESS_RECOVERY', '10m')),
      parse_dur(p.get('RAMP_STEP_DURATION', '60s')),
      int(p.get('RAMP_START_VUS', 10)),
      int(p.get('RAMP_STEP_VUS', 20)))
")

META_ESCAPED="$(sed "s/'/''/g" "${RUN_JSON}")"

{
    echo "BEGIN;"
    echo "DELETE FROM lr4_runs WHERE testid = '${TESTID}';"
    echo "INSERT INTO lr4_runs (testid, scenario, error_rate, started_at, finished_at, meta) VALUES ('${TESTID}', '${SCENARIO}', ${ERROR_RATE}, to_timestamp(${START_TS}), to_timestamp(${END_TS}), '${META_ESCAPED}'::jsonb);"
    echo "CREATE TEMP TABLE staging_k6 (metric_name text, timestamp double precision, metric_value double precision, \"check\" text, error text, error_code text, expected_response text, \"group\" text, method text, name text, proto text, scenario text, service text, status text, subproto text, tls_version text, url text, extra_tags text, metadata text) ON COMMIT DROP;"
    echo "\copy staging_k6 FROM stdin WITH (FORMAT csv, HEADER true);"
    gunzip -c "${CSV_GZ}"
    echo "\."
    cat <<EOF
WITH load_bounds AS (
    SELECT COALESCE(min(timestamp / 1000.0), ${START_TS}::double precision) AS load_start
    FROM staging_k6
    WHERE ("group" IS NULL OR "group" != '::setup')
      AND (extra_tags IS NULL OR extra_tags NOT LIKE '%phase=setup%')
      AND metric_name = 'http_req_duration'
)
INSERT INTO lr4_samples (testid, ts, metric, value, name, kind, phase, status, expected, variant)
SELECT
    '${TESTID}',
    to_timestamp(s.timestamp / 1000.0),
    s.metric_name,
    s.metric_value,
    NULLIF(s.name, ''),
    (regexp_match(s.extra_tags, '(?:^|&)kind=([^&]*)'))[1],
    CASE
        WHEN (regexp_match(s.extra_tags, '(?:^|&)phase=([^&]*)'))[1] IS NOT NULL
            THEN (regexp_match(s.extra_tags, '(?:^|&)phase=([^&]*)'))[1]
        WHEN '${SCENARIO}' = 'stress' THEN
            CASE
                WHEN (s.timestamp / 1000.0) - b.load_start < 10.0 THEN 'ramp_up'
                WHEN (s.timestamp / 1000.0) - b.load_start < 10.0 + ${BASE_DUR} THEN 'baseline'
                WHEN (s.timestamp / 1000.0) - b.load_start < 10.0 + ${BASE_DUR} + ${SPIKE_RAMP_DUR} THEN 'spike'
                WHEN (s.timestamp / 1000.0) - b.load_start < 10.0 + ${BASE_DUR} + ${SPIKE_RAMP_DUR} + ${HOLD_DUR} THEN 'hold'
                WHEN (s.timestamp / 1000.0) - b.load_start < 10.0 + ${BASE_DUR} + ${SPIKE_RAMP_DUR} + ${HOLD_DUR} + 10.0 THEN 'ramp_down'
                ELSE 'recovery'
            END
        WHEN '${SCENARIO}' = 'ramp' THEN
            ((${RAMP_START_VUS} + floor(greatest(0.0, (s.timestamp / 1000.0) - b.load_start) / (10.0 + ${RAMP_STEP_DUR})) * ${RAMP_STEP_VUS})::text || ' VU')
        ELSE '${SCENARIO}'
    END,
    NULLIF(s.status, '')::int,
    CASE WHEN lower(s.expected_response) = 'true' THEN true WHEN lower(s.expected_response) = 'false' THEN false ELSE NULL END,
    (regexp_match(s.extra_tags, '(?:^|&)variant=([^&]*)'))[1]
FROM staging_k6 s
CROSS JOIN load_bounds b
WHERE s.metric_name IN ('http_req_duration', 'vus')
  AND (s."group" IS NULL OR s."group" != '::setup')
  AND (s.extra_tags IS NULL OR s.extra_tags NOT LIKE '%phase=setup%');

COMMIT;
EOF
} | docker compose -p goflow-mon -f "${MON_COMPOSE}" exec -T results-db psql -U results -d results -v ON_ERROR_STOP=1 >/dev/null

FROM_MS=$(((START_TS - 15) * 1000))
TO_MS=$(((END_TS + 30) * 1000))
echo "--> Ссылка на дашборд ЛР4:"
echo "http://localhost:3000/d/lr4-load?var-testid=${TESTID}&from=${FROM_MS}&to=${TO_MS}"
