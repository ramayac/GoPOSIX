#!/bin/bash
# test/compliance/test_awk.sh — POSIX compliance tests for awk
set -uo pipefail

GOPOSIX="${GOPOSIX:-goposix}"
PASS=0
FAIL=0
TOTAL=0

check() {
    local name="$1" expected="$2" input="$3"
    shift 3
    ((TOTAL++))
    local actual
    actual=$(printf '%s' "$input" | "$GOPOSIX" awk "$@" 2>&1) || true
    if [ "$actual" = "$expected" ]; then
        echo "PASS: $name"
        ((PASS++))
    else
        echo "FAIL: $name"
        echo "  got:    '$actual'"
        echo "  wanted: '$expected'"
        ((FAIL++))
    fi
}

echo "=== GoPOSIX awk Compliance Tests ==="

# Basic field splitting
check "print first field"      "alice"  "alice:90"              '{ print $1 }' -F:
check "default whitespace FS"  "alice"  "alice 90 bob"          '{ print $1 }'

# BEGIN / END
check "BEGIN block"            "start"  ""                      'BEGIN { print "start" }'
check "END block (sum)"        "60"     $'10\n20\n30'           '{ sum += $1 } END { print sum }'

# Built-in variables
check "NR and NF" "1 3" "a b c" '{ print NR, NF }'

# Pattern matching
check "regex match" "hello" "hello world" '/hello/ { print $1 }'

# Built-in functions
check "length()"  "5"   "hello"  '{ print length($0) }'
check "substr()"  "bcd" "abcdef" '{ print substr($0, 2, 3) }'
check "split()"   "3 y" "x,y,z"  '{ n = split($0, a, ","); print n, a[2] }'
check "int()"     "42"  "42.9"   '{ print int($1) }'

# Arithmetic
check "addition" "15" "10 5" '{ print $1 + $2 }'

# Control flow
check "if/else (small)" "small" "3" '{ if ($1 > 5) print "big"; else print "small" }'
check "if/else (big)"   "big"   "8" '{ if ($1 > 5) print "big"; else print "small" }'

# Arrays
check "arrays" "1 99" "x 99" '{ a[$1]=$2 } END { print length(a), a["x"] }'

# -v flag
check "-v flag (match)"    "alice"  "alice 90"  '$2 > threshold { print $1 }' -v threshold=50
check "-v flag (no match)" ""       "bob 30"    '$2 > threshold { print $1 }' -v threshold=50

# -F flag concatenated
check "-F: concat" "alice" "alice:90" '{ print $1 }' -F:

# Syntax error
check "syntax error" \
  "awk: parse error at 1:2: expected } instead of EOF" \
  "" '{'

echo ""
echo "=== Results: $PASS/$TOTAL passed ==="
if [ "$FAIL" -gt 0 ]; then
    echo "FAILURES: $FAIL"
    exit 1
fi
exit 0
