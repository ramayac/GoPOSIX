#!/bin/sh
# =============================================================================
# Cat J — End-to-End Agent Loop Simulation.
# Simulates a typical AI agent task flow repeated N times:
#   ls → cat → grep → wc → find
# Compares GoPOSIX daemon vs BusyBox process-per-command.
# =============================================================================

set -u
. "$(dirname "$0")/lib/harness.sh"

SAMPLES=3
ITERATIONS=$(scaled 10 "$MAX_LOOP_ITERATIONS")

echo "# Cat J — End-to-End Agent Loop" >&2
echo "# scale=$BENCH_SCALE iterations=$ITERATIONS" >&2
echo "" >&2

WORKDIR="$BENCH_TMPDIR/agent_bench"
mkdir -p "$WORKDIR/workspace"

# Set up a realistic workspace.
if [ ! -f "$WORKDIR/workspace/README.md" ]; then
  echo "# Setting up agent workspace..." >&2
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

  # Create some .go files.
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

# Agent loop script — a single invocation of this sequence.
AGENT_SCRIPT='
cd /tmp/bench/agent_bench/workspace || exit 1
$LS -la . >/dev/null
$CAT README.md >/dev/null
$GREP "TODO" README.md >/dev/null
$WC -l README.md >/dev/null
$FIND . -name "*.go" >/dev/null
'

# === BusyBox process-per-command ===
echo "# BusyBox agent loop ($ITERATIONS iterations)" >&2
bench_run "agent_loop_${ITERATIONS}_busybox" "$SAMPLES" \
  "( for _i in \$(seq $ITERATIONS); do
      LS=/bin/busybox
      CAT=/bin/busybox
      GREP=/bin/busybox
      WC=/bin/busybox
      FIND=/bin/busybox
      eval \"$AGENT_SCRIPT\"
    done )"

# === GoPOSIX Daemon ===
SOCKET="$BENCH_TMPDIR/goposix-bench-j.sock"
echo "# Starting GoPOSIX daemon for agent loop..." >&2
rm -f "$SOCKET"
/bin/goposix daemon --socket "$SOCKET" &
DAEMON_PID=$!
sleep 1

if kill -0 "$DAEMON_PID" 2>/dev/null; then
  echo "# GoPOSIX daemon agent loop ($ITERATIONS iterations)" >&2

  # JSON-RPC payloads.
  LS_REQ='{"jsonrpc":"2.0","method":"goposix.ls","params":{"path":"/tmp/bench/agent_bench/workspace","flags":"-la"},"id":1}'
  CAT_REQ='{"jsonrpc":"2.0","method":"goposix.cat","params":{"path":"/tmp/bench/agent_bench/workspace/README.md"},"id":2}'
  GREP_REQ='{"jsonrpc":"2.0","method":"goposix.grep","params":{"pattern":"TODO","path":"/tmp/bench/agent_bench/workspace/README.md"},"id":3}'
  WC_REQ='{"jsonrpc":"2.0","method":"goposix.wc","params":{"path":"/tmp/bench/agent_bench/workspace/README.md","flags":"-l"},"id":4}'
  FIND_REQ='{"jsonrpc":"2.0","method":"goposix.find","params":{"path":"/tmp/bench/agent_bench/workspace","flags":"-name=*.go"},"id":5}'

  bench_run "agent_loop_${ITERATIONS}_goposix" "$SAMPLES" \
    "( for _i in \$(seq $ITERATIONS); do
        echo '$LS_REQ' | nc -w 2 -U $SOCKET >/dev/null 2>&1
        echo '$CAT_REQ' | nc -w 2 -U $SOCKET >/dev/null 2>&1
        echo '$GREP_REQ' | nc -w 2 -U $SOCKET >/dev/null 2>&1
        echo '$WC_REQ' | nc -w 2 -U $SOCKET >/dev/null 2>&1
        echo '$FIND_REQ' | nc -w 2 -U $SOCKET >/dev/null 2>&1
      done )"

  kill "$DAEMON_PID" 2>/dev/null || true
  wait "$DAEMON_PID" 2>/dev/null || true
else
  echo "ERROR: daemon failed to start" >&2
fi
rm -f "$SOCKET"

echo "" >&2
echo "# FINDING: Agent loop at $ITERATIONS iterations (scale=${BENCH_SCALE}×). GoPOSIX daemon expected to win 10–50×." >&2
echo "# FINDING: This is the benchmark that matters most for AI agent adoption." >&2
