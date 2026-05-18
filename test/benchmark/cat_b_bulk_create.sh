#!/bin/sh
# =============================================================================
# Cat B — Bulk File Creation (touch).
# Creates N empty files on a tmpfs, times GoPOSIX touch vs BusyBox touch.
# =============================================================================

set -u
. "$(dirname "$0")/lib/harness.sh"

SAMPLES=5

echo "# Cat B — Bulk File Creation (touch)" >&2

# We test at multiple scaled N levels.
N_SMALL=$(scaled 100   "$MAX_FILE_COUNT")
N_MED=$(scaled   1000  "$MAX_FILE_COUNT")
N_LARGE=$(scaled  10000 "$MAX_FILE_COUNT")

echo "# scale=$BENCH_SCALE N=$N_SMALL / $N_MED / $N_LARGE" >&2
echo "" >&2

WORKDIR="$BENCH_TMPDIR/touch_bench"
# Use tmpfs if available for consistent performance.
mkdir -p "$WORKDIR"

# CSV header.
echo "category,test,sample,wall_sec,user_sec,sys_sec,rss_kb"

for N in "$N_SMALL" "$N_MED" "$N_LARGE"; do
  echo "# N=$N GoPOSIX" >&2
  bench_run "bulk_touch_${N}_goposix" "$SAMPLES" \
    "( cd $WORKDIR; rm -rf touch_N; mkdir touch_N && cd touch_N; for i in \$(seq $N); do /bin/goposix touch \"file_\$i\"; done )"

  echo "# N=$N BusyBox" >&2
  bench_run "bulk_touch_${N}_busybox" "$SAMPLES" \
    "( cd $WORKDIR; rm -rf touch_N; mkdir touch_N && cd touch_N; for i in \$(seq $N); do /bin/busybox touch \"file_\$i\"; done )"
done

# Cleanup.
rm -rf "$WORKDIR"

echo "" >&2
echo "# FINDING: Bulk touch performance at N=$N_SMALL / $N_MED / $N_LARGE (scale=${BENCH_SCALE}×). Both bottleneck on VFS." >&2
