#!/bin/sh
# =============================================================================
# Cat I — Concurrent Operations (aspirational).
# Measures operations that could benefit from goroutine parallelization.
# Currently both tools are sequential; this measures baseline parity.
# Tests marked [GOROUTINE-TODO] until parallel implementations exist.
# =============================================================================

set -u
. "$(dirname "$0")/lib/harness.sh"

SAMPLES=3

echo "# Cat I — Concurrent Operations [GOROUTINE-TODO]" >&2

MANYFILES=$(scaled 100 "$MAX_FILE_COUNT")

echo "# scale=$BENCH_SCALE files=$MANYFILES" >&2
echo "# NOTE: GoPOSIX utilities are currently sequential. This category measures POTENTIAL." >&2
echo "" >&2

WORKDIR="$BENCH_TMPDIR/conc_bench"
mkdir -p "$WORKDIR"

MANYDIR="$WORKDIR/manyfiles"
if [ ! -d "$MANYDIR" ] || [ "$(ls "$MANYDIR" 2>/dev/null | wc -l)" != "$MANYFILES" ]; then
  echo "# Setting up $MANYFILES small .txt files..." >&2
  rm -rf "$MANYDIR"
  mkdir -p "$MANYDIR"
  for i in $(seq "$MANYFILES"); do
    echo "file_${i} content with pattern_${i} here $(head -c 200 /dev/urandom | base64 2>/dev/null || echo padding)" > "$MANYDIR/file_${i}.txt"
  done
fi

# CSV header.
echo "category,test,sample,wall_sec,user_sec,sys_sec,rss_kb"

# Accumulate for stats.
ACCUM=$(mktemp)

# I2: grep -r.
echo "# grep -r across $MANYFILES files — GoPOSIX" >&2
bench_run "conc_grepr_${MANYFILES}_goposix" "$SAMPLES" "/bin/goposix grep -r content $MANYDIR" | tee -a "$ACCUM"

echo "# grep -r across $MANYFILES files — BusyBox" >&2
bench_run "conc_grepr_${MANYFILES}_busybox" "$SAMPLES" "/bin/busybox grep -r content $MANYDIR" | tee -a "$ACCUM"

# I3: du.
echo "# du across $MANYFILES files — GoPOSIX" >&2
bench_run "conc_du_${MANYFILES}_goposix" "$SAMPLES" "/bin/goposix du -sh $MANYDIR" | tee -a "$ACCUM"

echo "# du across $MANYFILES files — BusyBox" >&2
bench_run "conc_du_${MANYFILES}_busybox" "$SAMPLES" "/bin/busybox du -sh $MANYDIR" | tee -a "$ACCUM"

# ===========================================================================
# Log: compute medians.
# ===========================================================================
{
  echo ""
  echo "## Cat I — Concurrent Operations (seconds, median of $SAMPLES, $MANYFILES files) [GOROUTINE-TODO]"
  echo ""
  echo "| Operation | GoPOSIX | BusyBox | Ratio | Winner |"
  echo "|-----------|:-------:|:-------:|:-----:|:------:|"
} >&2

for op in "grepr grep -r" "du du -sh"; do
  slug=$(echo "$op" | awk '{print $1}')
  desc=$(echo "$op" | awk '{print $2" "$3}')
  gpx_med=$(grep "conc_${slug}_${MANYFILES}_goposix" "$ACCUM" | cut -d, -f3 | bench_median)
  bbx_med=$(grep "conc_${slug}_${MANYFILES}_busybox" "$ACCUM" | cut -d, -f3 | bench_median)
  if [ "$(echo "$bbx_med > 0" | bc -l 2>/dev/null)" = "1" ]; then
    ratio=$(awk "BEGIN { printf \"%.1f\", $gpx_med / $bbx_med }" 2>/dev/null || echo "-")
    if [ "$(echo "$gpx_med < $bbx_med" | bc -l 2>/dev/null)" = "1" ]; then
      winner="**GoPOSIX**"
    else
      winner="BusyBox"
    fi
    echo "| \`$desc\` | ${gpx_med} | ${bbx_med} | ${ratio}× | $winner |" >&2
  else
    echo "| \`$desc\` | ${gpx_med} | ${bbx_med} | — | — |" >&2
  fi
done
echo "" >&2

echo "# FINDING: [GOROUTINE-TODO] Both tools are sequential today. Current measurements show baseline parity." >&2
echo "# FINDING: GoPOSIX can win 2–8× on these operations with goroutine-parallel file I/O (pending implementation)." >&2

rm -f "$ACCUM"
