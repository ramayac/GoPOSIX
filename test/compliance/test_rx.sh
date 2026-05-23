#!/bin/sh
# Compliance test for rx utility.
set -uo pipefail

PASS=0
FAIL=0

GOPOSIX="${GOPOSIX:-goposix}"

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

# Test 1: rx with missing filename fails
$GOPOSIX rx 2>/dev/null && rc=0 || rc=$?
if [ "$rc" -ne 0 ]; then
    PASS=$((PASS + 1))
else
    FAIL=$((FAIL + 1))
    echo "FAIL: rx no-args should fail"
fi

# Test 2: EOT-only = empty file transfer
OUT=$(printf '\x04' | $GOPOSIX rx "$TMPDIR/empty.out" 2>/dev/null | xxd -p | tr -d '\n')
if [ "$OUT" = "4306" ]; then
    PASS=$((PASS + 1))
else
    FAIL=$((FAIL + 1))
    echo "FAIL: rx EOT handshake got $OUT want 4306"
fi

# Test 3: rx produces output file
printf '\x04' | $GOPOSIX rx "$TMPDIR/file.out" 2>/dev/null > /dev/null
if [ -f "$TMPDIR/file.out" ]; then
    PASS=$((PASS + 1))
else
    FAIL=$((FAIL + 1))
    echo "FAIL: rx should create output file"
fi

echo "rx compliance: PASS=$PASS FAIL=$FAIL"
exit $FAIL
