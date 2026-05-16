# Phase 14a — JSON Gap Fill (8 Utilities)

> **Status:** COMPLETED | **Date:** 2026-05-15 | **Parent:** [14_xml_output.md](14_xml_output.md)

---

## Goal

Add `--json` structured output to the 8 utilities that previously lacked it
(or parsed `--json` manually outside the `common.FlagSpec` system). This closes the last
remaining gap before full XML rollout.

---

## Gap Inventory

| # | Utility | `--json` Status | Has Result Type? | Uses FlagSpec? | Root Cause |
|---|---------|----------------|-----------------|----------------|------------|
| 1 | `echo` | ✅ Fixed | `EchoResult` ✅ | ✅ Now uses spec | Was manual `for` loop over `os.Args` |
| 2 | `testcmd` | ✅ Fixed | `TestResult` ✅ | ✅ Pre-processes | Was stripping `--json`/`-j` at position 0 only |
| 3 | `sed` | ✅ Added | `SedResult` added | ✅ Added --json | Was never implemented |
| 4 | `tee` | ✅ Added | `TeeResult` added | ✅ Added --json | Was never implemented |
| 5 | `tr` | ✅ Added | `TrResult` added | ✅ Added --json | Was never implemented |
| 6 | `sleep` | ✅ Added | `SleepResult` added | ✅ Added --json | Was never implemented |
| 7 | `truefalse` | ✅ Added | `BoolResult` added | ✅ Added --json | Was never implemented |
| 8 | `yes` | ✅ Added | `YesResult` added | ✅ Added --json | Was documented as "does not support --json" |

---

## Per-Utility Implementation Notes

### 1. `echo` ✅ — FlagSpec Integration
- Replaced manual arg loop with `common.ParseFlags` + `spec`
- Flags: `-n`, `-e`, `-E`, `-j`/`--json` via FlagSpec
- Short flag bundling (`-ne`, `-en`) works via POSIX parser
- Fixed `fmt.Print`→`fmt.Fprint(out)` for proper test output capture

### 2. `testcmd` ✅ — Pre-process Extraction
- Pre-processes `--json`/`-j` at ANY position (not just first arg)
- `[` bracket form also handles `--json`/`-j`
- Expression arguments pass through untouched
- Added spec declaration for completeness

### 3. `sed` ✅ — SedResult Type
- Added `SedResult{Lines, LineCount, Changed, Scripts}`
- In JSON mode: captures engine output via `bytes.Buffer`, splits into lines
- Rejects `--json`+`--in-place` combination (mutually exclusive)

### 4. `tee` ✅ — TeeResult Type
- Added `TeeResult{BytesWritten, Files}`
- Uses `countingWriter` wrapper to track bytes
- In JSON mode: stdout→`io.Discard` (data captured in result)

### 5. `tr` ✅ — TrResult Type
- Added `TrResult{Lines, LineCount, BytesIn, BytesOut}`
- In JSON mode: reads full stdin, processes to buffer, reports byte counts

### 6. `sleep` ✅ — SleepResult Type
- Added `SleepResult{Duration, Requested, Interrupted}`
- Tracks requested vs actual sleep time via `time.Since`

### 7. `truefalse` ✅ — BoolResult Type
- Added `BoolResult{ExitCode, Value}`
- Both `true` and `false` accept `--json`/`-j`
- JSON mode: renders structured result instead of silent exit
- `true` → `{exitCode: 0, value: true}`, `false` → `{exitCode: 1, value: false}`

### 8. `yes` ✅ — YesResult Type
- Added `YesResult{String, Count, Truncated}`
- Supports `--json`/`-j` and `--count`/`-n`
- JSON mode: outputs ONLY the JSON envelope (text data in result)
- Text mode: preserves infinite loop with SIGPIPE handling

---

## Task Checklist

| # | Utility | Result Type | FlagSpec | run() Logic | Tests |
|---|---------|-------------|----------|-------------|-------|
| 1 | `echo` | `EchoResult` (exists) | ✅ FlagSpec replaces manual | ✅ JSON mode | ✅ Both `--json`/`-j` |
| 2 | `testcmd` | `TestResult` (exists) | ✅ Pre-process extraction | ✅ JSON mode | ✅ Both `--json`/`-j` |
| 3 | `sed` | `SedResult` | ✅ Add --json flag | ✅ Buffer lines | ✅ --json tests |
| 4 | `tee` | `TeeResult` | ✅ Add --json flag | ✅ Track bytes | ✅ --json tests |
| 5 | `tr` | `TrResult` | ✅ Add --json flag | ✅ Buffer + count | ✅ --json tests |
| 6 | `sleep` | `SleepResult` | ✅ Add --json flag | ✅ Track duration | ✅ --json tests |
| 7 | `truefalse` | `BoolResult` | ✅ Add --json flag | ✅ Render result | ✅ --json tests |
| 8 | `yes` | `YesResult` | ✅ Add --json + `--count` | ✅ Finite in json | ✅ --json tests |

---

## Verification

```bash
# All unit tests pass
go test ./pkg/echo/... ./pkg/testcmd/... ./pkg/sed/... ./pkg/tee/... \
        ./pkg/tr/... ./pkg/sleep/... ./pkg/truefalse/... ./pkg/yes/... -v

# Daemon integration tests pass
go test ./test/posix-json/... -v

# Full suite zero regressions
go test ./pkg/... ./internal/... -count=1
```
