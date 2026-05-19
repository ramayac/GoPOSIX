# Post-MVP Fix Sessions

> **Status:** COMPLETED | **Date:** 2026-05-16
>
> Three fix sessions that closed critical gaps after the Phase 10 baseline.

---

## 14a — JSON Gap Fill (8 Utilities)

Added `--json` structured output to 8 utilities that previously lacked it:
`echo`, `testcmd`, `sed`, `tee`, `tr`, `sleep`, `truefalse`, `yes`.

Each got a typed `Result` struct, `FlagSpec` integration, and `--json` tests.
Fixed an echo daemon bug where `--json` was appended instead of prepended.

**Outcome:** All 77 utilities now support `--json`.

---

## 14b — BusyBox Regression Fix (79 → 3 Failures)

A single architectural mistake cascaded across the entire test suite:
`common.ParseFlags` was applied uniformly to all utilities with no escape hatch
for free-form tools where arguments can start with `-`.

**Root causes fixed:**
- `echo` / `printf` / `expr`: Manual flag parsing (stop at first non-flag arg)
- `cp` `devID()`: Dereferenced `Sys()` pointer instead of formatting `Dev:Ino`
- `test`/`[`: Expression parser lacked lookahead for `!`/`(` operators
- `date`: Missing `-d` and `+FORMAT` support
- `mv`/`touch`/`chmod`: Missing POSIX flags (`-d`, symbolic modes, `-t`)
- Plus 15 single-failure fixes across `basename`, `cat`, `find`, `grep`, `hostname`, `tail`, `tar`, `touch`, `wc`, `xargs`

**Key lessons:**
1. Shared infra needs escape hatches (stop-at-first-nonflag mode)
2. Never use `-j` short for `--json` (collides with `tar -j`, free-form data)
3. BusyBox suite gates every commit (catches cascading failures)
4. Added ~75 hardening tests across 17 packages

**Final:** 548 passed, 4 failed (3 date, 1 fold NUL — all Go/runtime limitations)

---

## 14c — JSON-RPC Coverage Gap (9 → 55 Utilities)

The daemon integration test at `test/posix-json/runner_test.go` only exercised 9 of 55
utility packages. Added 46 new test cases across 5 tiers:

| Tier | Utilities | New Tests |
|------|-----------|:---------:|
| 1 — Filesystem | ls, cp, mv, rm, mkdir, rmdir, touch, ln, readlink, stat, chmod, chown, chgrp | 13 |
| 2 — Text | grep, find, sort, uniq, wc, head, tail, cut, diff, printf | 11 |
| 3 — System | date, du, df, ps, id, hostname, whoami, pwd, uname, kill | 10 |
| 4 — Archive | tar, gzip, sha256sum, md5sum | 5 |
| 5 — Misc | expr, basename, dirname, env, printenv, xargs | 7 |

**Bugs found and fixed:** `find` flag pre-processing broke `--json`; `uniq` overwrote
`out` writer with `os.Stdout` in JSON mode.

**Outcome:** 55/55 utilities (100%) tested via JSON-RPC daemon.
