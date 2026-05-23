#!/bin/sh
# Compliance test for cal utility.
# Tests ASCII calendar rendering.
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

# Test 1: cal for a specific month/year
RESULT=$($GOPOSIX cal 1 2024 | head -1)
check "cal 1 2024 header" "    January 2024" "$RESULT"

# Test 2: cal current month produces output
RESULT=$($GOPOSIX cal 2>/dev/null | wc -l)
if [ "$RESULT" -ge 5 ]; then
    check "cal current month has lines" "ge5" "ge5"
else
    check "cal current month has lines" "ge5" "$RESULT"
fi

# Test 3: cal year mode
RESULT=$($GOPOSIX cal 2024 2>/dev/null | grep -c "2024")
if [ "$RESULT" -ge 1 ]; then
    check "cal 2024 shows year" "found" "found"
else
    check "cal 2024 shows year" "found" "not found"
fi

echo "cal compliance: PASS=$PASS FAIL=$FAIL"
exit $FAIL
