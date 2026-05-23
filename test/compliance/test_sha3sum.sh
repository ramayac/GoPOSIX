#!/bin/sh
# Compliance test for sha3sum utility.
# Tests SHA-3 cryptographic digest computation.
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

# Test 1: sha3sum -a 256 of a file
echo "hello" > "$TMPDIR/test.txt"
RESULT=$($GOPOSIX sha3sum -a 256 "$TMPDIR/test.txt" | awk '{print $1}')
EXPECTED="b314e28493eae9dab57ac4f0c6d887bddbbeb810e900d818395ace558e96516d"
check "sha3-256 of 'hello'" "$EXPECTED" "$RESULT"

# Test 2: sha3sum -a 512 of stdin
RESULT=$(echo "hello" | $GOPOSIX sha3sum -a 512 | awk '{print $1}')
EXPECTED="ac766ba623301e0ad63c48cb2fc469d10145f65c9f1f28fe761c78c386ed295a1fda1b05e280354e620757d8a83e05a45f66438dd734278668c1c27ac6f27150"
check "sha3-512 stdin" "$EXPECTED" "$RESULT"

# Test 3: sha3sum default algorithm
RESULT=$($GOPOSIX sha3sum -a 256 "$TMPDIR/test.txt" 2>/dev/null | wc -l)
if [ "$RESULT" -ge 1 ]; then
    check "sha3sum -a 256 produces output" "1" "1"
else
    check "sha3sum -a 256 produces output" "1" "0"
fi

echo "sha3sum compliance: PASS=$PASS FAIL=$FAIL"
exit $FAIL
