# GoPOSIX — Open TODOs & Remaining Work

> **Last updated:** 2026-05-21 | **BusyBox:** 596 pass / 19 fail / 18 skip | **Coverage:** 76.6% | **--json:** 77/79 (patch ✅, dd deferred)

## Hardening IV — Remaining (0) - ALL RESOLVED ✅

All 27 architecture, security, and compliance gaps under Hardening IV have been fully resolved.

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

## Hardening IV — Resolved (27)

All 7 HIGH, 12 MEDIUM, and 8 LOW gaps are fully resolved. HIGH resolved: H1, H2, H3, H4, H5, H6, H7.
See [24_hardening_iv.md](24_hardening_iv.md) for full details.

**Also resolved same session:** `patch --json` — added flag, wired `Render`/`RenderError`.
4 new CLI tests, 78.0% coverage, race-clean.

## Remaining Failures (19)

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

### `date` — Resolved ✅

All `date` compliance failures are fully resolved by our custom POSIX `TZ` environment parser/evaluator and ordered error logging.

### `fold` — Resolved ✅

The `fold with NULs` and trailing newline compliance failures are fully resolved by introducing byte-splitting stream parsing.

## JSON-RPC Daemon Gaps

2 utilities lack explicit daemon integration tests in `test/posix-json/`: `tee`, `tr`.
(`testcmd` and `truefalse` are tested via `runner_test.go`; `daemon` is the daemon itself;
`patch` is tested via BusyBox. `tee` and `tr` are registered and dispatchable but lack
dedicated JSON-RPC sub-tests for their stdin-dependent success paths.)

## Planned & Deferred Work

All active planning phases, deferred architectural enhancements, completed transitions, and engine limitations are consolidated in a single central registry:

👉 **[wiki/deferred.md](deferred.md)**

Refer to that document for full details on:
* **Active Planning & Future Phases**: Phase 24 (Multi-Agent Observability), Phase 26 (Daemon Pipeline Composition).
* **Deferred Architectural Enhancements**: CWD Signature Refactoring (threading `CWD` through `Run()`), `dd --json` Output support, XML Output (`--xml`), Multi-Tenant Sandboxing, Smart CLI Forwarding (M5), StopAtFirstNonFlag Integration (`echo`/`printf`), CGroups v2 Per-Session Isolation.
* **Documented Limitations**: Go `regexp` vs POSIX BRE/ERE.
