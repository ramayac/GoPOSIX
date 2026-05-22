# GoPOSIX — Test Coverage & Compliance Matrix

> **Last updated:** 2026-05-22 | **BusyBox:** 679 pass / 20 fail / 22 skip | **Branch:** `feat/coverage-85` | **Overall Coverage:** 82.2%
>
> Canonical per-utility test status for all 86 utilities. Covers unit coverage,
> BusyBox integration tests, and JSON-RPC daemon tests. Replaces the former
> `posix_coverage.md` — this is now the single source of truth.

---

## Legend

| Symbol | Meaning |
|--------|---------|
| ✅ | Tests present and passing |
| ⚠️ | Partial coverage (some tests fail) |
| ❌ | No test coverage |
| — | Not applicable (no BusyBox tests exist for this utility) |

---

## Tier 1 — Trivial / Env

| Utility | Unit Coverage | BusyBox Tests | BusyBox Status | JSON-RPC |
|---------|:------------:|:-------------:|:--------------:|:--------:|
| `echo` | 97.8% | 11 | ✅ 11/11 | ✅ |
| `true` / `false` | 100.0% | 4 | ✅ 4/4 | ✅ |
| `yes` | 80.0% | — | — | ✅ |
| `whoami` | 78.9% | — | — | ✅ |
| `hostname` | 76.0% | 4 | ✅ 4/4 | ✅ |
| `hostid` | 96.3% | 1 | ✅ 1/1 | ✅ |
| `uname` | 76.7% | — | — | ✅ |
| `pwd` | 78.3% | 1 | ✅ 1/1 | ✅ |
| `printenv` | 100.0% | — | — | ✅ |
| `env` | 100.0% | — | — | ✅ |
| `which` | 86.0% | 1 | ✅ 1/1 | ✅ |

## Tier 2 — Filesystem

| Utility | Unit Coverage | BusyBox Tests | BusyBox Status | JSON-RPC |
|---------|:------------:|:-------------:|:--------------:|:--------:|
| `ls` | 87.0% | 5 | ✅ 5/5 | ✅ |
| `cat` | 89.6% | 1 | ✅ 1/1 | ✅ |
| `mkdir` | 73.0% | 2 | ✅ 2/2 | ✅ |
| `rmdir` | 92.6% | 1 | ✅ 1/1 | ✅ |
| `rm` | 87.3% | 1 | ✅ 1/1 | ✅ |
| `cp` | 78.0% | 14 | ✅ 14/14 | ✅ |
| `mv` | 76.0% | 14 | ✅ 14/14 | ✅ |
| `touch` | 82.6% | 3 | ✅ 3/3 | ✅ |
| `ln` | 79.3% | 6 | ✅ 6/6 | ✅ |
| `stat` | 100.0% | — | — | ✅ |
| `readlink` | 76.8% | 6 | ✅ 6/6 | ✅ |
| `realpath` | 94.7% | 10 | ⚠️ 7/10 (3 fail) | ✅ |
| `basename` | 85.7% | 2 | ✅ 2/2 | ✅ |
| `dirname` | 85.7% | 7 | ✅ 7/7 | ✅ |
| `tree` | 98.0% | 4 | ✅ 1/4 (3 skip) | ✅ |

## Tier 3 — Text Processing

| Utility | Unit Coverage | BusyBox Tests | BusyBox Status | JSON-RPC |
|---------|:------------:|:-------------:|:--------------:|:--------:|
| `head` | 94.3% | 4 | ✅ 4/4 | ✅ |
| `tail` | 87.1% | 3 | ✅ 3/3 | ✅ |
| `wc` | 93.2% | 5 | ✅ 5/5 | ✅ |
| `sort` | 85.2% | 27 | ✅ 27/27 | ✅ |
| `uniq` | 88.4% | 15 | ✅ 15/15 | ✅ |
| `tr` | 84.0% | 6 | ✅ 6/6 | ✅ |
| `cut` | 90.8% | 25 | ✅ 25/25 | ✅ |
| `tee` | 73.1% | 2 | ✅ 2/2 | ✅ |
| `grep` | 84.8% | 53 | ✅ 53/53 | ✅ |
| `sed` | 69.5% | 103 | ✅ 103/103 | ✅ |
| `rev` | 94.7% | 4 | ✅ 4/4 | ✅ |
| `tsort` | 84.3% | 20 | ✅ 20/20 | ✅ |

## Tier 4 — System & Process

| Utility | Unit Coverage | BusyBox Tests | BusyBox Status | JSON-RPC |
|---------|:------------:|:-------------:|:--------------:|:--------:|
| `ps` | 84.6% | — | — | ✅ |
| `kill` | 73.1% | — | — | ✅ |
| `sleep` | 78.1% | — | — | ✅ |
| `date` | 73.5% | 7 | ✅ 7/7 | ✅ |
| `uptime` | 88.5% | 1 | ✅ 1/1 | ✅ |
| `id` | 87.1% | 4 | ✅ 4/4 | ✅ |
| `chmod` | 68.3% | — | — | ✅ |
| `chown` | 71.8% | — | — | ✅ |
| `chgrp` | 70.0% | — | — | ✅ |
| `df` | 79.2% | — | — | ✅ |
| `du` | 83.9% | 6 | ✅ 6/6 | ✅ |
| `find` | 89.5% | 13 | ✅ 13/13 | ✅ |
| `xargs` | 74.5% | 12 | ✅ 12/12 | ✅ |
| `pidof` | 92.6% | 4 | ✅ 3/4 (1 skip) | ✅ |

## Tier 5 — Advanced / Agent Features

| Utility | Unit Coverage | BusyBox Tests | BusyBox Status | JSON-RPC |
|---------|:------------:|:-------------:|:--------------:|:--------:|
| `tar` | 67.5% | 18 | ✅ 18/18 | ✅ |
| `gzip` / `gunzip` | 64.7% | 4 | ✅ 4/4 | ✅ |
| `sha256sum` | 81.6% | — | — | ✅ |
| `sha1sum` | 89.1% | 1 | ✅ 1/1 | ✅ |
| `sha512sum` | 89.1% | — | — | ✅ |
| `sha3sum` | 89.4% | 2 | ✅ 2/2 | ✅ |
| `md5sum` | 79.6% | 2 | ✅ 2/2 | ✅ |
| `diff` | 72.5% | 20 | ✅ 20/20 | ✅ |
| `test` / `[` | 82.9% | — | — | ❌ |
| `printf` | 79.5% | 26 | ✅ 26/26 | ✅ |
| `expr` | 83.5% | 2 | ✅ 2/2 | ✅ |
| `awk` | 90.0% | 53 | ⚠️ 36/53 (17 fail) | ✅ |
| `shell` | 66.7% | — | — | ✅ |
| `wget` | 81.4% | 4 | ✅ 4/4 | ✅ |

## Tier 6 — Post-MVP (Phase 15–16, 18.3)

| Utility | Unit Coverage | BusyBox Tests | BusyBox Status | JSON-RPC |
|---------|:------------:|:-------------:|:--------------:|:--------:|
| `dd` | 88.8% | 6 | ✅ 6/6 | ✅ |
| `od` | 84.0% | 4 | ✅ 4/4 | ✅ |
| `patch` | 82.1% | 11 | ✅ 11/11 | ⚠️ |
| `unexpand` | 82.8% | 24 | ✅ 24/24 | ✅ |
| `comm` | 81.0% | 9 | ✅ 9/9 | ✅ |
| `paste` | 78.5% | 5 | ✅ 5/5 | ✅ |
| `fold` | 91.8% | 4 | ✅ 4/4 | ✅ |
| `sum` | 100.0% | 4 | ✅ 4/4 | ✅ |
| `nl` | 80.9% | 4 | ✅ 4/4 | ✅ |
| `expand` | 81.4% | 3 | ✅ 3/3 | ✅ |
| `cmp` | 76.0% | 1 | ✅ 1/1 | ✅ |
| `strings` | 91.5% | 1 | ✅ 1/1 | ✅ |
| `seq` | 87.1% | 21 | ✅ 21/21 | ✅ |
| `cal` | 85.8% | 1 | ✅ 1/1 | ✅ |
| `factor` | 93.9% | 13 | ✅ 13/13 | ✅ |


## Tier 7 — Stubs (Phase 17, in-progress)

| Utility | Unit Coverage | BusyBox Tests | BusyBox Status | JSON-RPC |
|---------|:------------:|:-------------:|:--------------:|:--------:|
| `cksum` | 76.4% | — | — | ✅ |
| `join` | 78.0% | — | — | ✅ |
| `link` | 90.0% | — | — | ✅ |
| `unlink` | 89.5% | — | — | ✅ |
| `logger` | 67.7% | — | — | ✅ |
| `logname` | 70.0% | — | — | ✅ |
| `mkfifo` | 92.9% | — | — | ✅ |
| `nice` | 85.7% | — | — | ✅ |
| `nohup` | 75.0% | — | — | ✅ |
| `split` | 86.3% | — | — | ✅ |
| `tty` | 60.0% | — | — | ✅ |
| `who` | 84.8% | — | — | ✅ |
| `daemon` | 82.4% | — | — | ❌ |

## SDK / Client Library

| Utility | Unit Coverage | BusyBox Tests | BusyBox Status | JSON-RPC |
|---------|:------------:|:-------------:|:--------------:|:--------:|
| `client` | 65.0% | — | — | — |

## Infrastructure

| Utility | Unit Coverage | BusyBox Tests | BusyBox Status | JSON-RPC |
|---------|:------------:|:-------------:|:--------------:|:--------:|
| `daemon` | 82.4% | — | — | ❌ |

---

## Summary

| Suite | Count | Status |
|-------|-------|--------|
| Total packages | 93 | 92 utilities + client SDK |
| Unit tests passing | 93/93 | 100% |
| BusyBox tests run | 699 | 699 total applicable tests |
| BusyBox passed | 679 | 97.1% (679 of 699) |
| BusyBox failed | 20 | 17 awk (goawk limits) + 3 realpath (symlinked environment mismatch) |
| BusyBox skipped | 22 | External deps (bzip2, xz, uudecode, tar, tree unicode, pidof init, etc.) |
| Daemon internal coverage | 65.2% | +28.7% from Phase 18, +0.6% from Phase C |
| JSON-RPC daemon tests | 81/92 | 88.0% (11 gaps: patch/daemon skipped) |
| Packages below 70% unit coverage | 7 | `client` (56.6%), `tty` (60.0%), `gzip` (64.7%), `tar` (65.3%), `shell` (66.7%), `logger` (67.7%), `sed` (67.9%) |

## Remaining Gaps

| # | Gap | Count |
|---|-----|-------|
| 1 | awk BusyBox failures | 17 (goawk v1.31.0 limitations) |
| 2 | realpath BusyBox failures | 3 (canonical path resolution limits in symlinked workspace) |
| 3 | JSON-RPC daemon tests missing | 11 utilities |
| 4 | Unit coverage < 60% | 1 package: `client` (56.6%) |

## Notes

- **BusyBox skipped (10):** All tar tests requiring bzip2/xz/uudecode (external deps)
- **Coverage gate:** CI enforces ≥80% overall (run `make cover-gate` for current; target ≥80% per Phase 28)
- **Tier 7 stubs:** Implemented as functional stubs; need hardening and BusyBox-style compliance tests
- **Phase 26/27 progress:** Implemented 15 new utilities (`which`, `realpath`, `seq`, `sha1sum`, `sha512sum`, `rev`, `uptime`, `wget`, `cal`, `hostid`, `factor`, `sha3sum`, `tree`, `tsort`, `pidof`) with statement coverage >= 80%. Brought overall coverage to 77.9%.
- **Phase 28 (feat/coverage-10):** Added 60+ new unit tests covering CLI glue layers (`run()`), infrastructure (dispatch, flags, filepath), utility edge cases (date, printf, wc, expr, diff, sort, tar), and observability. Overall coverage: 77.9% → 80.1%. Key wins: `true/false` 75% → 100%, `wc` 81.2% → 93.2%, dispatch 100%, flags 100%, filepath 100%, `printf` 65.6% → 79.0%, `sort` 82.5% → 85.2%. CI gate raised from 70% → 80%. Remaining gaps: `main()` (os.Exit), `client_helpers` (needs daemon), platform-specific code (`setProcTitle`, `RunDaemon`).
- **Phase 28.5 (feat/coverage-85):** Added 35+ new tests across Phase A (xargs 2→12, paste +5, join +5, tr +6, hostname +4) and Phase B (mkdir +3, mv +4, cp +1, diff +3, comm +6). Overall: 80.1% → 80.9%. Key wins: `xargs` 65.7% → 74.5%, `comm` 79.4% → 81.0%, `join` 76.8% → 78.0%. All 93 packages green. Next targets: `client_helpers` (131 blocks at 0%), `sed/execFlat` (36.7%), `tar` (104 uncovered). See [wiki/todos.md](todos.md) for full plan.
- **Phase 28.6 (feat/coverage-85 Phase D):** Added 20 new tests: sed (a/i/c/q/N/n/D/P/T/w commands, SubNum, \ delimiter, $ addr, +N range), tar (extract to stdout, verbose list), cp (symlink copy), date (last-week eval, non-leap Julian, complex TZ), printf (%c, %-width, %0pad). Overall: 81.5% → 82.2%. Key wins: `sed` 68% → 69.5%, `tar` 66.5% → 67.5%, `date` 72.5% → 73.5%. All 93 packages green.
