#!/bin/sh
# =============================================================================
# Cat J — End-to-End RPC Task Loop Simulation.
# Simulates a typical programmatic task flow repeated N times:
#   ls → cat → grep → wc → find
# Compares GoPOSIX daemon vs BusyBox process-per-command.
# =============================================================================

set -u
. "$(dirname "$0")/lib/harness.sh"

SAMPLES=3
ITERATIONS=$(scaled 10 "$MAX_LOOP_ITERATIONS")

echo "# Cat J — End-to-End RPC Task Loop" >&2
echo "# scale=$BENCH_SCALE iterations=$ITERATIONS" >&2
echo "" >&2

WORKDIR="$BENCH_TMPDIR/rpc_bench"
mkdir -p "$WORKDIR/workspace"

# Set up a realistic workspace.
if [ ! -f "$WORKDIR/workspace/README.md" ]; then
  echo "# Setting up workspace..." >&2
  cat > "$WORKDIR/workspace/README.md" << 'README_EOF'
# Sample Project

This is a test project for benchmarking.

## TODO
- TODO: implement feature A
- TODO: fix bug B
- TODO: write tests for C

## Done
- DONE: initial setup
- DONE: CI pipeline
README_EOF

  for i in $(seq 50); do
    cat > "$WORKDIR/workspace/module_${i}.go" << GO_EOF
package main

import "fmt"

// Module$i handles task $i.
// TODO: optimize loop in process$i()
func process$i() {
    fmt.Println("module $i running")
}
GO_EOF
  done
fi

# CSV header.
echo "category,test,sample,wall_sec,user_sec,sys_sec,rss_kb"

# Accumulate for stats.
ACCUM=$(mktemp)

# RPC task loop as a function-like command string (no eval complexity).
RPC_CMDS="cd /tmp/bench/rpc_bench/workspace && ls -la . >/dev/null && cat README.md >/dev/null && grep TODO README.md >/dev/null && wc -l README.md >/dev/null && find . -name '*.go' >/dev/null"

# === BusyBox process-per-command ===
echo "# BusyBox RPC task loop ($ITERATIONS iterations)" >&2
bench_run "rpc_loop_${ITERATIONS}_busybox" "$SAMPLES" \
  "( for _i in \$(seq $ITERATIONS); do
      PATH=/bin $RPC_CMDS
    done )" | tee -a "$ACCUM"

# === GoPOSIX Daemon ===
SOCKET="$BENCH_TMPDIR/goposix-bench-j.sock"
echo "# Starting GoPOSIX daemon for RPC task loop..." >&2
rm -f "$SOCKET"
/bin/goposix daemon --socket "$SOCKET" 2>/dev/null &
DAEMON_PID=$!
sleep 1

if kill -0 "$DAEMON_PID" 2>/dev/null; then
  echo "# GoPOSIX daemon RPC task loop ($ITERATIONS iterations)" >&2

  bench_run "rpc_loop_${ITERATIONS}_goposix" "$SAMPLES" \
    "( for _i in \$(seq $ITERATIONS); do
        echo '{\"jsonrpc\":\"2.0\",\"method\":\"goposix.ls\",\"params\":{\"path\":\"/tmp/bench/rpc_bench/workspace\",\"flags\":\"-la\"},\"id\":1}' | socat -T2 - UNIX-CONNECT:$SOCKET >/dev/null 2>&1
        echo '{\"jsonrpc\":\"2.0\",\"method\":\"goposix.cat\",\"params\":{\"path\":\"/tmp/bench/rpc_bench/workspace/README.md\"},\"id\":2}' | socat -T2 - UNIX-CONNECT:$SOCKET >/dev/null 2>&1
        echo '{\"jsonrpc\":\"2.0\",\"method\":\"goposix.grep\",\"params\":{\"pattern\":\"TODO\",\"path\":\"/tmp/bench/rpc_bench/workspace/README.md\"},\"id\":3}' | socat -T2 - UNIX-CONNECT:$SOCKET >/dev/null 2>&1
        echo '{\"jsonrpc\":\"2.0\",\"method\":\"goposix.wc\",\"params\":{\"path\":\"/tmp/bench/rpc_bench/workspace/README.md\",\"flags\":\"-l\"},\"id\":4}' | socat -T2 - UNIX-CONNECT:$SOCKET >/dev/null 2>&1
        echo '{\"jsonrpc\":\"2.0\",\"method\":\"goposix.find\",\"params\":{\"path\":\"/tmp/bench/rpc_bench/workspace\",\"flags\":\"-name=*.go\"},\"id\":5}' | socat -T2 - UNIX-CONNECT:$SOCKET >/dev/null 2>&1
      done )" | tee -a "$ACCUM"

  kill "$DAEMON_PID" 2>/dev/null || true
  wait "$DAEMON_PID" 2>/dev/null || true
else
  echo "ERROR: daemon failed to start" >&2
fi
rm -f "$SOCKET"

# ===========================================================================
# Log: compute medians.
# ===========================================================================
GPX_MED=$(grep "rpc_loop_${ITERATIONS}_goposix" "$ACCUM" | cut -d, -f3 | bench_median)
BBX_MED=$(grep "rpc_loop_${ITERATIONS}_busybox" "$ACCUM" | cut -d, -f3 | bench_median)

GPX_MED=${GPX_MED:-0}
BBX_MED=${BBX_MED:-0}

{
  echo ""
  echo "## Cat J — RPC Task Loop ($ITERATIONS iterations, median of $SAMPLES)"
  echo ""
  echo "| Mode | Time (s) | Per-Iteration (ms) |"
  echo "|------|:--------:|:------------------:|"
} >&2

if [ "$(echo "$GPX_MED > 0" | bc -l 2>/dev/null)" = "1" ]; then
  gpx_per=$(awk "BEGIN { printf \"%.2f\", ($GPX_MED / $ITERATIONS) * 1000 }" 2>/dev/null || echo "0")
  echo "| GoPOSIX Daemon | ${GPX_MED} | ${gpx_per} |" >&2
else
  echo "| GoPOSIX Daemon | — | — |" >&2
fi

if [ "$(echo "$BBX_MED > 0" | bc -l 2>/dev/null)" = "1" ]; then
  bbx_per=$(awk "BEGIN { printf \"%.2f\", ($BBX_MED / $ITERATIONS) * 1000 }" 2>/dev/null || echo "0")
  echo "| BusyBox Process | ${BBX_MED} | ${bbx_per} |" >&2
else
  echo "| BusyBox Process | — | — |" >&2
fi

echo "" >&2

if [ "$(echo "$BBX_MED > 0" | bc -l 2>/dev/null)" = "1" ] && [ "$(echo "$GPX_MED > 0" | bc -l 2>/dev/null)" = "1" ]; then
  ratio=$(awk "BEGIN { printf \"%.1f\", $GPX_MED / $BBX_MED }" 2>/dev/null || echo "-")
  if [ "$(echo "$GPX_MED < $BBX_MED" | bc -l 2>/dev/null)" = "1" ]; then
    echo "# FINDING: RPC task loop at $ITERATIONS iterations: GoPOSIX daemon wins (${GPX_MED}s vs ${BBX_MED}s, ${ratio}× faster)." >&2
  else
    echo "# FINDING: RPC task loop at $ITERATIONS iterations: BusyBox wins (${BBX_MED}s vs ${GPX_MED}s, ${ratio}× slower for GoPOSIX)." >&2
  fi
fi
echo "# FINDING: Per-iteration cost: GoPOSIX daemon ${gpx_per:-?}ms vs BusyBox ${bbx_per:-?}ms. This benchmark measures sustained RPC throughput." >&2

rm -f "$ACCUM"
