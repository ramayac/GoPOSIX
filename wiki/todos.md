# GoPOSIX — Open TODOs & Remaining Work

> **Last updated:** 2026-05-18 | **BusyBox:** 548 pass / 4 fail / 10 skip | **Coverage:** 75.1% | **Branch:** `main`

## Current State

| Metric | Value |
|--------|-------|
| Registered utilities | 77 |
| Unit test packages passing | 85/85 (100%) |
| BusyBox tests total | 541 |
| BusyBox passed | 548 |
| BusyBox failed | 4 |
| BusyBox skipped | 10 |
| **BusyBox pass rate** | **99.3%** |
| Overall unit coverage | 75.1% |

## Completed (all phases 00–18)

All 18 planned phases are complete. 77 utilities, 548/541 BusyBox tests, 85 test packages.

## Remaining Failures (4)

| # | Test | Utility | Root Cause | Fixable? |
|---|------|---------|------------|----------|
| 1 | `date-@-works` | date | Go `time` doesn't parse POSIX TZ strings | ❌ Custom parser |
| 2 | `date-timezone` | date | Same | ❌ Same |
| 3 | `date-works-1` | date | Error format mismatch | ⚠️ Cosmetic |
| 4 | `fold with NULs` | fold | Echo harness doesn't handle `\0` in `-e` mode | ⚠️ Echo limitation |

> **fold Unicode fixed** — `fold -sw66 with unicode input` now passes (rune-based column counting).

## Pending Work

### Low Unit Coverage (< 60%)

All packages now exceed 60%. The floor was raised on 2026-05-18:

| Utility | Before | After | Notes |
|---------|:------:|:-----:|-------|
| `split` | 60.3% | **86.3%** | CLI `run()` now tested with CWD-sensitive setup |
| `tty` | 54.3% | **60.0%** | Added `Run()` and combined flag tests; `ttyname()` remains CI-untestable |

### JSON-RPC Daemon Gaps

2 utilities lack explicit daemon integration tests in `test/posix-json/`:

`tee` `tr`

(`testcmd` and `truefalse` are tested in `runner_test.go`; `daemon` is the daemon itself;
`patch` is tested via BusyBox. `tee` and `tr` are registered and dispatchable but
lack dedicated JSON-RPC sub-tests for their stdin-dependent success paths.)

### Deferred

| Item | Doc |
|------|-----|
| `awk` implementation (Platinum gate) | [07a_awk.md](07a_awk.md) |
| XML output (`--xml`) | [14_xml_output.md](14_xml_output.md) |
| `date` TZ parsing | Go `time` package limitations |
| `fold` NUL handling | Echo harness limitation |
