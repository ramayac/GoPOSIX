#!/bin/sh
# Compliance test for mount utility.
# Tests listing mounts and --help (full mount needs root).
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

# Test 1: mount (no args) lists mounts
RESULT=$($GOPOSIX mount 2>/dev/null | wc -l)
if [ "$RESULT" -ge 1 ]; then
    check "mount lists mounts" "has-output" "has-output"
else
    check "mount lists mounts" "has-output" "no output"
fi

# Test 2: mount rejects unknown flag
$GOPOSIX mount --nonexistent-flag 2>/dev/null && rc=0 || rc=$?
if [ "$rc" -ne 0 ]; then
    check "mount unknown flag fails" "non-zero" "non-zero"
else
    check "mount unknown flag fails" "non-zero" "$rc"
fi

# Test 3: mount --json lists mounts
if $GOPOSIX mount --json >/dev/null 2>&1; then
    check "mount --json exits 0" "0" "0"
else
    check "mount --json exits 0" "0" "non-zero (may need root)"
fi

echo "mount compliance: PASS=$PASS FAIL=$FAIL"
exit $FAIL
