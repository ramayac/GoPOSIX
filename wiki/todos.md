# GoPOSIX — Open TODOs & Remaining Work

> **Last updated:** 2026-05-16 | **BusyBox:** 526 pass / 5 fail / 10 skip | **Branch:** `feat/post-mvp`

## Current State

| Metric | Value |
|--------|-------|
| Registered utilities | 74 |
| Unit test packages passing | 74/74 (100%) |
| BusyBox tests total | 541 |
| BusyBox passed | 526 |
| BusyBox failed | 5 |
| BusyBox skipped | 10 |
| **BusyBox pass rate** | **97.2%** |
| JSON-RPC daemon coverage | 59/74 (80%) |
| Overall unit coverage | ~72% |

## Active Plans

| Phase | Doc | Status |
|-------|-----|--------|
| Phase 15 | [15_post_mvp_tier1.md](15_post_mvp_tier1.md) — `dd` + `od` | PLANNING |
| Phase 16 | [16_post_mvp_tier2.md](16_post_mvp_tier2.md) — 9 text/stream utilities | ✅ COMPLETE |
| Phase 17 | [17_post_mvp_tier3.md](17_post_mvp_tier3.md) — 12 no-BusyBox utilities | ⚠️ STUBS (functional, need hardening) |
| Phase 18 | [18_quality_fixes.md](18_quality_fixes.md) — CI, patch, coverage, aliases | PLANNING |
| — | [test_coverage_matrix.md](test_coverage_matrix.md) — Full test status for all 74 utilities | LIVING DOC |

## Remaining Failures (5)

| # | Test | Utility | Root Cause | Fixable? |
|---|------|---------|------------|----------|
| 1 | `date-@-works` | date | Go `time` doesn't parse POSIX TZ strings | ❌ Needs custom parser |
| 2 | `date-timezone` | date | Same | ❌ Same |
| 3 | `date-works-1` | date | Error format mismatch (goposix vs BusyBox banner) | ⚠️ Cosmetic |
| 4 | `fold with NULs` | fold | NUL byte handling in word-wrap | ⚠️ Binary data issue |
| 5 | `fold -sw66 with unicode input` | fold | Rune-based word-break + column counting | ⚠️ Needs UTF-8 fix |

## JSON-RPC Daemon Gaps (15 utilities)

These utilities work via CLI but lack daemon integration tests in `test/posix-json/`:

`cmp` `comm` `daemon` `expand` `fold` `nl` `paste` `sed` `shell`
`strings` `sum` `tee` `testcmd` `truefalse` `unexpand`

## Low Unit Coverage (< 60%)

| Utility | Coverage |
|---------|----------|
| `diff` | 54.8% |
| `join` | 49.0% |
| `paste` | 46.2% |
| `shell` | 60.8% |
| `split` | 45.2% |
| `tty` | 54.3% |
| `who` | 54.5% |

## Skipped BusyBox Tests (10)

All 10 are tar tests requiring external compression tools (bzip2, xz, uudecode, pax) or hardlink detection not yet implemented.

## Deferred

| Item | Doc |
|------|-----|
| `awk` implementation (Platinum gate) | [07a_awk.md](07a_awk.md) |
| XML output (`--xml`) | [14_xml_output.md](14_xml_output.md) |
| GoPOSIXOS bootable distro | [prepare_to_goose.md](prepare_to_goose.md) |
