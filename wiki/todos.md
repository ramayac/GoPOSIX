# GoPOSIX — Open TODOs & Remaining Work

> **Last updated:** 2026-05-19 | **BusyBox:** 548 pass / 4 fail / 10 skip | **Coverage:** 76.7%

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

## Deferred

See [deferred.md](deferred.md) for the consolidated list. Key items:
- `awk` implementation (Platinum gate) → [07a_awk.md](07a_awk.md)
- XML output (`--xml`)
- Multi-tenant sandbox
- Multi-agent observability → [24_multi_agent_observability.md](24_multi_agent_observability.md)
- `date` TZ parsing (Go `time` package limitations)
- `fold` NUL handling (echo harness limitation)
