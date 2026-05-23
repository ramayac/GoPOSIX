#!/bin/sh
# Compliance test for mdev utility.
# Tests device manager (scan/dry-run modes; full operation needs root).
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

# Test 1: mdev -d (dry run/discovery) exits 0
if $GOPOSIX mdev -d >/dev/null 2>&1; then
    check "mdev -d exits 0" "0" "0"
else
    check "mdev -d exits 0" "0" "non-zero (may need /sys)"
fi

# Test 2: mdev -d (dry-run) doesn't crash
RESULT=$($GOPOSIX mdev -d 2>&1; echo "exit=$?")
case "$RESULT" in
    *"exit=0"*) check "mdev -d exits 0" "0" "0" ;;
    *)          check "mdev -d exits 0" "0" "non-zero (may need /sys)" ;;
esac

# Test 3: mdev -s (scan) in dry-run
if $GOPOSIX mdev -s -d 2>/dev/null; then
    check "mdev -s -d exits 0" "0" "0"
else
    check "mdev -s -d exits 0" "0" "non-zero (expected without root/sys)"
fi

echo "mdev compliance: PASS=$PASS FAIL=$FAIL"
exit $FAIL
