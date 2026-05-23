#!/bin/sh
# Compliance test for rev utility.
# Tests line reversal.
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

# Test 1: rev reverses a single line
RESULT=$(echo "hello" | $GOPOSIX rev)
check "rev 'hello'" "olleh" "$RESULT"

# Test 2: rev reverses file
printf "abc\n123\n" > "$TMPDIR/input.txt"
RESULT=$($GOPOSIX rev "$TMPDIR/input.txt" | tr '\n' '|')
check "rev file" "cba|321|" "$RESULT"

# Test 3: rev handles empty input
RESULT=$(printf "" | $GOPOSIX rev)
check "rev empty" "" "$RESULT"

echo "rev compliance: PASS=$PASS FAIL=$FAIL"
exit $FAIL
