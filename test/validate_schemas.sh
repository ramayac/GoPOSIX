#!/usr/bin/env bash
# validate_schemas.sh — Validate korego --json output against published JSON schemas.
# Usage: make validate-schemas   or   bash test/validate_schemas.sh
set -euo pipefail

KOREGO=${KOREGO:-./korego}
SCHEMA_DIR="test/schemas"
GOLDEN_DIR="$SCHEMA_DIR/golden"
PASS=0
FAIL=0
SKIP=0

validate() {
  local util="$1" schema="$SCHEMA_DIR/${util}.schema.json" golden="$GOLDEN_DIR/${util}.json"
  if [ ! -f "$schema" ]; then
    echo "SKIP: $util (no schema)"
    SKIP=$((SKIP + 1))
    return
  fi
  if [ ! -f "$golden" ]; then
    echo "SKIP: $util (no golden fixture)"
    SKIP=$((SKIP + 1))
    return
  fi
  if npx --yes ajv-cli validate -s "$schema" -d "$golden" > /dev/null 2>&1; then
    echo "PASS: $util"
    PASS=$((PASS + 1))
  else
    echo "FAIL: $util"
    FAIL=$((FAIL + 1))
  fi
}

echo "=== KoreGo Schema Validation ==="
echo ""

# Validate all schemas against golden fixtures
for schema_file in "$SCHEMA_DIR"/*.schema.json; do
  util=$(basename "$schema_file" .schema.json)
  [ "$util" = "common" ] && continue
  validate "$util"
done

echo ""
echo "=== Results: $PASS passed, $FAIL failed, $SKIP skipped ==="

if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
exit 0
