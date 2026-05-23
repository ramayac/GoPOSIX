#!/bin/sh
# Compliance test for ar utility.
# Tests basic create, list, print, replace, extract, and delete operations.
set -uo pipefail

PASS=0
FAIL=0

check() {
    local name="$1"
    local expected="$2"
    local actual="$3"
    if [ "$actual" = "$expected" ]; then
        PASS=$((PASS + 1))
    else
        FAIL=$((FAIL + 1))
        echo "FAIL: $name"
        echo "  expected: $expected"
        echo "  actual:   $actual"
    fi
}

AR="${AR:-ar}"
GOPOSIX_AR="${GOPOSIX_AR:-goposix}"

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
cd "$TMPDIR"

# Create test files
echo "hello world" > hello.txt
echo "goodbye world" > goodbye.txt

# Test 1: Create archive and list
$GOPOSIX_AR ar rc test.a hello.txt goodbye.txt
RESULT=$($GOPOSIX_AR ar t test.a | sort | tr '\n' '|')
check "ar create and list" "goodbye.txt|hello.txt|" "$RESULT"

# Test 2: Print file from archive
RESULT=$($GOPOSIX_AR ar p test.a hello.txt)
check "ar print file" "hello world" "$RESULT"

# Test 3: Extract file
rm -f hello.txt goodbye.txt
$GOPOSIX_AR ar x test.a
check "ar extract hello.txt" "hello world" "$(cat hello.txt)"
check "ar extract goodbye.txt" "goodbye world" "$(cat goodbye.txt)"

# Test 4: Delete member
$GOPOSIX_AR ar d test.a hello.txt
RESULT=$($GOPOSIX_AR ar t test.a)
check "ar delete: only goodbye.txt remains" "goodbye.txt" "$RESULT"

# Test 5: Replace/update member
echo "updated content" > goodbye.txt
$GOPOSIX_AR ar r test.a goodbye.txt
RESULT=$($GOPOSIX_AR ar p test.a goodbye.txt)
check "ar replace: updated content" "updated content" "$RESULT"

echo "ar compliance: PASS=$PASS FAIL=$FAIL"
exit $FAIL
