#!/bin/sh
# Compliance test for factor utility.
# Tests prime factorization.
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

# Test 1: factor a prime number
RESULT=$($GOPOSIX factor 7)
check "factor 7" "7: 7" "$RESULT"

# Test 2: factor a composite
RESULT=$($GOPOSIX factor 12)
check "factor 12" "12: 2 2 3" "$RESULT"

# Test 3: factor from stdin
RESULT=$(echo "6 15" | $GOPOSIX factor)
check "factor stdin" "6: 2 3
15: 3 5" "$RESULT"

# Test 4: factor 1
RESULT=$($GOPOSIX factor 1)
check "factor 1" "1:" "$RESULT"

echo "factor compliance: PASS=$PASS FAIL=$FAIL"
exit $FAIL
