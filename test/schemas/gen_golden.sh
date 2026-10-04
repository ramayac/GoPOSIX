#!/usr/bin/env bash
# Generate golden fixtures for all utilies missing them.
# Run from repo root: bash test/schemas/gen_golden.sh
set -euo pipefail

GOPOSIX="${GOPOSIX:-./goposix}"
GOPOSIX_ABS="$(cd "$(dirname "$GOPOSIX")" && pwd)/$(basename "$GOPOSIX")"
GOLDEN_DIR="test/schemas/golden"
GOLDEN_ABS="$(pwd)/$GOLDEN_DIR"
TMPDIR=$(mktemp -d -t goposix-golden.XXXXXX)
cleanup() { rm -rf "$TMPDIR"; }
trap cleanup EXIT

mkdir -p "$GOLDEN_DIR"

# Commands that exit non-zero (false, makedevs without root, ...) still
# produce valid JSON envelopes on stdout. Collect their names instead of
# aborting under set -e.
GEN_FAILED=""

gen() {
  local util json
  util="$1"
  json="$GOLDEN_DIR/${util}.json"
  shift
  echo -n "  $util ... "
  if "$GOPOSIX" "$util" "$@" > "$json" 2>/dev/null; then
    echo "OK"
  else
    echo "FAIL (exit $?)" >&2
    GEN_FAILED="$GEN_FAILED $util"
  fi
}

gen_stdin() {
  local util stdin json
  util="$1"
  stdin="$2"
  json="$GOLDEN_DIR/${util}.json"
  shift 2
  echo -n "  $util (stdin) ... "
  if printf '%b' "$stdin" | "$GOPOSIX" "$util" "$@" > "$json" 2>/dev/null; then
    echo "OK"
  else
    echo "FAIL (exit $?)" >&2
    GEN_FAILED="$GEN_FAILED $util"
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

# --- Phase 28 F12: schemas added during the POSIX command audit ---
echo "hello world" > "$TMPDIR/f12.txt"

gen_stdin bc       '1+2'         --json
gen_stdin hexdump  'hello'       --json
gen       dc       --json -e '2 2+p'
gen_stdin tsort    'a b\nb c\n'  --json
gen_stdin rev      'hello\n'     --json
gen_stdin xxd      'hi'          --json
gen       seq      --json 3
gen       factor   --json 21
gen       which    --json sh
gen       cal      --json 2 2020
gen       uptime   --json
gen       hostid   --json
gen       realpath --json "$TMPDIR/f12.txt"
gen       tree     --json -L 2 "$TMPDIR"
gen       shell    --json -c 'echo hi'
gen       taskset  --json -p $$
gen       mount    --json

# pidof needs a live process
sleep 5 &
PIDOF_SPID=$!
gen pidof --json sleep
kill "$PIDOF_SPID" 2>/dev/null || true

gen       sha512sum --json "$TMPDIR/f12.txt"
gen       sha3sum   --json "$TMPDIR/f12.txt"
gen       sha1sum   --json "$TMPDIR/f12.txt"

# Decompress family: system compressors provide the input files.
bzip2 -kc "$TMPDIR/f12.txt" > "$TMPDIR/f12.txt.bz2"
lzma  -kc "$TMPDIR/f12.txt" > "$TMPDIR/f12.txt.lzma"
# bzcat is cat mode: decompressed bytes and the envelope share stdout, so
# keep the trailing JSON line. The redirect paths are absolute because the
# subshell changes the working directory.
"$GOPOSIX_ABS" bzcat --json "$TMPDIR/f12.txt.bz2" 2>/dev/null | tail -1 > "$GOLDEN_ABS/bzcat.json"
( cd "$TMPDIR" && rm -f f12.txt && "$GOPOSIX_ABS" bunzip2 --json f12.txt.bz2 > "$GOLDEN_ABS/bunzip2.json" 2>/dev/null )
( cd "$TMPDIR" && rm -f f12.txt && "$GOPOSIX_ABS" unlzma --json f12.txt.lzma > "$GOLDEN_ABS/unlzma.json" 2>/dev/null )
# uncompress has no .Z compressor on this host: the unknown-suffix shape
# (files[0].error) is a valid fixture for the schema.
"$GOPOSIX" uncompress --json -c "$TMPDIR/f12.txt" > "$GOLDEN_DIR/uncompress.json" 2>/dev/null || true

gen_stdin cryptpw  'secret'      --json
gen       uuencode  --json "$TMPDIR/f12.txt" out.bin
"$GOPOSIX" uuencode "$TMPDIR/f12.txt" out.bin > "$TMPDIR/enc.uue"
gen       uudecode  --json -o "$TMPDIR/dec.txt" "$TMPDIR/enc.uue"

# Archive tools
mkdir -p "$TMPDIR/z"
cp "$TMPDIR/f12.txt" "$TMPDIR/z/a.txt"
python3 -c "import zipfile,sys; z=zipfile.ZipFile('$TMPDIR/t.zip','w'); z.write('$TMPDIR/z/a.txt','a.txt'); z.close()"
gen       unzip     --json -l "$TMPDIR/t.zip"
ar rcs "$TMPDIR/libt.a" "$TMPDIR/f12.txt" 2>/dev/null
gen       ar        --json t "$TMPDIR/libt.a"
( cd "$TMPDIR" && printf '%s\n' z/a.txt | "$GOPOSIX_ABS" cpio -o -F out.cpio 2>/dev/null && "$GOPOSIX_ABS" cpio -i -t --json < out.cpio > "$GOLDEN_ABS/cpio.json" 2>/dev/null )

# Device tools (no root: makedevs reports failures inside the envelope)
printf 'devname c 0600 0 0 1 3\n' > "$TMPDIR/devtable.txt"
mkdir -p "$TMPDIR/md-root"
gen makedevs --json -d "$TMPDIR/devtable.txt" "$TMPDIR/md-root" || true
gen mdev     --json -d
gen mkfs.minix --json "$TMPDIR/minix.img" 1000
gen start-stop-daemon --json -K -n nonexistent-proc-xyz || true

# wget needs network
gen wget --json -O "$TMPDIR/wget.html" https://example.com

gen who     --json

# rx has no fixture generator: an XMODEM sender is not available in this
# script. test/schemas/golden/rx.json is a hand-written error envelope
# (data null, error RECEIVE_ERROR) that matches rx.schema.json.

echo ""
echo "Done. Golden fixtures in $GOLDEN_DIR/"
ls -la "$GOLDEN_DIR"/*.json | wc -l
echo "fixtures total"
if [ -n "$GEN_FAILED" ]; then
  echo "Non-zero exits (fixture may still be valid):$GEN_FAILED"
fi
