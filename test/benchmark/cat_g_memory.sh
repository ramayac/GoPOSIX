#!/bin/sh
# =============================================================================
# Cat G — Memory Footprint (RSS).
# Measures RSS for single invocation, idle daemon, loaded daemon, and BusyBox.
# =============================================================================

set -u
. "$(dirname "$0")/lib/harness.sh"

SAMPLES=3
CONCURRENT=$(scaled 100 "$MAX_DAEMON_REQUESTS")

echo "# Cat G — Memory Footprint (RSS)" >&2
echo "# scale=$BENCH_SCALE concurrent=$CONCURRENT" >&2
echo "" >&2

SOCKET="$BENCH_TMPDIR/goposix-bench-g.sock"

# CSV header.
echo "category,test,sample,wall_sec,user_sec,sys_sec,rss_kb"

# Accumulate for stats.
ACCUM=$(mktemp)
echo "category,test,sample,wall_sec,user_sec,sys_sec,rss_kb" > "$ACCUM"

# G1: Single echo invocation RSS.
for tool in goposix busybox; do
  binary="/bin/$tool"
  echo "# G1: Single echo RSS — $tool" >&2
  for i in $(seq "$SAMPLES"); do
    timing=$( { time -f "%e %U %S %M" "$binary" echo hello >/dev/null; } 2>&1 )
    read -r wall user sys rss rest <<-TIMING_EOF
		$timing
		TIMING_EOF
    : "${rss:=0}"
    row="mem_single_echo_${tool},$i,$wall,$user,$sys,$rss"
    echo "$row"
    echo "$row" >> "$ACCUM"
    sleep 0.5
  done
done

# G2: Daemon idle RSS.
echo "# G2: Daemon idle RSS" >&2
rm -f "$SOCKET"
/bin/goposix daemon --socket "$SOCKET" 2>/dev/null &
DAEMON_PID=$!
sleep 1

if kill -0 "$DAEMON_PID" 2>/dev/null; then
  for i in $(seq "$SAMPLES"); do
    rss=$(ps -o rss= -p "$DAEMON_PID" 2>/dev/null | tr -d ' ' || echo "0")
    row="mem_daemon_idle_goposix,$i,0,0,0,$rss"
    echo "$row"
    echo "$row" >> "$ACCUM"
    sleep 0.5
  done

  # G3: Daemon under concurrent load.
  echo "# G3: Daemon under load ($CONCURRENT concurrent)" >&2
  JSON_REQ='{"jsonrpc":"2.0","method":"goposix.echo","params":{"text":"hello"},"id":1}'

  for i in $(seq "$SAMPLES"); do
    socat_pids=""
    for j in $(seq "$CONCURRENT"); do
      echo "$JSON_REQ" | socat -T2 - UNIX-CONNECT:"$SOCKET" >/dev/null 2>&1 &
      socat_pids="$socat_pids $!"
    done
    sleep 0.3
    rss=$(ps -o rss= -p "$DAEMON_PID" 2>/dev/null | tr -d ' ' || echo "0")
    row="mem_daemon_loaded_${CONCURRENT}_goposix,$i,0,0,0,$rss"
    echo "$row"
    echo "$row" >> "$ACCUM"
    for pid in $socat_pids; do
      wait "$pid" 2>/dev/null || true
    done
    sleep 1
  done

  kill "$DAEMON_PID" 2>/dev/null || true
  wait "$DAEMON_PID" 2>/dev/null || true
fi
rm -f "$SOCKET"

# G4: BusyBox peak RSS during sequential calls.
echo "# G4: BusyBox sequential peak RSS ($CONCURRENT calls)" >&2
for i in $(seq "$SAMPLES"); do
  (
    for j in $(seq "$CONCURRENT"); do
      /bin/busybox echo hello >/dev/null
    done
  ) &
  PID=$!
  PEAK=0
  while kill -0 "$PID" 2>/dev/null; do
    rss=$(ps -o rss= -p "$PID" 2>/dev/null | tr -d ' ' || echo "0")
    [ "$rss" -gt "$PEAK" ] && PEAK="$rss"
    sleep 0.05
  done
  wait "$PID"
  row="mem_busybox_sequential_${CONCURRENT},$i,0,0,0,$PEAK"
  echo "$row"
  echo "$row" >> "$ACCUM"
  sleep 0.5
done

# ===========================================================================
# Log: compute medians.
# ===========================================================================
{
  echo ""
  echo "## Cat G — Memory Footprint (RSS KB, median of $SAMPLES)"
  echo ""
  echo "| Scenario | GoPOSIX | BusyBox | Ratio | Winner |"
  echo "|----------|:-------:|:-------:|:-----:|:------:|"
} >&2

# Single echo
GPX_SINGLE=$(grep "mem_single_echo_goposix" "$ACCUM" | cut -d, -f6 | bench_median)
BBX_SINGLE=$(grep "mem_single_echo_busybox" "$ACCUM" | cut -d, -f6 | bench_median)
r_echo=$(awk "BEGIN { printf \"%.1f\", ${GPX_SINGLE:-0} / ${BBX_SINGLE:-1} }" 2>/dev/null || echo "-")
echo "| Single echo | ${GPX_SINGLE} | ${BBX_SINGLE} | ${r_echo}× | BusyBox |" >&2

# Daemon idle
GPX_IDLE=$(grep "mem_daemon_idle_goposix" "$ACCUM" | cut -d, -f6 | bench_median)
echo "| Daemon idle | ${GPX_IDLE} | — | — | — |" >&2

# Daemon loaded
GPX_LOAD=$(grep "mem_daemon_loaded_${CONCURRENT}_goposix" "$ACCUM" | cut -d, -f6 | bench_median)
echo "| Daemon ($CONCURRENT req) | ${GPX_LOAD} | — | — | — |" >&2

# BusyBox sequential
BBX_SEQ=$(grep "mem_busybox_sequential_${CONCURRENT}" "$ACCUM" | cut -d, -f6 | bench_median)
echo "| Sequential ($CONCURRENT calls) | — | ${BBX_SEQ} | — | — |" >&2

echo "" >&2

echo "# FINDING: Single invocation RSS: GoPOSIX ${GPX_SINGLE}KB vs BusyBox ${BBX_SINGLE}KB (${r_echo}×). BusyBox wins per-invocation." >&2
echo "# FINDING: GoPOSIX daemon idle: ${GPX_IDLE}KB, under load: ${GPX_LOAD}KB. Fixed cost amortized over many calls." >&2

rm -f "$ACCUM"
