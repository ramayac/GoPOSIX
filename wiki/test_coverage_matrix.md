# GoPOSIX — Test Coverage Matrix

> **Last updated:** 2026-05-18 (✅ verified via `go test -cover` on all packages) | **BusyBox:** 548 pass / 4 fail / 10 skip | **Branch:** `main`
>
> Complete test coverage status for all 78 registered packages across unit tests,
> BusyBox integration tests, and JSON-RPC daemon tests. Coverage % are live-verified,
> not stale estimates.

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
| `true` / `false` | 75.0% | 4 | ✅ 4/4 | ✅ |
| `yes` | 80.0% | — | — | ✅ |
| `whoami` | 68.4% | — | — | ✅ |
| `hostname` | 74.5% | 4 | ✅ 4/4 | ✅ |
| `uname` | 76.7% | — | — | ✅ |
| `pwd` | 78.3% | 1 | ✅ 1/1 | ✅ |
| `printenv` | 100.0% | — | — | ✅ |
| `env` | 100.0% | — | — | ✅ |

## Tier 2 — Filesystem

| Utility | Unit Coverage | BusyBox Tests | BusyBox Status | JSON-RPC |
|---------|:------------:|:-------------:|:--------------:|:--------:|
| `ls` | 85.4% | 5 | ✅ 5/5 | ✅ |
| `cat` | 88.7% | 1 | ✅ 1/1 | ✅ |
| `mkdir` | 70.6% | 2 | ✅ 2/2 | ✅ |
| `rmdir` | 92.6% | 1 | ✅ 1/1 | ✅ |
| `rm` | 82.4% | 1 | ✅ 1/1 | ✅ |
| `cp` | 76.4% | 14 | ✅ 14/14 | ✅ |
| `mv` | 74.0% | 14 | ✅ 14/14 | ✅ |
| `touch` | 82.6% | 3 | ✅ 3/3 | ✅ |
| `ln` | 81.5% | 6 | ✅ 6/6 | ✅ |
| `stat` | 100.0% | — | — | ✅ |
| `readlink` | 81.2% | 6 | ✅ 6/6 | ✅ |
| `basename` | 85.7% | 2 | ✅ 2/2 | ✅ |
| `dirname` | 85.7% | 7 | ✅ 7/7 | ✅ |

## Tier 3 — Text Processing

| Utility | Unit Coverage | BusyBox Tests | BusyBox Status | JSON-RPC |
|---------|:------------:|:-------------:|:--------------:|:--------:|
| `head` | 94.1% | 4 | ✅ 4/4 | ✅ |
| `tail` | 87.0% | 3 | ✅ 3/3 | ✅ |
| `wc` | 81.2% | 5 | ✅ 5/5 | ✅ |
| `sort` | 82.5% | 27 | ✅ 27/27 | ✅ |
| `uniq` | 88.3% | 15 | ✅ 15/15 | ✅ |
| `tr` | 82.5% | 6 | ✅ 6/6 | ✅ |
| `cut` | 61.5% | 25 | ✅ 25/25 | ✅ |
| `tee` | 72.5% | 2 | ✅ 2/2 | ✅ |
| `grep` | 85.9% | 53 | ✅ 53/53 | ✅ |
| `sed` | 67.0% | 103 | ✅ 103/103 | ✅ |

## Tier 4 — System & Process

| Utility | Unit Coverage | BusyBox Tests | BusyBox Status | JSON-RPC |
|---------|:------------:|:-------------:|:--------------:|:--------:|
| `ps` | 84.6% | — | — | ✅ |
| `kill` | 73.1% | — | — | ✅ |
| `sleep` | 78.1% | — | — | ✅ |
| `date` | 76.4% | 7 | ⚠️ 4/7 (3 fail) | ✅ |
| `id` | 87.1% | 4 | ✅ 4/4 | ✅ |
| `chmod` | 68.3% | — | — | ✅ |
| `chown` | 71.8% | — | — | ✅ |
| `chgrp` | 70.0% | — | — | ✅ |
| `df` | 79.2% | — | — | ✅ |
| `du` | 83.9% | 6 | ✅ 6/6 | ✅ |
| `find` | 89.5% | 13 | ✅ 13/13 | ✅ |
| `xargs` | 65.3% | 12 | ✅ 12/12 | ✅ |

## Tier 5 — Advanced / Agent Features

| Utility | Unit Coverage | BusyBox Tests | BusyBox Status | JSON-RPC |
|---------|:------------:|:-------------:|:--------------:|:--------:|
| `tar` | 65.3% | 18 | ✅ 18/18 | ✅ |
| `gzip` / `gunzip` | 64.2% | 4 | ✅ 4/4 | ✅ |
| `sha256sum` | 69.4% | — | — | ✅ |
| `md5sum` | 65.3% | 2 | ✅ 2/2 | ✅ |
| `diff` | 71.0% | 20 | ✅ 20/20 | ✅ |
| `test` / `[` | 82.9% | — | — | ❌ |
| `printf` | 65.6% | 26 | ✅ 26/26 | ✅ |
| `expr` | 82.6% | 2 | ✅ 2/2 | ✅ |
| `shell` | 60.8% | — | — | ✅ |

## Tier 6 — Post-MVP (Phase 15–16, 18.3)

| Utility | Unit Coverage | BusyBox Tests | BusyBox Status | JSON-RPC |
|---------|:------------:|:-------------:|:--------------:|:--------:|
| `dd` | 81.4% | 6 | ✅ 6/6 | ✅ |
| `od` | 84.0% | 4 | ✅ 4/4 | ✅ |
| `patch` | 76.7% | 11 | ✅ 11/11 | ⚠️ |
| `unexpand` | 81.9% | 24 | ✅ 24/24 | ✅ |
| `comm` | 70.1% | 9 | ✅ 9/9 | ✅ |
| `paste` | 76.9% | 5 | ✅ 5/5 | ✅ |
| `fold` | 92.0% | 5 | ⚠️ 4/5 (1 fail) | ✅ |
| `sum` | 100.0% | 4 | ✅ 4/4 | ✅ |
| `nl` | 73.5% | 4 | ✅ 4/4 | ✅ |
| `expand` | 79.7% | 3 | ✅ 3/3 | ✅ |
| `cmp` | 61.5% | 1 | ✅ 1/1 | ✅ |
| `strings` | 90.1% | 1 | ✅ 1/1 | ✅ |

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
| Total packages | 78 | 77 utilities + client SDK |
| Unit tests passing | 78/78 | 100% |
| BusyBox tests run | 552 | 541 applicable + 11 extra (run from 548 passing + 4 failing) |
| BusyBox passed | 548 | 99.3% (548 of 552) |
| BusyBox failed | 4 | 3 date (Go TZ limits + cosmetic) + 1 fold (NUL handling) |
| BusyBox skipped | 10 | External deps (bzip2, xz, uudecode) |
| Daemon internal coverage | 64.6% | +28.7% from Phase 18 |
| JSON-RPC daemon tests | 73/77 | 95% (4 gaps: daemon, tee, testcmd, truefalse; patch skipped) |
| Packages below 70% unit coverage | 17 | See [20_hardening_ii.md](20_hardening_ii.md) §20.13 for full list |

## Remaining Gaps

| # | Gap | Count |
|---|-----|-------|
| 1 | date BusyBox failures | 3 (Go TZ limits + cosmetic) |
| 2 | fold NUL | 1 (echo harness limitation) |
| 3 | JSON-RPC daemon tests missing | 4 utilities (daemon, tee, testcmd, truefalse); patch skipped |
| 4 | Unit coverage < 60% | 2 packages: `client` (55.4%), `tty` (60.0% — exactly at 60%) |
| 5 | `awk` not implemented | Platinum gate (deferred) |

## Notes

- **BusyBox skipped (10):** All tar tests requiring bzip2/xz/uudecode (external deps)
- **Coverage gate:** CI enforces ≥70% overall (run `make cover-gate` for current; target ≥75% per Phase 20)
- **Tier 7 stubs:** Implemented as functional stubs; need hardening and BusyBox-style compliance tests
- **Client SDK** (`pkg/client`): 1,341 LOC Go SDK for agent JSON-RPC clients. Not a utility itself but included in coverage tracking. 55.4% coverage — Phase 20c targets ≥70%.
- **Stale number warning:** Prior version had stale entries for `split` (was 60.3% → now 86.3%), `nl` (was 62.2% → now 73.5%), `dd` (was 86.4% → now 81.4%), `tty` (was 54.3% → now 60.0%). Run `make cover-pkg` before updating this matrix.
