#!/bin/sh
# =============================================================================
# Cat B — Bulk File Creation (touch).
# Creates N empty files on a tmpfs, times GoPOSIX touch vs BusyBox touch.
# =============================================================================

set -u
. "$(dirname "$0")/lib/harness.sh"

SAMPLES=5

echo "# Cat B — Bulk File Creation (touch)" >&2

N_SMALL=$(scaled 100   "$MAX_FILE_COUNT")
N_MED=$(scaled   1000  "$MAX_FILE_COUNT")
N_LARGE=$(scaled  10000 "$MAX_FILE_COUNT")

echo "# scale=$BENCH_SCALE N=$N_SMALL / $N_MED / $N_LARGE" >&2
echo "" >&2

WORKDIR="$BENCH_TMPDIR/touch_bench"
mkdir -p "$WORKDIR"

# CSV header.
echo "category,test,sample,wall_sec,user_sec,sys_sec,rss_kb"

# Accumulate for stats.
ACCUM=$(mktemp)

for N in "$N_SMALL" "$N_MED" "$N_LARGE"; do
  echo "# N=$N GoPOSIX" >&2
  bench_run "bulk_touch_${N}_goposix" "$SAMPLES" \
    "( cd $WORKDIR; rm -rf touch_N; mkdir touch_N && cd touch_N; seq 1 $N | xargs -P4 -I{} /bin/goposix touch file_{} )" | tee -a "$ACCUM"

  echo "# N=$N BusyBox" >&2
  bench_run "bulk_touch_${N}_busybox" "$SAMPLES" \
    "( cd $WORKDIR; rm -rf touch_N; mkdir touch_N && cd touch_N; seq 1 $N | xargs -P4 -I{} /bin/busybox touch file_{} )" | tee -a "$ACCUM"
done

rm -rf "$WORKDIR"

# ===========================================================================
# Log: compute medians from accumulated data.
# ===========================================================================
{
  echo ""
  echo "## Cat B — Bulk File Creation (seconds, median of $SAMPLES)"
  echo ""
  echo "| N Files | GoPOSIX | BusyBox | Ratio | Winner |"
  echo "|--------:|:-------:|:-------:|:-----:|:------:|"
} >&2

for N in "$N_SMALL" "$N_MED" "$N_LARGE"; do
  gpx_med=$(grep "bulk_touch_${N}_goposix," "$ACCUM" | cut -d, -f3 | bench_median)
  bbx_med=$(grep "bulk_touch_${N}_busybox," "$ACCUM" | cut -d, -f3 | bench_median)
  if [ "$(echo "$bbx_med > 0" | bc -l 2>/dev/null)" = "1" ]; then
    ratio=$(awk "BEGIN { printf \"%.1f\", $gpx_med / $bbx_med }" 2>/dev/null || echo "-")
    if [ "$(echo "$gpx_med < $bbx_med" | bc -l 2>/dev/null)" = "1" ]; then
      winner="**GoPOSIX**"
    else
      winner="BusyBox"
    fi
    echo "| $N | ${gpx_med} | ${bbx_med} | ${ratio}× | $winner |" >&2
  else
    echo "| $N | ${gpx_med} | ${bbx_med} | — | — |" >&2
  fi
done
echo "" >&2

# Summary finding using largest N.
GPX_BIG=$(grep "bulk_touch_${N_LARGE}_goposix," "$ACCUM" | cut -d, -f3 | bench_median)
BBX_BIG=$(grep "bulk_touch_${N_LARGE}_busybox," "$ACCUM" | cut -d, -f3 | bench_median)
RATIO_BIG=$(awk "BEGIN { printf \"%.1f\", ${GPX_BIG:-0} / ${BBX_BIG:-1} }" 2>/dev/null || echo "-")
echo "# FINDING: Bulk touch at N=$N_LARGE: GoPOSIX ${GPX_BIG}s vs BusyBox ${BBX_BIG}s (${RATIO_BIG}×). Both bottleneck on VFS." >&2

rm -f "$ACCUM"
