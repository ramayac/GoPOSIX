#!/bin/sh
# Compliance test for ash utility.
# ash is an alias for the native shell interpreter.
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

# Test 1: ash -c runs inline scripts
RESULT=$(goposix ash -c 'echo hello ash')
check "ash -c inline" "hello ash" "$RESULT"

# Test 2: ash runs scripts from a file
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
cat > "$TMPDIR/test.sh" << 'SCRIPT'
echo script output
SCRIPT
RESULT=$(goposix ash "$TMPDIR/test.sh")
check "ash script file" "script output" "$RESULT"

# Test 3: ash returns correct exit code
goposix ash -c 'exit 0'
check "ash exit 0" "0" "$?"

goposix ash -c 'exit 42'
check "ash exit 42" "42" "$?"

echo "ash compliance: PASS=$PASS FAIL=$FAIL"
exit $FAIL
