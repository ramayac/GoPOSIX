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

## JSON-RPC Daemon Gaps — Resolved ✅

All daemon integration tests for `tee` and `tr` have been implemented and verified under `test/posix-json/tier2_text_test.go` (on branch `feat/hardening-part3`).

## Planned & Deferred Work

All active planning phases, deferred architectural enhancements, completed transitions, and engine limitations are consolidated in a single central registry:

👉 **[wiki/deferred.md](deferred.md)**

Refer to that document for full details.

## 🚀 CWD Signature Refactoring (Global State Elimination) — Resolved ✅

Standardizing all 79 utility `Run` signatures to accept a context-relative `cwd string` parameter and achieving lock-free concurrent execution safety.

### Phase 1: Core Registry & Execution Layers
- `[x]` Refactor `Command` in `internal/dispatch/dispatch.go` to unified signature.
- `[x]` Refactor `goposix.go` to retrieve and forward process physical CWD.
- `[x]` Refactor `internal/daemon/server.go` to retrieve and forward session CWD.
- `[x]` Refactor `internal/shell/interpreter.go` to eliminate `execMu` / `chdirMu` and forward interpreter dir.

### Phase 2: Batch Refactoring of 79 Utilities

#### Batch A (Structural Foundation)
- `[x]` `pwd`
- `[x]` `basename`
- `[x]` `dirname`
- `[x]` `truefalse`
- `[x]` `testcmd`
- `[x]` `tty`
- `[x]` `hostname`
- `[x]` `id`
- `[x]` `whoami`
- `[x]` `logname`
- `[x]` `uname`

#### Batch B (Standard I/O Streams)
- `[x]` `echo`
- `[x]` `printf`
- `[x]` `yes`
- `[x]` `sleep`
- `[x]` `env`
- `[x]` `printenv`
- `[x]` `nice`
- `[x]` `nohup`
- `[x]` `kill`
- `[x]` `ps`

#### Batch C (Basic Text Parsers)
- `[x]` `cat`
- `[x]` `head`
- `[x]` `tail`
- `[x]` `wc`
- `[x]` `tee`
- `[x]` `tr`
- `[x]` `fold`
- `[x]` `expand`
- `[x]` `unexpand`
- `[x]` `strings`
- `[x]` `uniq`

#### Batch D (Advanced Text Parsers)
- `[x]` `grep`
- `[x]` `sed`
- `[x]` `awk`
- `[x]` `cut`
- `[x]` `paste`
- `[x]` `join`
- `[x]` `sort`
- `[x]` `split`
- `[x]` `diff`
- `[x]` `cmp`
- `[x]` `comm`

#### Batch E (Filesystem Operations)
- `[x]` `ls`
- `[x]` `stat`
- `[x]` `chmod`
- `[x]` `chown`
- `[x]` `chgrp`
- `[x]` `mkdir`
- `[x]` `rmdir`
- `[x]` `touch`
- `[x]` `rm`
- `[x]` `mv`
- `[x]` `cp`
- `[x]` `ln`
- `[x]` `link`
- `[x]` `unlink`
- `[x]` `readlink`

#### Batch F (Archival & Hashing)
- `[x]` `tar`
- `[x]` `gzip`
- `[x]` `dd`
- `[x]` `cksum`
- `[x]` `sum`
- `[x]` `md5sum`
- `[x]` `sha256sum`
- `[x]` `xargs`

#### Batch G (System & Sandbox)
- `[x]` `df`
- `[x]` `du`
- `[x]` `shell`
- `[x]` `patch`
- `[x]` `daemon`
- `[x]` `date`
- `[x]` `mkfifo`
