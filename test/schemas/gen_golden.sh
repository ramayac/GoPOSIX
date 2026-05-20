#!/usr/bin/env bash
# Generate golden fixtures for all utilies missing them.
# Run from repo root: bash test/schemas/gen_golden.sh
set -euo pipefail

GOPOSIX="${GOPOSIX:-./goposix}"
GOLDEN_DIR="test/schemas/golden"
TMPDIR=$(mktemp -d -t goposix-golden.XXXXXX)
cleanup() { rm -rf "$TMPDIR"; }
trap cleanup EXIT

mkdir -p "$GOLDEN_DIR"

gen() {
  local util="$1" json="$GOLDEN_DIR/${util}.json"
  shift
  echo -n "  $util ... "
  if "$GOPOSIX" "$util" "$@" > "$json" 2>/dev/null; then
    echo "OK"
  else
    echo "FAIL (exit $?)" >&2
    return 1
  fi
}

gen_stdin() {
  local util="$1" stdin="$2" json="$GOLDEN_DIR/${util}.json"
  shift 2
  echo -n "  $util (stdin) ... "
  if printf '%s' "$stdin" | "$GOPOSIX" "$util" "$@" > "$json" 2>/dev/null; then
    echo "OK"
  else
    echo "FAIL (exit $?)" >&2
    return 1
  fi
}

# --- Simple utilities (no args, no stdin) ---
gen sleep    --json 0.001
gen true     --json
gen false    --json
gen yes      --json -n 1
gen logname  --json
gen tty      --json

# --- File creation/destruction (use temp dir) ---
echo "test content" > "$TMPDIR/testfile"
ln -s "$TMPDIR/testfile" "$TMPDIR/testlink"
mkdir "$TMPDIR/testdir"

gen mkfifo   --json "$TMPDIR/testfifo"
gen link     --json "$TMPDIR/testfile" "$TMPDIR/hardlink"
gen touch    --json "$TMPDIR/touchfile"
gen rmdir    --json "$TMPDIR/testdir"
gen unlink   --json "$TMPDIR/testlink"

# --- File reading utilities ---
echo -e "hello\nworld\nhello" > "$TMPDIR/data.txt"
echo -e "alpha\nbeta\ngamma" > "$TMPDIR/data2.txt"
echo -e "alpha\nbeta\nhello" > "$TMPDIR/sorted1.txt"
echo -e "alpha\ngamma\nhello" > "$TMPDIR/sorted2.txt"
echo "binary content here" > "$TMPDIR/binfile"

gen cksum    --json "$TMPDIR/data.txt"
gen sum      --json "$TMPDIR/data.txt"
gen strings  --json "$TMPDIR/binfile"

# --- Stdin-consuming utilities ---
gen_stdin sort    "c\na\nb\n"         --json
gen_stdin wc      "a b c\nd e f\n"   --json
gen_stdin uniq    "a\na\nb\nc\n"     --json
gen_stdin tr      "abc"              --json a-z A-Z
gen_stdin tee     "hello\n"          --json "$TMPDIR/tee_out.txt"
gen_stdin fold    "hello world"      --json -w 5
gen_stdin expand  "a\tb\nc\td\n"     --json
gen_stdin unexpand "a    b\nc    d\n" --json
gen_stdin nl      "line1\nline2\n"   --json
gen_stdin paste   "a\nb\n"           --json -
gen_stdin sed     "hello world"      --json 's/hello/goodbye/'
gen_stdin awk     "1 2 3\n4 5 6"    --json '{print $1}'
gen_stdin od      "abcdef"           --json

# --- Multi-file utilities ---
gen cmp     --json "$TMPDIR/data.txt" "$TMPDIR/data.txt"
gen comm    --json "$TMPDIR/sorted1.txt" "$TMPDIR/sorted2.txt"
gen join    --json "$TMPDIR/sorted1.txt" "$TMPDIR/sorted2.txt"

# --- Complex utilities ---
gen nice    --json -n 5 true
gen nohup   --json true
gen logger  --json "goposix golden fixture test"
gen split   --json "$TMPDIR/data.txt" "$TMPDIR/split_prefix"
gen dd      --json if="$TMPDIR/data.txt" bs=4 count=1
echo "patch content" > "$TMPDIR/orig.txt"
diff -u "$TMPDIR/data2.txt" "$TMPDIR/data.txt" > "$TMPDIR/patch.diff" 2>/dev/null || true
gen patch   --json "$TMPDIR/data2.txt" "$TMPDIR/patch.diff"

echo ""
echo "Done. Golden fixtures in $GOLDEN_DIR/"
ls -la "$GOLDEN_DIR"/*.json | wc -l
echo "fixtures total"
