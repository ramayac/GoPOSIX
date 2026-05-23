#!/bin/sh
# Compliance test for sha512sum utility.
# Tests SHA-512 cryptographic digest computation.
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

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

# Test 1: sha512sum of a file
echo "hello" > "$TMPDIR/test.txt"
RESULT=$($GOPOSIX sha512sum "$TMPDIR/test.txt" | awk '{print $1}')
EXPECTED="e7c22b994c59d9cf2b48e549b1e24666636045930d3da7c1acb299d1c3b7f931f94aae41edda2c2b207a36e10f8bcb8d45223e54878f5b316e7ce3b6bc019629"
check "sha512sum of 'hello'" "$EXPECTED" "$RESULT"

# Test 2: sha512sum of stdin
RESULT=$(echo "hello" | $GOPOSIX sha512sum | awk '{print $1}')
check "sha512sum stdin" "$EXPECTED" "$RESULT"

# Test 3: sha512sum check mode
echo "$EXPECTED  $TMPDIR/test.txt" > "$TMPDIR/checksums"
RESULT=$($GOPOSIX sha512sum -c "$TMPDIR/checksums" 2>&1 | grep -c "OK")
check "sha512sum -c OK" "1" "$RESULT"

echo "sha512sum compliance: PASS=$PASS FAIL=$FAIL"
exit $FAIL
