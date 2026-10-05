# GoPOSIX — Test Coverage & Compliance Matrix

> **Last updated:** 2026-10-04 | **BusyBox:** 870 pass / 17 fail / 30 skip | **Branch:** `audit/whatsleft` | **Overall Coverage:** 88.1% | **JSON-RPC:** 115/115 (100.0%)
>
> Coverage values refreshed from the tree on 2026-10-04 (P2).
> Canonical per-utility test status for all 115 utilities. Covers unit coverage,
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
| `echo` | 100.0% | 11 | ✅ 11/11 | ✅ |
| `true` / `false` | 100.0% | 4 | ✅ 4/4 | ✅ |
| `yes` | 95.8% | — | — | ✅ |
| `whoami` | 100.0% | — | — | ✅ |
| `hostname` | 98.1% | 4 | ✅ 4/4 | ✅ |
| `hostid` | 98.0% | 1 | ✅ 1/1 | ✅ |
| `uname` | 97.6% | — | — | ✅ |
| `pwd` | 100.0% | 1 | ✅ 1/1 | ✅ |
| `printenv` | 100.0% | — | — | ✅ |
| `env` | 100.0% | — | — | ✅ |
| `which` | 94.1% | 1 | ✅ 1/1 | ✅ |

## Tier 2 — Filesystem

| Utility | Unit Coverage | BusyBox Tests | BusyBox Status | JSON-RPC |
|---------|:------------:|:-------------:|:--------------:|:--------:|
| `ls` | 88.1% | 5 | ✅ 5/5 | ✅ |
| `cat` | 89.5% | 1 | ✅ 1/1 | ✅ |
| `mkdir` | 100.0% | 2 | ✅ 2/2 | ✅ |
| `rmdir` | 100.0% | 1 | ✅ 1/1 | ✅ |
| `rm` | 87.1% | 1 | ✅ 1/1 | ✅ |
| `cp` | 90.6% | 14 | ✅ 14/14 | ✅ |
| `mv` | 95.9% | 14 | ✅ 14/14 | ✅ |
| `touch` | 91.2% | 3 | ✅ 3/3 | ✅ |
| `ln` | 100.0% | 6 | ✅ 6/6 | ✅ |
| `stat` | 100.0% | — | — | ✅ |
| `readlink` | 81.1% | 6 | ✅ 6/6 | ✅ |
| `realpath` | 96.8% | 10 | ✅ 10/10 | ✅ |
| `basename` | 95.0% | 2 | ✅ 2/2 | ✅ |
| `dirname` | 100.0% | 7 | ✅ 7/7 | ✅ |
| `tree` | 97.9% | 4 | ✅ 4/4 | ✅ |

## Tier 3 — Text Processing

| Utility | Unit Coverage | BusyBox Tests | BusyBox Status | JSON-RPC |
|---------|:------------:|:-------------:|:--------------:|:--------:|
| `head` | 94.2% | 4 | ✅ 4/4 | ✅ |
| `tail` | 88.6% | 3 | ✅ 3/3 | ✅ |
| `wc` | 88.6% | 5 | ✅ 5/5 | ✅ |
| `sort` | 86.1% | 27 | ✅ 27/27 | ✅ |
| `uniq` | 88.3% | 15 | ✅ 15/15 | ✅ |
| `tr` | 90.8% | 6 | ✅ 6/6 | ✅ |
| `cut` | 90.8% | 25 | ✅ 25/25 | ✅ |
| `tee` | 92.2% | 2 | ✅ 2/2 | ✅ |
| `grep` | 84.8% | 53 | ✅ 53/53 | ✅ |
| `sed` | 80.0% | 103 | ✅ 103/103 | ✅ |
| `rev` | 98.2% | 4 | ✅ 4/4 | ✅ |
| `tsort` | 87.0% | 20 | ✅ 20/20 | ✅ |

## Tier 4 — System & Process

| Utility | Unit Coverage | BusyBox Tests | BusyBox Status | JSON-RPC |
|---------|:------------:|:-------------:|:--------------:|:--------:|
| `ps` | 100.0% | — | — | ✅ |
| `kill` | 100.0% | — | — | ✅ |
| `sleep` | 93.5% | — | — | ✅ |
| `date` | 83.9% | 7 | ✅ 7/7 | ✅ |
| `uptime` | 92.2% | 1 | ✅ 1/1 | ✅ |
| `id` | 94.5% | 4 | ✅ 4/4 | ✅ |
| `chmod` | 92.6% | — | — | ✅ |
| `chown` | 100.0% | — | — | ✅ |
| `chgrp` | 95.7% | — | — | ✅ |
| `df` | 95.7% | — | — | ✅ |
| `du` | 90.4% | 6 | ✅ 6/6 | ✅ |
| `find` | 89.7% | 13 | ✅ 13/13 | ✅ |
| `xargs` | 94.1% | 12 | ✅ 12/12 | ✅ |
| `pidof` | 96.8% | 4 | ✅ 4/4 | ✅ |

## Tier 5 — Advanced / Agent Features

| Utility | Unit Coverage | BusyBox Tests | BusyBox Status | JSON-RPC |
|---------|:------------:|:-------------:|:--------------:|:--------:|
| `tar` | 82.6% | 31 | ✅ 31/31 | ✅ |
| `gzip` / `gunzip` | 87.3% | 4 | ✅ 4/4 | ✅ |
| `sha256sum` | 100.0% | — | — | ✅ |
| `sha1sum` | 100.0% | 1 | ✅ 1/1 | ✅ |
| `sha512sum` | 100.0% | — | — | ✅ |
| `sha3sum` | 100.0% | 2 | ✅ 2/2 | ✅ |
| `md5sum` | 100.0% | 2 | ✅ 2/2 | ✅ |
| `diff` | 90.3% | 20 | ✅ 20/20 | ✅ |
| `test` / `[` | 92.7% | — | — | ✅ |
| `printf` | 89.7% | 26 | ✅ 26/26 | ✅ |
| `expr` | 86.8% | 2 | ✅ 2/2 | ✅ |
| `awk` | 90.0% | 53 | ⚠️ 36/53 (17 fail, deferred) | ✅ |
| `shell` | 95.6% | — | — | ✅ |
| `wget` | 86.2% | 4 | ✅ 4/4 | ✅ |

## Tier 6 — Post-MVP (Phase 15–16, 18.3)

| Utility | Unit Coverage | BusyBox Tests | BusyBox Status | JSON-RPC |
|---------|:------------:|:-------------:|:--------------:|:--------:|
| `dd` | 88.8% | 6 | ✅ 6/6 | ✅ |
| `od` | 85.2% | 4 | ✅ 4/4 | ✅ |
| `patch` | 82.7% | 11 | ✅ 11/11 | ⚠️ |
| `unexpand` | 84.3% | 24 | ✅ 24/24 | ✅ |
| `comm` | 88.7% | 9 | ✅ 9/9 | ✅ |
| `paste` | 88.3% | 5 | ✅ 5/5 | ✅ |
| `fold` | 91.7% | 4 | ✅ 4/4 | ✅ |
| `sum` | 100.0% | 4 | ✅ 4/4 | ✅ |
| `nl` | 97.0% | 4 | ✅ 4/4 | ✅ |
| `expand` | 84.5% | 3 | ✅ 3/3 | ✅ |
| `cmp` | 82.1% | 1 | ✅ 1/1 | ✅ |
| `strings` | 94.3% | 1 | ✅ 1/1 | ✅ |
| `seq` | 92.6% | 21 | ✅ 21/21 | ✅ |
| `cal` | 86.9% | 1 | ✅ 1/1 | ✅ |
| `factor` | 96.2% | 13 | ✅ 13/13 | ✅ |


## Tier 7 — Stubs (Phase 17, in-progress)

| Utility | Unit Coverage | BusyBox Tests | BusyBox Status | JSON-RPC |
|---------|:------------:|:-------------:|:--------------:|:--------:|
| `cksum` | 94.4% | — | — | ✅ |
| `join` | 89.6% | — | — | ✅ |
| `link` | 100.0% | — | — | ✅ |
| `unlink` | 100.0% | — | — | ✅ |
| `logger` | 98.5% | — | — | ✅ |
| `logname` | 100.0% | — | — | ✅ |
| `mkfifo` | 100.0% | — | — | ✅ |
| `nice` | 90.2% | — | — | ✅ |
| `nohup` | 93.5% | — | — | ✅ |
| `split` | 92.5% | — | — | ✅ |
| `tty` | 100.0% | — | — | ✅ |
| `who` | 89.1% | — | — | ✅ |
| `daemon` | 93.8% | — | — | —¹ |

¹ The `daemon` command manages the daemon process itself. `--json` and JSON-RPC do not apply to it (documented exemption, Phase 28 F15).

## Tier 8 — Phase 26 Tier 4 + Phase 27 (High-Complexity & Privileged)

| Utility | Unit Coverage | BusyBox Tests | BusyBox Status | JSON-RPC |
|---------|:------------:|:-------------:|:--------------:|:--------:|
| `bunzip2` | 100.0% | 11 | ✅ 11/11 | ✅ |
| `bzcat` | 100.0% | 3 | ✅ 3/3 | ✅ |
| `unlzma` | 100.0% | 3 | ✅ 3/3 | ✅ |
| `uncompress` | 100.0% | 1 | ✅ 1/1 | ✅ |
| `unzip` | 82.1% | 4 | ✅ 4/4 | ✅ |
| `uuencode` | 88.8% | 19 | ✅ 19/19 | ✅ |
| `uudecode` | 84.3% | — | — | ✅ |
| `taskset` | 86.9% | 3 | ✅ 3/3 | ✅ |
| `start-stop-daemon` | 81.0% | 4 | ✅ 4/4 | ✅ |
| `cryptpw` | 82.9% | 7 | ✅ 7/7 | ✅ |
| `makedevs` | 87.8% | 1 | ⚠️ 0/1 (1 skip) | ⚠️ skip |
| `ar` | 81.6% | 2 | ✅ 2/2 | ✅ |
| `cpio` | 83.4% | 2 | ✅ 2/9 (7 skip) | ✅ |
| `ash` | — | 0 | ⚠️ 0/1 (1 skip) | ⚠️ skip |
| `mount` | 80.4% | 0 | ⚠️ 0/1 (1 skip) | ⚠️ skip |
| `mdev` | 87.5% | 0 | ⚠️ 0/12 (12 skip) | ⚠️ skip |
| `dc` | 89.3% | 36 | ✅ 36/36 | ✅ |
| `rx` | 89.1% | 1 | ✅ 1/1 | ✅ |
| `hexdump` | 84.6% | 3 | ✅ 3/3 | ✅ |
| `xxd` | 86.3% | 7 | ✅ 7/7 | ✅ |
| `bc` | 84.8% | 81 | ✅ 81/81 | ✅ |
| `mkfs.minix` | 87.8% | 1 | ✅ 1/1 | ✅ |
## Infrastructure

| Utility | Unit Coverage | BusyBox Tests | BusyBox Status | JSON-RPC |
|---------|:------------:|:-------------:|:--------------:|:--------:|
| `daemon` | 93.8% | — | — | —¹ |

¹ The `daemon` command manages the daemon process itself. `--json` and JSON-RPC do not apply to it (documented exemption, Phase 28 F15).

| Suite | Count | Status |
|-------|-------|--------|
| Total packages | 115 | 115 utilities |
| Unit tests passing | 115/115 | 100% |
| BusyBox tests run | 919 | 919 total applicable tests |
| BusyBox passed | 877 | 98.1% (877 of 919) |
| BusyBox failed | 16 | 16 awk (deferred) |
| BusyBox skipped | 25 | 13 mdev (root), 7 cpio, 2 mount/makedevs (root), 1 ash, 2 awk (deferred) |
| Overall statement coverage | 84.1% | Checked via make cover-gate |
| JSON-RPC daemon tests | 115/115 | 100.0% (all 115 utilities implemented and registered) |
| Packages below 70% unit coverage | 0 | None (all packages ≥70%) |
## Remaining Gaps

See [todos.md](todos.md) for the canonical list of remaining work:

- awk: 17 BusyBox failures (deferred — goawk v1.31.0 engine limitations)
- Coverage: 13 packages below 80% (blocked by syscall/I/O error mocking)
- Alpine daemon target: planning


## Notes

- **bc**: All 81 BusyBox tests pass (100% compliance rate). Replaced the complex native big.Float math routines with the fully standard Gavin Howard/POSIX math library parsed and executed dynamically by the interpreter, achieving absolute precision-scale compatibility. Unit test coverage reached 80.9%. ✅
- **ar**: Archive creation now passes all BusyBox tests (2/2). Feature flags enabled. ✅
- **unzip**: Corrupted archive handling passes all BusyBox tests (4/4). Added `scanCorruptedZip()` for local file header extraction from damaged zips. ✅
- **tree**: All 4 BusyBox tests pass including Unicode box-drawing output. ✅
- **tar**: All 31 BusyBox tests now passing (100% compliance). Symlink safety with pre-scan conflict detection, hardlink dedup for symlinks, XZ compression auto-detect. ✅
- **dc**: All 36 BusyBox tests pass (100% compliance rate). Fixed recursive macro stack overflow, scale-aware modulus/divmod operations, and mathematical zero formatting quirks. Added full support for multi-character extended register mode (`-x`). Unit test coverage reached 87.8%. ✅
- **pidof**: All 4 tests pass including `-o init` (FEATURE_PIDOF_OMIT enabled). ✅
- **cryptpw**: All 7 tests pass including SHA-256/512 with rounds (USE_BB_CRYPT_SHA flag enabled). Unit coverage increased 80.6% → 82.4% with 6 new test functions. ✅
- **realpath**: All 10 BusyBox tests pass (previously 3 failures — resolved). ✅
- **rx**: The intermittent flakiness in the XMODEM integration test was traced back to GoPOSIX's `hexdump` buffering partial reads and splitting outputs across lines. Hardened `hexdump` to buffer standard input to `blockSize` using `io.ReadFull`. Extended `rx` unit tests with extensive error path tests raising unit statement coverage from 72.4% to 86.2%. ✅
- **Coverage gate:** CI enforces ≥80% overall (run `make cover-gate` for current)
- **JSON-RPC alias coverage added:** `egrep`, `fgrep` (grep aliases), `gunzip` (gzip alias) tested via daemon.
- **Phase 26/27 compliance tests:** 28 `test/compliance/test_<name>.sh` scripts written. 84 assertions, 0 failures.
