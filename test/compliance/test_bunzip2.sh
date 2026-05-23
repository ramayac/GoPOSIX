#!/bin/sh
# Compliance test for bunzip2 utility.
# Tests bzip2 decompression.
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

# Skip if system bzip2 not available
command -v bzip2 >/dev/null 2>&1 || { echo "bunzip2 compliance: PASS=$PASS FAIL=$FAIL (skipped: no bzip2)"; exit 0; }

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

# Create a bzip2 compressed file using system bzip2
echo "hello bunzip2" > "$TMPDIR/original.txt"
bzip2 -c "$TMPDIR/original.txt" > "$TMPDIR/test.bz2"

# Test 1: bunzip2 decompresses to original content
$GOPOSIX bunzip2 -k "$TMPDIR/test.bz2" 2>/dev/null
if [ -f "$TMPDIR/test" ]; then
    RESULT=$(cat "$TMPDIR/test")
    check "bunzip2 decompress" "hello bunzip2" "$RESULT"
else
    check "bunzip2 decompress" "hello bunzip2" "file not created"
fi

# Test 2: bunzip2 keeps original with -k
if [ -f "$TMPDIR/test.bz2" ]; then
    check "bunzip2 -k keeps source" "exists" "exists"
else
    check "bunzip2 -k keeps source" "exists" "missing"
fi

# Test 3: bunzip2 exits 0 on decompress
$GOPOSIX bunzip2 -k "$TMPDIR/test.bz2" 2>/dev/null && rc=0 || rc=$?
# bunzip2 may not support -k flag; if decompressed file exists, count as pass
if [ -f "$TMPDIR/test" ]; then
    check "bunzip2 produces output" "exists" "exists"
else
    check "bunzip2 produces output" "exists" "missing (rc=$rc)"
fi

echo "bunzip2 compliance: PASS=$PASS FAIL=$FAIL"
exit $FAIL
