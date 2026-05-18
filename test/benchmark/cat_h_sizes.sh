#!/bin/sh
# =============================================================================
# Cat H — Binary & Image Size (static analysis, no runtime).
# Always runs regardless of BENCH_SCALE (scale has no effect).
# =============================================================================

set -u
. "$(dirname "$0")/lib/harness.sh"

echo "# Cat H — Binary & Image Size" >&2
echo "" >&2

# Binary sizes.
GPX_SIZE=$(stat -c%s /bin/goposix 2>/dev/null || echo 0)
BBX_SIZE=$(stat -c%s /bin/busybox 2>/dev/null || echo 0)

# Symlink counts.
GPX_COUNT=$(/bin/goposix --list-commands 2>/dev/null | wc -l || echo 0)
BBX_COUNT=$(/bin/busybox --list 2>/dev/null | wc -l || echo 0)

# ===========================================================================
# CSV output
# ===========================================================================
# header row
echo "category,test,sample,wall_sec,user_sec,sys_sec,rss_kb"

# We report sizes as rows with zero timing.
echo "cat_h,binary_size_goposix,1,0,0,0,$GPX_SIZE"
echo "cat_h,binary_size_busybox,1,0,0,0,$BBX_SIZE"

# Symlink counts (treat as "timed" — wall=count, for table formatting).
echo "cat_h,command_count_goposix,1,$GPX_COUNT,0,0,0"
echo "cat_h,command_count_busybox,1,$BBX_COUNT,0,0,0"

# ===========================================================================
# Log output
# ===========================================================================
{
  echo "# FINDING: BusyBox ${BBX_SIZE}B, GoPOSIX ${GPX_SIZE}B ($(awk "BEGIN { printf \"%.1f\", ${GPX_SIZE}/${BBX_SIZE} }")×)"
  echo ""
  echo "## Binary Size"
  echo ""
  echo "| Tool | Size (bytes) | Size (human) |"
  echo "|------|:------------:|-------------|"
  echo "| GoPOSIX | $GPX_SIZE | $(numfmt --to=iec $GPX_SIZE 2>/dev/null || echo "${GPX_SIZE}B") |"
  echo "| BusyBox | $BBX_SIZE | $(numfmt --to=iec $BBX_SIZE 2>/dev/null || echo "${BBX_SIZE}B") |"
  echo "| Ratio | $(awk "BEGIN { printf \"%.1f\", ${GPX_SIZE}/${BBX_SIZE} }")× | — |"
  echo ""
  echo "## Command Count (symlinks)"
  echo ""
  echo "| Tool | Count |"
  echo "|------|------:|"
  echo "| GoPOSIX | $GPX_COUNT |"
  echo "| BusyBox | $BBX_COUNT |"
  echo ""
} >&2
