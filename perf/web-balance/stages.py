import collections
import csv
import gzip
import sys

WINDOW = 30


def main(path):
    windows = collections.defaultdict(list)
    start = None
    with gzip.open(path, "rt") as f:
        for row in csv.DictReader(f):
            if row["metric_name"] != "http_req_duration" or "kind=read" not in row["extra_tags"]:
                continue
            ts = int(row["timestamp"])
            start = start or ts
            windows[(ts - start) // WINDOW].append(float(row["metric_value"]))

    print(f"{'window':<10} {'rps':>6} {'med':>7} {'p95':>7} {'p99':>7} {'max':>8}")
    for k in sorted(windows):
        values = sorted(windows[k])
        n = len(values)

        def q(p):
            return values[min(n - 1, int(p * n))]

        print(f"{k * WINDOW:>4}-{(k + 1) * WINDOW:<4}s {n / WINDOW:6.0f} {q(.5):7.2f} {q(.95):7.2f} {q(.99):7.2f} {values[-1]:8.2f}")


if __name__ == "__main__":
    main(sys.argv[1])
