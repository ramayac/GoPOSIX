#!/bin/sh
# =============================================================================
# Cat J — End-to-End RPC Task Loop Simulation.
# Simulates a typical programmatic task flow repeated N times:
#   ls → cat → grep → wc → find
# Three modes compared:
#   1. GoPOSIX daemon via Go SDK (persistent connection, 5 typed calls/iter)
#   2. GoPOSIX daemon via socat (one connection per call — legacy, kept for comparison)
#   3. BusyBox process-per-command (fork+exec per command)
# =============================================================================

set -u
. "$(dirname "$0")/lib/harness.sh"

SAMPLES=3
ITERATIONS=$(scaled 10 "$MAX_LOOP_ITERATIONS")

echo "# Cat J — End-to-End RPC Task Loop (3 modes)" >&2
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

SOCKET="$BENCH_TMPDIR/goposix-bench-j.sock"

# === Start daemon once for all GoPOSIX modes. ===
echo "# Starting GoPOSIX daemon..." >&2
rm -f "$SOCKET"
/bin/goposix daemon --socket "$SOCKET" 2>/dev/null &
DAEMON_PID=$!
sleep 1

if ! kill -0 "$DAEMON_PID" 2>/dev/null; then
  echo "ERROR: daemon failed to start" >&2
  exit 1
fi

# === Mode 1: GoPOSIX daemon via Go SDK (persistent connection, typed calls) ===
echo "# GoPOSIX SDK RPC task loop ($ITERATIONS iterations)" >&2
for i in $(seq "$SAMPLES"); do
  /bench/bench-sdk-client -socket "$SOCKET" -op rpc-loop -workspace "$WORKDIR/workspace" $ITERATIONS 2>/dev/null
  sleep 1
done | tee -a "$ACCUM"

# === Mode 2: GoPOSIX daemon via socat (one connection per call — legacy) ===
LS_REQ='{"jsonrpc":"2.0","method":"goposix.ls","params":{"path":"/tmp/bench/rpc_bench/workspace","flags":"-la"},"id":1}'
CAT_REQ='{"jsonrpc":"2.0","method":"goposix.cat","params":{"path":"/tmp/bench/rpc_bench/workspace/README.md"},"id":2}'
GREP_REQ='{"jsonrpc":"2.0","method":"goposix.grep","params":{"pattern":"TODO","path":"/tmp/bench/rpc_bench/workspace/README.md"},"id":3}'
WC_REQ='{"jsonrpc":"2.0","method":"goposix.wc","params":{"path":"/tmp/bench/rpc_bench/workspace/README.md","flags":"-l"},"id":4}'
FIND_REQ='{"jsonrpc":"2.0","method":"goposix.find","params":{"path":"/tmp/bench/rpc_bench/workspace","flags":"-name=*.go"},"id":5}'

echo "# GoPOSIX socat RPC task loop ($ITERATIONS iterations)" >&2
bench_run "rpc_loop_socat_${ITERATIONS}_goposix" "$SAMPLES" \
  "( for _i in \$(seq $ITERATIONS); do
      echo '$LS_REQ' | socat -T2 - UNIX-CONNECT:$SOCKET >/dev/null 2>&1
      echo '$CAT_REQ' | socat -T2 - UNIX-CONNECT:$SOCKET >/dev/null 2>&1
      echo '$GREP_REQ' | socat -T2 - UNIX-CONNECT:$SOCKET >/dev/null 2>&1
      echo '$WC_REQ' | socat -T2 - UNIX-CONNECT:$SOCKET >/dev/null 2>&1
      echo '$FIND_REQ' | socat -T2 - UNIX-CONNECT:$SOCKET >/dev/null 2>&1
    done )" | tee -a "$ACCUM"

# Kill daemon before testing BusyBox.
kill "$DAEMON_PID" 2>/dev/null || true
wait "$DAEMON_PID" 2>/dev/null || true
rm -f "$SOCKET"
sleep 1

# === Mode 3: BusyBox process-per-command ===
RPC_CMDS="cd $WORKDIR/workspace && ls -la . >/dev/null && cat README.md >/dev/null && grep TODO README.md >/dev/null && wc -l README.md >/dev/null && find . -name '*.go' >/dev/null"

echo "# BusyBox RPC task loop ($ITERATIONS iterations)" >&2
bench_run "rpc_loop_fork_${ITERATIONS}_busybox" "$SAMPLES" \
  "( for _i in \$(seq $ITERATIONS); do PATH=/bin $RPC_CMDS; done )" | tee -a "$ACCUM"

# ===========================================================================
# Log: compute medians, emit table + findings.
# ===========================================================================
GPX_SDK_MED=$(grep "daemon_sdk_rpc-loop_${ITERATIONS}," "$ACCUM" | cut -d, -f3 | bench_median)
GPX_SOCAT_MED=$(grep "rpc_loop_socat_${ITERATIONS}_goposix," "$ACCUM" | cut -d, -f3 | bench_median)
BBX_MED=$(grep "rpc_loop_fork_${ITERATIONS}_busybox," "$ACCUM" | cut -d, -f3 | bench_median)

GPX_SDK_MED=${GPX_SDK_MED:-0}
GPX_SOCAT_MED=${GPX_SOCAT_MED:-0}
BBX_MED=${BBX_MED:-0}

{
  echo ""
  echo "## Cat J — RPC Task Loop ($ITERATIONS iterations, median of $SAMPLES)"
  echo ""
  echo "| Mode | Time (s) | Per-Iteration (ms) | vs BusyBox |"
  echo "|------|:--------:|:------------------:|:----------:|"
} >&2

if [ "$(echo "$BBX_MED > 0" | bc -l 2>/dev/null)" = "1" ]; then
  bbx_per=$(awk "BEGIN { printf \"%.2f\", ($BBX_MED / $ITERATIONS) * 1000 }" 2>/dev/null || echo "?")
  echo "| BusyBox Fork | ${BBX_MED} | ${bbx_per} | baseline |" >&2
else
  echo "| BusyBox Fork | — | — | — |" >&2
fi

if [ "$(echo "$GPX_SOCAT_MED > 0" | bc -l 2>/dev/null)" = "1" ]; then
  gpx_socat_per=$(awk "BEGIN { printf \"%.2f\", ($GPX_SOCAT_MED / $ITERATIONS) * 1000 }" 2>/dev/null || echo "?")
  echo "| GPX Daemon (socat) | ${GPX_SOCAT_MED} | ${gpx_socat_per} | — |" >&2
else
  echo "| GPX Daemon (socat) | — | — | — |" >&2
fi

if [ "$(echo "$GPX_SDK_MED > 0" | bc -l 2>/dev/null)" = "1" ]; then
  gpx_sdk_per=$(awk "BEGIN { printf \"%.2f\", ($GPX_SDK_MED / $ITERATIONS) * 1000 }" 2>/dev/null || echo "?")
  echo "| **GPX Daemon (SDK)** | ${GPX_SDK_MED} | ${gpx_sdk_per} | — |" >&2
else
  echo "| **GPX Daemon (SDK)** | — | — | — |" >&2
fi

echo "" >&2

# Findings.
if [ "$(echo "$GPX_SDK_MED > 0" | bc -l 2>/dev/null)" = "1" ] && [ "$(echo "$BBX_MED > 0" | bc -l 2>/dev/null)" = "1" ]; then
  if [ "$(echo "$GPX_SDK_MED < $BBX_MED" | bc -l 2>/dev/null)" = "1" ]; then
    ratio=$(awk "BEGIN { printf \"%.1f\", $BBX_MED / $GPX_SDK_MED }" 2>/dev/null || echo "?")
    echo "# FINDING: RPC task loop ($ITERATIONS iterations): GoPOSIX SDK wins — ${GPX_SDK_MED}s vs BusyBox ${BBX_MED}s (${ratio}× faster)." >&2
  else
    ratio=$(awk "BEGIN { printf \"%.1f\", $GPX_SDK_MED / $BBX_MED }" 2>/dev/null || echo "?")
    echo "# FINDING: RPC task loop ($ITERATIONS iterations): BusyBox wins — ${BBX_MED}s vs GoPOSIX SDK ${GPX_SDK_MED}s (${ratio}× faster)." >&2
  fi
fi
echo "# FINDING: Per-iteration: Go SDK ${gpx_sdk_per:-?}ms vs socat ${gpx_socat_per:-?}ms vs BusyBox ${bbx_per:-?}ms. SDK eliminates 5 socat connections per iteration." >&2

rm -f "$ACCUM"
