#!/bin/sh
# Compliance test for tsort utility.
# Tests topological sort.
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

# Test 1: simple topological sort
RESULT=$(printf "a b\nb c\n" | $GOPOSIX tsort 2>/dev/null | tr '\n' ' ')
check "tsort a-b b-c" "a b c " "$RESULT"

# Test 2: tsort from file
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
printf "x y\ny z\n" > "$TMPDIR/edges.txt"
RESULT=$($GOPOSIX tsort "$TMPDIR/edges.txt" 2>/dev/null | tr '\n' ' ')
check "tsort file" "x y z " "$RESULT"

# Test 3: tsort detects cycle (exits non-zero)
if printf "a b\nb a\n" | $GOPOSIX tsort >/dev/null 2>&1; then
    check "tsort cycle exits non-zero" "non-zero" "0"
else
    check "tsort cycle exits non-zero" "non-zero" "non-zero"
fi

echo "tsort compliance: PASS=$PASS FAIL=$FAIL"
exit $FAIL
