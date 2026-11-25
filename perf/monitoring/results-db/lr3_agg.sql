SET work_mem = '256MB';

DELETE FROM lr3_dist WHERE bench_id = :'bench';
DELETE FROM lr3_hist WHERE bench_id = :'bench';
DELETE FROM lr3_timeline WHERE bench_id = :'bench';

\set q '{0,0.01,0.02,0.03,0.04,0.05,0.06,0.07,0.08,0.09,0.1,0.11,0.12,0.13,0.14,0.15,0.16,0.17,0.18,0.19,0.2,0.21,0.22,0.23,0.24,0.25,0.26,0.27,0.28,0.29,0.3,0.31,0.32,0.33,0.34,0.35,0.36,0.37,0.38,0.39,0.4,0.41,0.42,0.43,0.44,0.45,0.46,0.47,0.48,0.49,0.5,0.51,0.52,0.53,0.54,0.55,0.56,0.57,0.58,0.59,0.6,0.61,0.62,0.63,0.64,0.65,0.66,0.67,0.68,0.69,0.7,0.71,0.72,0.73,0.74,0.75,0.76,0.77,0.78,0.79,0.8,0.81,0.82,0.83,0.84,0.85,0.86,0.87,0.88,0.89,0.9,0.91,0.92,0.93,0.94,0.95,0.96,0.97,0.98,0.99,0.999,0.9999,1}'
\set bins 40

INSERT INTO lr3_dist (bench_id, version, scenario, size, op, run, ops_count, min_ns, max_ns, mean_ns, std_ns, q, p_ns)
SELECT
  r.bench_id, r.version, r.scenario, r.size, s.op,
  CASE WHEN GROUPING(r.run) = 1 THEN 0 ELSE r.run END,
  count(*),
  min(s.latency_ns),
  max(s.latency_ns),
  avg(s.latency_ns),
  stddev(s.latency_ns),
  :'q'::float8[],
  percentile_cont(:'q'::float8[]) WITHIN GROUP (ORDER BY s.latency_ns)
FROM lr3_samples s
JOIN lr3_runs r ON r.id = s.run_id
WHERE r.bench_id = :'bench' AND s.phase = 'measure'
GROUP BY GROUPING SETS (
  (r.bench_id, r.version, r.scenario, r.size, s.op, r.run),
  (r.bench_id, r.version, r.scenario, r.size, s.op)
);

CREATE TEMP TABLE lr3_bounds ON COMMIT DROP AS
SELECT
  scenario, size, op, run,
  floor(lo / width) * width AS start,
  width,
  ceil((hi - floor(lo / width) * width) / width)::int AS nbins
FROM (
  SELECT
    scenario, size, op, run, lo, hi,
    CASE
      WHEN raw / mag <= 1 THEN 1
      WHEN raw / mag <= 2 THEN 2
      WHEN raw / mag <= 2.5 THEN 2.5
      WHEN raw / mag <= 5 THEN 5
      ELSE 10
    END * mag AS width
  FROM (
    SELECT *, power(10, floor(log(raw))) AS mag
    FROM (
      SELECT *, GREATEST((hi - lo) / :bins, 1) AS raw
      FROM (
        SELECT
          r.scenario, r.size, s.op,
          CASE WHEN GROUPING(r.run) = 1 THEN 0 ELSE r.run END AS run,
          percentile_cont(0.001) WITHIN GROUP (ORDER BY s.latency_ns) AS lo,
          percentile_cont(0.99) WITHIN GROUP (ORDER BY s.latency_ns) AS hi
        FROM lr3_samples s
        JOIN lr3_runs r ON r.id = s.run_id
        WHERE r.bench_id = :'bench' AND s.phase = 'measure'
        GROUP BY GROUPING SETS ((r.scenario, r.size, s.op, r.run), (r.scenario, r.size, s.op))
      ) p
    ) w
  ) m
) b;

INSERT INTO lr3_hist (bench_id, scenario, size, op, run, bin, lo_us, hi_us, v23, v25)
SELECT
  :'bench', b.scenario, b.size, b.op, b.run, g.bin,
  (b.start + (g.bin - 1) * b.width) / 1000,
  (b.start + g.bin * b.width) / 1000,
  COALESCE(c.v23, 0),
  COALESCE(c.v25, 0)
FROM lr3_bounds b
CROSS JOIN LATERAL generate_series(1, b.nbins) AS g(bin)
LEFT JOIN (
  SELECT
    b.scenario, b.size, b.op, b.run,
    floor((s.latency_ns - b.start) / b.width)::int + 1 AS bin,
    count(*) FILTER (WHERE r.version = 'go1.23') AS v23,
    count(*) FILTER (WHERE r.version = 'go1.25') AS v25
  FROM lr3_samples s
  JOIN lr3_runs r ON r.id = s.run_id
  JOIN lr3_bounds b ON b.scenario = r.scenario AND b.size = r.size AND b.op = s.op AND b.run IN (0, r.run)
  WHERE r.bench_id = :'bench' AND s.phase = 'measure'
  GROUP BY 1, 2, 3, 4, 5
) c ON c.scenario = b.scenario AND c.size = b.size AND c.op = b.op AND c.run = b.run AND c.bin = g.bin;

INSERT INTO lr3_timeline (bench_id, scenario, size, op, run, t_s, p50_23_ns, max_23_ns, p50_25_ns, max_25_ns, avg_23_ns, avg_25_ns)
SELECT
  :'bench', r.scenario, r.size, s.op,
  CASE WHEN GROUPING(r.run) = 1 THEN 0 ELSE r.run END,
  round((floor(s.t_ms / 50) * 0.05)::numeric, 2),
  percentile_cont(0.5) WITHIN GROUP (ORDER BY s.latency_ns) FILTER (WHERE r.version = 'go1.23'),
  max(s.latency_ns) FILTER (WHERE r.version = 'go1.23'),
  percentile_cont(0.5) WITHIN GROUP (ORDER BY s.latency_ns) FILTER (WHERE r.version = 'go1.25'),
  max(s.latency_ns) FILTER (WHERE r.version = 'go1.25'),
  avg(s.latency_ns) FILTER (WHERE r.version = 'go1.23'),
  avg(s.latency_ns) FILTER (WHERE r.version = 'go1.25')
FROM lr3_samples s
JOIN lr3_runs r ON r.id = s.run_id
WHERE r.bench_id = :'bench'
GROUP BY GROUPING SETS (
  (r.scenario, r.size, s.op, floor(s.t_ms / 50), r.run),
  (r.scenario, r.size, s.op, floor(s.t_ms / 50))
);
