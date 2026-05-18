#!/bin/sh
# =============================================================================
# Cat A — Single-Invocation Latency (cold start overhead).
# Measures pure startup cost of GoPOSIX vs BusyBox.
# Uses nanosecond-resolution timing (date +%s%N) because these commands
# complete in <20ms where BusyBox 'time -f' rounds to 0.00.
# =============================================================================

set -u
. "$(dirname "$0")/lib/harness.sh"

SAMPLES=10

echo "# Cat A — Single-Invocation Cold-Start Latency" >&2
echo "# scale=$BENCH_SCALE samples=$SAMPLES (nanosecond wall-clock, no RSS)" >&2
echo "" >&2

# CSV header.
echo "category,test,sample,wall_sec,user_sec,sys_sec,rss_kb"

# Accumulate all CSV rows for stats computation.
ACCUM=$(mktemp)
echo "category,test,sample,wall_sec,user_sec,sys_sec,rss_kb" > "$ACCUM"

for cmd_raw in "true true" "echo hello echo_hello" "pwd pwd" "whoami whoami"; do
  cmd=$(echo "$cmd_raw" | awk '{print $1}')
  label=$(echo "$cmd_raw" | awk '{print $2}')

  # GoPOSIX — nanosecond wall clock.
  bench_run_fine "startup_${label}_goposix" "$SAMPLES" "/bin/goposix" "$cmd" | tee -a "$ACCUM"
  # BusyBox — nanosecond wall clock.
  bench_run_fine "startup_${label}_busybox" "$SAMPLES" "/bin/busybox" "$cmd" | tee -a "$ACCUM"
done

# ===========================================================================
# Log: compute medians from accumulated data, emit findings + table.
# ===========================================================================
{
  echo ""
  echo "## Cat A — Single-Invocation Latency (ms, median of $SAMPLES)"
  echo ""
  echo "| Test | GoPOSIX (ms) | BusyBox (ms) | Ratio | Winner |"
  echo "|------|:------------:|:------------:|:-----:|:------:|"
} >&2

for cmd_raw in "true true" "echo hello echo_hello" "pwd pwd" "whoami whoami"; do
  cmd=$(echo "$cmd_raw" | awk '{print $1}')
  label=$(echo "$cmd_raw" | awk '{print $2}')

  # Compute median wall time in seconds, convert to ms.
  gpx_secs=$(grep "startup_${label}_goposix" "$ACCUM" | cut -d, -f3 | bench_median)
  bbx_secs=$(grep "startup_${label}_busybox" "$ACCUM" | cut -d, -f3 | bench_median)

  gpx_ms=$(awk "BEGIN { printf \"%.2f\", ${gpx_secs:-0} * 1000 }" 2>/dev/null || echo "0.00")
  bbx_ms=$(awk "BEGIN { printf \"%.2f\", ${bbx_secs:-0} * 1000 }" 2>/dev/null || echo "0.00")

  if [ "$(echo "$bbx_secs > 0" | bc -l 2>/dev/null)" = "1" ]; then
    ratio=$(awk "BEGIN { printf \"%.1f\", $gpx_secs / $bbx_secs }" 2>/dev/null || echo "-")
    if [ "$(echo "$gpx_secs < $bbx_secs" | bc -l 2>/dev/null)" = "1" ]; then
      winner="**GoPOSIX**"
      echo "# FINDING: GoPOSIX $cmd wins (${gpx_ms}ms vs ${bbx_ms}ms, ${ratio}× faster)" >&2
    else
      winner="BusyBox"
      echo "# FINDING: BusyBox $cmd wins (${bbx_ms}ms vs ${gpx_ms}ms, ${ratio}× faster)" >&2
    fi
    echo "| $cmd | ${gpx_ms} | ${bbx_ms} | ${ratio}× | $winner |" >&2
  else
    echo "# FINDING: $cmd: GoPOSIX ${gpx_ms}ms vs BusyBox ${bbx_ms}ms (times too fast to ratio)" >&2
    echo "| $cmd | ${gpx_ms} | ${bbx_ms} | — | — |" >&2
  fi
done
echo "" >&2
rm -f "$ACCUM"
