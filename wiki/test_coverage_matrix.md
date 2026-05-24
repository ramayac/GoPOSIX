# GoPOSIX — Test Coverage & Compliance Matrix

> **Last updated:** 2026-05-23 | **BusyBox:** 788 pass / 52 fail / 53 skip | **Branch:** `feat/more-tools` | **Overall Coverage:** 82.3% | **JSON-RPC:** 115/115 (100.0%)
>
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
| `echo` | 97.8% | 11 | ✅ 11/11 | ✅ |
| `true` / `false` | 100.0% | 4 | ✅ 4/4 | ✅ |
| `yes` | 80.0% | — | — | ✅ |
| `whoami` | 78.9% | — | — | ✅ |
| `hostname` | 78.2% | 4 | ✅ 4/4 | ✅ |
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
| `mkdir` | 85.3% | 2 | ✅ 2/2 | ✅ |
| `rmdir` | 92.6% | 1 | ✅ 1/1 | ✅ |
| `rm` | 87.3% | 1 | ✅ 1/1 | ✅ |
| `cp` | 77.6% | 14 | ✅ 14/14 | ✅ |
| `mv` | 84.0% | 14 | ✅ 14/14 | ✅ |
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
| `tr` | 90.4% | 6 | ✅ 6/6 | ✅ |
| `cut` | 90.8% | 25 | ✅ 25/25 | ✅ |
| `tee` | 73.1% | 2 | ✅ 2/2 | ✅ |
| `grep` | 84.8% | 53 | ✅ 53/53 | ✅ |
| `sed` | 80.1% | 103 | ✅ 103/103 | ✅ |
| `rev` | 94.7% | 4 | ✅ 4/4 | ✅ |
| `tsort` | 84.3% | 20 | ✅ 20/20 | ✅ |

## Tier 4 — System & Process

| Utility | Unit Coverage | BusyBox Tests | BusyBox Status | JSON-RPC |
|---------|:------------:|:-------------:|:--------------:|:--------:|
| `ps` | 84.6% | — | — | ✅ |
| `kill` | 73.1% | — | — | ✅ |
| `sleep` | 78.1% | — | — | ✅ |
| `date` | 79.3% | 7 | ✅ 7/7 | ✅ |
| `uptime` | 88.5% | 1 | ✅ 1/1 | ✅ |
| `id` | 87.1% | 4 | ✅ 4/4 | ✅ |
| `chmod` | 68.3% | — | — | ✅ |
| `chown` | 71.8% | — | — | ✅ |
| `chgrp` | 70.0% | — | — | ✅ |
| `df` | 79.2% | — | — | ✅ |
| `du` | 83.9% | 6 | ✅ 6/6 | ✅ |
| `find` | 89.5% | 13 | ✅ 13/13 | ✅ |
| `xargs` | 94.1% | 12 | ✅ 12/12 | ✅ |
| `pidof` | 92.6% | 4 | ✅ 3/4 (1 skip) | ✅ |

## Tier 5 — Advanced / Agent Features

| Utility | Unit Coverage | BusyBox Tests | BusyBox Status | JSON-RPC |
|---------|:------------:|:-------------:|:--------------:|:--------:|
| `tar` | 69.4% | 18 | ✅ 18/18 | ✅ |
| `gzip` / `gunzip` | 64.7% | 4 | ✅ 4/4 | ✅ |
| `sha256sum` | 81.6% | — | — | ✅ |
| `sha1sum` | 89.1% | 1 | ✅ 1/1 | ✅ |
| `sha512sum` | 89.1% | — | — | ✅ |
| `sha3sum` | 89.4% | 2 | ✅ 2/2 | ✅ |
| `md5sum` | 79.6% | 2 | ✅ 2/2 | ✅ |
| `diff` | 73.9% | 20 | ✅ 20/20 | ✅ |
| `test` / `[` | 82.9% | — | — | ❌ |
| `printf` | 83.7% | 26 | ✅ 26/26 | ✅ |
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
| `comm` | 88.8% | 9 | ✅ 9/9 | ✅ |
| `paste` | 88.5% | 5 | ✅ 5/5 | ✅ |
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
| `join` | 80.6% | — | — | ✅ |
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

## Tier 8 — Phase 26 Tier 4 + Phase 27 (High-Complexity & Privileged)

| Utility | Unit Coverage | BusyBox Tests | BusyBox Status | JSON-RPC |
|---------|:------------:|:-------------:|:--------------:|:--------:|
| `bunzip2` | 82.7% | 11 | ✅ 11/11 | ✅ |
| `bzcat` | 90.6% | 3 | ✅ 3/3 | ✅ |
| `unlzma` | 83.3% | 3 | ✅ 3/3 | ✅ |
| `uncompress` | 84.1% | 1 | ✅ 1/1 | ✅ |
| `unzip` | 87.5% | 1 | ⚠️ 1/4 (3 skip) | ✅ |
| `uuencode` | 88.3% | 19 | ✅ 19/19 | ✅ |
| `uudecode` | 80.5% | — | — | ✅ |
| `taskset` | 86.4% | 3 | ✅ 3/3 | ✅ |
| `start-stop-daemon` | 82.1% | 4 | ✅ 4/4 | ✅ |
| `cryptpw` | 80.6% | 3 | ⚠️ 3/7 (4 skip) | ✅ |
| `makedevs` | 87.3% | 1 | ⚠️ 0/1 (1 skip) | ⚠️ skip |
| `ar` | 80.0% | 0 | ⚠️ 0/23 (23 skip) | ✅ |
| `cpio` | 79.4% | 2 | ⚠️ 0/9 (2 fail, 7 skip) | ✅ |
| `ash` | — | 0 | ⚠️ 0/1 (1 skip) | ⚠️ skip |
| `mount` | 80.6% | 0 | ⚠️ 0/1 (1 skip) | ⚠️ skip |
| `mdev` | 87.4% | 0 | ⚠️ 0/12 (12 skip) | ⚠️ skip |
| `dc` | 90.3% | 3 | 🟡 3/13 (dc testsuite: 3 ✓ / 10 wrapping+scale diffs) | ✅ |
| `rx` | 72.4% | 0 | ⚠️ 0/1 (1 skip) | ⚠️ skip |
| `hexdump` | 83.6% | 3 | ✅ 3/3 | ✅ |
| `xxd` | 86.4% | 7 | ✅ 7/7 | ✅ |
| `bc` | 64.3% | 81 | ⚠️ 49/81 (32 fail) | ✅ |
| `mkfs.minix` | 82.5% | 1 | ⚠️ 0/1 (1 fail: od -i) | ✅ |
## SDK / Client Library

| Utility | Unit Coverage | BusyBox Tests | BusyBox Status | JSON-RPC |
|---------|:------------:|:-------------:|:--------------:|:--------:|
| `client` | 76.6% | — | — | — |

## Infrastructure

| Utility | Unit Coverage | BusyBox Tests | BusyBox Status | JSON-RPC |
|---------|:------------:|:-------------:|:--------------:|:--------:|
| `daemon` | 82.4% | — | — | ❌ |

---

## Summary

| Suite | Count | Status |
|-------|-------|--------|
| Total packages | 113 | 112 utilities + client SDK |
| Unit tests passing | 113/113 | 100% |
| BusyBox tests run | 893 | 893 total applicable tests |
| BusyBox passed | 788 | 88.2% (788 of 893) |
| BusyBox failed | 52 | 16 awk + 2 cpio + 1 pidof + 32 bc + 1 mkfs.minix (harness uses od -i) |
| BusyBox skipped | 53 | External deps (bzip2, xz, uudecode, tar, tree unicode, pidof init, ar needs system ar, mount/mdev need root, etc.) |
| Overall statement coverage | 82.3% | Checked via make cover-gate |
| JSON-RPC daemon tests | 115/115 | 100.0% (all 115 utilities implemented and registered) |
| Packages below 70% unit coverage | 4 | `tty` (60.0%), `gzip` (64.7%), `tar` (69.4%), `bc` (64.3%) |

## Remaining Gaps

| # | Gap | Count |
|---|-----|-------|
| 1 | awk BusyBox failures | 16 (goawk v1.31.0 limitations) |
| 2 | cpio BusyBox failures | 2 (block count output not emitted by cavaliergopher/cpio) |
| 3 | pidof BusyBox failure | 1 (exit code mismatch in test env) |
| 4 | bc BusyBox failures | 32 (formatting and precision/scale differences) |
| 5 | mkfs.minix BusyBox failure | 1 (harness uses od -i which GoPOSIX od doesn't support) |
| 6 | Unit coverage < 80% | 2 packages: `cpio` (79.4%), `bc` (64.3%) |

## Notes

- **BusyBox skipped (10):** All tar tests requiring bzip2/xz/uudecode (external deps)
- **Coverage gate:** CI enforces ≥80% overall (run `make cover-gate` for current; target ≥80% per Phase 28)
- **Tier 7 stubs:** Implemented as functional stubs; need hardening and BusyBox-style compliance tests
- **Phase 26 (Tiers 1–4):** 25 utilities + 1 companion (`uudecode`) implemented. Tier 1: `which`, `realpath`, `seq`, `sha1sum`, `sha512sum`. Tier 2: `rev`, `uptime`, `wget`, `cal`. Tier 3: `hostid`, `factor`, `sha3sum`, `tree`, `tsort`, `pidof`. Tier 4: `bunzip2`, `bzcat`, `unlzma`, `uncompress`, `unzip`, `uuencode`, `uudecode`, `taskset`, `start-stop-daemon`, `cryptpw`, `makedevs`.
- **Phase 27 (Tier 5):** 11 of 11 implemented: `ar`, `cpio`, `ash`, `mount`, `mdev`, `dc`, `rx`, `hexdump`, `xxd`, `bc`, `mkfs.minix`.
- **Phase 26/27 JSON-RPC tests:** 31 new daemon tests added in `test/posix-json/tier8_phase26_27_test.go` (25 running + 6 skipped). Includes `dc` add + complex.
- **JSON-RPC alias coverage added:** `egrep`, `fgrep` (grep aliases), `gunzip` (gzip alias) now tested via daemon.
- **Phase 26/27 compliance tests:** 28 `test/compliance/test_<name>.sh` scripts written. 84 assertions, 0 failures. 1 test skipped (uncompress needs system `compress`).
- **Phase 28 (feat/coverage-10):** Added 60+ new unit tests covering CLI glue layers (`run()`), infrastructure (dispatch, flags, filepath), utility edge cases (date, printf, wc, expr, diff, sort, tar), and observability. Overall coverage: 77.9% → 80.1%. Key wins: `true/false` 75% → 100%, `wc` 81.2% → 93.2%, dispatch 100%, flags 100%, filepath 100%, `printf` 65.6% → 79.0%, `sort` 82.5% → 85.2%. CI gate raised from 70% → 80%. Remaining gaps: `main()` (os.Exit), `client_helpers` (needs daemon), platform-specific code (`setProcTitle`, `RunDaemon`).
- **Phase 28.5 (feat/coverage-85):** Added 35+ new tests across Phase A (xargs 2→12, paste +5, join +5, tr +6, hostname +4) and Phase B (mkdir +3, mv +4, cp +1, diff +3, comm +6). Overall: 80.1% → 80.9%. Key wins: `xargs` 65.7% → 74.5%, `comm` 79.4% → 81.0%, `join` 76.8% → 78.0%. All 93 packages green. Next targets: `client_helpers` (131 blocks at 0%), `sed/execFlat` (36.7%), `tar` (104 uncovered). See [wiki/todos.md](todos.md) for full plan.
- **Phase 28.6 (feat/coverage-85 Phase D):** Added 20 new tests: sed (a/i/c/q/N/n/D/P/T/w commands, SubNum, \ delimiter, $ addr, +N range), tar (extract to stdout, verbose list), cp (symlink copy), date (last-week eval, non-leap Julian, complex TZ), printf (%c, %-width, %0pad). Overall: 81.5% → 82.2%. All 93 packages green.
- **Phase 28.7 (feat/coverage-85 Phase E):** Added 16 tests: date formatDate (%e/%I/%m/%S/%y/%T), parseDateString compact/sec/@epoch/time-only/Zulu/invalid; printf %*d/%.*f/%5d/%.5d/%e/length-mods/exhausted-args. Overall: 82.2% → 82.4%. All 93 green.
