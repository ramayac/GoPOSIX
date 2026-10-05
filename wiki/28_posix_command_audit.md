---
status: current
description: "Phase 28: POSIX command audit plan, matrix, and remaining work."
references: [source:wiki/test_coverage_matrix.md, source:test/posix-json]
---

# Phase 28 — POSIX Command Audit (Plan & Matrix)

> **Created:** 2026-10-03 | **Commands:** 115 | **Status:** ✅ all findings F1–F16 and P1–P7 resolved — PR #44 (F8–F14) and PR #46 (5d JSON) merged, PR #47 (F16/P1/P2/P4) in review · remaining work: §6 |
>
> Companion to [wiki/test_coverage_matrix.md](test_coverage_matrix.md) (test status) and [wiki/todos.md](todos.md) (open work).
> This page is the plan and tracking matrix for a one-shot audit of every POSIX command.

---

## 1. Purpose

Audit all 115 registered commands in `pkg/`. For each command, decide a verdict:
**KEEP** (no change), **IMPROVE** (small changes), or **REFACTOR** (structural change).

The audit does not add features. It removes duplication, fixes invariant violations,
and lifts test quality.

## 2. Method

For each command, check these 10 items:

1. **Writer injection** — the `run()` path uses the injected `stdout`/`stderr` writers, not `os.Stdout`/`os.Stderr`.
2. **Flag parsing** — uses `common.ParseFlags` from [pkg/common/flags.go](../pkg/common/flags.go), or a documented manual parser for free-form tools. No stdlib `flag`.
3. **JSON output** — `--json` goes through `common.Render` with the injected writer; schema exists in [wiki/json_schema.md](json_schema.md).
4. **Error handling** — POSIX exit codes, errors to stderr, no `panic` in the run path.
5. **Layers** — library logic separated from CLI glue (the `catRun()` pattern in [pkg/cat/cat.go](../pkg/cat/cat.go)) so the daemon can reuse the core.
6. **Coverage** — unit coverage ≥ 80%, including the CLI glue layer.
7. **BusyBox** — the BusyBox suite passes for the command (`make testsuite`).
8. **Code quality** — no dead code, no duplicated helpers, no needless complexity.
9. **Platform** — no CGO, statically buildable, cross-compile safe.
10. **Daemon** — the command works over JSON-RPC without process-local state assumptions.

## 3. Findings — all resolved

Repository-wide findings from the pre-audit scan and the 5d JSON work. Every one is done.
The short form below is the record; PR numbers point at the full change sets.

| ID | Finding | Result (PR) |
|----|---------|-------------|
| F1 | 53 commands hardcoded `os.Stderr`/`os.Stdout` | Injected writers everywhere; daemon returns separate stdout/stderr buffers (#43) |
| F2 | Digest family duplication (~1,400 LOC) | Shared [pkg/common/digest.go](../pkg/common/digest.go); 5 tools 1,485 → 348 LOC (#43) |
| F3 | Coverage below 80% in 4 packages | All ≥ 80% (#43, #46) |
| F4 | BusyBox failures | 17 `awk` fails deferred (upstream goawk limits) — all others pass (#43) |
| F5 | Signal parsing duplication | `common.ParseSignal` (31-signal table) replaces the local copy (#43) |
| F6 | `panic` in bc startup | `NewInterpreter` returns an error; `mathLibSource` seam (#43) |
| F7 | Compression family overlap (~950 LOC) | Shared [pkg/common/decompress.go](../pkg/common/decompress.go); 4 tools 950 → ~230 LOC (#43) |
| F8 | Parser helpers duplicated across 4 packages | `common.TokenCursor` shared; bc/sed keep local cursors (different grammars) (#44) |
| F9 | `big.Rat` helpers duplicated between bc and dc | `common.RatToInt64`/`RatTruncate` (#44) |
| F10 | Flag errors ignored `--json` in 83 of 93 commands | `common.HasJSONFlag` + `RenderFlagError` at all 93 `ParseFlags` call sites; `factor` exempt (no flag-error path) (#44) |
| F11 | Escape-sequence logic duplicated 3× | `common.ExpandEscapes` (`EscapeFormat`/`EscapeArg`) wraps printf and echo (#44) |
| F12 | 39 commands had no published JSON schema | 37 new schemas + golden fixtures; `dd` and `daemon` documented exemptions; `mkfs.minix` name trap (schema uses dispatch name) (#46) |
| F13 | UID/GID lookup duplicated between chgrp and chown | `common.LookupUID`/`LookupGID` (#44) |
| F14 | Human-size formatting duplicated between ls and du | `common.HumanSize` with a `round` option (ls truncates, du rounds) (#44) |
| F15 | `shell` broke the "`--json` for all" contract | `shell --json` returns `{exitCode, stdout, stderr}` in inline/file/pipe modes; daemon test asserts it; `dd` stays the documented exception (#46) |
| F16 | Raw payload mixed with the envelope on stdout | JSON stdout modes capture the payload (50 MB cap) and embed it as base64 `content` (decompress core, `gzip`, `cpio -o`) (#47) |
| P1 | `logger` package-global `stderrWriter` | `Run` takes the injected `errOut io.Writer`; global deleted (#47) |
| P2 | Matrix coverage data stale | All 115 coverage cells refreshed from a `go test -cover` run; overall 88.1% (#47) |
| P3 | `awk` BusyBox failures | Deferred with reasons (upstream engine limits) (#43) |
| P4 | Concurrent `make testsuite` runs corrupt each other | Harness uses per-run `mktemp -d` link dirs and `.tmpdir.<applet>.$$`; tracked `runtest-tempdir-links/` deleted from the repo; concurrent runs verified (#47) |
| P5 | AGENTS.md duplicated test counts (drifted) | Numbers now live only in test_coverage_matrix.md (#43) |
| P6 | Two wrong BusyBox cells; alias names broke the score | Cells corrected; lookup matches aliases (`true`/`false`, `gzip`/`gunzip`, `test`/`[`) (#43) |
| P7 | 31 pre-existing schemas had no golden fixture | `gen_golden.sh` fixed (`set -u` bug, `%b` escapes, absolute paths) and extended; 114 → 115 fixtures; 0 skips (#46) |

JSON error-path work from 5d (not numbered findings): `wget`, `which`, `seq`, `pidof`, `mdev`,
`rx`, `shell` usage errors now honour `--json`; `rx` no longer writes handshake bytes to stdout
in JSON mode; `who` emits `users: []` instead of `null` (#46).

## 4. Phases & Priorities

| Phase | Scope | Status |
|-------|-------|--------|
| 0 | Plan + matrix snapshot | ✅ DONE |
| 1 | F1 mechanical fix: injected writers (53 commands) | ✅ DONE (#43) |
| 2 | F2 digest consolidation + F5 signal consolidation | ✅ DONE (#43) |
| 3 | F3 coverage drive | ✅ DONE (#43, #46) |
| 4 | Deep audit of XL/L commands (one PR each) | ▶️ 1 of 7 done (`printf` = KEEP ✅); 6 open in §6 |
| 5 | Sweep of M/S commands (batched) | ⏳ 13 open verdicts in §6 |
| 6 | Close all open verdicts | ⏳ matrix rows below + §6 |

## 5. Audit Matrix — remaining rows

All 115 rows have a verdict and notes. The rows below are the only ones with open work
(6 from Phase 4, 13 from Phase 5). The other 96 rows are `KEEP` / `KEEP ✅` with no open
findings; their coverage and BusyBox numbers live in
[wiki/test_coverage_matrix.md](test_coverage_matrix.md).

**Tier by LOC:** XL ≥ 1500 · L 700–1499 · M 350–699 · S < 350 (non-test lines).
**PreAudit** = pre-audit priority score (higher = stronger candidate; methodology in the
2026-10-03 preflight notes, git history of this file).

| Command | Tier | Score | Verdict | Open work |
|---------|:----:|:-----:|:-------:|-----------|
| `bc` | XL | 5.50 | IMPROVE | `eval` 486 lines, `NextToken` 284 lines |
| `sed` | L | 3.75 | IMPROVE | `parseInstruction` 266 lines, `execFlat` 184 lines, coverage 80.1% |
| `date` | L | 3.75 | IMPROVE | `parsePOSIXTZ` 263-line closure nest — lift inner funcs |
| `tar` | L | 3.50 | IMPROVE | `extractArchiveStream` 196 lines, `createArchiveStream` 183 lines |
| `dc` | L | 3.25 | IMPROVE | `evalDC` 569-line god function |
| `diff` | L | 2.25 | IMPROVE | deep audit open (F1+F3 already applied) |
| `grep` | M | 2.75 | REFACTOR | `grepRun` 432-line god function |
| `patch` | M | 3.75 | IMPROVE | `Run` 120 lines |
| `hexdump` | M | 3.50 | IMPROVE | `Run` 202 lines, thin tests |
| `start-stop-daemon` | M | 3.25 | IMPROVE | `run` 237 lines |
| `unzip` | M | 3.00 | IMPROVE | `run` 341 lines (god function) |
| `uudecode` | S | 3.00 | IMPROVE | `run` 282 lines (god function), no BusyBox tests |
| `wget` | S | 2.50 | IMPROVE | `http.Client{}` has no timeout (daemon blocking risk) |
| `xxd` | M | 2.25 | IMPROVE | `reverseStandard`/`reversePlain` repeat the hex-scan loop |
| `sort` | M | 2.00 | IMPROVE | exported `Run` takes unexported types |
| `uuencode` | S | 2.00 | IMPROVE | `run` 167 lines |
| `taskset` | S | 1.75 | IMPROVE | `run` 162 lines |
| `rx` | S | 1.25 | IMPROVE | 1 flaky BusyBox test (handshake race, deferred) |
| `xargs` | S | 1.00 | IMPROVE | `xargsRun` 159 lines, 29 branches |

## 6. What's Left

All numbered findings are resolved. The remaining work is the per-command deep audit:

### Phase 4 — one PR per XL/L command (6 open, ordered by score)

`bc` → `sed` → `date` → `tar` → `dc` → `diff` (`printf` done).

### Phase 5 — M/S sweep (13 open verdicts)

`grep` (REFACTOR) plus `patch`, `hexdump`, `start-stop-daemon`, `unzip`, `uudecode`,
`wget`, `xxd`, `sort`, `uuencode`, `taskset`, `rx`, `xargs`.

### Housekeeping

- ✅ README links the Phase 28 plan.
- ✅ `wiki/test_coverage_matrix.md` refreshed (P2).
- Phase 6: close the 19 verdicts above.

## 7. Definition of Done

- Every row has a verdict and notes. ✅ (115/115)
- No row shows `REFACTOR` or `IMPROVE` for a finding that is already complete.
- Findings F1–F16 and P1–P7 are resolved or explicitly deferred with reasons. ✅
- Commands with `PreAudit = 0` (100% coverage and a passing BusyBox suite) need no change.
- The `PreAudit` score and the matrix data match the current tree.
- `make test`, `make testsuite`, `go vet`, `go fmt` all pass. ✅ (per PR)
- Overall coverage stays ≥ 80% (CI gate). ✅ (88.1%)
- [wiki/test_coverage_matrix.md](test_coverage_matrix.md) and [wiki/todos.md](todos.md) updated. ✅
