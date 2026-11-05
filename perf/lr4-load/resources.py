import json
import os
import sys
import urllib.parse
import urllib.request

QUERIES = {
    "cpu_cores": 'sum by (svc) (label_replace(rate(container_cpu_usage_seconds_total{container_label_com_docker_compose_service!=""}[15s]), "svc", "$1", "container_label_com_docker_compose_service", "(.*)"))',
    "ram_bytes": 'sum by (svc) (label_replace(container_memory_working_set_bytes{container_label_com_docker_compose_service!=""}, "svc", "$1", "container_label_com_docker_compose_service", "(.*)"))',
    "disk_read_Bps": 'sum by (svc) (label_replace(rate(container_blkio_device_usage_total{operation="Read",container_label_com_docker_compose_service!=""}[15s]), "svc", "$1", "container_label_com_docker_compose_service", "(.*)"))',
    "disk_write_Bps": 'sum by (svc) (label_replace(rate(container_blkio_device_usage_total{operation="Write",container_label_com_docker_compose_service!=""}[15s]), "svc", "$1", "container_label_com_docker_compose_service", "(.*)"))',
    "net_rx_Bps": 'sum by (svc) (label_replace(rate(container_network_receive_bytes_total{container_label_com_docker_compose_service!=""}[15s]), "svc", "$1", "container_label_com_docker_compose_service", "(.*)"))',
    "net_tx_Bps": 'sum by (svc) (label_replace(rate(container_network_transmit_bytes_total{container_label_com_docker_compose_service!=""}[15s]), "svc", "$1", "container_label_com_docker_compose_service", "(.*)"))',
}

def series_stats(vals: list[float]) -> dict[str, float]:
    if not vals:
        return {"min": 0.0, "max": 0.0, "mean": 0.0, "median": 0.0, "p95": 0.0}
    s = sorted(vals)
    n = len(s)
    median = s[n // 2] if n % 2 == 1 else (s[n // 2 - 1] + s[n // 2]) / 2.0
    idx95 = int(round(0.95 * (n - 1)))
    return {
        "min": float(s[0]),
        "max": float(s[-1]),
        "mean": float(sum(s) / n),
        "median": float(median),
        "p95": float(s[min(idx95, n - 1)]),
    }

def query_prom(base_url: str, query: str, start: float, end: float) -> list[dict]:
    params = urllib.parse.urlencode({"query": query, "start": f"{start:.3f}", "end": f"{end:.3f}", "step": "5s"})
    url = f"{base_url.rstrip('/')}/api/v1/query_range?{params}"
    req = urllib.request.Request(url, headers={"User-Agent": "lr4-resources"})
    try:
        with urllib.request.urlopen(req, timeout=15) as resp:
            data = json.loads(resp.read().decode("utf-8"))
            if data.get("status") == "success":
                return data.get("data", {}).get("result", [])
    except Exception as e:
        sys.stderr.write(f"Prometheus query error: {e}\n")
    return []

def main():
    if len(sys.argv) < 2:
        sys.stderr.write("Usage: resources.py <run_dir>\n")
        sys.exit(1)
    run_dir = sys.argv[1]
    with open(os.path.join(run_dir, "run.json"), "r", encoding="utf-8") as f:
        run_data = json.load(f)
    start_t = float(run_data["start"]) - 15.0
    end_t = float(run_data["end"]) + 30.0
    prom_url = os.environ.get("PROMETHEUS_URL", "http://127.0.0.1:9090")

    metrics_by_svc: dict[str, dict[str, list[float]]] = {}
    for metric_name, query in QUERIES.items():
        res = query_prom(prom_url, query, start_t, end_t)
        for item in res:
            svc = item.get("metric", {}).get("svc", "default")
            vals = [float(v[1]) for v in item.get("values", [])]
            if svc not in metrics_by_svc:
                metrics_by_svc[svc] = {k: [] for k in QUERIES}
            metrics_by_svc[svc][metric_name] = vals

    result = {}
    for svc in sorted(metrics_by_svc.keys()):
        result[svc] = {m: series_stats(metrics_by_svc[svc].get(m, [])) for m in QUERIES}

    with open(os.path.join(run_dir, "resources.json"), "w", encoding="utf-8") as f:
        json.dump(result, f, indent=2)

if __name__ == "__main__":
    main()
