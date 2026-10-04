# POSIX Command Audit — Plan & Matrix

> **Created:** 2026-10-03 | **Branch:** `audit/posix-commands` | **Commands:** 115 | **Status:** IN PROGRESS (F1/F2/F3 done)
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

### F4 — BusyBox failures

`awk` has 17 failing tests (upstream goawk engine limits, deferred — see [wiki/deferred.md](deferred.md)).
`rx` has 1 flaky test (handshake race). All other ⚠️ entries are root-required skips.

### F5 — Signal parsing duplication

`start-stop-daemon` keeps a partial `parseSignal` (7 names). [pkg/common/signal.go](../pkg/common/signal.go)
has the full table (added for `kill` in PR #41). Consolidate.

### F6 — `panic` in bc startup

[pkg/bc/bc.go](../pkg/bc/bc.go) calls `panic` if the embedded math library fails to load at init.
Fail-fast is acceptable for an embedded asset, but a returned error is cleaner. Decide during audit.

### F7 — Compression family overlap

`gzip`, `uncompress`, `unlzma`, `bunzip2`, `bzcat` all wrap stdlib `compress/*` readers. Check for
duplicated wrapper logic (header checks, multi-stream handling).

## 4. Phases & Priorities

| Phase | Scope | Commands | Verify |
|-------|-------|----------|--------|
| 0 | This plan + matrix snapshot | — | ✅ committed on `audit/posix-commands` |
| 1 | F1 mechanical fix: injected writers | 53 | ✅ 871/16/30 BusyBox, unit + vet clean |
| 2 | F2 digest consolidation + F5 signal consolidation | 8 | ✅ digest core; F5 still open |
| 3 | F3 coverage drive | 4 | ✅ whoami 100, hostname 98.2, diff 89.5, gzip 87.3 |
| 4 | Deep audit of XL/L commands (one PR each) | 24 | per-command checklist + suites |
| 5 | Sweep of M/S commands (batched) | 87 | per-command checklist + suites |
| 6 | Decide F6/F7 and close all open verdicts | — | matrix 100% filled |

## 5. Audit Matrix

**Tier by LOC:** XL ≥ 1500 · L 700–1499 · M 350–699 · S < 350

**Columns:** LOC = non-test lines. T-LOC = test lines. Cov = unit coverage. BB = BusyBox status
(from [wiki/test_coverage_matrix.md](test_coverage_matrix.md)). IO = F1 hardcoded I/O. Verdict filled during audit.

| Command | Tier | LOC | T-LOC | Cov | BB | IO | Verdict | Notes |
|---------|:----:|----:|------:|----:|:--:|:--:|:-------:|-------|
| `bc` | XL | 3128 | 547 | 83.4% | ✅ 81/81 | — |  |  |
| `tar` | XL | 2635 | 1521 | 82.1% | ✅ 31/31 | — |  |  |
| `sed` | XL | 2241 | 1060 | 80.1% | ✅ 103/103 | — |  |  |
| `dc` | XL | 1777 | 628 | 89.0% | ✅ 36/36 | — |  |  |
| `printf` | L | 1494 | 744 | 83.7% | ✅ 26/26 | — |  |  |
| `grep` | L | 1487 | 839 | 84.8% | ✅ 53/53 | — |  |  |
| `date` | L | 1381 | 635 | 82.6% | ✅ 7/7 | — |  |  |
| `diff` | L | 1365 | 529 | 89.5% | ✅ 20/20 | — | IMPROVE | F1+F3: tests 89.5% |
| `sort` | L | 1190 | 537 | 86.2% | ✅ 27/27 | — |  |  |
| `testcmd` | L | 1012 | 602 | 88.4% | — | — |  |  |
| `ar` | L | 1000 | 516 | 81.6% | ✅ 2/2 | — |  |  |
| `patch` | L | 981 | 381 | 82.1% | ✅ 11/11 | — |  |  |
| `od` | L | 953 | 431 | 84.5% | ✅ 4/4 | — |  |  |
| `cpio` | L | 905 | 449 | 82.0% | ✅ 2/9 (7 skip) | — |  |  |
| `unzip` | L | 861 | 403 | 80.5% | ✅ 4/4 | — |  |  |
| `expr` | L | 832 | 385 | 83.9% | ✅ 2/2 | — |  |  |
| `ls` | L | 820 | 377 | 88.6% | ✅ 5/5 | — |  |  |
| `hexdump` | L | 798 | 236 | 84.7% | ✅ 3/3 | — |  |  |
| `awk` | L | 766 | 558 | 90.0% | ⚠️ 36/53 (17 fail, deferred) | — |  |  |
| `gzip` | L | 763 | 457 | 87.3% | ✅ 4/4 | — | IMPROVE | F3: tests 87.3% |
| `head` | L | 738 | 528 | 94.3% | ✅ 4/4 | — |  |  |
| `fold` | L | 719 | 469 | 91.8% | ✅ 4/4 | — |  |  |
| `cp` | L | 714 | 385 | 82.6% | ✅ 14/14 | — |  |  |
| `cat` | M | 684 | 464 | 89.6% | ✅ 1/1 | — |  |  |
| `start-stop-daemon` | M | 634 | 240 | 82.1% | ✅ 4/4 | — |  |  |
| `split` | M | 621 | 351 | 86.3% | — | — |  |  |
| `xxd` | M | 610 | 239 | 86.4% | ✅ 7/7 | — |  |  |
| `comm` | M | 591 | 346 | 88.8% | ✅ 9/9 | — |  |  |
| `cryptpw` | M | 576 | 335 | 82.4% | ✅ 7/7 | — |  |  |
| `who` | M | 573 | 383 | 84.8% | — | — |  |  |
| `wc` | M | 570 | 263 | 88.0% | ✅ 5/5 | — |  |  |
| `join` | M | 568 | 264 | 80.6% | — | — |  |  |
| `seq` | M | 547 | 263 | 89.7% | ✅ 21/21 | — |  |  |
| `find` | M | 544 | 293 | 89.8% | ✅ 13/13 | — |  |  |
| `unexpand` | M | 544 | 287 | 82.8% | ✅ 24/24 | — |  |  |
| `dd` | M | 540 | 208 | 88.8% | ✅ 6/6 | — |  |  |
| `uudecode` | M | 540 | 217 | 84.6% | — | — |  |  |
| `logger` | M | 530 | 311 | 98.6% | — | — |  |  |
| `tail` | M | 522 | 294 | 87.1% | ✅ 3/3 | — |  |  |
| `factor` | M | 511 | 165 | 93.9% | ✅ 13/13 | — |  |  |
| `sha3sum` | M | 114 | 252 | 90.2% | ✅ 2/2 | — | REFACTOR | F1+F2: shared digest core + -a autodetect |
| `mount` | M | 495 | 178 | 80.6% | ⚠️ 0/1 (1 skip) | — |  |  |
| `cal` | M | 492 | 144 | 85.8% | ✅ 1/1 | — |  |  |
| `makedevs` | M | 491 | 156 | 87.3% | ⚠️ 0/1 (1 skip) | — |  |  |
| `cut` | M | 485 | 219 | 90.8% | ✅ 25/25 | — |  |  |
| `cmp` | M | 479 | 286 | 82.3% | ✅ 1/1 | — |  |  |
| `taskset` | M | 476 | 191 | 86.4% | ✅ 3/3 | — |  |  |
| `uncompress` | M | 469 | 179 | 84.1% | ✅ 1/1 | — |  |  |
| `paste` | M | 467 | 240 | 88.5% | ✅ 5/5 | — |  |  |
| `rx` | M | 461 | 279 | 86.2% | ✅ 1/1 | — |  |  |
| `tree` | M | 460 | 212 | 98.0% | ✅ 4/4 | — |  |  |
| `readlink` | M | 459 | 251 | 81.2% | ✅ 6/6 | — |  |  |
| `mdev` | M | 455 | 146 | 87.4% | ⚠️ 0/12 (12 skip) | — |  |  |
| `unlzma` | M | 454 | 204 | 83.3% | ✅ 3/3 | — |  |  |
| `echo` | M | 452 | 258 | 97.8% | ✅ 11/11 | — |  |  |
| `hostid` | M | 444 | 284 | 96.3% | ✅ 1/1 | — |  |  |
| `bunzip2` | M | 442 | 194 | 82.7% | ✅ 11/11 | — |  |  |
| `mkfs_minix` | M | 433 | 121 | 86.4% | — | — |  |  |
| `uniq` | M | 432 | 239 | 88.4% | ✅ 15/15 | — |  |  |
| `shell` | M | 416 | 229 | 90.2% | — | — |  |  |
| `tr` | M | 413 | 148 | 90.8% | ✅ 6/6 | — |  |  |
| `sha256sum` | M | 58 | 208 | 100.0% | — | — | REFACTOR | F1+F2: shared digest core, 100% cov |
| `touch` | M | 407 | 265 | 82.6% | ✅ 3/3 | — |  |  |
| `md5sum` | M | 58 | 202 | 100.0% | ✅ 2/2 | — | REFACTOR | F1+F2: shared digest core, 100% cov |
| `realpath` | M | 399 | 211 | 94.7% | ✅ 10/10 | — |  |  |
| `xargs` | M | 390 | 177 | 94.1% | ✅ 12/12 | — |  |  |
| `uuencode` | M | 385 | 134 | 88.3% | ✅ 19/19 | — |  |  |
| `sha1sum` | M | 59 | 177 | 100.0% | ✅ 1/1 | — | REFACTOR | F1+F2: shared digest core, 100% cov |
| `sha512sum` | M | 59 | 176 | 100.0% | — | — | REFACTOR | F1+F2: shared digest core, 100% cov |
| `strings` | M | 375 | 223 | 91.5% | ✅ 1/1 | — |  |  |
| `sum` | M | 370 | 238 | 100.0% | ✅ 4/4 | — |  |  |
| `wget` | M | 363 | 172 | 81.4% | ✅ 4/4 | — |  |  |
| `kill` | M | 359 | 198 | 100.0% | — | — |  |  |
| `rm` | M | 350 | 198 | 87.3% | ✅ 1/1 | — |  |  |
| `mv` | S | 335 | 223 | 84.0% | ✅ 14/14 | — |  |  |
| `chmod` | S | 327 | 155 | 80.5% | — | — |  |  |
| `nl` | S | 324 | 182 | 97.1% | ✅ 4/4 | — |  |  |
| `pwd` | S | 321 | 219 | 100.0% | ✅ 1/1 | — |  |  |
| `uptime` | S | 315 | 169 | 88.5% | ✅ 1/1 | — |  |  |
| `tee` | S | 314 | 185 | 92.3% | ✅ 2/2 | — |  |  |
| `tty` | S | 309 | 201 | 100.0% | — | — |  |  |
| `hostname` | S | 305 | 166 | 98.2% | ✅ 4/4 | — | IMPROVE | F1+F3: function seams, 98.2% |
| `bzcat` | S | 304 | 142 | 90.6% | ✅ 3/3 | — |  |  |
| `cksum` | S | 290 | 135 | 85.5% | — | — |  |  |
| `expand` | S | 271 | 149 | 81.4% | ✅ 3/3 | — |  |  |
| `du` | S | 268 | 122 | 88.7% | ✅ 6/6 | — |  |  |
| `pidof` | S | 262 | 114 | 96.7% | ✅ 4/4 | — |  |  |
| `stat` | S | 257 | 115 | 100.0% | — | — |  |  |
| `rev` | S | 254 | 128 | 94.7% | ✅ 4/4 | — |  |  |
| `id` | S | 248 | 117 | 94.6% | ✅ 4/4 | — |  |  |
| `nohup` | S | 225 | 111 | 80.9% | — | — |  |  |
| `which` | S | 222 | 96 | 86.0% | ✅ 1/1 | — |  |  |
| `tsort` | S | 218 | 73 | 84.3% | ✅ 20/20 | — |  |  |
| `yes` | S | 208 | 124 | 80.0% | — | — |  |  |
| `uname` | S | 201 | 51 | 86.0% | — | — |  |  |
| `mkdir` | S | 192 | 113 | 85.3% | ✅ 2/2 | — |  |  |
| `rmdir` | S | 191 | 122 | 92.6% | ✅ 1/1 | — |  |  |
| `nice` | S | 180 | 73 | 85.7% | — | — |  |  |
| `ln` | S | 176 | 98 | 82.8% | ✅ 6/6 | — |  |  |
| `sleep` | S | 170 | 88 | 87.5% | — | — |  |  |
| `mkfifo` | S | 168 | 87 | 92.9% | — | — |  |  |
| `chown` | S | 164 | 55 | 92.3% | — | — |  |  |
| `env` | S | 159 | 86 | 100.0% | — | — |  |  |
| `printenv` | S | 157 | 78 | 100.0% | — | — |  |  |
| `truefalse` | S | 148 | 87 | 100.0% | ✅ 4/4 | — |  |  |
| `unlink` | S | 146 | 82 | 89.5% | — | — |  |  |
| `link` | S | 143 | 79 | 90.0% | — | — |  |  |
| `chgrp` | S | 141 | 55 | 83.3% | — | — |  |  |
| `basename` | S | 136 | 74 | 85.7% | ✅ 2/2 | — |  |  |
| `whoami` | S | 135 | 71 | 100.0% | — | — | IMPROVE | F1+F3: userCurrent seam, 100% |
| `daemon` | S | 125 | 70 | 82.4% | — | — |  |  |
| `logname` | S | 124 | 57 | 80.0% | — | — |  |  |
| `df` | S | 112 | 37 | 87.5% | — | — |  |  |
| `dirname` | S | 104 | 54 | 85.7% | ✅ 7/7 | — |  |  |
| `ps` | S | 89 | 29 | 84.6% | — | — |  |  |

## 6. Definition of Done

- Every row in the matrix has a verdict and notes.
- Findings F1–F7 are resolved or explicitly deferred with reasons.
- `make test`, `make testsuite`, `go vet`, `go fmt` all pass.
- Overall coverage stays ≥ 80% (CI gate).
- [wiki/test_coverage_matrix.md](test_coverage_matrix.md) and [wiki/todos.md](todos.md) updated.
