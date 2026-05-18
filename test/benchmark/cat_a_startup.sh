#!/bin/sh
# =============================================================================
# Cat A — Single-Invocation Latency (cold start overhead).
# Measures pure startup cost of GoPOSIX vs BusyBox.
# Scale has minimal effect here — always 10 samples per command.
# =============================================================================

set -u
. "$(dirname "$0")/lib/harness.sh"

SAMPLES=10
CMDS="true echo_hello pwd whoami"

echo "# Cat A — Single-Invocation Cold-Start Latency" >&2
echo "# scale=$BENCH_SCALE samples=$SAMPLES" >&2
echo "" >&2

# CSV header.
echo "category,test,sample,wall_sec,user_sec,sys_sec,rss_kb"

# Temp file to accumulate all CSV rows for stats computation.
ACCUM=$(mktemp)
echo "category,test,sample,wall_sec,user_sec,sys_sec,rss_kb" > "$ACCUM"

for cmd_raw in "true true" "echo hello echo_hello" "pwd pwd" "whoami whoami"; do
  cmd=$(echo "$cmd_raw" | awk '{print $1}')
  label=$(echo "$cmd_raw" | awk '{print $2}')

  # GoPOSIX
  bench_run "startup_${label}_goposix" "$SAMPLES" "/bin/goposix $cmd" | tee -a "$ACCUM"
  # BusyBox
  bench_run "startup_${label}_busybox" "$SAMPLES" "/bin/busybox $cmd" | tee -a "$ACCUM"
done

# ===========================================================================
# Log: compute medians and emit findings to stderr.
# ===========================================================================
{
  echo "# FINDING: See median table below."
  echo ""
  echo "## Cat A — Single-Invocation Latency (seconds, median of $SAMPLES)"
  echo ""
  echo "| Test | GoPOSIX | BusyBox | Ratio | Winner |"
  echo "|------|:-------:|:-------:|:-----:|:------:|"
} >&2

for cmd_raw in "true true" "echo hello echo_hello" "pwd pwd" "whoami whoami"; do
  cmd=$(echo "$cmd_raw" | awk '{print $1}')
  label=$(echo "$cmd_raw" | awk '{print $2}')

  gpx_med=$(grep "startup_${label}_goposix" "$ACCUM" | cut -d, -f3 | bench_median)
  bbx_med=$(grep "startup_${label}_busybox" "$ACCUM" | cut -d, -f3 | bench_median)

  if [ "$(echo "$bbx_med > 0" | bc -l 2>/dev/null)" = "1" ]; then
    ratio=$(awk "BEGIN { printf \"%.1f\", $gpx_med / $bbx_med }")
    echo "| $cmd | ${gpx_med}s | ${bbx_med}s | ${ratio}× | BusyBox |" >&2
  else
    echo "| $cmd | ${gpx_med}s | ${bbx_med}s | — | — |" >&2
  fi
done
echo "" >&2
rm -f "$ACCUM"
