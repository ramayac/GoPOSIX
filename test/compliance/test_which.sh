#!/bin/sh
# Compliance test for which utility.
# Tests PATH lookup for known commands.
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

# Test 1: which finds a command in PATH
RESULT=$($GOPOSIX which ls 2>/dev/null | head -1)
case "$RESULT" in
    */ls) check "which finds ls" "found" "found" ;;
    *)    check "which finds ls" "found" "not found: $RESULT" ;;
esac

# Test 2: which returns nothing for nonexistent command
RESULT=$($GOPOSIX which nonexistent_cmd_xyzzy 2>/dev/null)
check "which nonexistent returns empty" "" "$RESULT"

# Test 3: which -a shows all matches
RESULT=$($GOPOSIX which -a ls 2>/dev/null | wc -l)
if [ "$RESULT" -gt 0 ]; then
    check "which -a returns results" "1" "1"
else
    check "which -a returns results" "1" "0"
fi

echo "which compliance: PASS=$PASS FAIL=$FAIL"
exit $FAIL
