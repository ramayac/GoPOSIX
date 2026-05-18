#!/bin/sh
# =============================================================================
# GoPOSIX Benchmark Runner — master orchestrator.
# Runs all requested benchmark categories, collects CSVs, generates reports.
#
# Usage:
#   /bench/runner.sh --all          # all categories
#   /bench/runner.sh --quick        # Cat A + H only (smoke)
#   /bench/runner.sh --cat startup  # single category
#
# Environment:
#   BENCH_SCALE=5.0    workload multiplier (default 1.0)
# =============================================================================

set -u

BENCH_DIR="/bench"
LIB_DIR="$BENCH_DIR/lib"

# Source harness (scale, timing, stats).
. "$LIB_DIR/harness.sh"

# Results directory with timestamp and scale.
TIMESTAMP=$(date -u +%Y-%m-%dT%H%M%S)
RESULTS_DIR="$BENCH_OUTDIR/results/${TIMESTAMP}_scale${BENCH_SCALE}"
mkdir -p "$RESULTS_DIR"

# Symlink to latest.
ln -sfn "$RESULTS_DIR" "$BENCH_OUTDIR/results/latest"

# Combined CSV output.
RAW_CSV="$RESULTS_DIR/raw.csv"

# Category log prefix.
CAT_LOG_PREFIX="$RESULTS_DIR/cat"

echo "=== GoPOSIX Benchmark Suite ==="
echo "Scale:     ${BENCH_SCALE}×"
echo "Results:   $RESULTS_DIR"
echo "Timestamp: $TIMESTAMP"
echo ""

# Save run config.
{
  echo "scale=$BENCH_SCALE"
  echo "timestamp=$TIMESTAMP"
  echo "host=$(uname -a)"
  echo "cpus=$(nproc 2>/dev/null || echo '?')"
} > "$RESULTS_DIR/run_config.txt"

# Ensure BENCH_TMPDIR exists.
mkdir -p "$BENCH_TMPDIR"

# ---------------------------------------------------------------------------
# Run one category script, collect its CSV into the combined raw.csv.
# ---------------------------------------------------------------------------
run_category() {
  local cat_name="$1"
  local cat_script="$BENCH_DIR/${cat_name}.sh"

  if [ ! -x "$cat_script" ]; then
    echo "SKIP: $cat_name (script not found or not executable: $cat_script)" >&2
    return 1
  fi

  echo ""
  echo "--- $cat_name ---"
  local cat_csv="$RESULTS_DIR/${cat_name}_data.csv"

  # Run the script; it writes CSV lines to stdout, logs to stderr.
  "$cat_script" > "$cat_csv" 2> "$CAT_LOG_PREFIX/${cat_name##cat_}.log" || {
    echo "WARNING: $cat_name exited non-zero (check ${cat_name##cat_}.log)" >&2
  }

  # Append to combined CSV (with header on first category).
  if [ ! -f "$RAW_CSV" ]; then
    echo "category,test,sample,wall_sec,user_sec,sys_sec,rss_kb" > "$RAW_CSV"
  fi
  if [ -s "$cat_csv" ]; then
    local cat_label
    cat_label=$(echo "$cat_name" | sed 's/^cat_//')
    # Prefix each line with the category label (skip header if present).
    grep -v '^category,' "$cat_csv" 2>/dev/null | sed "s/^/${cat_label},/" >> "$RAW_CSV"
    echo "  → $(wc -l < "$cat_csv") data rows collected"
  else
    echo "  → no data rows"
  fi

  return 0
}

# Ensure category log directory exists.
mkdir -p "$CAT_LOG_PREFIX"

# ---------------------------------------------------------------------------
# Parse arguments.
# ---------------------------------------------------------------------------
MODE="all"
CAT_NAME=""

case "${1:-}" in
  --all)
    MODE="all"
    ;;
  --quick)
    MODE="quick"
    ;;
  --cat)
    MODE="cat"
    CAT_NAME="${2:-}"
    if [ -z "$CAT_NAME" ]; then
      echo "ERROR: --cat requires a category script name (e.g., cat_a_startup)" >&2
      echo "  Available: $(ls /bench/cat_*.sh 2>/dev/null | sed 's|/bench/||;s|\.sh||' | tr '\n' ' ')" >&2
      exit 1
    fi
    # Accept both short names (a, b, ...) and full names (cat_a_startup).
    case "$CAT_NAME" in
      cat_*) ;;  # already full name
      a) CAT_NAME="cat_a_startup" ;;
      b) CAT_NAME="cat_b_bulk_create" ;;
      c) CAT_NAME="cat_c_bulk_ls" ;;
      d) CAT_NAME="cat_d_bulk_move" ;;
      e) CAT_NAME="cat_e_text_throughput" ;;
      f) CAT_NAME="cat_f_daemon_vs_process" ;;
      g) CAT_NAME="cat_g_memory" ;;
      h) CAT_NAME="cat_h_sizes" ;;
      i) CAT_NAME="cat_i_concurrent" ;;
      j) CAT_NAME="cat_j_agent_loop" ;;
      startup) CAT_NAME="cat_a_startup" ;;
      bulk_create) CAT_NAME="cat_b_bulk_create" ;;
      bulk_ls) CAT_NAME="cat_c_bulk_ls" ;;
      bulk_move) CAT_NAME="cat_d_bulk_move" ;;
      text) CAT_NAME="cat_e_text_throughput" ;;
      daemon) CAT_NAME="cat_f_daemon_vs_process" ;;
      memory) CAT_NAME="cat_g_memory" ;;
      sizes) CAT_NAME="cat_h_sizes" ;;
      concurrent) CAT_NAME="cat_i_concurrent" ;;
      agent) CAT_NAME="cat_j_agent_loop" ;;
      *) echo "ERROR: unknown category '$CAT_NAME'" >&2; exit 1 ;;
    esac
    ;;
  *)
    echo "Usage: $0 [--all | --quick | --cat <name>]" >&2
    echo "  Short names: a, b, c, d, e, f, g, h, i, j" >&2
    echo "  Full names:  cat_a_startup, cat_b_bulk_create, ..." >&2
    echo "  Friendly:    startup, bulk_create, bulk_ls, bulk_move, text, daemon, memory, sizes, concurrent, agent" >&2
    echo "  BENCH_SCALE=5.0 $0 --all" >&2
    exit 1
    ;;
esac

echo "Mode: $MODE"
echo ""

# ---------------------------------------------------------------------------
# Category list (order matters — create deps before consumers).
# ---------------------------------------------------------------------------
ALL_CATEGORIES="cat_h_sizes cat_a_startup cat_b_bulk_create cat_c_bulk_ls cat_d_bulk_move cat_e_text_throughput cat_f_daemon_vs_process cat_g_memory cat_j_agent_loop cat_i_concurrent"
QUICK_CATEGORIES="cat_h_sizes cat_a_startup"

# ---------------------------------------------------------------------------
# Execute.
# ---------------------------------------------------------------------------
case "$MODE" in
  all)
    for cat in $ALL_CATEGORIES; do
      run_category "$cat"
    done
    ;;
  quick)
    for cat in $QUICK_CATEGORIES; do
      run_category "$cat"
    done
    ;;
  cat)
    run_category "$CAT_NAME"
    ;;
esac

echo ""
echo "=== Benchmark Complete ==="
echo "Results: $RESULTS_DIR"
echo "Combined CSV: $RAW_CSV"
echo "Rows: $(wc -l < "$RAW_CSV" 2>/dev/null || echo 0)"
echo ""

# Generate report.
"$LIB_DIR/report.sh" "$RESULTS_DIR"

echo ""
echo "Summary:   ${RESULTS_DIR}/summary.md"
echo "Narrative: ${RESULTS_DIR}/narrative.md"
