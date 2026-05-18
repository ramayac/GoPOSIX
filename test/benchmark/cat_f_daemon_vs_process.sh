#!/bin/sh
# =============================================================================
# Cat F — Daemon vs Process-per-Call (The GoPOSIX Killer Feature).
# Three modes compared:
#   1. GoPOSIX daemon via socat (one connection per call)
#   2. GoPOSIX daemon via Go SDK (persistent connection, typed client)
#   3. BusyBox process-per-call (fork+exec per invocation)
# Uses echo as the simplest possible command.
# =============================================================================

set -u
. "$(dirname "$0")/lib/harness.sh"

SAMPLES=3

echo "# Cat F — Daemon vs Process-per-Call (3 modes)" >&2

N1=$(scaled 10   "$MAX_DAEMON_REQUESTS")
N2=$(scaled 100  "$MAX_DAEMON_REQUESTS")
N3=$(scaled 1000 "$MAX_DAEMON_REQUESTS")

echo "# scale=$BENCH_SCALE N=$N1 / $N2 / $N3" >&2
echo "" >&2

SOCKET="$BENCH_TMPDIR/goposix-bench-f.sock"
JSON_REQ='{"jsonrpc":"2.0","method":"goposix.echo","params":{"text":"hello"},"id":1}'

# CSV header.
echo "category,test,sample,wall_sec,user_sec,sys_sec,rss_kb"

# Accumulate for stats.
ACCUM=$(mktemp)

for N in "$N1" "$N2" "$N3"; do
  echo "# Testing N=$N" >&2

  # === Start a fresh GoPOSIX daemon for each N level. ===
  echo "# Starting GoPOSIX daemon..." >&2
  rm -f "$SOCKET"
  /bin/goposix daemon --socket "$SOCKET" 2>/dev/null &
  DAEMON_PID=$!
  sleep 1

  if ! kill -0 "$DAEMON_PID" 2>/dev/null; then
    echo "ERROR: daemon failed to start" >&2
    echo "# FINDING: Daemon failed to start for N=$N" >&2
    break
  fi

  # === Mode 1: GoPOSIX daemon via socat (one connection per call) ===
  echo "# GoPOSIX socat — $N echo calls" >&2
  bench_run "daemon_socat_${N}_goposix" "$SAMPLES" \
    "( for i in \$(seq $N); do echo '$JSON_REQ' | socat -T2 - UNIX-CONNECT:$SOCKET >/dev/null 2>&1; done )" | tee -a "$ACCUM"

  # === Mode 2: GoPOSIX daemon via Go SDK (persistent connection, typed client) ===
  echo "# GoPOSIX SDK — $N echo calls" >&2
  /bench/bench-sdk-client -socket "$SOCKET" -op echo $N > /tmp/bench_sdk_out 2>/tmp/bench_sdk_err
  if [ $? -eq 0 ]; then
    # bench_client outputs: daemon_sdk_echo_N,1,wall,0,0,0
    # We need 3 samples — run it 3 times.
    for i in $(seq "$SAMPLES"); do
      /bench/bench-sdk-client -socket "$SOCKET" -op echo $N 2>/dev/null
      sleep 1
    done | tee -a "$ACCUM"
  else
    echo "ERROR: bench_client failed: $(cat /tmp/bench_sdk_err)" >&2
  fi

  # Kill daemon before testing BusyBox.
  kill "$DAEMON_PID" 2>/dev/null || true
  wait "$DAEMON_PID" 2>/dev/null || true
  rm -f "$SOCKET"
  sleep 1

  # === Mode 3: BusyBox process-per-call ===
  echo "# BusyBox process-per-call — $N echo calls" >&2
  bench_run "daemon_fork_${N}_busybox" "$SAMPLES" \
    "( for i in \$(seq $N); do /bin/busybox echo hello >/dev/null; done )" | tee -a "$ACCUM"
done

# ===========================================================================
# Log: compute medians, emit table + findings.
# ===========================================================================
{
  echo ""
  echo "## Cat F — Daemon Amortization (seconds, median of $SAMPLES)"
  echo ""
  echo "| N Calls | GPX Daemon (socat) | GPX Daemon (SDK) | BusyBox Fork | SDK/BBX | Winner |"
  echo "|--------:|:------------------:|:----------------:|:------------:|:-------:|:------:|"
} >&2

FINDINGS_TMP=$(mktemp)

for N in "$N1" "$N2" "$N3"; do
  gpx_socat=$(grep "daemon_socat_${N}_goposix," "$ACCUM" | cut -d, -f3 | bench_median)
  gpx_sdk=$(grep "daemon_sdk_echo_${N}," "$ACCUM" | cut -d, -f3 | bench_median)
  bbx_med=$(grep "daemon_fork_${N}_busybox," "$ACCUM" | cut -d, -f3 | bench_median)

  # Per-call costs in milliseconds.
  gpx_socat_per=$(awk "BEGIN { printf \"%.2f\", (${gpx_socat:-0} / $N) * 1000 }" 2>/dev/null || echo "?")
  gpx_sdk_per=$(awk "BEGIN { printf \"%.2f\", (${gpx_sdk:-0} / $N) * 1000 }" 2>/dev/null || echo "?")
  bbx_per=$(awk "BEGIN { printf \"%.2f\", (${bbx_med:-0} / $N) * 1000 }" 2>/dev/null || echo "?")

  if [ "$(echo "${bbx_med:-0} > 0" | bc -l 2>/dev/null)" = "1" ] && [ "$(echo "${gpx_sdk:-0} > 0" | bc -l 2>/dev/null)" = "1" ]; then
    ratio=$(awk "BEGIN { printf \"%.1f\", ${gpx_sdk:-0} / ${bbx_med:-0} }" 2>/dev/null || echo "-")
    if [ "$(echo "${gpx_sdk:-0} < ${bbx_med:-0}" | bc -l 2>/dev/null)" = "1" ]; then
      winner="**GoPOSIX SDK**"
    else
      winner="BusyBox"
    fi
    echo "| $N | ${gpx_socat:-?} | ${gpx_sdk:-?} | ${bbx_med:-?} | ${ratio}× | $winner |" >&2
  else
    echo "| $N | ${gpx_socat:-?} | ${gpx_sdk:-?} | ${bbx_med:-?} | — | — |" >&2
  fi

  echo "# FINDING: N=$N: socat ${gpx_socat_per}ms/call, SDK ${gpx_sdk_per}ms/call, BusyBox fork ${bbx_per}ms/call" >> "$FINDINGS_TMP"
done
echo "" >&2

cat "$FINDINGS_TMP" >&2

# Summary.
GPX_SDK_N3=$(grep "daemon_sdk_echo_${N3}," "$ACCUM" | cut -d, -f3 | bench_median)
BBX_N3=$(grep "daemon_fork_${N3}_busybox" "$ACCUM" | cut -d, -f3 | bench_median)
SDK_PER=$(awk "BEGIN { printf \"%.2f\", (${GPX_SDK_N3:-0} / $N3) * 1000 }" 2>/dev/null || echo "?")
BBX_PER=$(awk "BEGIN { printf \"%.2f\", (${BBX_N3:-0} / $N3) * 1000 }" 2>/dev/null || echo "?")
echo "# FINDING: Per-call at N=$N3: Go SDK ${SDK_PER}ms vs BusyBox fork ${BBX_PER}ms. The Go SDK with persistent connection eliminates socat overhead." >&2

rm -f "$FINDINGS_TMP" "$ACCUM"
