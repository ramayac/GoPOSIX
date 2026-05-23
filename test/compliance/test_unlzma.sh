#!/bin/sh
# Compliance test for unlzma utility.
# Tests LZMA decompression.
set -uo pipefail

PASS=0
FAIL=0

check() {
    local name="$1" expected="$2" actual="$3"
    if [ "$actual" = "$expected" ]; then
        PASS=$((PASS + 1))
    else
        FAIL=$((FAIL + 1))
        echo "FAIL: $name"
        echo "  expected: $expected"
        echo "  actual:   $actual"
    fi
}

GOPOSIX="${GOPOSIX:-goposix}"

command -v xz >/dev/null 2>&1 || { echo "unlzma compliance: PASS=$PASS FAIL=$FAIL (skipped: no xz)"; exit 0; }

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

echo "hello unlzma" | xz -c --format=lzma > "$TMPDIR/test.lzma"

# Test 1: unlzma decompresses
$GOPOSIX unlzma -k "$TMPDIR/test.lzma" 2>/dev/null
OUTFILE="${TMPDIR}/test"
if [ -f "$OUTFILE" ]; then
    RESULT=$(cat "$OUTFILE")
    check "unlzma decompress" "hello unlzma" "$RESULT"
else
    check "unlzma decompress" "hello unlzma" "file not created"
fi

# Test 2: -k keeps source
if [ -f "$TMPDIR/test.lzma" ]; then
    check "unlzma -k keeps source" "exists" "exists"
else
    check "unlzma -k keeps source" "exists" "missing"
fi

echo "unlzma compliance: PASS=$PASS FAIL=$FAIL"
exit $FAIL
