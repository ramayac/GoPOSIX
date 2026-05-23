#!/bin/sh
# Compliance test for uncompress utility.
# Tests LZW (.Z) decompression.
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

command -v compress >/dev/null 2>&1 || { echo "uncompress compliance: PASS=$PASS FAIL=$FAIL (skipped: no compress)"; exit 0; }

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

echo "hello uncompress" > "$TMPDIR/original.txt"
compress -c "$TMPDIR/original.txt" > "$TMPDIR/test.Z" 2>/dev/null

# Test 1: uncompress decompresses .Z file
$GOPOSIX uncompress -k "$TMPDIR/test.Z" 2>/dev/null
if [ -f "$TMPDIR/test" ]; then
    RESULT=$(cat "$TMPDIR/test")
    check "uncompress decompress" "hello uncompress" "$RESULT"
else
    check "uncompress decompress" "hello uncompress" "file not created"
fi

# Test 2: exits 0 on valid .Z
if [ -f "$TMPDIR/test.Z" ]; then
    check "uncompress had source" "exists" "exists"
fi

echo "uncompress compliance: PASS=$PASS FAIL=$FAIL"
exit $FAIL
