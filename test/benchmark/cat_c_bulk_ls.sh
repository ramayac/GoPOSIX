#!/bin/sh
# =============================================================================
# Cat C — Bulk Directory Listing (ls).
# Creates N files, then times `ls -1` and `ls -la`.
# =============================================================================

set -u
. "$(dirname "$0")/lib/harness.sh"

SAMPLES=5

echo "# Cat C — Bulk Directory Listing (ls)" >&2

N1=$(scaled 1000  "$MAX_FILE_COUNT")
N2=$(scaled 10000 "$MAX_FILE_COUNT")

echo "# scale=$BENCH_SCALE N=$N1 / $N2" >&2
echo "" >&2

WORKDIR="$BENCH_TMPDIR/ls_bench"
mkdir -p "$WORKDIR"

# CSV header.
echo "category,test,sample,wall_sec,user_sec,sys_sec,rss_kb"

for N in "$N1" "$N2"; do
  # Pre-create the file tree (using busybox touch for speed — this is setup, not measured).
  echo "# Pre-creating $N files..." >&2
  rm -rf "$WORKDIR/ls_N"
  mkdir -p "$WORKDIR/ls_N"
  for i in $(seq "$N"); do
    /bin/busybox touch "$WORKDIR/ls_N/file_$i"
  done

  echo "# N=$N ls -1 GoPOSIX" >&2
  bench_run "bulk_ls1_${N}_goposix" "$SAMPLES" "/bin/goposix ls -1 $WORKDIR/ls_N"

  echo "# N=$N ls -1 BusyBox" >&2
  bench_run "bulk_ls1_${N}_busybox" "$SAMPLES" "/bin/busybox ls -1 $WORKDIR/ls_N"

  echo "# N=$N ls -la GoPOSIX" >&2
  bench_run "bulk_lsla_${N}_goposix" "$SAMPLES" "/bin/goposix ls -la $WORKDIR/ls_N"

  echo "# N=$N ls -la BusyBox" >&2
  bench_run "bulk_lsla_${N}_busybox" "$SAMPLES" "/bin/busybox ls -la $WORKDIR/ls_N"

  rm -rf "$WORKDIR/ls_N"
done

rm -rf "$WORKDIR"

echo "" >&2
echo "# FINDING: BusyBox ls uses getdents64 directly; GoPOSIX uses os.ReadDir + sort. Sorting overhead measurable at scale." >&2
