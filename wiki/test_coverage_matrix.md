# GoPOSIX — Test Coverage Matrix

> **Last updated:** 2026-05-16 | **BusyBox:** 526 pass / 5 fail / 10 skip | **Branch:** `feat/post-mvp`
>
> Complete test coverage status for all 74 registered utilities across unit tests,
> BusyBox integration tests, and JSON-RPC daemon tests.

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
| `tee` | 72.5% | 2 | ✅ 2/2 | ❌ |
| `grep` | 86.3% | 53 | ✅ 53/53 | ✅ |
| `sed` | 67.0% | 103 | ✅ 103/103 | ❌ |

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
| `gzip` / `gunzip` | 63.5% | 4 | ✅ 4/4 | ✅ |
| `sha256sum` | 69.4% | — | — | ✅ |
| `md5sum` | 65.3% | 2 | ✅ 2/2 | ✅ |
| `diff` | 54.8% | 20 | ✅ 20/20 | ✅ |
| `test` / `[` | 82.9% | — | — | ❌ |
| `printf` | 65.6% | 26 | ✅ 26/26 | ✅ |
| `expr` | 82.6% | 2 | ✅ 2/2 | ✅ |
| `shell` | 60.8% | — | — | ❌ |

## Tier 6 — Post-MVP (Phase 15–16)

| Utility | Unit Coverage | BusyBox Tests | BusyBox Status | JSON-RPC |
|---------|:------------:|:-------------:|:--------------:|:--------:|
| `unexpand` | 81.9% | 24 | ✅ 24/24 | ❌ |
| `comm` | 70.1% | 9 | ✅ 9/9 | ❌ |
| `paste` | 46.2% | 5 | ✅ 5/5 | ❌ |
| `fold` | 78.6% | 5 | ⚠️ 3/5 (2 fail) | ❌ |
| `sum` | — | 4 | ✅ 4/4 | ❌ |
| `nl` | — | 4 | ✅ 4/4 | ❌ |
| `expand` | — | 3 | ✅ 3/3 | ❌ |
| `cmp` | — | 1 | ✅ 1/1 | ❌ |
| `strings` | — | 1 | ✅ 1/1 | ❌ |

## Tier 7 — Stubs (Phase 17, in-progress)

| Utility | Unit Coverage | BusyBox Tests | BusyBox Status | JSON-RPC |
|---------|:------------:|:-------------:|:--------------:|:--------:|
| `cksum` | 76.4% | — | — | ✅ |
| `join` | 49.0% | — | — | ✅ |
| `link` | 90.0% | — | — | ✅ |
| `unlink` | 89.5% | — | — | ✅ |
| `logger` | 61.5% | — | — | ✅ |
| `logname` | 70.0% | — | — | ✅ |
| `mkfifo` | 92.9% | — | — | ✅ |
| `nice` | 85.7% | — | — | ✅ |
| `nohup` | 68.2% | — | — | ✅ |
| `split` | 45.2% | — | — | ✅ |
| `tty` | 54.3% | — | — | ✅ |
| `who` | 54.5% | — | — | ✅ |
| `daemon` | 82.4% | — | — | ❌ |

## Infrastructure

| Utility | Unit Coverage | BusyBox Tests | BusyBox Status | JSON-RPC |
|---------|:------------:|:-------------:|:--------------:|:--------:|
| `daemon` | 82.4% | — | — | ❌ |

---

## Summary

| Suite | Count | Status |
|-------|-------|--------|
| Total utilities | 74 | All registered |
| Unit tests passing | 74/74 | 100% |
| BusyBox tests total | 541 | — |
| BusyBox passed | 526 | 97.2% |
| BusyBox failed | 5 | 3 date + 2 fold |
| BusyBox skipped | 10 | External deps |
| JSON-RPC daemon tests | 59/74 | 80% (15 gaps) |

## Remaining Gaps

| # | Gap | Count |
|---|-----|-------|
| 1 | date BusyBox failures | 3 (Go TZ limits + cosmetic) |
| 2 | fold NUL/Unicode | 2 (binary data handling) |
| 3 | JSON-RPC daemon tests missing | 15 utilities (see ❌ marks above) |
| 4 | Unit coverage < 60% | 7 utilities (diff, join, paste, shell, split, tty, who) |
| 5 | `awk` not implemented | Platinum gate (deferred) |

## Notes

- **BusyBox skipped (10):** All tar tests requiring bzip2/xz/uudecode (external deps)
- **Coverage gate:** CI enforces ≥70% overall (current: ~72%)
- **Tier 7 stubs:** Implemented as functional stubs; need hardening and BusyBox-style compliance tests
