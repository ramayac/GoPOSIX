#!/bin/sh
# Compliance test for tree utility.
# Tests directory tree display.
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
mkdir -p "$TMPDIR/sub"
touch "$TMPDIR/file.txt" "$TMPDIR/sub/nested.txt"

# Test 1: tree shows directory name
RESULT=$($GOPOSIX tree "$TMPDIR" 2>/dev/null | head -1)
check "tree shows root dir" "$TMPDIR" "$RESULT"

# Test 2: tree shows files
RESULT=$($GOPOSIX tree "$TMPDIR" 2>/dev/null | grep -c "file.txt")
check "tree finds file.txt" "1" "$RESULT"

# Test 3: tree exits 0 on valid dir
if $GOPOSIX tree "$TMPDIR" >/dev/null 2>&1; then
    check "tree exits 0" "0" "0"
else
    check "tree exits 0" "0" "1"
fi

echo "tree compliance: PASS=$PASS FAIL=$FAIL"
exit $FAIL
