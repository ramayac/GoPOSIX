#!/bin/sh
# Compliance test for tar utility.
# Tests --json output matches schema requirements.
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
# Resolve to absolute path before cd
case "$GOPOSIX" in
    /*) ;;
    *) GOPOSIX=$(command -v "$GOPOSIX" || echo "$GOPOSIX") ;;
esac

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

cd "$TMPDIR" || exit 1

# Create a simple tar archive
echo "test content" > testfile.txt
$GOPOSIX tar -c -f test.tar testfile.txt 2>/dev/null

# Test 1: --json list output is valid JSON with name field
JSON=$($GOPOSIX tar -t -f test.tar --json 2>/dev/null)
if echo "$JSON" | grep -q '"name"'; then
    check "tar --json list has name field" "ok" "ok"
else
    check "tar --json list has name field" "ok" "fail"
fi

# Test 2: --json list output has size field
if echo "$JSON" | grep -q '"size"'; then
    check "tar --json list has size field" "ok" "ok"
else
    check "tar --json list has size field" "ok" "fail"
fi

# Test 3: --json list output has mode field
if echo "$JSON" | grep -q '"mode"'; then
    check "tar --json list has mode field" "ok" "ok"
else
    check "tar --json list has mode field" "ok" "fail"
fi

# Test 4: tar create with --json produces JSON output
JSON2=$($GOPOSIX tar -c -f test2.tar testfile.txt --json 2>/dev/null)
if [ -n "$JSON2" ]; then
    check "tar --json create produces output" "ok" "ok"
else
    check "tar --json create produces output" "ok" "fail"
fi

# Test 5: tar exits 0 on success
if $GOPOSIX tar -t -f test.tar >/dev/null 2>&1; then
    check "tar list exits 0" "ok" "ok"
else
    check "tar list exits 0" "ok" "fail"
fi

echo "tar compliance: PASS=$PASS FAIL=$FAIL"
exit $FAIL
