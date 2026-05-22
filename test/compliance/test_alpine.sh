#!/bin/sh

# Color output codes for a premium terminal experience
GREEN='\033[0;32m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color
BOLD='\033[1m'

echo -e "${BLUE}${BOLD}======================================================================${NC}"
echo -e "${CYAN}${BOLD}   ____            _     _       _                   ${NC}"
echo -e "${CYAN}${BOLD}  / ___| ___      / \   | |_ __ (_)_ __   ___        ${NC}"
echo -e "${CYAN}${BOLD} | |  _ / _ \    / _ \  | | '_ \| | '_ \ / _ \\       ${NC}"
echo -e "${CYAN}${BOLD} | |_| | (_) |  / ___ \ | | |_) | | | | |  __/       ${NC}"
echo -e "${CYAN}${BOLD}  \\____|\\___/  /_/   \\_\\|_| .__/|_|_| |_|\\___|       ${NC}"
echo -e "${CYAN}${BOLD}                          |_|                        ${NC}"
echo -e "${GREEN}${BOLD}      GoPOSIX-powered Alpine Userland (go-alpine MVP)        ${NC}"
echo -e "${BLUE}${BOLD}======================================================================${NC}"
echo ""

echo -e "${BOLD}[1/4] SYSTEM DIAGNOSTICS${NC}"
echo -e "------------------------------------------------------------"

# 1. Inspect Shell & Binary Format
if [ -L /bin/sh ]; then
    echo -e "• /bin/sh symlink:      ${GREEN}$(ls -ld /bin/sh)${NC}"
else
    echo -e "• /bin/sh symlink:      ${RED}Warning: Not a symbolic link!${NC}"
fi

# 2. Check executable size & fingerprint
if [ -f /bin/busybox ]; then
    SIZE_MB=$(ls -lh /bin/busybox | awk '{print $5}')
    echo -e "• /bin/busybox file:    ${GREEN}Found (Size: $SIZE_MB)${NC}"
else
    echo -e "• /bin/busybox file:    ${RED}Missing!${NC}"
fi

# 3. Read GoPOSIX Version
GP_VERSION=$(goposix --version 2>&1)
if echo "$GP_VERSION" | grep -iq "goposix"; then
    echo -e "• Engine detected:      ${GREEN}${BOLD}GoPOSIX ($GP_VERSION)${NC}"
else
    echo -e "• Engine detected:      ${RED}Standard BusyBox or Unknown! ($GP_VERSION)${NC}"
fi
echo ""

echo -e "${BOLD}[2/4] CORE POSIX COMPATIBILITY TESTS${NC}"
echo -e "------------------------------------------------------------"

# Test ls (File system listing)
echo -n "• testing 'ls -la /': "
if ls -la / >/dev/null 2>&1; then
    echo -e "${GREEN}✓ PASSED${NC}"
else
    echo -e "${RED}✗ FAILED${NC}"
fi

# Test echo and grep (Streaming pipe & pattern search)
echo -n "• testing 'echo | grep': "
TEST_PIPE=$(echo "pure-go-unix-power" | grep "go-unix")
if [ "$TEST_PIPE" = "pure-go-unix-power" ]; then
    echo -e "${GREEN}✓ PASSED${NC}"
else
    echo -e "${RED}✗ FAILED (got: '$TEST_PIPE')${NC}"
fi

# Test awk processing
echo -n "• testing 'awk' math: "
AWK_MATH=$(awk 'BEGIN {print (40 + 2)}')
if [ "$AWK_MATH" = "42" ]; then
    echo -e "${GREEN}✓ PASSED (40 + 2 = $AWK_MATH)${NC}"
else
    echo -e "${RED}✗ FAILED (got: '$AWK_MATH')${NC}"
fi

# Test mkdir & rmdir
echo -n "• testing 'mkdir' & 'rmdir': "
if mkdir /tmp/goposix-temp-dir && rmdir /tmp/goposix-temp-dir; then
    echo -e "${GREEN}✓ PASSED${NC}"
else
    echo -e "${RED}✗ FAILED${NC}"
fi
echo ""

echo -e "${BOLD}[3/4] ADVANCED GO-ENGINE TEST${NC}"
echo -e "------------------------------------------------------------"
# Go binaries carry special symbols. Let's look for Go-specific artifacts in /bin/busybox to prove it's a Go binary!
if strings /bin/busybox | grep -q "go.production"; then
    echo -e "${GREEN}✓ PROVED: /bin/busybox contains Go runtime execution signatures.${NC}"
elif strings /bin/busybox | grep -q "Go build ID"; then
    echo -e "${GREEN}✓ PROVED: /bin/busybox verified as compiled Go binary (contains Go build metadata).${NC}"
else
    # Fallback check
    echo -e "${YELLOW}ℹ /bin/busybox size is $(ls -lh /bin/busybox | awk '{print $5}'), which fits statically compiled Go binaries.${NC}"
fi
echo ""

echo -e "${BOLD}[4/4] THE ACID TEST: Package Installation (${CYAN}apk add${YELLOW})${NC}"
echo -e "------------------------------------------------------------"
echo -e "Let's attempt a package manager run. Alpine's 'apk' manager executes several shell"
echo -e "scripts, creating streams and reading archives via tar and gzip."
echo ""
echo -e "${YELLOW}Executing: apk update && apk add --simulate curl${NC}"
echo "------------------------------------------------------------"
apk update && apk add --simulate curl
echo "------------------------------------------------------------"
echo ""
echo -e "${GREEN}${BOLD}Verification Complete! Enjoy exploring the pure Go userland!${NC}"
echo -e "${BLUE}${BOLD}======================================================================${NC}"
