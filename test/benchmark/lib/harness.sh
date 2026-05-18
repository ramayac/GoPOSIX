#!/bin/sh
# =============================================================================
# GoPOSIX Benchmark Harness — shared timing, stats, and scale functions.
# Source in every benchmark category script.
# =============================================================================

# Scale factor: single knob controlling all workload sizes.
# Default 1.0 (standard). Set via environment: BENCH_SCALE=5.0 make bench-all
: "${BENCH_SCALE:=1.0}"

# Directory for temporary benchmark data (e.g., file trees, text files).
: "${BENCH_TMPDIR:=/tmp/bench}"

# Results output directory (set by runner.sh).
: "${BENCH_OUTDIR:=/data}"

# Hard caps to prevent runaway runs.
MAX_FILE_COUNT=500000
MAX_TEXT_MB=10240
MAX_DAEMON_REQUESTS=100000
MAX_LOOP_ITERATIONS=1000

# ---------------------------------------------------------------------------
# Scale a base value by BENCH_SCALE, clamp to [1, cap].
# Usage: FILE_COUNT=$(scaled 1000)   # → 1000 at SCALE=1, 5000 at SCALE=5
#        FILE_COUNT=$(scaled 1000 10000)  # max cap 10,000
# ---------------------------------------------------------------------------
scaled() {
  local base="$1"
  local cap="${2:-500000}"
  local val
  val=$(awk "BEGIN { v = int($base * $BENCH_SCALE + 0.5); if (v < 1) v = 1; printf \"%d\", v }")
  if [ "$val" -gt "$cap" ]; then
    echo "WARNING: scaled value $val (base=$base × scale=$BENCH_SCALE) capped at $cap" >&2
    val=$cap
  fi
  printf "%d" "$val"
}

# Human-readable config dump for report headers.
bench_config() {
  echo "scale=$BENCH_SCALE"
  echo "tmpdir=$BENCH_TMPDIR"
  echo "outdir=$BENCH_OUTDIR"
  echo "host=$(uname -m) $(nproc 2>/dev/null || echo 1)cpu"
}

# ---------------------------------------------------------------------------
# Run a command N times, collect timing metrics, write CSV to stdout.
# Usage: bench_run "label" 10 "command string"
# Columns: label,sample,wall_sec,user_sec,sys_sec,rss_kb
#
# Uses BusyBox 'time -f' which writes formatted output to stderr.
# ---------------------------------------------------------------------------
bench_run() {
  local label="$1"
  local samples="$2"
  local cmd="$3"

  # Warmup: 3 discarded runs
  for _i in 1 2 3; do
    eval "$cmd" >/dev/null 2>&1
  done

  # Measured runs
  local i
  for i in $(seq "$samples"); do
    local timing
    # BusyBox time writes formatted output to stderr, stdout from cmd discarded.
    timing=$( { time -f "%e %U %S %M" sh -c "$cmd" >/dev/null; } 2>&1 )
    local wall user sys rss rest
    read -r wall user sys rss rest <<-TIMING_EOF
		$timing
		TIMING_EOF
    # Handle case where rss might be missing (some BusyBox builds).
    : "${rss:=0}"
    echo "$label,$i,$wall,$user,$sys,$rss"
    sleep 1
  done
}

# ---------------------------------------------------------------------------
# Quick single-shot timing (no warmup, no stats).
# Usage: wall_sec=$(bench_time "command string")
# ---------------------------------------------------------------------------
bench_time() {
  local cmd="$1"
  local timing
  timing=$( { time -f "%e" sh -c "$cmd" >/dev/null; } 2>&1 )
  echo "$timing"
}

# ---------------------------------------------------------------------------
# Compute median from space-delimited numeric column.
# Usage: echo "5 2 8" | bench_median → 5
# ---------------------------------------------------------------------------
bench_median() {
  sort -n | awk '{
    arr[NR]=$1
  } END {
    if (NR % 2 == 1)
      print arr[(NR+1)/2]
    else
      print (arr[NR/2] + arr[NR/2+1]) / 2
  }'
}

# ---------------------------------------------------------------------------
# Compute min, median, p95, max from CSV column.
# Input: CSV with the target column as field $col.
# Output: min median p95 max
# ---------------------------------------------------------------------------
bench_stats() {
  local col="${1:-3}"  # default: wall-clock (column 3)
  local data
  data=$(cut -d, -f"$col" | sort -n)
  if [ -z "$data" ]; then
    echo "0 0 0 0"
    return
  fi
  local min med p95 max n
  min=$(echo "$data" | head -1)
  max=$(echo "$data" | tail -1)
  n=$(echo "$data" | wc -l)
  med=$(echo "$data" | bench_median)
  # p95: ceil(0.95 * n)
  local idx
  idx=$(awk "BEGIN { printf \"%d\", int(0.95 * $n + 0.9999) }")
  [ "$idx" -lt 1 ] && idx=1
  p95=$(echo "$data" | sed -n "${idx}p")
  echo "$min $med $p95 $max"
}

# ---------------------------------------------------------------------------
# Print a markdown table row from a comma-separated header + CSV data.
# Usage: bench_md_table "Test,GoPOSIX,BusyBox,Ratio,Winner" < csv_data
# CSV data: testname,$col3_goposix,$col3_busybox
# ---------------------------------------------------------------------------
bench_md_table() {
  local header="$1"
  echo ""
  # Print header
  echo "$header" | awk -F, '{
    printf "|";
    for(i=1;i<=NF;i++) printf " %s |", $i;
    printf "\n"
  }'
  # Print separator
  echo "$header" | awk -F, '{
    printf "|";
    for(i=1;i<=NF;i++) printf " --- |";
    printf "\n"
  }'
  # Print data rows
  while IFS=',' read -r test gx bb; do
    if [ -z "$test" ]; then continue; fi
    local ratio winner
    if [ "$(echo "$bb > 0" | bc -l 2>/dev/null)" = "1" ]; then
      ratio=$(awk "BEGIN { printf \"%.1f\", $gx / $bb }" 2>/dev/null || echo "-")
      if [ "$(echo "$gx < $bb" | bc -l 2>/dev/null)" = "1" ]; then
        winner="**GoPOSIX**"
      elif [ "$(echo "$gx > $bb" | bc -l 2>/dev/null)" = "1" ]; then
        winner="BusyBox"
      else
        winner="Tie"
      fi
      printf "| %s | %.4f | %.4f | %s× | %s |\n" "$test" "$gx" "$bb" "$ratio" "$winner"
    else
      printf "| %s | %.4f | %.4f | — | — |\n" "$test" "$gx" "$bb"
    fi
  done
  echo ""
}

# ---------------------------------------------------------------------------
# Print a scaled-parameters summary for the report.
# ---------------------------------------------------------------------------
bench_print_config() {
  echo ""
  echo "## Run Configuration"
  echo ""
  echo "| Parameter | Value |"
  echo "|-----------|-------|"
  echo "| BENCH_SCALE | ${BENCH_SCALE}× |"
  echo "| Host | $(uname -m) |"
  echo "| CPUs | $(nproc 2>/dev/null || echo '?') |"
  echo "| Temp dir | $BENCH_TMPDIR |"
  echo ""
}
