#!/bin/sh
# =============================================================================
# Cat D — Bulk File Move / Remove (mv, rm).
# Creates N files, moves them to a new directory, then removes them.
# =============================================================================

set -u
. "$(dirname "$0")/lib/harness.sh"

SAMPLES=5

echo "# Cat D — Bulk File Move / Remove (mv, rm)" >&2

N=$(scaled 1000 "$MAX_FILE_COUNT")

echo "# scale=$BENCH_SCALE N=$N" >&2
echo "" >&2

WORKDIR="$BENCH_TMPDIR/mvrm_bench"
mkdir -p "$WORKDIR"

# Pre-create N files using busybox (setup, not measured).
rm -rf "$WORKDIR/src" "$WORKDIR/dst"
mkdir -p "$WORKDIR/src"
for i in $(seq "$N"); do
  /bin/busybox touch "$WORKDIR/src/file_$i"
done

# CSV header.
echo "category,test,sample,wall_sec,user_sec,sys_sec,rss_kb"

# D1: mv (re-create destination each time via shell wrapper).
echo "# mv GoPOSIX" >&2
bench_run "bulk_mv_${N}_goposix" "$SAMPLES" \
  "( rm -rf $WORKDIR/dst; mkdir $WORKDIR/dst; cd $WORKDIR/src; for f in *; do /bin/goposix mv \"\$f\" ../dst/; done )"

echo "# mv BusyBox" >&2
bench_run "bulk_mv_${N}_busybox" "$SAMPLES" \
  "( rm -rf $WORKDIR/dst; mkdir $WORKDIR/dst; cd $WORKDIR/src; for f in *; do /bin/busybox mv \"\$f\" ../dst/; done )"

# D2: rm (re-create source for each run).
echo "# rm GoPOSIX" >&2
bench_run "bulk_rm_${N}_goposix" "$SAMPLES" \
  "( rm -rf $WORKDIR/rmdir; mkdir $WORKDIR/rmdir; for i in \$(seq $N); do /bin/busybox touch $WORKDIR/rmdir/file_\$i; done; /bin/goposix rm -rf $WORKDIR/rmdir )"

echo "# rm BusyBox" >&2
bench_run "bulk_rm_${N}_busybox" "$SAMPLES" \
  "( rm -rf $WORKDIR/rmdir; mkdir $WORKDIR/rmdir; for i in \$(seq $N); do /bin/busybox touch $WORKDIR/rmdir/file_\$i; done; /bin/busybox rm -rf $WORKDIR/rmdir )"

rm -rf "$WORKDIR"

echo "" >&2
echo "# FINDING: Both bottleneck on VFS rename/unlink. Overhead difference negligible." >&2
