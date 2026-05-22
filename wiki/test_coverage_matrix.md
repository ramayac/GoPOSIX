# GoPOSIX — Test Coverage & Compliance Matrix

> **Last updated:** 2026-05-22 | **BusyBox:** 679 pass / 20 fail / 22 skip | **Branch:** `feat/coverage-10` | **Overall Coverage:** 79.6%
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
| `hostname` | 74.5% | 4 | ✅ 4/4 | ✅ |
| `hostid` | 96.3% | 1 | ✅ 1/1 | ✅ |
| `uname` | 76.7% | — | — | ✅ |
| `pwd` | 78.3% | 1 | ✅ 1/1 | ✅ |
| `printenv` | 100.0% | — | — | ✅ |
| `env` | 100.0% | — | — | ✅ |
| `which` | 86.0% | 1 | ✅ 1/1 | ✅ |

## Tier 2 — Filesystem

| Utility | Unit Coverage | BusyBox Tests | BusyBox Status | JSON-RPC |
|---------|:------------:|:-------------:|:--------------:|:--------:|
| `ls` | 85.4% | 5 | ✅ 5/5 | ✅ |
| `cat` | 88.7% | 1 | ✅ 1/1 | ✅ |
| `mkdir` | 70.6% | 2 | ✅ 2/2 | ✅ |
| `rmdir` | 92.6% | 1 | ✅ 1/1 | ✅ |
| `rm` | 82.4% | 1 | ✅ 1/1 | ✅ |
| `cp` | 77.0% | 14 | ✅ 14/14 | ✅ |
| `mv` | 74.0% | 14 | ✅ 14/14 | ✅ |
| `touch` | 82.6% | 3 | ✅ 3/3 | ✅ |
| `ln` | 81.5% | 6 | ✅ 6/6 | ✅ |
| `stat` | 100.0% | — | — | ✅ |
| `readlink` | 81.2% | 6 | ✅ 6/6 | ✅ |
| `realpath` | 94.7% | 10 | ⚠️ 7/10 (3 fail) | ✅ |
| `basename` | 85.7% | 2 | ✅ 2/2 | ✅ |
| `dirname` | 85.7% | 7 | ✅ 7/7 | ✅ |
| `tree` | 98.0% | 4 | ✅ 1/4 (3 skip) | ✅ |

## Tier 3 — Text Processing

| Utility | Unit Coverage | BusyBox Tests | BusyBox Status | JSON-RPC |
|---------|:------------:|:-------------:|:--------------:|:--------:|
| `head` | 94.1% | 4 | ✅ 4/4 | ✅ |
| `tail` | 87.0% | 3 | ✅ 3/3 | ✅ |
| `wc` | 93.2% | 5 | ✅ 5/5 | ✅ |
| `sort` | 82.5% | 27 | ✅ 27/27 | ✅ |
| `uniq` | 88.3% | 15 | ✅ 15/15 | ✅ |
| `tr` | 82.5% | 6 | ✅ 6/6 | ✅ |
| `cut` | 61.5% | 25 | ✅ 25/25 | ✅ |
| `tee` | 72.5% | 2 | ✅ 2/2 | ✅ |
| `grep` | 85.9% | 53 | ✅ 53/53 | ✅ |
| `sed` | 67.0% | 103 | ✅ 103/103 | ✅ |
| `rev` | 94.7% | 4 | ✅ 4/4 | ✅ |
| `tsort` | 84.3% | 20 | ✅ 20/20 | ✅ |

## Tier 4 — System & Process

| Utility | Unit Coverage | BusyBox Tests | BusyBox Status | JSON-RPC |
|---------|:------------:|:-------------:|:--------------:|:--------:|
| `ps` | 84.6% | — | — | ✅ |
| `kill` | 73.1% | — | — | ✅ |
| `sleep` | 78.1% | — | — | ✅ |
| `date` | 71.0% | 7 | ✅ 7/7 | ✅ |
| `uptime` | 88.5% | 1 | ✅ 1/1 | ✅ |
| `id` | 87.1% | 4 | ✅ 4/4 | ✅ |
| `chmod` | 68.3% | — | — | ✅ |
| `chown` | 71.8% | — | — | ✅ |
| `chgrp` | 70.0% | — | — | ✅ |
| `df` | 79.2% | — | — | ✅ |
| `du` | 83.9% | 6 | ✅ 6/6 | ✅ |
| `find` | 89.5% | 13 | ✅ 13/13 | ✅ |
| `xargs` | 65.3% | 12 | ✅ 12/12 | ✅ |
| `pidof` | 92.6% | 4 | ✅ 3/4 (1 skip) | ✅ |

## Tier 5 — Advanced / Agent Features

| Utility | Unit Coverage | BusyBox Tests | BusyBox Status | JSON-RPC |
|---------|:------------:|:-------------:|:--------------:|:--------:|
| `tar` | 65.3% | 18 | ✅ 18/18 | ✅ |
| `gzip` / `gunzip` | 64.2% | 4 | ✅ 4/4 | ✅ |
| `sha256sum` | 69.4% | — | — | ✅ |
| `sha1sum` | 89.1% | 1 | ✅ 1/1 | ✅ |
| `sha512sum` | 89.1% | — | — | ✅ |
| `sha3sum` | 89.4% | 2 | ✅ 2/2 | ✅ |
| `md5sum` | 65.3% | 2 | ✅ 2/2 | ✅ |
| `diff` | 73.0% | 20 | ✅ 20/20 | ✅ |
| `test` / `[` | 82.9% | — | — | ❌ |
| `printf` | 65.6% | 26 | ✅ 26/26 | ✅ |
| `expr` | 82.6% | 2 | ✅ 2/2 | ✅ |
| `awk` | 88.1% | 53 | ⚠️ 36/53 (17 fail) | ✅ |
| `shell` | 60.8% | — | — | ✅ |
| `wget` | 81.4% | 4 | ✅ 4/4 | ✅ |

## Tier 6 — Post-MVP (Phase 15–16, 18.3)

| Utility | Unit Coverage | BusyBox Tests | BusyBox Status | JSON-RPC |
|---------|:------------:|:-------------:|:--------------:|:--------:|
| `dd` | 83.0% | 6 | ✅ 6/6 | ✅ |
| `od` | 84.0% | 4 | ✅ 4/4 | ✅ |
| `patch` | 76.7% | 11 | ✅ 11/11 | ⚠️ |
| `unexpand` | 81.9% | 24 | ✅ 24/24 | ✅ |
| `comm` | 72.5% | 9 | ✅ 9/9 | ✅ |
| `paste` | 76.9% | 5 | ✅ 5/5 | ✅ |
| `fold` | 91.8% | 4 | ✅ 4/4 | ✅ |
| `sum` | 100.0% | 4 | ✅ 4/4 | ✅ |
| `nl` | 73.5% | 4 | ✅ 4/4 | ✅ |
| `expand` | 79.7% | 3 | ✅ 3/3 | ✅ |
| `cmp` | 63.5% | 1 | ✅ 1/1 | ✅ |
| `strings` | 90.1% | 1 | ✅ 1/1 | ✅ |
| `seq` | 87.1% | 21 | ✅ 21/21 | ✅ |
| `cal` | 85.8% | 1 | ✅ 1/1 | ✅ |
| `factor` | 93.9% | 13 | ✅ 13/13 | ✅ |


## Tier 7 — Stubs (Phase 17, in-progress)

| Utility | Unit Coverage | BusyBox Tests | BusyBox Status | JSON-RPC |
|---------|:------------:|:-------------:|:--------------:|:--------:|
| `cksum` | 76.4% | — | — | ✅ |
| `join` | 76.8% | — | — | ✅ |
| `link` | 90.0% | — | — | ✅ |
| `unlink` | 89.5% | — | — | ✅ |
| `logger` | 61.5% | — | — | ✅ |
| `logname` | 70.0% | — | — | ✅ |
| `mkfifo` | 92.9% | — | — | ✅ |
| `nice` | 85.7% | — | — | ✅ |
| `nohup` | 68.2% | — | — | ✅ |
| `split` | 86.3% | — | — | ✅ |
| `tty` | 60.0% | — | — | ✅ |
| `who` | 84.8% | — | — | ✅ |
| `daemon` | 82.4% | — | — | ❌ |

## SDK / Client Library

| Utility | Unit Coverage | BusyBox Tests | BusyBox Status | JSON-RPC |
|---------|:------------:|:-------------:|:--------------:|:--------:|
| `client` | 55.4% | — | — | — |

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
| Daemon internal coverage | 65.0% | +28.7% from Phase 18, +0.4% from feat/coverage-10 |
| JSON-RPC daemon tests | 81/92 | 88.0% (11 gaps: patch/daemon skipped) |
| Packages below 70% unit coverage | 5 | See [20_hardening_ii.md](20_hardening_ii.md) §20.13 for details |

## Remaining Gaps

| # | Gap | Count |
|---|-----|-------|
| 1 | awk BusyBox failures | 17 (goawk v1.31.0 limitations) |
| 2 | realpath BusyBox failures | 3 (canonical path resolution limits in symlinked workspace) |
| 3 | JSON-RPC daemon tests missing | 11 utilities |
| 4 | Unit coverage < 60% | 0 packages (was 1: `client` now at ~56%) |

## Notes

- **BusyBox skipped (10):** All tar tests requiring bzip2/xz/uudecode (external deps)
- **Coverage gate:** CI enforces ≥70% overall (run `make cover-gate` for current; target ≥75% per Phase 20)
- **Tier 7 stubs:** Implemented as functional stubs; need hardening and BusyBox-style compliance tests
- **Phase 26/27 progress:** Implemented 15 new utilities (`which`, `realpath`, `seq`, `sha1sum`, `sha512sum`, `rev`, `uptime`, `wget`, `cal`, `hostid`, `factor`, `sha3sum`, `tree`, `tsort`, `pidof`) with statement coverage >= 80%. Brought overall coverage to 77.9%.
- **Phase 28 (feat/coverage-10):** Added 40+ new unit tests covering CLI glue layers (`run()`), infrastructure (dispatch, flags, filepath), utility edge cases (date, printf, wc, expr, diff), and observability. Overall coverage: 77.9% → 79.6%. Key wins: `true/false` 75% → 100%, `wc` 81.2% → 93.2%, dispatch 100%, flags 100%, filepath 100%. Remaining gaps: `main()` (os.Exit), `client_helpers` (needs daemon), platform-specific code (`setProcTitle`, `RunDaemon`).
