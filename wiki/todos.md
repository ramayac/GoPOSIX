# GoPOSIX — Open TODOs & Remaining Work

> **Last updated:** 2026-05-20 | **BusyBox:** 548 pass / 4 fail / 10 skip | **Coverage:** 76.7%

## Hardening IV — Remaining (3 HIGH)

| # | Item | Doc |
|---|------|-----|
| H1 | `session.setCwd` bypasses `SecurePath` — validate path before storing | [24_hardening_iv.md](24_hardening_iv.md) |
| H4 | Systemic `os.Stderr` hardcoding — 11/79 utilities fixed, ~68 remain | [24_hardening_iv.md](24_hardening_iv.md) |
| H5 | `rm --no-preserve-root` not in flag spec (one-line fix) | [24_hardening_iv.md](24_hardening_iv.md) |

## Hardening IV — Resolved (24)

All MEDIUM (12) and LOW (8) resolved. HIGH resolved: H2, H3, H6, H7.
See [24_hardening_iv.md](24_hardening_iv.md) for full resolution table.

## Remaining Failures (4)

| # | Test | Utility | Root Cause | Fixable? |
|---|------|---------|------------|----------|
| 1 | `date-@-works` | date | Go `time` doesn't parse POSIX TZ strings | ❌ Custom parser |
| 2 | `date-timezone` | date | Same | ❌ Same |
| 3 | `date-works-1` | date | Error format mismatch | ⚠️ Cosmetic |
| 4 | `fold with NULs` | fold | Echo harness doesn't handle `\0` in `-e` mode | ⚠️ Echo limitation |

## JSON-RPC Daemon Gaps

2 utilities lack explicit daemon integration tests in `test/posix-json/`: `tee`, `tr`.
(`testcmd` and `truefalse` are tested via `runner_test.go`; `daemon` is the daemon itself;
`patch` is tested via BusyBox. `tee` and `tr` are registered and dispatchable but lack
dedicated JSON-RPC sub-tests for their stdin-dependent success paths.)

## Planned

| # | Item | Doc | Status |
|---|------|-----|--------|
| 1 | Daemon stdin support (40+ stdin-consuming utilities unreachable via SDK) | [deferred.md](deferred.md) | 🔴 ACTIVE |
| 2 | Daemon pipeline composition (`goposix.pipe` RPC method) | [deferred.md](deferred.md) | 🟡 After stdin land |
| 3 | CWD refactor — eliminate `os.Chdir()` from shell by threading CWD through `dispatch.Command.Run` | [24_hardening_iv.md](24_hardening_iv.md) §H6 | 🟢 Deferred (mutex workaround in place) |

## Deferred

See [deferred.md](deferred.md) for the consolidated list. Key items:
- XML output (`--xml`)
- Multi-tenant sandbox
- Multi-agent observability
- `date` TZ parsing (Go `time` package limitations)
- `fold` NUL handling (echo harness limitation)
