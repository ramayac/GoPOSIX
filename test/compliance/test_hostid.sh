#!/bin/sh
# Compliance test for hostid utility.
# Tests host identifier output.
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

# Test 1: hostid outputs 8 hex chars
RESULT=$($GOPOSIX hostid 2>/dev/null | tr -d '\n')
LEN=${#RESULT}
if [ "$LEN" -eq 8 ]; then
    check "hostid is 8 chars" "8" "$LEN"
else
    check "hostid is 8 chars" "8" "$LEN ($RESULT)"
fi

# Test 2: hostid only contains hex chars
case "$RESULT" in
    *[!0-9a-fA-F]*) check "hostid is hex" "hex" "non-hex: $RESULT" ;;
    *)              check "hostid is hex" "hex" "hex" ;;
esac

# Test 3: hostid exits 0
if $GOPOSIX hostid >/dev/null 2>&1; then
    check "hostid exits 0" "0" "0"
else
    check "hostid exits 0" "0" "1"
fi

echo "hostid compliance: PASS=$PASS FAIL=$FAIL"
exit $FAIL
