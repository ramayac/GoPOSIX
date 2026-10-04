# Phase 28 — POSIX Command Audit (Plan & Matrix)

> **Created:** 2026-10-03 | **Branch:** `audit/posix-commands` | **Commands:** 115 | **Status:** PHASE 4 (deep audit of XL/L commands) — F8–F14 shipped in PR #44, 5d JSON work shipped in PR #46 (`audit/5d-json`) · remaining work: §6 |
> **Preflight:** 2026-10-03 — matrix refreshed from the tree, `PreAudit` score added (see §5a). Corrected XL/L scope: 7 commands, not 24.
>
> Companion to [wiki/test_coverage_matrix.md](test_coverage_matrix.md) (test status) and [wiki/todos.md](todos.md) (open work).
> This page is the plan and tracking matrix for a one-shot audit of every POSIX command.

---

## 1. Purpose

Audit all 115 registered commands in `pkg/`. For each command, evaluate if the code can be
improved or refactored. Decide a verdict: **KEEP** (no change), **IMPROVE** (small changes),
or **REFACTOR** (structural change).

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

## 3. Repo-Wide Findings (pre-audit)

These findings apply to many commands at once. They were found by scanning the whole tree before
the per-command audit:

### F1 — 53 commands hardcode `os.Stderr`/`os.Stdout` ✅ DONE

53 of 115 command packages write errors directly to `os.Stderr` (or output to `os.Stdout`)
inside the `run()` function instead of the injected writer. This breaks the **Standardized
Output** invariant in [AGENTS.md](../AGENTS.md) and makes the CLI glue layer untestable. Fix is
mechanical: use `common.RenderError` / the injected writers. Pattern already exists in
[pkg/cat/cat.go](../pkg/cat/cat.go).

**Affected (53):** `basename`, `chgrp`, `chmod`, `chown`, `cksum`, `cp`, `daemon`, `date`, `df`, `diff`, `dirname`, `du`, `env`, `expr`, `hostname`, `join`, `kill`, `link`, `ln`, `logger`, `logname`, `md5sum`, `mkdir`, `mkfifo`, `mv`, `nice`, `nohup`, `od`, `printenv`, `printf`, `ps`, `pwd`, `readlink`, `rev`, `rm`, `rmdir`, `sed`, `sha1sum`, `sha256sum`, `sha512sum`, `sleep`, `split`, `stat`, `tail`, `tar`, `testcmd`, `touch`, `truefalse`, `uname`, `unlink`, `who`, `whoami`, `yes`.

**Resolution:** all error messages now go through the injected `stderr` writer. Special cases:
`nice`/`nohup` thread the writers into their `Run()` library functions (subprocess stdio);
`od` gained an `errOut` parameter; `tar` gained `errOut` on `createArchiveStream`; `sed`'s test
helper passes `io.Discard`. The `logger` package keeps its `stderrWriter` seam (already
swapped by `run()`). The daemon now passes **separate** stdout/stderr buffers and returns
a `stderr` field in JSON-RPC responses (see [rpc_api.md](rpc_api.md)).

### F2 — Digest family duplication ✅ DONE

`md5sum`, `sha1sum`, `sha256sum`, `sha512sum`, `sha3sum`, `cksum`, `sum` are near-copies.
Each repeats file walking, check-mode (`-c`) verification, and CLI parsing. Total ~1,400 LOC
with the same structure. Candidate for a shared digest core in `pkg/common`.

**Resolution:** new [pkg/common/digest.go](../pkg/common/digest.go) provides `DigestHashMode`,
`DigestCheckMode`, and `DigestReader`. `md5sum`, `sha1sum`, `sha256sum`, `sha512sum`, and
`sha3sum` are now thin wrappers (1,485 → 348 LOC, −77%). `sha3sum` keeps its `-a` algorithm
flag and per-line autodetection via `CheckResolveAlg`. Per-package behaviors preserved
(md5sum/sha256sum error on `-c` without files; sha1sum/sha512sum/sha3sum read stdin).
`cksum`/`sum` were left as-is: different formats (CRC), no check mode, already compliant.

### F3 — Coverage below 80% ✅ DONE

Four packages are below the per-package target: `diff` (73.9%), `gzip` (72.7%), `hostname` (78.2%),
`whoami` (78.9%). The project gate (80% overall) still passes.

**Resolution:** `whoami` 100.0% and `hostname` 98.2% via function seams (`userCurrent`,
`osHostname`, `netLookupHost`, `netLookupAddr` — the PR #42 `pkg/pwd` pattern).
`diff` 89.5%: dir-vs-file, stdin, `-q`/`-u` formats, non-regular files, `-N` missing-file
paths, hunk trimming. `gzip` 87.3%: stdin/dash modes, create/process failures, garbage
input. `pkg/common` also lifted 68.9% → 93.7% with direct digest-core tests.

### F4 — BusyBox failures ✅ ACKNOWLEDGED (awk deferred)

`awk` has 16 failing tests (upstream goawk engine limits, deferred — see [wiki/deferred.md](deferred.md)).
`rx` has 1 flaky test (handshake race). All other ⚠️ entries are root-required skips.

### F5 — Signal parsing duplication ✅ DONE

`start-stop-daemon` kept a partial `parseSignal` (7 names). Now deleted: the call site uses
`common.ParseSignal` from [pkg/common/signal.go](../pkg/common/signal.go) — full 31-signal Linux table,
SIG-prefix tolerance, case-insensitivity, whitespace trimming, numeric parsing. Error message
format unchanged. Tests extended with signals the old parser rejected (CONT, STOP, PWR, SYS,
WINCH, SIGPIPE, lowercase, padded numbers).

### F6 — `panic` in bc startup ✅ DONE

`NewInterpreter` now returns `(*Interpreter, error)` instead of calling `panic` twice.
`Run` propagates the error. `bcRun` already renders it (`RenderError` in JSON mode).
A `mathLibSource` seam lets tests break the embedded library. Coverage 83.6%.

### F7 — Compression family overlap ✅ DONE

New [pkg/common/decompress.go](../pkg/common/decompress.go) provides `DecompressMode`.
`unlzma`, `bunzip2`, `uncompress`, `bzcat` are thin wrappers: 950 → ~230 LOC (−76%).
Per-tool behaviors preserved: suffix tables, `-c/-f/-k/-q` flags, quiet mode, corrupt-data
messages, `dcompress` panic recovery, log suppression, cat mode. CLI output parity verified
against the pre-refactor binary. All four packages now at 100% coverage.
`gzip` stays separate: dual compress/decompress mode, different logic.

## 4. Phases & Priorities

> **Current step: Phase 4** — deep audit of the XL/L commands, one PR each.
> The preflight refresh (§5a) found 7 XL/L commands under the stated non-test LOC definition:
> `bc`, `sed`, `printf`, `date`, `tar`, `dc`, `diff`. The previous count of 24 used total
> lines (code + test). Order the work by `PreAudit` score in §5.

| Phase | Scope | Commands | Verify | Status |
|-------|-------|----------|--------|--------|
| 0 | This plan + matrix snapshot | — | committed on `audit/posix-commands` | ✅ DONE |
| 1 | F1 mechanical fix: injected writers | 53 | `make test` + `make testsuite` + `go vet` | ✅ DONE (870/17/30) |
| 2 | F2 digest consolidation + F5 signal consolidation | 8 | BusyBox suite + output parity | ✅ DONE (F2 + F5) |
| 3 | F3 coverage drive | 4 | `make cover-pkg` ≥ 80% | ✅ DONE (whoami 100, hostname 98.2, diff 89.5, gzip 87.3) |
| 4 | Deep audit of XL/L commands (one PR each) | 7 | per-command checklist + suites | ▶️ IN PROGRESS — F8–F11, F13, F14 done in PR #44 (`audit/f8-f11-json-parser-core`); F12, F15, and the JSON-RPC daemon-test gap done in PR #46 (`audit/5d-json`); per-command deep audit (god functions) still open |
| 5 | Sweep of M/S commands (batched) | 87 | per-command checklist + suites | ⏳ pending |
| 6 | Decide F6/F7 and close all open verdicts | — | matrix 100% filled | ⚠️ F6+F7 done; matrix still filling |

## 5. Audit Matrix

**Tier by LOC:** XL ≥ 1500 · L 700–1499 · M 350–699 · S < 350 (non-test lines)

**Columns:** LOC = non-test lines. T-LOC = test lines. Cov = unit coverage. BB = BusyBox status
(from [wiki/test_coverage_matrix.md](test_coverage_matrix.md)). IO = F1 hardcoded I/O.
**PreAudit** = pre-audit priority score (§5a); higher means a stronger candidate.

**Verdict values:** `KEEP` = audited, no change needed. `KEEP ✅` = a prior finding was completed
and needs no more work. `IMPROVE` = small changes still open. `REFACTOR` = structural change still
open. Blank = not audited yet. The verdict states the **decision that is still open**. A completed
finding therefore moves the verdict to `KEEP ✅`. The `Notes` column keeps the finding reference.
For example, `md5sum` was refactored under F2 (completed), so its verdict is `KEEP ✅`, not
`REFACTOR`.

> **Preflight refresh (2026-10-03):** every LOC, T-LOC, coverage, and tier value below is measured
> from the current tree. The earlier snapshot was stale: 110 of 115 rows drifted by more than 25%,
> and the old `LOC` column actually held total lines (code + test) although the header claimed
> non-test lines. Tiers now follow the stated definition on non-test lines, so the XL/L set shrank
> from 24 to 7 commands (`bc`, `sed`, `printf`, `date`, `tar`, `dc`, `diff`).

| Command | Tier | LOC | T-LOC | Cov | BB | IO | PreAudit | Verdict | Notes |
|---------|:----:|----:|------:|----:|:--:|:--:|:--------:|:-------:|-------|
| `bc` | XL | 2587 | 590 | 83.7% | ✅ 81/81 | — | 5.50 | IMPROVE | F6 ✅ NewInterpreter returns error (no panic); mathLibSource seam · F8 ✅ `peek`/`next` cursor shared with `testcmd` (token cursor); F9 ✅ `RatToInt64`/`RatTruncate` moved to `pkg/common/rat.go` · F10 ✅ flag errors honour `--json` · F12 ✅ schema + fixture + daemon test (tier9) · Preflight open: `eval` 486 lines, `NextToken` 284 lines |
| `sed` | L | 1181 | 1060 | 80.1% | ✅ 103/103 | — | 3.75 | IMPROVE | Preflight: `parseInstruction` 266 lines, `execFlat` 184 lines · F10 ✅ flag errors honour `--json` · coverage 80.1% is the lowest of the large tools |
| `printf` | L | 749 | 744 | 83.7% | ✅ 26/26 | — | 3.75 | KEEP ✅ | F11 ✅ `processEscapes`/`processEscapesForB` now wrap `common.ExpandEscapes` · F10 ✅ flag errors honour `--json` · Preflight: no other open items |
| `date` | L | 746 | 661 | 83.8% | ✅ 7/7 | — | 3.75 | IMPROVE | Preflight: `parsePOSIXTZ` is a 263-line closure nest · lift the inner funcs for testability · F10 ✅ flag errors honour `--json` |
| `patch` | M | 600 | 381 | 82.1% | ✅ 11/11 | — | 3.75 | IMPROVE | Preflight: `Run` 120 lines · F10 ✅ flag errors honour `--json` |
| `mount` | S | 317 | 178 | 80.6% | ⚠️ 0/1 (1 skip) | — | 3.75 | KEEP ✅ | Preflight: small and clean, `mountRun` is injectable · all BusyBox cases need root · F12 ✅ schema + fixture + daemon test (tier9) |
| `tar` | L | 1114 | 1537 | 82.3% | ✅ 31/31 | — | 3.50 | IMPROVE | Preflight: `extractArchiveStream` 196 lines, `createArchiveStream` 183 lines · F10 ✅ flag errors honour `--json` · BusyBox 33/33 isolated |
| `hexdump` | M | 562 | 236 | 84.7% | ✅ 3/3 | — | 3.50 | IMPROVE | Preflight: `Run` 202 lines · thin tests (T-LOC/LOC 0.42) · F10 ✅ flag errors honour `--json` · F12 ✅ schema + fixture + daemon test (tier9) |
| `dc` | L | 1149 | 628 | 89.0% | ✅ 36/36 | — | 3.25 | IMPROVE | Preflight: `evalDC` is a 569-line god function · F9 ✅ `RatToInt64`/`RatTruncate` moved to `pkg/common/rat.go` · F10 ✅ flag errors honour `--json` · F12 ✅ schema + fixture (daemon test present) · BusyBox 36/36 |
| `start-stop-daemon` | M | 366 | 242 | 80.7% | ✅ 4/4 | — | 3.25 | IMPROVE | F5 ✅ · F12 ✅ schema + fixture (daemon test present) · Preflight: `run` 237 lines · BusyBox 4/4 |
| `unzip` | M | 458 | 403 | 80.5% | ✅ 4/4 | — | 3.00 | IMPROVE | Preflight: `run` 341 lines (god function) · F12 ✅ schema + fixture (daemon test present) |
| `cpio` | M | 456 | 449 | 82.0% | ✅ 2/9 (7 skip) | — | 3.00 | KEEP ✅ | Preflight: balanced helpers, no duplication found · 7 of 9 BusyBox cases are root skips · F12 ✅ schema + fixture + daemon test · F16 ⏳ `-o --json` mixes the archive stream with the envelope on stdout |
| `uudecode` | S | 323 | 217 | 84.6% | — | — | 3.00 | IMPROVE | Preflight: `run` 282 lines (god function) · no BusyBox tests · F12 ✅ schema + fixture (daemon test present) |
| `tsort` | S | 145 | 73 | 84.3% | ✅ 20/20 | — | 3.00 | KEEP ✅ | Preflight: thin tests · F10 ✅ flag errors honour `--json` · F12 ✅ schema + fixture + daemon test |
| `grep` | M | 648 | 839 | 84.8% | ✅ 53/53 | — | 2.75 | REFACTOR | Preflight: `grepRun` is a 432-line god function (flags + pattern compile + traversal + output) · F10 ✅ flag errors honour `--json` |
| `ar` | M | 484 | 516 | 81.6% | ✅ 2/2 | — | 2.75 | KEEP ✅ | Preflight: `arRun` 88 lines, balanced · F10 ✅ flag errors honour `--json` · F12 ✅ schema + fixture (daemon test present) |
| `makedevs` | S | 335 | 156 | 87.3% | ⚠️ 0/1 (1 skip) | — | 2.75 | KEEP ✅ | F12 ✅ schema + fixture + daemon test (tier9) · BusyBox case needs root |
| `mdev` | S | 309 | 146 | 87.4% | ⚠️ 0/12 (12 skip) | — | 2.75 | KEEP ✅ | F12 ✅ schema + fixture + daemon test (tier9) · 5d.1 ✅ usage error honours `--json` · BusyBox cases need root |
| `nohup` | S | 121 | 110 | 80.9% | — | — | 2.75 | KEEP | Preflight: clean; `Run` library function and `terminalCheck` seam · schema and daemon test present |
| `logger` | S | 219 | 311 | 98.6% | — | ⚠ global | 2.50 | IMPROVE | Preflight: package-global `stderrWriter` (daemon re-entrancy risk); replace it with an injected writer (P1) · schema and daemon test present |
| `wget` | S | 191 | 172 | 81.4% | ✅ 4/4 | — | 2.50 | IMPROVE | Preflight: `http.Client{}` has no timeout, so it can block the daemon · F12 ✅ schema + fixture + daemon test (tier9) · 5d.1 ✅ missing-URL error honours `--json` |
| `diff` | L | 836 | 824 | 90.3% | ✅ 20/20 | — | 2.25 | IMPROVE | F1+F3: tests 89.5% · Preflight: LOC corrected; now L tier, not XL. F1+F3 applied; deep audit open. |
| `xxd` | M | 371 | 239 | 86.4% | ✅ 7/7 | — | 2.25 | IMPROVE | Preflight: `reverseStandard` (99) and `reversePlain` (85) repeat the hex-scan loop · F12 ✅ schema + fixture + daemon test (tier9) |
| `mkfs_minix` | S | 312 | 121 | 86.4% | ✅ 1/1 | — | 2.25 | KEEP ✅ | Preflight: `Run` 177 lines (observation for Phase 5) · F10 ✅ flag errors honour `--json` · F12 ✅ schema (`mkfs.minix.schema.json`, dispatch name) + fixture + daemon test (tier9) |
| `unexpand` | S | 257 | 287 | 82.8% | ✅ 24/24 | — | 2.25 | KEEP | Preflight: clean; its `Transform` is the inverse of `expand`'s, not a copy · schema and daemon test present |
| `cryptpw` | S | 241 | 335 | 82.4% | ✅ 7/7 | — | 2.25 | KEEP ✅ | F12 ✅ schema + fixture (daemon test present) · flag errors honour `--json` |
| `readlink` | S | 208 | 251 | 81.2% | ✅ 6/6 | — | 2.25 | KEEP | Preflight: clean; `evalSymlinksUnder` 78 lines · schema and daemon test present |
| `cmp` | S | 193 | 286 | 82.3% | ✅ 1/1 | — | 2.25 | KEEP | Preflight: clean; `Compare` differs from `comm`'s (bytes vs sorted merge) · schema and daemon test present |
| `expand` | S | 122 | 149 | 81.4% | ✅ 3/3 | — | 2.25 | KEEP | Preflight: clean, 122 lines · schema and daemon test present |
| `sort` | M | 653 | 537 | 86.2% | ✅ 27/27 | — | 2.00 | IMPROVE | Preflight: exported `Run` takes unexported types (`lineItem`, `keySpec`), so it is unusable as an API · schema and daemon test present |
| `od` | M | 522 | 443 | 85.3% | ✅ 4/4 | — | 2.00 | KEEP | Preflight: balanced; `Run` 65 lines · schema and daemon test present |
| `expr` | M | 446 | 397 | 87.1% | ✅ 2/2 | — | 2.00 | KEEP ✅ | F8 ✅ `peek`/`next`/`done` now use `common.TokenCursor` · F10 ✅ flag errors honour `--json` · schema and daemon test present |
| `ls` | M | 443 | 377 | 88.6% | ✅ 5/5 | — | 2.00 | KEEP ✅ | F14 ✅ `common.HumanSize` (truncated) · schema and daemon test present |
| `cal` | S | 348 | 144 | 85.8% | ✅ 1/1 | — | 2.00 | KEEP ✅ | F12 ✅ schema + fixture (daemon test present) · `RenderMonth` 118 lines (observation for Phase 5) |
| `uuencode` | S | 251 | 134 | 88.3% | ✅ 19/19 | — | 2.00 | IMPROVE | Preflight: `run` 167 lines · F12 ✅ schema + fixture (daemon test present) · flag errors honour `--json` |
| `dd` | S | 332 | 208 | 88.8% | ✅ 6/6 | — | 1.75 | KEEP ✅ | F15 ✅ documented exception: no `--json` output (see [wiki/json_schema.md](json_schema.md)) · `Run` 212 lines (observation for Phase 5) · daemon test present |
| `taskset` | S | 285 | 191 | 86.4% | ✅ 3/3 | — | 1.75 | IMPROVE | Preflight: `run` 162 lines · F12 ✅ schema + fixture (daemon test present) |
| `uname` | S | 152 | 69 | 93.0% | — | — | 1.75 | KEEP | Preflight: clean; platform-split `Run()` in uname_linux.go/uname_darwin.go · schema and daemon test present |
| `which` | S | 126 | 96 | 86.0% | ✅ 1/1 | — | 1.75 | KEEP ✅ | F12 ✅ schema + fixture + daemon test · 5d.1 ✅ missing-argument error honours `--json` · small and clean otherwise |
| `daemon` | S | 54 | 86 | 94.1% | — | — | 1.75 | KEEP ✅ | Preflight: daemon control command · F15 ✅ documented exemption: `--json` and JSON-RPC do not apply |
| `testcmd` | M | 410 | 625 | 92.8% | — | — | 1.50 | KEEP ✅ | F8 ✅ `peek`/`next`/`done` now use `common.TokenCursor` · F10 ✅ flag errors honour `--json` · `--json` works and the daemon test asserts the bool result; schema is `test.schema.json` |
| `factor` | S | 346 | 165 | 93.9% | ✅ 13/13 | — | 1.50 | KEEP ✅ | F12 ✅ schema + fixture (daemon test present) · F10 ✅ its `PreProcess` turns unknown flags into positional args, so no flag-error path exists |
| `wc` | S | 311 | 274 | 88.6% | ✅ 5/5 | — | 1.50 | KEEP | Preflight: clean; `CountProper` 93 lines · schema and daemon test present |
| `join` | S | 304 | 314 | 89.7% | — | — | 1.50 | KEEP | Preflight: clean · schema and daemon test present |
| `seq` | S | 284 | 263 | 89.7% | ✅ 21/21 | — | 1.50 | KEEP ✅ | F12 ✅ schema + fixture (daemon test present) · 5d.1 ✅ usage errors honour `--json` · `run` 133 lines (observation for Phase 5) |
| `tr` | S | 265 | 148 | 90.8% | ✅ 6/6 | — | 1.50 | KEEP | Preflight: clean · schema and daemon test present |
| `who` | S | 193 | 400 | 89.4% | — | — | 1.50 | KEEP | Preflight: clean · schema and daemon test present |
| `chown` | S | 109 | 66 | 97.4% | — | — | 1.50 | KEEP ✅ | F13 ✅ `common.LookupUID`/`common.LookupGID` · schema and daemon test present |
| `nice` | S | 107 | 85 | 90.5% | — | — | 1.50 | KEEP | Preflight: clean · schema and daemon test present |
| `df` | S | 74 | 48 | 95.8% | — | — | 1.50 | KEEP | Preflight: clean · schema and daemon test present |
| `gzip` | S | 306 | 575 | 87.3% | ✅ 4/4 | — | 1.25 | KEEP ✅ | F3: tests 87.3% |
| `find` | S | 251 | 293 | 89.8% | ✅ 13/13 | — | 1.25 | KEEP | Preflight: clean; keeps the documented `-exec` argument pre-processing · schema and daemon test present |
| `comm` | S | 245 | 346 | 88.8% | ✅ 9/9 | — | 1.25 | KEEP | Preflight: clean · schema and daemon test present |
| `tail` | S | 228 | 305 | 88.7% | ✅ 3/3 | — | 1.25 | KEEP | Preflight: clean; keeps the `-N` to `-n N` argument pre-processing · schema and daemon test present |
| `paste` | S | 227 | 240 | 88.5% | ✅ 5/5 | — | 1.25 | KEEP | Preflight: clean · schema and daemon test present |
| `cat` | S | 220 | 464 | 89.6% | ✅ 1/1 | — | 1.25 | KEEP | Preflight: clean; canonical `catRun()` injectable entry · schema and daemon test present |
| `awk` | S | 208 | 558 | 90.0% | ⚠️ 36/53 (17 fail, deferred) | — | 1.25 | KEEP | Preflight: the 17 BusyBox fails are deferred goawk engine limits · schema and daemon test present |
| `uniq` | S | 193 | 239 | 88.4% | ✅ 15/15 | — | 1.25 | KEEP | Preflight: clean · schema and daemon test present |
| `rx` | S | 182 | 279 | 86.2% | ✅ 1/1 | — | 1.25 | IMPROVE | F12 ✅ schema + fixture + daemon test (tier9) · 5d.1 ✅ error path honours `--json`, handshake bytes no longer pollute stdout in JSON mode · 1 flaky BusyBox test (handshake race, deferred) |
| `rm` | S | 152 | 198 | 87.3% | ✅ 1/1 | — | 1.25 | KEEP | Preflight: clean; root protection present · schema and daemon test present |
| `pidof` | S | 148 | 114 | 96.7% | ✅ 4/4 | — | 1.25 | KEEP ✅ | F12 ✅ schema + fixture (daemon test present) · 5d.1 ✅ missing-operand error honours `--json` · `run` 112 lines (observation for Phase 5) |
| `uptime` | S | 146 | 169 | 88.5% | ✅ 1/1 | — | 1.25 | KEEP ✅ | F12 ✅ schema + fixture (daemon test present) · `run` 96 lines (observation for Phase 5) |
| `chgrp` | S | 86 | 77 | 96.7% | — | — | 1.25 | KEEP ✅ | F13 ✅ `common.LookupGID` · schema and daemon test present |
| `split` | S | 273 | 375 | 92.6% | — | — | 1.00 | KEEP | Preflight: clean layering (`Run` library plus `run` CLI) · schema and daemon test present |
| `cut` | S | 266 | 219 | 90.8% | ✅ 25/25 | — | 1.00 | KEEP | Preflight: clean layering (`Run` library plus `cutRun` CLI) · schema and daemon test present |
| `tree` | S | 248 | 212 | 98.0% | ✅ 4/4 | — | 1.00 | KEEP ✅ | F12 ✅ schema + fixture (daemon test present) · `buildTree` 87 lines (observation for Phase 5) |
| `xargs` | S | 213 | 177 | 94.1% | ✅ 12/12 | — | 1.00 | IMPROVE | Preflight: `xargsRun` is 159 lines with 29 branches · schema and daemon test present |
| `shell` | S | 187 | 229 | 90.2% | — | — | 1.00 | KEEP ✅ | F15 ✅ `--json` support in inline, file, and pipe modes; envelope data `{exitCode, stdout, stderr}`; the daemon test asserts it (tier9) · F12 ✅ schema + fixture · 5d.1 ✅ error paths honour `--json` |
| `chmod` | S | 172 | 193 | 92.7% | — | — | 1.00 | KEEP | Preflight: clean; `applySymbolicMode` 63 lines · schema and daemon test present |
| `cksum` | S | 155 | 157 | 94.5% | — | — | 1.00 | KEEP | Preflight: clean; `Run` library layer plus a `posixCRC` helper · schema and daemon test present |
| `du` | S | 145 | 133 | 91.9% | ✅ 6/6 | — | 1.00 | KEEP ✅ | F14 ✅ `common.HumanSize` (rounded) · schema and daemon test present |
| `id` | S | 131 | 117 | 94.6% | ✅ 4/4 | — | 1.00 | KEEP | Preflight: clean; single 97-line `run` · schema and daemon test present |
| `yes` | S | 84 | 146 | 96.0% | — | — | 1.00 | KEEP | Preflight: clean, 47-line `run` · schema and daemon test present |
| `sleep` | S | 81 | 100 | 93.8% | — | — | 1.00 | KEEP | Preflight: clean, 53-line `run` · schema and daemon test present |
| `ps` | S | 60 | 40 | 100.0% | — | — | 1.00 | KEEP | Preflight: clean, 30-line `run`, 100% coverage · schema and daemon test present |
| `cp` | S | 329 | 460 | 90.7% | ✅ 14/14 | — | 0.75 | KEEP | Preflight: clean; `run` 124 lines · schema and daemon test present |
| `fold` | S | 250 | 469 | 91.8% | ✅ 4/4 | — | 0.75 | KEEP | Preflight: clean and small · schema and daemon test present |
| `head` | S | 210 | 528 | 94.3% | ✅ 4/4 | — | 0.75 | KEEP | Preflight: clean; `headRun` 108 lines · schema and daemon test present |
| `echo` | S | 194 | 258 | 97.8% | ✅ 11/11 | — | 0.75 | KEEP ✅ | F11 ✅ `processEscapes` now wraps `common.ExpandEscapes` · schema and daemon test present |
| `realpath` | S | 188 | 211 | 94.7% | ✅ 10/10 | — | 0.75 | KEEP ✅ | F12 ✅ schema + fixture (daemon test present) · `resolvePathFlags` 95 lines (observation for Phase 5) |
| `hostid` | S | 160 | 284 | 96.3% | ✅ 1/1 | — | 0.75 | KEEP ✅ | F12 ✅ schema + fixture (daemon test present) · small and clean otherwise |
| `strings` | S | 152 | 223 | 91.5% | ✅ 1/1 | — | 0.75 | KEEP | Preflight: clean; `stringsRun` 70 lines · schema and daemon test present |
| `hostname` | S | 146 | 310 | 98.2% | ✅ 4/4 | — | 0.75 | KEEP ✅ | F1+F3: function seams, 98.2% |
| `nl` | S | 142 | 182 | 97.1% | ✅ 4/4 | — | 0.75 | KEEP | Preflight: clean; `nlRun` 71 lines · schema and daemon test present |
| `touch` | S | 142 | 302 | 91.3% | ✅ 3/3 | — | 0.75 | KEEP | Preflight: clean; `run` 74 lines · schema and daemon test present |
| `stat` | S | 141 | 115 | 100.0% | — | — | 0.75 | KEEP | Preflight: clean, 100% coverage, no function over 60 lines · schema and daemon test present |
| `tee` | S | 129 | 185 | 92.3% | ✅ 2/2 | — | 0.75 | KEEP | Preflight: clean; `teeRun` 81 lines · schema and daemon test present |
| `rev` | S | 126 | 139 | 98.2% | ✅ 4/4 | — | 0.75 | KEEP ✅ | F12 ✅ schema + fixture (daemon test present) · `run` 85 lines (observation for Phase 5) |
| `mv` | S | 112 | 246 | 96.0% | ✅ 14/14 | — | 0.75 | KEEP | Preflight: clean and small · schema and daemon test present |
| `printenv` | S | 79 | 78 | 100.0% | — | — | 0.75 | KEEP | Preflight: clean, 100% coverage · schema and daemon test present |
| `basename` | S | 61 | 97 | 95.2% | ✅ 2/2 | — | 0.75 | KEEP | Preflight: clean and small · schema and daemon test present |
| `kill` | S | 160 | 198 | 100.0% | — | — | 0.50 | KEEP | Preflight: clean, 100% coverage; `run` 79 lines · schema and daemon test present |
| `tty` | S | 108 | 201 | 100.0% | — | — | 0.50 | KEEP | Preflight: clean, 100% coverage · schema and daemon test present |
| `mkfifo` | S | 81 | 99 | 100.0% | — | — | 0.50 | KEEP | Preflight: clean, 100% coverage · schema and daemon test present |
| `env` | S | 73 | 86 | 100.0% | — | — | 0.50 | KEEP | Preflight: clean, 100% coverage · schema and daemon test present |
| `logname` | S | 70 | 77 | 100.0% | — | — | 0.50 | KEEP | Preflight: clean, 100% coverage · schema and daemon test present |
| `whoami` | S | 66 | 130 | 100.0% | — | — | 0.50 | KEEP ✅ | F1+F3: userCurrent seam, 100% |
| `link` | S | 64 | 91 | 100.0% | — | — | 0.50 | KEEP | Preflight: clean, 100% coverage · schema and daemon test present |
| `unlink` | S | 63 | 94 | 100.0% | — | — | 0.50 | KEEP | Preflight: clean, 100% coverage · schema and daemon test present |
| `sha512sum` | S | 59 | 176 | 100.0% | — | — | 0.50 | KEEP ✅ | F1+F2: shared digest core, 100% cov · F12 ✅ schema + fixture |
| `sha256sum` | S | 58 | 208 | 100.0% | — | — | 0.50 | KEEP ✅ | F1+F2: shared digest core, 100% cov |
| `sum` | S | 132 | 238 | 100.0% | ✅ 4/4 | — | 0.00 | KEEP | Preflight: 100% coverage and a passing BusyBox suite; schema and daemon test present |
| `sha3sum` | S | 114 | 294 | 100.0% | ✅ 2/2 | — | 0.00 | KEEP ✅ | F1+F2: shared digest core + -a autodetect · F12 ✅ schema + fixture |
| `pwd` | S | 102 | 219 | 100.0% | ✅ 1/1 | — | 0.00 | KEEP | Preflight: 100% coverage and a passing BusyBox suite; schema and daemon test present |
| `mkdir` | S | 79 | 138 | 100.0% | ✅ 2/2 | — | 0.00 | KEEP | Preflight: 100% coverage and a passing BusyBox suite; schema and daemon test present |
| `ln` | S | 78 | 122 | 100.0% | ✅ 6/6 | — | 0.00 | KEEP | Preflight: 100% coverage and a passing BusyBox suite; schema and daemon test present |
| `rmdir` | S | 69 | 133 | 100.0% | ✅ 1/1 | — | 0.00 | KEEP | Preflight: 100% coverage and a passing BusyBox suite; schema and daemon test present |
| `truefalse` | S | 60 | 87 | 100.0% | ✅ 4/4 | — | 0.00 | KEEP | Preflight: 100% coverage and a passing BusyBox suite; schemas `true` and `false` present |
| `sha1sum` | S | 59 | 177 | 100.0% | ✅ 1/1 | — | 0.00 | KEEP ✅ | F1+F2: shared digest core, 100% cov · F12 ✅ schema + fixture |
| `md5sum` | S | 58 | 202 | 100.0% | ✅ 2/2 | — | 0.00 | KEEP ✅ | F1+F2: shared digest core, 100% cov |
| `bunzip2` | S | 51 | 194 | 100.0% | ✅ 11/11 | — | 0.00 | KEEP ✅ | F7: shared decompress core, 100% cov · F12 ✅ schema + fixture · F16 ⏳ `-c --json` mixes the data stream with the envelope |
| `uncompress` | S | 51 | 179 | 100.0% | ✅ 1/1 | — | 0.00 | KEEP ✅ | F7: shared decompress core + recover/log suppress, 100% cov · F12 ✅ schema + fixture · F16 ⏳ `-c --json` mixes the data stream with the envelope |
| `dirname` | S | 49 | 66 | 100.0% | ✅ 7/7 | — | 0.00 | KEEP | Preflight: 100% coverage and a passing BusyBox suite; schema and daemon test present |
| `unlzma` | S | 47 | 204 | 100.0% | ✅ 3/3 | — | 0.00 | KEEP ✅ | F7: shared decompress core, 100% cov · F12 ✅ schema + fixture · F16 ⏳ `-c --json` mixes the data stream with the envelope |
| `bzcat` | S | 39 | 142 | 100.0% | ✅ 3/3 | — | 0.00 | KEEP ✅ | F7: shared decompress core (cat mode), 100% cov · F12 ✅ schema + fixture · F16 ⏳ cat mode always mixes the data stream with the envelope on stdout |

### 5a. Pre-Audit Score (preflight)

The score ranks commands by how likely they are to need a refactor or an improvement.
A high score is a **priority signal, not a verdict**. The range is 0–10.

| Signal | Points | Source |
|--------|-------:|--------|
| Coverage < 80% | +3.0 | `go test -cover` |
| Coverage 80–84.9% | +2.0 | `go test -cover` |
| Coverage 85–89.9% | +1.0 | `go test -cover` |
| Coverage 90–99.9% | +0.5 | `go test -cover` |
| Size XL | +2.5 | LOC |
| Size L | +1.5 | LOC |
| Size M | +0.75 | LOC |
| Size S | +0.25 | LOC |
| T-LOC / LOC < 0.4 | +1.0 | LOC |
| T-LOC / LOC < 0.6 | +0.75 | LOC |
| T-LOC / LOC < 0.8 | +0.5 | LOC |
| T-LOC / LOC < 1.0 | +0.25 | LOC |
| Hardcoded `os.Stdout`/`os.Stderr` in the run path | +1.5 | repo scan |
| JSON-RPC not covered (❌) | +0.75 | matrix |
| JSON-RPC partial (⚠️) | +0.5 | matrix |
| BusyBox failure | +1.5 | `make testsuite` |
| BusyBox failure, explicitly deferred | +0.5 | [wiki/deferred.md](deferred.md) |
| BusyBox root skip, or no BusyBox test | +0.25 | matrix |

**Exemption rule:** a command with **100% coverage and a passing BusyBox suite scores 0**.
Such a command is validated, so it is the weakest candidate for change. 14 commands qualify
(`sum`, `pwd`, `mkdir`, `ln`, `rmdir`, `dirname`, `truefalse`, and the seven F1/F2/F7 refactors
`sha3sum`, `sha1sum`, `md5sum`, `bunzip2`, `uncompress`, `unlzma`, `bzcat`).

**Preflight result:** 1 command ≥ 5.0 · 13 commands 3.0–4.9 · 21 commands 2.0–2.9 ·
80 commands < 2.0 (14 of those are exempt).

**Top candidates (PreAudit ≥ 3.0):** `bc`, `sed`, `printf`, `date`, `patch`, `mount`, `tar`,
`hexdump`, `dc`, `start-stop-daemon`, `unzip`, `cpio`, `uudecode`, `tsort`. All 14 are inspected.
Their verdicts and notes are in the matrix above. Two are `REFACTOR` (`bc`, `dc`), ten are
`IMPROVE`, and two are `KEEP` (`mount`, `cpio` — their scores came from tests that need root,
not from the code).

### 5b. Preflight Findings

#### F8 — Hand-rolled parser helpers duplicated across 4 packages ✅ DONE
`bc`, `expr`, `sed`, and `testcmd` each define `peek`, `next`, `parseOr`, `parseAnd`, and
`parsePrimary`. The lexer and operator-precedence logic is repeated in all four. This is the
highest-value structural item the preflight found. Move the logic to a shared expression core in
[pkg/common](../pkg/common).

**Resolution:** new [pkg/common/cursor.go](../pkg/common/cursor.go) provides `TokenCursor`
(`Peek`/`Next`/`Done`/`Lookahead`/`Has`). `expr` and `testcmd` now embed it. `bc` and `sed`
keep their local cursors: they walk different token types (`Token` and byte) with different
grammars, so a shared cursor would not reduce duplication there. The `parseOr`/`parseAnd`/
`parsePrimary` chains differ in token type and semantics, so they stay per-package.

#### F9 — `big.Rat` helpers duplicated between `bc` and `dc` ✅ DONE
`ratToInt64`, `truncateRat`, and `formatRat` exist in both [pkg/bc](../pkg/bc) and
[pkg/dc](../pkg/dc), with the same truncate-toward-zero semantics. Move them to one core, for
example `pkg/common/rat.go`. This finding sets the verdict for both commands.

**Resolution:** new [pkg/common/rat.go](../pkg/common/rat.go) provides `RatToInt64` and
`RatTruncate`. Both packages call them; the local copies are deleted. `formatRat` stays
per-package: `bc` supports non-decimal output bases and `forceNeg`, `dc` supports `negZero`
and per-number fractional digits, so their formatters are not interchangeable.

#### F10 — The JSON contract breaks on flag-parse errors in 83 of 93 commands ✅ DONE
`--json` must return the JSON envelope. A bad flag is detected before `jsonMode` is known, so a
command must pre-scan the arguments. Only 10 packages do. The other 83 print plain text.
For example, `goposix unzip --json --nope` returns the JSON envelope, but
`goposix tsort --json --nope` returns `tsort: unknown flag: --nope`. Add the pre-scan and the
flag-error block to [pkg/common](../pkg/common) and use it in every command.

**Resolution:** new `common.HasJSONFlag` and `common.RenderFlagError` in
[pkg/common/flagerr.go](../pkg/common/flagerr.go). `RenderFlagError(name, args, err, errOut,
exitCode)` pre-scans `--json`, writes the JSON error envelope (code `FLAG_ERROR`) when set,
and writes a plain `<name>: <message>` line otherwise; the caller supplies the historical exit
code. All 93 `ParseFlags` call sites in `pkg/` now use it, including `date` (keeps the BusyBox
banner in the plain path), `gzip` (`cmdName`), and the shared digest/decompress cores.
`factor` is the one exception: its `PreProcess` turns any unknown flag into a positional
argument, so a flag-parse error cannot occur there and the helper call is not needed.

#### F11 — Escape-sequence logic is duplicated three times ✅ DONE
[pkg/printf/printf.go](../pkg/printf/printf.go) holds two near-identical processors
(`processEscapes`, 88 lines, and `processEscapesForB`, 77 lines). [pkg/echo](../pkg/echo) holds a
third copy. Merge them into one helper with a `\c` option.

**Resolution:** new `common.ExpandEscapes(s, mode)` in
[pkg/common/escape.go](../pkg/common/escape.go). `EscapeFormat` is the printf format-string
semantics (`\0NNN` octal, `\c` passed through); `EscapeArg` is the echo -e / printf `%b`
semantics (bare `\0` = NUL, `\1`–`\7` octal). `printf` and `echo` now wrap this helper.

#### F12 — 39 commands have no published JSON schema ✅ DONE
[wiki/json_schema.md](json_schema.md) requires a schema for every `--json` utility. 39 of the 115
audited commands have no file in `test/schemas/`: `bc`, `mount`, `hexdump`, `dc`,
`start-stop-daemon`, `unzip`, `cpio`, `uudecode`, `tsort`, `ar`, `makedevs`, `mdev`,
`mkfs_minix`, `wget`, `xxd`, `cryptpw`, `cal`, `uuencode`, `dd`, `taskset`, `which`, `daemon`,
`factor`, `seq`, `rx`, `pidof`, `uptime`, `tree`, `shell`, `realpath`, `hostid`, `rev`,
`sha512sum`, `sha3sum`, `sha1sum`, `bunzip2`, `uncompress`, `unlzma`, `bzcat`.
Add the schema and a fixture for each command.

**Resolution:** 37 new schemas plus golden fixtures. `dd` (no `--json` support) and
`daemon` (control command) are documented exemptions in [wiki/json_schema.md](json_schema.md).
`make validate-schemas` now reports 114 passed, 0 failed, 0 skipped. Details in §5d.

**Name trap:** a schema can exist under a different name than the command. `testcmd` uses
`test.schema.json`, and `truefalse` uses `true.schema.json` and `false.schema.json`. Match on the
alias before you report a schema as missing. The same trap applied to `mkfs_minix`: the
dispatch name is `mkfs.minix`, so the schema is `mkfs.minix.schema.json`.

#### F13 — UID/GID lookup helpers duplicated between `chgrp` and `chown` ✅ DONE
`lookupGID` is byte-identical in [pkg/chgrp](../pkg/chgrp) and [pkg/chown](../pkg/chown), and
`chown` also carries `lookupUID`. Both try a number first, then the name. Move them to
[pkg/common](../pkg/common) as `LookupUID` and `LookupGID`. This finding sets the verdict for both
commands.

**Resolution:** new [pkg/common/idlookup.go](../pkg/common/idlookup.go) provides `LookupUID` and
`LookupGID`. `chgrp` and `chown` now call them; the local copies are deleted. The `chgrp` unit
tests call `common.LookupGID` directly.

#### F14 — Human-readable size formatting duplicated between `ls` and `du` ✅ DONE
[pkg/ls](../pkg/ls) and [pkg/du](../pkg/du) each define `humanSize`, with different output
(`1.0K` versus `1K`). Move one helper to [pkg/common](../pkg/common) with a format option. This
finding sets the verdict for both commands.

**Resolution:** new `common.HumanSize(n, round)` in
[pkg/common/humansize.go](../pkg/common/humansize.go). `ls` passes `round=false` (truncated),
`du` passes `round=true` (rounded). The actual output difference was truncation versus rounding,
not the `1.0K`/`1K` pair in the original note. Both commands keep thin local wrappers so their
unit tests still call `humanSize`.

#### F15 — Two commands break the "`--json` for all" contract ✅ DONE
[wiki/json_schema.md](json_schema.md) states that all utilities support `--json`. `dd` is the
documented exception. `shell` is an undocumented one: `goposix shell --json -c 'echo hi'` treats
`--json` as a file name and fails. The shell daemon test only logs its result and asserts nothing,
so the gap stays hidden. Either add `--json` to `shell` and assert it in the test, or document the
exemption in [wiki/json_schema.md](json_schema.md).

**Resolution:** `shell` now parses `--json` in all three modes (inline, script file, pipe) and
renders the envelope with `data = {exitCode, stdout, stderr}`. The script's own exit code travels
in `data.exitCode` and as the process exit code. The daemon test in
[test/posix-json/tier9_json_contract_test.go](../test/posix-json/tier9_json_contract_test.go)
asserts both a successful script and a failing one. `dd` remains the documented exception.

#### P1 — `logger` uses a package-global writer ⏳ OPEN
[pkg/logger/logger.go](../pkg/logger/logger.go) keeps a package-level `stderrWriter` that defaults
to `os.Stderr` and is swapped by `run()`. This is process-local mutable state and violates audit
item 10 (daemon safety). Replace it with an injected writer.

#### P2 — Matrix data was stale ✅ ACKNOWLEDGED (plan fixed; companion matrix still stale)
Coverage, LOC, and tier values did not match the tree. The `start-stop-daemon` row was also
misaligned (its `Verdict` text sat in the IO column). Both are corrected here.
[wiki/test_coverage_matrix.md](test_coverage_matrix.md) is stale in the same way (for example
`chmod` 68.3% vs 92.7% measured, `cp` 77.6% vs 90.7%). This preflight takes only the BusyBox
column from it. Refreshing the companion matrix is open work.

#### P3 — `awk` BusyBox failures are deferred ✅ ACKNOWLEDGED
The 17 `awk` failures are upstream goawk engine limits, not refactor work. The score reflects this.

#### P4 — Concurrent `make testsuite` runs corrupt each other ⏳ OPEN (workflow hazard)
[runtest](../test/busybox_testsuite/runtest) deletes and re-creates the shared
`runtest-tempdir-links` directory, makes a shared `busybox` symlink, and rebuilds `goposix`.
Two runs at the same time therefore interfere. Observed during this preflight: a full run reported
92 failures while another agent worked in the same checkout. A re-run of one applet in isolation
passed 19/19. Treat a high failure count as suspicious when more than one agent is active, and
re-run the applet alone before reporting a regression.

#### P5 — [AGENTS.md](../AGENTS.md) duplicated test counts and they drifted ✅ FIXED
AGENTS.md §5 reported 831 passed / 54 failed, with 7 failures in `dc` and 7 in `tar`. The
canonical matrix ([wiki/test_coverage_matrix.md](test_coverage_matrix.md)) and isolated re-runs
both report 0 (`dc` 36/36, `tar` 33/33). AGENTS.md §4a also carried a fixed coverage percentage.

**Resolution:** AGENTS.md no longer repeats test counts or the coverage percentage. It points to
[wiki/test_coverage_matrix.md](test_coverage_matrix.md) as the single source. Keep the numbers in
one place only. The same rule applies to the coverage gate: quote the gate, not the current value.

#### P6 — Two BusyBox cells were wrong, and alias names broke the score ✅ FIXED
1. `mkfs_minix` showed `⚠️ 0/1 (1 skip)`. An isolated run passes 1/1, and the companion matrix
   agrees. The cell is now `✅ 1/1`, so the score drops 2.50 → 2.25.
2. The BusyBox lookup keyed on the command name. Three matrix rows name two applets in one cell
   (`true`/`false`, `gzip`/`gunzip`, `test`/`[`), so the lookup missed them. `truefalse` looked
   unverified, and `gzip` lost its pass. The lookup now matches every backticked alias.
3. The matrix marks `test` as JSON-RPC ❌. `test/posix-json/runner_test.go` asserts
   `goposix.test` and checks the bool result, so the mark is stale.

**Effect:** `truefalse` 0.50 → 0.00 (exempt), `gzip` 1.50 → 1.25, `mkfs_minix` 2.50 → 2.25.

## 5c. PR #44 Change Log (F8–F14)

Branch `audit/f8-f11-json-parser-core`, merged into `audit/posix-commands` via PR #44.

| Finding | Tool(s) touched | What changed |
|---------|-----------------|--------------|
| F8 | `expr`, `testcmd`, `pkg/common` | new `common.TokenCursor`; `expr`/`testcmd` embed it. `bc`/`sed` keep local cursors (different token types and grammars). |
| F9 | `bc`, `dc`, `pkg/common` | new `common.RatToInt64`/`common.RatTruncate`; `bc`/`dc` call them. |
| F10 | all 93 `ParseFlags` call sites | new `common.HasJSONFlag`/`common.RenderFlagError`; flag errors return the JSON envelope. `factor` needs no flag-error path (its `PreProcess` turns unknown flags into positionals). |
| F11 | `printf`, `echo`, `pkg/common` | new `common.ExpandEscapes` with `EscapeFormat`/`EscapeArg` modes. |
| F13 | `chgrp`, `chown`, `pkg/common` | new `common.LookupUID`/`common.LookupGID`. |
| F14 | `ls`, `du`, `pkg/common` | new `common.HumanSize` with a `round` option (`ls` truncates, `du` rounds). |

Test notes: shared helpers in `pkg/common` are unit-tested to 100% statement coverage. Every
command that routes flag errors through `common.RenderFlagError` has a JSON bad-flag test.

## 5d. JSON Changes

All JSON-related findings from the audit, with their current status.

| ID | Issue | Scope | Status | Notes |
|----|-------|-------|--------|-------|
| F10 | Flag errors ignore `--json` and print plain text | 93 `ParseFlags` call sites | ✅ DONE (PR #44) | `common.HasJSONFlag` + `common.RenderFlagError`; every call site now returns the JSON envelope. `factor` is exempt (its `PreProcess` turns unknown flags into positionals). |
| F12 | No published JSON schema | 39 commands: `bc`, `mount`, `hexdump`, `dc`, `start-stop-daemon`, `unzip`, `cpio`, `uudecode`, `tsort`, `ar`, `makedevs`, `mdev`, `mkfs_minix`, `wget`, `xxd`, `cryptpw`, `cal`, `uuencode`, `dd`, `taskset`, `which`, `daemon`, `factor`, `seq`, `rx`, `pidof`, `uptime`, `tree`, `shell`, `realpath`, `hostid`, `rev`, `sha512sum`, `sha3sum`, `sha1sum`, `bunzip2`, `uncompress`, `unlzma`, `bzcat` | ✅ DONE (PR #46) | 37 new schemas in `test/schemas/` plus golden fixtures. Name trap resolved: `mkfs_minix` registers as `mkfs.minix`, so the schema is `mkfs.minix.schema.json`. `dd` (documented exception, no `--json`) and `daemon` (control command, `--json` and JSON-RPC do not apply) are documented exemptions in [wiki/json_schema.md](json_schema.md). `make validate-schemas` now reports 114 passed, 0 failed, 0 skipped. |
| F15 | Breaks the "`--json` for all" contract | `dd`, `shell` | ✅ DONE (PR #46) | `shell` now parses `--json` and returns the envelope (`data = {exitCode, stdout, stderr}`) in all three modes (inline, file, pipe); the script's own exit code travels in `data.exitCode` and as the process exit code. The daemon test asserts the envelope. `dd` stays the documented exception. |
| — | Missing JSON-RPC daemon test | commands marked ❌ in the matrix | ✅ DONE (PR #46) | New `test/posix-json/tier9_json_contract_test.go` covers `bc`, `mount`, `hexdump`, `makedevs`, `mdev`, `mkfs.minix`, `wget`, `xxd`, `rx`, and `shell` over JSON-RPC. |

### 5d.1 JSON error-path fixes (found during 5d work)

While building the schemas and daemon tests, the audit found usage-error paths that printed
plain text even with `--json`. The JSON contract applies to every output, so these paths now
call `common.RenderError` when `--json` is present:

| Command | Error path | Envelope code |
|---------|------------|---------------|
| `wget` | missing URL | `MISSING_ARGUMENT` |
| `which` | missing argument | `MISSING_ARGUMENT` |
| `seq` | wrong argument count, non-numeric argument | `INVALID_ARGUMENT` |
| `pidof` | missing operand | `MISSING_ARGUMENT` |
| `mdev` | no `-s`/`-d` and no hotplug env | `USAGE` |
| `rx` | no sender (handshake timeout), protocol bytes no longer pollute stdout in JSON mode | `RECEIVE_ERROR` |
| `shell` | missing `-c` argument, unreadable script file | `MISSING_ARGUMENT`, `SHELL_ERROR` |

### 5d.2 Open JSON follow-ups (new findings, not fixed here)

| ID | Issue | Notes |
|----|-------|-------|
| F16 | Raw payload mixes with the envelope on stdout | `bzcat --json`, `bunzip2/unlzma/uncompress -c --json`, `gzip -c --json`, and `cpio -o --json` write the raw data stream to stdout before the JSON envelope. Over JSON-RPC the daemon cannot parse the mixed output. Fixing this changes F7-core behavior, so it is recorded here for the Phase 4/5 deep audit of those tools. The `bzcat` golden fixture keeps only the trailing JSON line for this reason. |
| P7 | 31 pre-existing schemas had no golden fixture | Regenerating fixtures with `gen_golden.sh` (after fixing its `set -u` bug and the `%b` escape handling) closed 30 of 31 gaps. `who` was the last: it emitted `"users": null` with no utmp file, which its schema rejects. `pkg/who` now emits `[]`; the schema and fixture validate. No open item remains. |

## 6. What's Left (after PR #46)

Snapshot taken on `audit/whatsleft` after PR #46 (`audit/5d-json`) merged into `main`.
All JSON-related audit work (F10, F12, F15, daemon tests) is done. The remaining work:

### Phase 4 — deep audit of XL/L commands (6 of 7 open)

`printf` is done (`KEEP ✅`). One PR per command, ordered by `PreAudit` score:

| Command | Score | Open work |
|---------|:-----:|-----------|
| `bc` | 5.50 | `eval` 486 lines, `NextToken` 284 lines |
| `sed` | 3.75 | `parseInstruction` 266 lines, `execFlat` 184 lines, coverage 80.1% (lowest of the large tools) |
| `date` | 3.75 | `parsePOSIXTZ` 263-line closure nest — lift the inner funcs for testability |
| `tar` | 3.50 | `extractArchiveStream` 196 lines, `createArchiveStream` 183 lines |
| `dc` | 3.25 | `evalDC` 569-line god function |
| `diff` | 2.25 | deep audit open (F1+F3 already applied) |

### Phase 5 — sweep of M/S commands (14 verdicts open here; 20 total with Phase 4)

| Command | Score | Open work |
|---------|:-----:|-----------|
| `grep` | 2.75 | **REFACTOR** — `grepRun` 432-line god function |
| `patch` | 3.75 | `Run` 120 lines |
| `hexdump` | 3.50 | `Run` 202 lines, thin tests |
| `start-stop-daemon` | 3.25 | `run` 237 lines |
| `unzip` | 3.00 | `run` 341 lines (god function) |
| `uudecode` | 3.00 | `run` 282 lines (god function), no BusyBox tests |
| `logger` | 2.50 | P1 — package-global `stderrWriter` |
| `wget` | 2.50 | `http.Client{}` has no timeout (daemon blocking risk) |
| `xxd` | 2.25 | `reverseStandard`/`reversePlain` repeat the hex-scan loop |
| `sort` | 2.00 | exported `Run` takes unexported types |
| `uuencode` | 2.00 | `run` 167 lines |
| `taskset` | 1.75 | `run` 162 lines |
| `rx` | 1.25 | 1 flaky BusyBox test (handshake race, deferred) |
| `xargs` | 1.00 | `xargsRun` 159 lines, 29 branches |

### Open findings

| ID | Status | What |
|----|--------|------|
| F16 | ⏳ NEW (PR #46) | Raw payload mixes with the envelope on stdout: `bzcat --json`, `bunzip2/unlzma/uncompress -c --json`, `gzip -c --json`, `cpio -o --json`. Over JSON-RPC the daemon cannot parse the mixed output. |
| P1 | ⏳ OPEN | `logger` package-global `stderrWriter` (daemon re-entrancy). |
| P2 | ⏳ OPEN | [wiki/test_coverage_matrix.md](test_coverage_matrix.md) coverage numbers are stale (only the JSON-RPC column was refreshed during 5d). |
| P4 | ⏳ OPEN | Concurrent `make testsuite` runs corrupt each other (shared `runtest-tempdir-links`). |

### Housekeeping

- ✅ README now links the Phase 28 plan (added on `audit/whatsleft`).
- Phase 6: close all open verdicts once Phases 4 and 5 are done.
- Matrix rows marked `KEEP ✅` with "(observation for Phase 5)" notes (`mkfs_minix`, `cal`, `seq`, `pidof`, `uptime`, `tree`, `realpath`, `rev`, `dd` size notes) do not block the plan; they only point the Phase 5 sweep at the largest `run` functions.

## 7. Definition of Done

- Every row has a verdict and notes. ✅ (115/115)
- No row shows `REFACTOR` or `IMPROVE` for a finding that is already complete.
- Findings F1–F15 and P1 are resolved or explicitly deferred with reasons.
- Commands with `PreAudit = 0` (100% coverage and a passing BusyBox suite) need no change.
- The `PreAudit` score and the matrix data match the current tree.
- `make test`, `make testsuite`, `go vet`, `go fmt` all pass.
- Overall coverage stays ≥ 80% (CI gate).
- [wiki/test_coverage_matrix.md](test_coverage_matrix.md) and [wiki/todos.md](todos.md) updated.
