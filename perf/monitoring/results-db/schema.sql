CREATE TABLE IF NOT EXISTS lr3_runs (
  id serial primary key,
  bench_id text,
  version text,
  scenario text,
  size int,
  run int,
  meta jsonb,
  unique (bench_id, version, scenario, size, run)
);

CREATE TABLE IF NOT EXISTS lr3_samples (
  run_id int references lr3_runs on delete cascade,
  t_ms double precision,
  latency_ns bigint,
  op text,
  phase text
);

CREATE INDEX IF NOT EXISTS idx_lr3_samples_run_phase_op ON lr3_samples (run_id, phase, op);

CREATE TABLE IF NOT EXISTS lr3_stats (
  run_id int references lr3_runs on delete cascade,
  bench_id text,
  version text,
  scenario text,
  size int,
  run int,
  op text,
  p50_ns double precision,
  p99_ns double precision,
  ops_count bigint,
  primary key (run_id, op)
);

CREATE INDEX IF NOT EXISTS idx_lr3_stats_bench ON lr3_stats (bench_id, scenario, size, op);

CREATE TABLE IF NOT EXISTS lr3_dist (
  bench_id text,
  version text,
  scenario text,
  size int,
  op text,
  run int,
  ops_count bigint,
  min_ns bigint,
  max_ns bigint,
  mean_ns double precision,
  std_ns double precision,
  q double precision[],
  p_ns double precision[],
  primary key (bench_id, scenario, size, op, run, version)
);

CREATE TABLE IF NOT EXISTS lr3_hist (
  bench_id text,
  scenario text,
  size int,
  op text,
  run int,
  bin int,
  lo_us double precision,
  hi_us double precision,
  v23 bigint,
  v25 bigint,
  primary key (bench_id, scenario, size, op, run, bin)
);

CREATE TABLE IF NOT EXISTS lr3_timeline (
  bench_id text,
  scenario text,
  size int,
  op text,
  run int,
  t_s double precision,
  p50_23_ns double precision,
  max_23_ns bigint,
  p50_25_ns double precision,
  max_25_ns bigint,
  avg_23_ns double precision,
  avg_25_ns double precision,
  primary key (bench_id, scenario, size, op, run, t_s)
);

ALTER TABLE lr3_hist ADD COLUMN IF NOT EXISTS hi_us double precision;
ALTER TABLE lr3_timeline ADD COLUMN IF NOT EXISTS avg_23_ns double precision;
ALTER TABLE lr3_timeline ADD COLUMN IF NOT EXISTS avg_25_ns double precision;

CREATE TABLE IF NOT EXISTS lr3_live (
  ts timestamptz,
  bench_id text,
  version text,
  scenario text,
  size int,
  run int,
  phase text,
  op text,
  bucket int,
  lo_ns bigint,
  hi_ns bigint,
  n bigint
);

CREATE INDEX IF NOT EXISTS idx_lr3_live_ts ON lr3_live (ts);
CREATE INDEX IF NOT EXISTS idx_lr3_live_run ON lr3_live (bench_id, scenario, size, run, version);

CREATE TABLE IF NOT EXISTS lr4_runs (
  testid text primary key,
  scenario text,
  error_rate real,
  started_at timestamptz,
  finished_at timestamptz,
  meta jsonb
);

CREATE TABLE IF NOT EXISTS lr4_samples (
  testid text references lr4_runs on delete cascade,
  ts timestamptz,
  metric text,
  value double precision,
  name text,
  kind text,
  phase text,
  status int,
  expected boolean,
  variant text
);

CREATE INDEX IF NOT EXISTS idx_lr4_samples_testid_metric_ts ON lr4_samples (testid, metric, ts);
CREATE INDEX IF NOT EXISTS idx_lr4_samples_testid_kind ON lr4_samples (testid, kind);
