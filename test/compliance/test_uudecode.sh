#!/bin/sh
# Compliance test for uudecode utility.
# Tests text-to-binary decoding.
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

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

# Create a uuencoded file
echo "hello decode" | $GOPOSIX uuencode out.bin > "$TMPDIR/test.uue"

# Test 1: uudecode produces output file
$GOPOSIX uudecode -o "$TMPDIR/result.bin" "$TMPDIR/test.uue" 2>/dev/null
if [ -f "$TMPDIR/result.bin" ]; then
    RESULT=$(cat "$TMPDIR/result.bin")
    check "uudecode roundtrip" "hello decode" "$RESULT"
else
    check "uudecode roundtrip" "hello decode" "file not created"
fi

# Test 2: uudecode from stdin
echo "hello decode" | $GOPOSIX uuencode - > "$TMPDIR/stdin.uue"
$GOPOSIX uudecode -o "$TMPDIR/stdout.bin" < "$TMPDIR/stdin.uue" 2>/dev/null
if [ -f "$TMPDIR/stdout.bin" ]; then
    RESULT=$(cat "$TMPDIR/stdout.bin")
    check "uudecode stdin" "hello decode" "$RESULT"
fi

# Test 3: uudecode exits 0
if $GOPOSIX uudecode -o /dev/null "$TMPDIR/test.uue" 2>/dev/null; then
    check "uudecode exits 0" "0" "0"
else
    check "uudecode exits 0" "0" "1"
fi

echo "uudecode compliance: PASS=$PASS FAIL=$FAIL"
exit $FAIL
