#!/bin/sh
# Compliance test for realpath utility.
# Tests canonical path resolution.
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

# Test 1: resolve an absolute path
RESULT=$($GOPOSIX realpath /tmp)
check "realpath /tmp" "/tmp" "$RESULT"

# Test 2: resolve a relative path
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
cd "$TMPDIR"
mkdir -p a/b
ln -s a/b link
RESULT=$($GOPOSIX realpath link)
check "realpath symlink" "$TMPDIR/a/b" "$RESULT"

# Test 3: resolve multiple paths
echo "hello" > "$TMPDIR/file.txt"
RESULT=$($GOPOSIX realpath file.txt /tmp 2>/dev/null | wc -l)
check "realpath multiple paths" "2" "$RESULT"

echo "realpath compliance: PASS=$PASS FAIL=$FAIL"
exit $FAIL
