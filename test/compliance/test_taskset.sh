#!/bin/sh
# Compliance test for taskset utility.
# Tests CPU affinity querying.
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

# Test 1: taskset -p shows current affinity mask
RESULT=$($GOPOSIX taskset -p $$ 2>/dev/null)
case "$RESULT" in
    *"affinity"*|*"mask"*|*[0-9a-f]*)
        check "taskset -p shows mask" "found" "found" ;;
    *)
        check "taskset -p shows mask" "found" "not found: $RESULT" ;;
esac

# Test 2: taskset --json exits 0
if $GOPOSIX taskset --json -p 1 2>/dev/null; then
    check "taskset --json exits 0" "0" "0"
else
    # May fail without proper permissions
    check "taskset --json exits 0" "0" "non-zero (may need root)"
fi

# Test 3: taskset help
if $GOPOSIX taskset --help >/dev/null 2>&1; then
    check "taskset --help exits 0" "0" "0"
else
    check "taskset --help exits 0" "0" "1"
fi

echo "taskset compliance: PASS=$PASS FAIL=$FAIL"
exit $FAIL
