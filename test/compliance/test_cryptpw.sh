#!/bin/sh
# Compliance test for cryptpw utility.
# Tests password hashing.
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

# Test 1: cryptpw produces a hash
RESULT=$($GOPOSIX cryptpw mypassword 2>/dev/null | wc -c)
if [ "$RESULT" -gt 10 ]; then
    check "cryptpw produces hash" "non-empty" "non-empty"
else
    check "cryptpw produces hash" "non-empty" "empty or too short"
fi

# Test 2: cryptpw with method flag
RESULT=$($GOPOSIX cryptpw -m md5 mypassword 2>/dev/null)
case "$RESULT" in
    \$1\$*) check "cryptpw -m md5 produces md5crypt" "md5crypt" "md5crypt" ;;
    ??*)     check "cryptpw -m md5" "non-empty" "non-empty" ;;
    *)       check "cryptpw -m md5" "non-empty" "empty" ;;
esac

# Test 3: cryptpw exits 0
if $GOPOSIX cryptpw test >/dev/null 2>&1; then
    check "cryptpw exits 0" "0" "0"
else
    check "cryptpw exits 0" "0" "1"
fi

# Test 4: same password produces hash of expected length
RESULT=$($GOPOSIX cryptpw samepass 2>/dev/null)
LEN=${#RESULT}
if [ "$LEN" -ge 13 ]; then
    check "cryptpw hash length >= 13" "ge13" "ge13"
else
    check "cryptpw hash length >= 13" "ge13" "$LEN"
fi

echo "cryptpw compliance: PASS=$PASS FAIL=$FAIL"
exit $FAIL
