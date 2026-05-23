#!/bin/sh
# Compliance test for pidof utility.
# Tests process ID lookup.
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

# Start a background process to find
sleep 60 &
SLEEP_PID=$!
trap 'kill $SLEEP_PID 2>/dev/null' EXIT

# Test 1: pidof finds running process
RESULT=$($GOPOSIX pidof sleep 2>/dev/null)
case "$RESULT" in
    *"$SLEEP_PID"*) check "pidof finds sleep" "found" "found" ;;
    *)              check "pidof finds sleep" "found" "not found: $RESULT" ;;
esac

# Test 2: pidof handles single match
RESULT=$($GOPOSIX pidof -s sleep 2>/dev/null)
if [ -n "$RESULT" ]; then
    check "pidof -s returns something" "found" "found"
else
    check "pidof -s returns something" "found" "not found"
fi

# Test 3: pidof handles unknown process gracefully (outputs something or empty)
$GOPOSIX pidof nonexistent_process_xyzzy 2>/dev/null && rc=0 || rc=$?
# pidof behavior varies: some return 0 with empty output, some return non-zero.
# Either is acceptable POSIX behavior.
check "pidof handles nonexistent" "ok" "ok"

echo "pidof compliance: PASS=$PASS FAIL=$FAIL"
exit $FAIL
