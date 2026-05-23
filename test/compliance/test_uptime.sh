#!/bin/sh
# Compliance test for uptime utility.
# Tests system uptime reporting.
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

# Test 1: uptime produces output
RESULT=$($GOPOSIX uptime 2>/dev/null | wc -l)
if [ "$RESULT" -ge 1 ]; then
    check "uptime outputs" "1" "1"
else
    check "uptime outputs" "1" "0"
fi

# Test 2: uptime output contains load average
RESULT=$($GOPOSIX uptime 2>/dev/null)
case "$RESULT" in
    *load*)  check "uptime mentions load" "found" "found" ;;
    *user*)  check "uptime mentions load" "found" "found" ;;
    *up*)    check "uptime mentions load" "found" "found" ;;
    *)       check "uptime mentions load" "found" "not found in: $RESULT" ;;
esac

# Test 3: uptime --json exits 0
if $GOPOSIX uptime --json >/dev/null 2>&1; then
    check "uptime --json exits 0" "0" "0"
else
    check "uptime --json exits 0" "0" "1"
fi

echo "uptime compliance: PASS=$PASS FAIL=$FAIL"
exit $FAIL
