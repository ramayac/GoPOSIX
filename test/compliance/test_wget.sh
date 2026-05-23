#!/bin/sh
# Compliance test for wget utility.
# Tests non-interactive HTTP download.
# Note: wget requires network; these tests use --help and validation.
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

# Test 1: wget with no URL exits non-zero
$GOPOSIX wget 2>/dev/null && rc=0 || rc=$?
if [ "$rc" -ne 0 ]; then
    check "wget no-URL fails" "non-zero" "non-zero"
else
    check "wget no-URL fails" "non-zero" "$rc"
fi

# Test 2: wget with invalid URL fails
$GOPOSIX wget "http://0.0.0.0:1/nope" 2>/dev/null && rc=0 || rc=$?
if [ "$rc" -ne 0 ]; then
    check "wget bad URL fails" "non-zero" "non-zero"
else
    check "wget bad URL fails" "non-zero" "$rc"
fi

# Test 3: wget unknown flag exits non-zero
if $GOPOSIX wget --nonexistent-flag-xyz 2>/dev/null; then
    check "wget bad flag exits non-zero" "0" "1"
else
    check "wget bad flag exits non-zero" "1" "1"
fi

# Test 4: wget invalid URL exits non-zero
$GOPOSIX wget "http://0.0.0.0:1/nonexistent" 2>/dev/null && rc=0 || rc=$?
if [ "$rc" -ne 0 ]; then
    check "wget bad URL fails" "non-zero" "non-zero"
else
    check "wget bad URL fails" "non-zero" "zero"
fi

echo "wget compliance: PASS=$PASS FAIL=$FAIL"
exit $FAIL
