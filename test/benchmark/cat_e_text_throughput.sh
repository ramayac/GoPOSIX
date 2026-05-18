#!/bin/sh
# =============================================================================
# Cat E — Text Processing Throughput (cat, grep, wc, sort).
# Generates a scaled text file, then times various operations.
# =============================================================================

set -u
. "$(dirname "$0")/lib/harness.sh"

SAMPLES=5

echo "# Cat E — Text Processing Throughput" >&2

TEXT_MB=$(scaled 100 "$MAX_TEXT_MB")
SMALL_FILES=$(scaled 1000 "$MAX_FILE_COUNT")

echo "# scale=$BENCH_SCALE text=${TEXT_MB}MB small_files=${SMALL_FILES}" >&2
echo "" >&2

WORKDIR="$BENCH_TMPDIR/text_bench"
mkdir -p "$WORKDIR"

# Generate the big text file (setup — not measured).
BIGFILE="$WORKDIR/big.txt"
if [ ! -f "$BIGFILE" ] || [ "$(stat -c%s "$BIGFILE" 2>/dev/null)" != "$((TEXT_MB * 1048576))" ]; then
  echo "# Generating ${TEXT_MB}MB text file..." >&2
  LINES=$((TEXT_MB * 10000))
  awk -v n="$LINES" 'BEGIN {
    srand(1);
    for(i=1;i<=n;i++) {
      printf "line_%d the quick brown fox jumped over the lazy dog pattern_%d\n", i, (i%100)
    }
  }' > "$BIGFILE"
  echo "# Generated $(wc -c < "$BIGFILE") bytes, $(wc -l < "$BIGFILE") lines" >&2
fi

# Generate many small files for grep -r.
MANYDIR="$WORKDIR/manyfiles"
if [ ! -d "$MANYDIR" ] || [ "$(ls "$MANYDIR" 2>/dev/null | wc -l)" != "$SMALL_FILES" ]; then
  echo "# Generating $SMALL_FILES small files for grep -r..." >&2
  rm -rf "$MANYDIR"
  mkdir -p "$MANYDIR"
  for i in $(seq "$SMALL_FILES"); do
    echo "file_${i} content with pattern_${i} here" > "$MANYDIR/file_${i}.txt"
  done
fi

# CSV header.
echo "category,test,sample,wall_sec,user_sec,sys_sec,rss_kb"

# Accumulate for stats.
ACCUM=$(mktemp)

# E1 — cat
echo "# cat ${TEXT_MB}MB GoPOSIX" >&2
bench_run "text_cat_${TEXT_MB}mb_goposix" "$SAMPLES" "/bin/goposix cat $BIGFILE" | tee -a "$ACCUM"
echo "# cat ${TEXT_MB}MB BusyBox" >&2
bench_run "text_cat_${TEXT_MB}mb_busybox" "$SAMPLES" "/bin/busybox cat $BIGFILE" | tee -a "$ACCUM"

# E2 — wc -l
echo "# wc -l ${TEXT_MB}MB GoPOSIX" >&2
bench_run "text_wc_${TEXT_MB}mb_goposix" "$SAMPLES" "/bin/goposix wc -l $BIGFILE" | tee -a "$ACCUM"
echo "# wc -l ${TEXT_MB}MB BusyBox" >&2
bench_run "text_wc_${TEXT_MB}mb_busybox" "$SAMPLES" "/bin/busybox wc -l $BIGFILE" | tee -a "$ACCUM"

# E3 — grep
echo "# grep ${TEXT_MB}MB GoPOSIX" >&2
bench_run "text_grep_${TEXT_MB}mb_goposix" "$SAMPLES" "/bin/goposix grep -c line_500 $BIGFILE" | tee -a "$ACCUM"
echo "# grep ${TEXT_MB}MB BusyBox" >&2
bench_run "text_grep_${TEXT_MB}mb_busybox" "$SAMPLES" "/bin/busybox grep -c line_500 $BIGFILE" | tee -a "$ACCUM"

# E4 — sort (CPU-bound, could be slow at high scale)
echo "# sort ${TEXT_MB}MB GoPOSIX" >&2
bench_run "text_sort_${TEXT_MB}mb_goposix" "$SAMPLES" "/bin/goposix sort $BIGFILE" | tee -a "$ACCUM"
echo "# sort ${TEXT_MB}MB BusyBox" >&2
bench_run "text_sort_${TEXT_MB}mb_busybox" "$SAMPLES" "/bin/busybox sort $BIGFILE" | tee -a "$ACCUM"

# E5 — grep -r across many small files
echo "# grep -r ${SMALL_FILES} files GoPOSIX" >&2
bench_run "text_grepr_${SMALL_FILES}f_goposix" "$SAMPLES" "/bin/goposix grep -r pattern $MANYDIR" | tee -a "$ACCUM"
echo "# grep -r ${SMALL_FILES} files BusyBox" >&2
bench_run "text_grepr_${SMALL_FILES}f_busybox" "$SAMPLES" "/bin/busybox grep -r pattern $MANYDIR" | tee -a "$ACCUM"

# ===========================================================================
# Log: compute medians.
# ===========================================================================
{
  echo ""
  echo "## Cat E — Text Processing Throughput (seconds, median of $SAMPLES, ${TEXT_MB}MB)"
  echo ""
  echo "| Operation | GoPOSIX | BusyBox | Ratio | Winner |"
  echo "|-----------|:-------:|:-------:|:-----:|:------:|"
} >&2

for op in "cat cat" "wc wc -l" "grep grep -c" "sort sort"; do
  slug=$(echo "$op" | awk '{print $1}')
  desc=$(echo "$op" | awk '{print $2" "$3}')
  gpx_med=$(grep "text_${slug}_${TEXT_MB}mb_goposix" "$ACCUM" | cut -d, -f3 | bench_median)
  bbx_med=$(grep "text_${slug}_${TEXT_MB}mb_busybox" "$ACCUM" | cut -d, -f3 | bench_median)
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

# grep -r is separate (different file count).
GPX_GR=$(grep "text_grepr_${SMALL_FILES}f_goposix" "$ACCUM" | cut -d, -f3 | bench_median)
BBX_GR=$(grep "text_grepr_${SMALL_FILES}f_busybox" "$ACCUM" | cut -d, -f3 | bench_median)
if [ "$(echo "$BBX_GR > 0" | bc -l 2>/dev/null)" = "1" ]; then
  ratio_gr=$(awk "BEGIN { printf \"%.1f\", $GPX_GR / $BBX_GR }" 2>/dev/null || echo "-")
  if [ "$(echo "$GPX_GR < $BBX_GR" | bc -l 2>/dev/null)" = "1" ]; then
    winner_gr="**GoPOSIX**"
  else
    winner_gr="BusyBox"
  fi
  echo "| \`grep -r\` ($SMALL_FILES files) | ${GPX_GR} | ${BBX_GR} | ${ratio_gr}× | $winner_gr |" >&2
else
  echo "| \`grep -r\` ($SMALL_FILES files) | ${GPX_GR} | ${BBX_GR} | — | — |" >&2
fi
echo "" >&2

# Emit findings for interesting comparisons.
GPX_GREP=$(grep "text_grep_${TEXT_MB}mb_goposix" "$ACCUM" | cut -d, -f3 | bench_median)
BBX_GREP=$(grep "text_grep_${TEXT_MB}mb_busybox" "$ACCUM" | cut -d, -f3 | bench_median)
RATIO_GREP=$(awk "BEGIN { printf \"%.1f\", ${GPX_GREP:-0} / ${BBX_GREP:-1} }" 2>/dev/null || echo "-")

GPX_SORT=$(grep "text_sort_${TEXT_MB}mb_goposix" "$ACCUM" | cut -d, -f3 | bench_median)
BBX_SORT=$(grep "text_sort_${TEXT_MB}mb_busybox" "$ACCUM" | cut -d, -f3 | bench_median)
RATIO_SORT=$(awk "BEGIN { printf \"%.1f\", ${GPX_SORT:-0} / ${BBX_SORT:-1} }" 2>/dev/null || echo "-")

echo "# FINDING: grep on ${TEXT_MB}MB: GoPOSIX ${GPX_GREP}s vs BusyBox ${BBX_GREP}s (${RATIO_GREP}×). GoPOSIX RE2 engine vs BusyBox POSIX ERE." >&2
echo "# FINDING: sort on ${TEXT_MB}MB: GoPOSIX ${GPX_SORT}s vs BusyBox ${BBX_SORT}s (${RATIO_SORT}×). GoPOSIX uses in-memory sort (high RSS), BusyBox uses external merge." >&2
echo "# FINDING: grep -r across ${SMALL_FILES} files: GoPOSIX ${GPX_GR}s vs BusyBox ${BBX_GR}s (${RATIO_GR:-?}×)." >&2

rm -f "$ACCUM"
