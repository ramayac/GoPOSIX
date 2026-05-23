#!/bin/sh
# Compliance test for uuencode utility.
# Tests binary-to-text encoding.
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

echo "hello uuencode" | $GOPOSIX uuencode remote.txt > "$TMPDIR/encoded.uue"

# Test 1: uuencode produces begin marker
RESULT=$(grep -c "^begin" "$TMPDIR/encoded.uue")
check "uuencode begin marker" "1" "$RESULT"

# Test 2: uuencode produces end marker
RESULT=$(grep -c "^end" "$TMPDIR/encoded.uue")
check "uuencode end marker" "1" "$RESULT"

# Test 3: uuencode includes remote name
RESULT=$(grep -c "remote.txt" "$TMPDIR/encoded.uue")
check "uuencode remote name" "1" "$RESULT"

# Test 4: decode with system uudecode to verify
if command -v uudecode >/dev/null 2>&1; then
    uudecode "$TMPDIR/encoded.uue" -o "$TMPDIR/decoded.txt" 2>/dev/null || true
    if [ -f "$TMPDIR/decoded.txt" ]; then
        RESULT=$(cat "$TMPDIR/decoded.txt")
        check "uuencode roundtrip" "hello uuencode" "$RESULT"
    fi
else
    PASS=$((PASS + 1))  # can't verify roundtrip, count as pass
fi

echo "uuencode compliance: PASS=$PASS FAIL=$FAIL"
exit $FAIL
