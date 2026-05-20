# GoPOSIX — Open TODOs & Remaining Work

> **Last updated:** 2026-05-20 | **BusyBox:** 590 pass / 21 fail / 18 skip | **Coverage:** 76.6% | **--json:** 77/79 (patch ✅, dd deferred)

## Hardening IV — Remaining (3 HIGH)

| # | Item | Doc |
|---|------|-----|
| H1 | `session.setCwd` bypasses `SecurePath` — validate path before storing | [24_hardening_iv.md](24_hardening_iv.md) |
| H4 | Systemic `os.Stderr` hardcoding — 11/79 utilities fixed, ~68 remain | [24_hardening_iv.md](24_hardening_iv.md) |
| H5 | `rm --no-preserve-root` not in flag spec (one-line fix) | [24_hardening_iv.md](24_hardening_iv.md) |

## Phase 25: Daemon Stdin — Resolved ✅

`dispatch.Command.Run` signature expanded to include `stdin io.Reader`. `GoposixParams`
gained a `Stdin` field. All 76 utility `run()` functions and 69 test files updated mechanically.
Daemon now passes stdin through to stdin-consuming utilities (grep, sed, sort, wc, tr, etc.).

| # | Item | Status |
|---|------|--------|
| ✅ | Shell redirect bug: empty `cwd` resolved to `/tutu.txt` instead of CWD | Fixed — `openHandler` falls back to `os.Getwd()` |
| ✅ | `dispatch.Command.Run` signature: `(args, stdin io.Reader, stdout io.Writer)` | Implemented |
| ✅ | `GoposixParams.Stdin` field + daemon plumbing | Implemented |
| ✅ | 76 utility `run()` + 69 test file call sites updated | Complete |
| ✅ | Daemon stdin integration test (`TestDaemonStdinSupport`) | Added |

## Hardening IV — Resolved (25)

All MEDIUM (12) and LOW (8) resolved. HIGH resolved: H2, H3, H6, H7.
See [24_hardening_iv.md](24_hardening_iv.md) for full resolution table.

**Also resolved same session:** `patch --json` — added flag, wired `Render`/`RenderError`.
4 new CLI tests, 78.0% coverage, race-clean.

## Remaining Failures (21)

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

### `date` — 3 failures (Go `time` package limitations)

| # | Test | Root Cause | Fixable? |
|---|------|------------|----------|
| 1 | `date-@-works` | Go `time` doesn't parse POSIX TZ strings | ❌ Custom parser |
| 2 | `date-timezone` | Same | ❌ Same |
| 3 | `date-works-1` | Error format mismatch | ⚠️ Cosmetic |

### `fold` — 1 failure

| # | Test | Root Cause | Fixable? |
|---|------|------------|----------|
| 1 | `fold with NULs` | Echo harness doesn't handle `\0` in `-e` mode | ⚠️ Echo limitation |

## JSON-RPC Daemon Gaps

2 utilities lack explicit daemon integration tests in `test/posix-json/`: `tee`, `tr`.
(`testcmd` and `truefalse` are tested via `runner_test.go`; `daemon` is the daemon itself;
`patch` is tested via BusyBox. `tee` and `tr` are registered and dispatchable but lack
dedicated JSON-RPC sub-tests for their stdin-dependent success paths.)

## Planned

| # | Item | Doc | Status |
|---|------|-----|--------|
| 1 | Daemon pipeline composition (`goposix.pipe` RPC method) | [deferred.md](deferred.md) | 🟡 After stdin land |
| 2 | CWD refactor — eliminate `os.Chdir()` from shell by threading CWD through `dispatch.Command.Run` | [24_hardening_iv.md](24_hardening_iv.md) §H6 | 🟢 Deferred (mutex workaround in place) |

## Deferred

See [deferred.md](deferred.md) for the consolidated list. Key items:
- `dd --json` — manual `key=value` operand parsing makes flag injection non-trivial.
  Needs result struct design + operand parser changes. Estimated ~30–60 min.
- XML output (`--xml`)
- Multi-tenant sandbox
- Multi-agent observability
- `date` TZ parsing (Go `time` package limitations)
- `fold` NUL handling (echo harness limitation)
