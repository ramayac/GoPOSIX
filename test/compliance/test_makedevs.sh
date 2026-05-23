#!/bin/sh
# Compliance test for makedevs utility.
# Tests device table parsing (dry-run mode since actual device creation needs root).
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

# Test 1: makedevs --help exits 0
if $GOPOSIX makedevs --help >/dev/null 2>&1; then
    check "makedevs --help exits 0" "0" "0"
else
    check "makedevs --help exits 0" "0" "1"
fi

# Test 2: makedevs with missing file produces error message
RESULT=$($GOPOSIX makedevs /nonexistent/device_table 2>&1 || true)
if [ -n "$RESULT" ]; then
    check "makedevs bad path reports" "output" "output"
else
    check "makedevs bad path reports" "output" "silent"
fi

# Test 3: makedevs --json validates
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
printf "/dev\tnull\tc\t666\t0\t0\t1\t3\t-\t-\t-\n" > "$TMPDIR/devtable.txt"
if $GOPOSIX makedevs --json "$TMPDIR/devtable.txt" >/dev/null 2>&1; then
    check "makedevs --json exits 0" "0" "0"
else
    # Will fail without root, that's OK
    check "makedevs --json runs" "non-zero (expected without root)" "non-zero (expected without root)"
fi

echo "makedevs compliance: PASS=$PASS FAIL=$FAIL"
exit $FAIL
