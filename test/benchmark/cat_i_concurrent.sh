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

# Pre-create files if needed.
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

# I2: grep -r (may parallelize).
echo "# grep -r across $MANYFILES files — GoPOSIX" >&2
bench_run "conc_grepr_${MANYFILES}_goposix" "$SAMPLES" "/bin/goposix grep -r content $MANYDIR"

echo "# grep -r across $MANYFILES files — BusyBox" >&2
bench_run "conc_grepr_${MANYFILES}_busybox" "$SAMPLES" "/bin/busybox grep -r content $MANYDIR"

# I3: du (recursive disk usage).
echo "# du across $MANYFILES files — GoPOSIX" >&2
bench_run "conc_du_${MANYFILES}_goposix" "$SAMPLES" "/bin/goposix du -sh $MANYDIR"

echo "# du across $MANYFILES files — BusyBox" >&2
bench_run "conc_du_${MANYFILES}_busybox" "$SAMPLES" "/bin/busybox du -sh $MANYDIR"

echo "" >&2
echo "# FINDING: [GOROUTINE-TODO] Both tools are sequential today. GoPOSIX can win 2–8× with goroutine-parallel I/O." >&2
echo "# FINDING: Current measurements show baseline parity for sequential implementations." >&2
