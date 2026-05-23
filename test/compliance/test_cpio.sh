#!/bin/sh
# Compliance test for cpio utility.
# Tests create (-o), list (-t), and extract (-i) operations.
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

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
cd "$TMPDIR"

# Create test files
echo "hello cpio" > file_a.txt
echo "goodbye cpio" > file_b.txt

# Test 1: Create archive
printf "file_a.txt\nfile_b.txt\n" | goposix cpio -o > test.cpio
check "cpio create: archive non-empty" "1" "$(test -s test.cpio && echo 1 || echo 0)"

# Test 2: List archive
RESULT=$(goposix cpio -it < test.cpio | sort | tr '\n' '|')
check "cpio list" "file_a.txt|file_b.txt|" "$RESULT"

# Test 3: Extract archive
mkdir extract_dir
cd extract_dir
goposix cpio -id < ../test.cpio
check "cpio extract file_a.txt" "hello cpio" "$(cat file_a.txt)"
check "cpio extract file_b.txt" "goodbye cpio" "$(cat file_b.txt)"
cd ..

echo "cpio compliance: PASS=$PASS FAIL=$FAIL"
exit $FAIL
