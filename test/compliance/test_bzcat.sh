#!/bin/sh
# Compliance test for bzcat utility.
# Tests bzip2 decompression to stdout.
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

command -v bzip2 >/dev/null 2>&1 || { echo "bzcat compliance: PASS=$PASS FAIL=$FAIL (skipped: no bzip2)"; exit 0; }

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

echo "hello bzcat" | bzip2 -c > "$TMPDIR/test.bz2"

# Test 1: bzcat outputs decompressed content
RESULT=$($GOPOSIX bzcat "$TMPDIR/test.bz2" 2>/dev/null)
check "bzcat stdout" "hello bzcat" "$RESULT"

# Test 2: bzcat exits 0
if $GOPOSIX bzcat "$TMPDIR/test.bz2" >/dev/null 2>&1; then
    check "bzcat exits 0" "0" "0"
else
    check "bzcat exits 0" "0" "1"
fi

echo "bzcat compliance: PASS=$PASS FAIL=$FAIL"
exit $FAIL
