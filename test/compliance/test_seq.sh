#!/bin/sh
# Compliance test for seq utility.
# Tests numeric sequence generation.
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

# Test 1: simple range
RESULT=$($GOPOSIX seq 1 5 | tr '\n' '|')
check "seq 1 5" "1|2|3|4|5|" "$RESULT"

# Test 2: first + increment + last
RESULT=$($GOPOSIX seq 0 2 10 | tr '\n' '|')
check "seq 0 2 10" "0|2|4|6|8|10|" "$RESULT"

# Test 3: descending
RESULT=$($GOPOSIX seq 5 -1 1 | tr '\n' '|')
check "seq 5 -1 1" "5|4|3|2|1|" "$RESULT"

# Test 4: single value
RESULT=$($GOPOSIX seq 1 1)
check "seq 1 1" "1" "$RESULT"

echo "seq compliance: PASS=$PASS FAIL=$FAIL"
exit $FAIL
