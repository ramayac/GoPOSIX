#!/bin/sh
# Compliance test for dc utility.
# Tests desk calculator (RPN arbitrary precision).
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

# Test 1: basic addition
RESULT=$($GOPOSIX dc -e'10 20+p')
check "dc add" "30" "$RESULT"

# Test 2: multiplication
RESULT=$($GOPOSIX dc -e'8 8*p')
check "dc multiply" "64" "$RESULT"

# Test 3: complex expression
RESULT=$($GOPOSIX dc -e'8 8*2 2+/p')
check "dc complex" "16" "$RESULT"

# Test 4: scale and division
RESULT=$($GOPOSIX dc -e'2k 7 3/p')
check "dc scale" "2.33" "$RESULT"

# Test 5: power
RESULT=$($GOPOSIX dc -e'2 10^p')
check "dc power" "1024" "$RESULT"

# Test 6: sqrt
RESULT=$($GOPOSIX dc -e'16vp')
check "dc sqrt" "4" "$RESULT"

# Test 7: string
RESULT=$($GOPOSIX dc -e'[Hello, World!]pR')
check "dc string" "Hello, World!" "$RESULT"

# Test 8: stack depth
RESULT=$($GOPOSIX dc -e'1 2 3zpR')
check "dc depth" "3" "$RESULT"

echo "dc compliance: PASS=$PASS FAIL=$FAIL"
exit $FAIL
