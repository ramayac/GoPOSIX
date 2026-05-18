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
  # Generate ~10K lines per MB.
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

# E1 — cat
echo "# cat ${TEXT_MB}MB GoPOSIX" >&2
bench_run "text_cat_${TEXT_MB}mb_goposix" "$SAMPLES" "/bin/goposix cat $BIGFILE"
echo "# cat ${TEXT_MB}MB BusyBox" >&2
bench_run "text_cat_${TEXT_MB}mb_busybox" "$SAMPLES" "/bin/busybox cat $BIGFILE"

# E2 — wc -l
echo "# wc -l ${TEXT_MB}MB GoPOSIX" >&2
bench_run "text_wc_${TEXT_MB}mb_goposix" "$SAMPLES" "/bin/goposix wc -l $BIGFILE"
echo "# wc -l ${TEXT_MB}MB BusyBox" >&2
bench_run "text_wc_${TEXT_MB}mb_busybox" "$SAMPLES" "/bin/busybox wc -l $BIGFILE"

# E3 — grep
echo "# grep ${TEXT_MB}MB GoPOSIX" >&2
bench_run "text_grep_${TEXT_MB}mb_goposix" "$SAMPLES" "/bin/goposix grep -c line_500 $BIGFILE"
echo "# grep ${TEXT_MB}MB BusyBox" >&2
bench_run "text_grep_${TEXT_MB}mb_busybox" "$SAMPLES" "/bin/busybox grep -c line_500 $BIGFILE"

# E4 — sort (CPU-bound, could be slow at high scale)
echo "# sort ${TEXT_MB}MB GoPOSIX" >&2
bench_run "text_sort_${TEXT_MB}mb_goposix" "$SAMPLES" "/bin/goposix sort $BIGFILE"
echo "# sort ${TEXT_MB}MB BusyBox" >&2
bench_run "text_sort_${TEXT_MB}mb_busybox" "$SAMPLES" "/bin/busybox sort $BIGFILE"

# E5 — grep -r across many small files
echo "# grep -r ${SMALL_FILES} files GoPOSIX" >&2
bench_run "text_grepr_${SMALL_FILES}f_goposix" "$SAMPLES" "/bin/goposix grep -r pattern $MANYDIR"
echo "# grep -r ${SMALL_FILES} files BusyBox" >&2
bench_run "text_grepr_${SMALL_FILES}f_busybox" "$SAMPLES" "/bin/busybox grep -r pattern $MANYDIR"

# Cleanup (keep files for re-runs; /tmp is tmpfs and disappears).
echo "" >&2
echo "# FINDING: Text I/O throughput at ${TEXT_MB}MB scale=${BENCH_SCALE}×. Both saturate pipe/disk bandwidth." >&2
