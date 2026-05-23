#!/bin/sh
# Compliance test for unzip utility.
# Tests ZIP archive extraction.
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

command -v zip >/dev/null 2>&1 || { echo "unzip compliance: PASS=$PASS FAIL=$FAIL (skipped: no zip)"; exit 0; }

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

echo "hello zip" > "$TMPDIR/hello.txt"
( cd "$TMPDIR" && zip -q test.zip hello.txt )

# Test 1: unzip -l lists files
RESULT=$($GOPOSIX unzip -l "$TMPDIR/test.zip" 2>/dev/null | grep -c "hello.txt")
check "unzip -l finds hello.txt" "1" "$RESULT"

# Test 2: unzip extracts files
EXTRACT_DIR="$TMPDIR/extracted"
mkdir -p "$EXTRACT_DIR"
$GOPOSIX unzip -d "$EXTRACT_DIR" "$TMPDIR/test.zip" 2>/dev/null
if [ -f "$EXTRACT_DIR/hello.txt" ]; then
    RESULT=$(cat "$EXTRACT_DIR/hello.txt")
    check "unzip extract" "hello zip" "$RESULT"
else
    check "unzip extract" "hello zip" "file not found"
fi

# Test 3: unzip exits 0 on success
if $GOPOSIX unzip -l "$TMPDIR/test.zip" >/dev/null 2>&1; then
    check "unzip -l exits 0" "0" "0"
else
    check "unzip -l exits 0" "0" "1"
fi

echo "unzip compliance: PASS=$PASS FAIL=$FAIL"
exit $FAIL
