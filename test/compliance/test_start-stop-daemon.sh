#!/bin/sh
# Compliance test for start-stop-daemon utility.
# Tests daemon management (test mode and error handling).
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

# Test 1: --test mode runs without starting anything
if $GOPOSIX start-stop-daemon --start --test --exec /bin/true 2>/dev/null; then
    check "start-stop-daemon --test exits 0" "0" "0"
else
    check "start-stop-daemon --test exits 0" "0" "1"
fi

# Test 2: fails without --start or --stop
if $GOPOSIX start-stop-daemon --exec /bin/true 2>/dev/null; then
    check "start-stop-daemon no action fails" "non-zero" "0"
else
    check "start-stop-daemon no action fails" "non-zero" "non-zero"
fi

# Test 3: --help works
if $GOPOSIX start-stop-daemon --help >/dev/null 2>&1; then
    check "start-stop-daemon --help" "0" "0"
else
    check "start-stop-daemon --help" "0" "1"
fi

echo "start-stop-daemon compliance: PASS=$PASS FAIL=$FAIL"
exit $FAIL
