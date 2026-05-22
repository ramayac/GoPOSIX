# GoPOSIX — Open TODOs & Remaining Work

> **Last updated:** 2026-05-22 | **BusyBox:** 679 pass / 20 fail / 22 skip | **Coverage:** 77.9% | **--json:** 81/92 (patch ✅, dd deferred)

## Remaining Failures (20)

### `awk` — 17 failures (goawk v1.31.0 limitations)

| # | Test | Root Cause |
|---|------|------------|
| 1 | `awk bitwise op` | goawk doesn't implement bitwise operators |
| 2 | `awk properly handles undefined function` | goawk parse error on undefined functions |
| 3 | `awk unused function args are evaluated` | goawk evaluation order difference |
| 4 | `awk hex const 1` | goawk doesn't support hex constants |
| 5 | `awk hex const 2` | Same |
| 6 | `awk oct const` | goawk doesn't support octal constants |
| 7 | `awk handles non-existing file correctly` | goawk error handling difference |
| 8 | `awk nested loops with the same variable` | goawk scoping difference |
| 9–12 | `awk func arg parsing 1–4` | goawk function argument parsing |
| 13 | `awk handles empty ()` | goawk empty arg list handling |
| 14 | `awk break` | goawk break statement |
| 15 | `awk continue` | goawk continue statement |
| 16 | `awk negative field access` | goawk negative field access |
| 17 | `awk backslash+newline` | goawk line continuation handling |

### `realpath` — 3 failures (canonical path resolution limits in symlinked workspace)

| # | Test | Root Cause |
|---|------|------------|
| 18 | `realpath on non-existent local file 1` | Path canonicalization behavior difference on non-existent paths |
| 19 | `realpath on link to non-existent file 1` | Path canonicalization behavior difference on non-existent paths |
| 20 | `realpath on link to non-existent file 3` | Path canonicalization behavior difference on non-existent paths |

## Planned & Deferred Work

All active planning phases, deferred architectural enhancements, completed transitions, and engine limitations are consolidated in a single central registry:

👉 **[wiki/deferred.md](deferred.md)**

Refer to that document for full details.

### Alpine Daemon Mode

| # | Item | Status |
|---|------|--------|
| — | Daemon-in-Alpine: `alpine-mvp` image runs CLI-only (shell). Adding daemon mode requires entrypoint change + user setup + BusyBox override decision. | PLANNING — see [alpine_plan.md § Daemon Mode](alpine_plan.md#daemon-mode-in-alpine) |
