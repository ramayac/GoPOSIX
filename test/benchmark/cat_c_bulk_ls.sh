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

# Accumulate for stats.
ACCUM=$(mktemp)

for N in "$N1" "$N2"; do
  echo "# Pre-creating $N files..." >&2
  rm -rf "$WORKDIR/ls_N"
  mkdir -p "$WORKDIR/ls_N"
  for i in $(seq "$N"); do
    /bin/busybox touch "$WORKDIR/ls_N/file_$i"
  done

  echo "# N=$N ls -1 GoPOSIX" >&2
  bench_run "bulk_ls1_${N}_goposix" "$SAMPLES" "/bin/goposix ls -1 $WORKDIR/ls_N" | tee -a "$ACCUM"

  echo "# N=$N ls -1 BusyBox" >&2
  bench_run "bulk_ls1_${N}_busybox" "$SAMPLES" "/bin/busybox ls -1 $WORKDIR/ls_N" | tee -a "$ACCUM"

  echo "# N=$N ls -la GoPOSIX" >&2
  bench_run "bulk_lsla_${N}_goposix" "$SAMPLES" "/bin/goposix ls -la $WORKDIR/ls_N" | tee -a "$ACCUM"

  echo "# N=$N ls -la BusyBox" >&2
  bench_run "bulk_lsla_${N}_busybox" "$SAMPLES" "/bin/busybox ls -la $WORKDIR/ls_N" | tee -a "$ACCUM"

  rm -rf "$WORKDIR/ls_N"
done

rm -rf "$WORKDIR"

# ===========================================================================
# Log: compute medians.
# ===========================================================================
{
  echo ""
  echo "## Cat C — Bulk Directory Listing (seconds, median of $SAMPLES)"
  echo ""
  echo "| N | Command | GoPOSIX | BusyBox | Ratio | Winner |"
  echo "|---:|---------|:-------:|:-------:|:-----:|:------:|"
} >&2

for N in "$N1" "$N2"; do
  for cmd in "ls1 ls -1" "lsla ls -la"; do
    scmd=$(echo "$cmd" | awk '{print $1}')
    ful=$(echo "$cmd" | awk '{print $2" "$3}')
    gpx_med=$(grep "bulk_${scmd}_${N}_goposix," "$ACCUM" | cut -d, -f3 | bench_median)
    bbx_med=$(grep "bulk_${scmd}_${N}_busybox," "$ACCUM" | cut -d, -f3 | bench_median)
    if [ "$(echo "$bbx_med > 0" | bc -l 2>/dev/null)" = "1" ]; then
      ratio=$(awk "BEGIN { printf \"%.1f\", $gpx_med / $bbx_med }" 2>/dev/null || echo "-")
      if [ "$(echo "$gpx_med < $bbx_med" | bc -l 2>/dev/null)" = "1" ]; then
        winner="**GoPOSIX**"
      else
        winner="BusyBox"
      fi
      echo "| $N | \`$ful\` | ${gpx_med} | ${bbx_med} | ${ratio}× | $winner |" >&2
    else
      echo "| $N | \`$ful\` | ${gpx_med} | ${bbx_med} | — | — |" >&2
    fi
  done
done
echo "" >&2

# Summary finding: ls -la at large N.
GPX_LSLA=$(grep "bulk_lsla_${N2}_goposix," "$ACCUM" | cut -d, -f3 | bench_median)
BBX_LSLA=$(grep "bulk_lsla_${N2}_busybox," "$ACCUM" | cut -d, -f3 | bench_median)
RATIO_LSLA=$(awk "BEGIN { printf \"%.1f\", ${GPX_LSLA:-0} / ${BBX_LSLA:-1} }" 2>/dev/null || echo "-")
echo "# FINDING: ls -la at N=$N2: GoPOSIX ${GPX_LSLA}s vs BusyBox ${BBX_LSLA}s (${RATIO_LSLA}×). BusyBox uses getdents64 directly; GoPOSIX pays os.ReadDir + sort overhead." >&2

rm -f "$ACCUM"
