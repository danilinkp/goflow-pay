#!/usr/bin/env bash
set -euo pipefail

REPORTS_DIR="${REPORTS_DIR:-reports}"
PCAP_FILE="$REPORTS_DIR/traffic_capture.pcap"

mkdir -p "$REPORTS_DIR"
rm -f "$PCAP_FILE"

# Determine capture utility
if command -v dumpcap >/dev/null 2>&1; then
    echo "--> Starting packet capture via dumpcap (interface: lo, port: 8080)..."
    dumpcap -i lo -f "tcp port 8080" -w "$PCAP_FILE" -q > /dev/null 2>&1 &
    CAP_PID=$!
elif command -v tcpdump >/dev/null 2>&1; then
    echo "--> Starting packet capture via tcpdump (interface: lo, port: 8080)..."
    tcpdump -i lo port 8080 -w "$PCAP_FILE" > /dev/null 2>&1 &
    CAP_PID=$!
else
    echo "ERROR: neither dumpcap nor tcpdump found!"
    exit 1
fi

cleanup() {
    if [ -n "${CAP_PID:-}" ] && kill -0 "$CAP_PID" 2>/dev/null; then
        kill -INT "$CAP_PID" 2>/dev/null || true
        wait "$CAP_PID" 2>/dev/null || true
    fi
    ./scripts/test_stand_down.sh > /dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM

# Start test stand
./scripts/test_stand_up.sh

# Short pause to ensure capture socket is ready
sleep 0.5

echo "--> Executing E2E test scenario..."
go test -v -count=1 -tags=e2e ./tests/e2e/...

# Finish capture cleanly to flush pcap buffers
kill -INT "$CAP_PID" 2>/dev/null || true
wait "$CAP_PID" 2>/dev/null || true
CAP_PID=""

echo "--> Packet capture completed successfully: $PCAP_FILE"
ls -lh "$PCAP_FILE"
