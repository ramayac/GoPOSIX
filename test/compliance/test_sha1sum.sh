#!/bin/sh
# Compliance test for sha1sum utility.
# Tests SHA-1 cryptographic digest computation.
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

# Test 1: sha1sum of a file
echo "hello" > "$TMPDIR/test.txt"
RESULT=$($GOPOSIX sha1sum "$TMPDIR/test.txt" | awk '{print $1}')
check "sha1sum of 'hello'" "f572d396fae9206628714fb2ce00f72e94f2258f" "$RESULT"

# Test 2: sha1sum of stdin
RESULT=$(echo "hello" | $GOPOSIX sha1sum | awk '{print $1}')
check "sha1sum stdin" "f572d396fae9206628714fb2ce00f72e94f2258f" "$RESULT"

# Test 3: sha1sum check mode
echo "f572d396fae9206628714fb2ce00f72e94f2258f  $TMPDIR/test.txt" > "$TMPDIR/checksums"
RESULT=$($GOPOSIX sha1sum -c "$TMPDIR/checksums" 2>&1 | grep -c "OK")
check "sha1sum -c OK" "1" "$RESULT"

echo "sha1sum compliance: PASS=$PASS FAIL=$FAIL"
exit $FAIL
