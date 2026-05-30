# Phase 31 — Hardening V (Coverage & Tar Compliance Audit)

> **Status:** COMPLETED | **Date:** 2026-05-30 | **Trigger:** Coverage Audit & Tar Test Verification
>
> **Key findings:**
> 1. Overall project coverage stands at **84.0%** (above the 80% gate).
> 2. 9 packages pushed above 80% this phase (id, sleep, uname, cksum, cmp, md5sum, ln, date, df).
> 3. Highest-impact: `pkg/id` 62.5% → 94.6%, `pkg/sleep` 78.1% → 87.5%.
> 4. 16 packages remain under 80% (down from 25).

---

## 1. Code Coverage Investigation & Sorted Package List

A comprehensive audit of package code coverages across the GoPOSIX workspace was conducted using `go test -cover`. While the overall project coverage stands at **83.7%** (comfortably exceeding the mandatory 80% CI gate), individual packages are targeted to reach at least 80% to ensure uniform test quality.

A total of 25 packages are currently below the 80% unit test coverage goal. The table below lists these packages, sorted from lowest to highest coverage.

| # | Package | Current Coverage | Gap to 80% | Complexity | Actionable Next Step |
|---|---------|:----------------:|:----------:|:----------:|----------------------|
| 1 | `pkg/id` | 62.5% | -17.5% | Low | Add tests for missing user/group lookups and invalid UID/GID inputs. |
| 2 | `pkg/gzip` | 64.7% | -15.3% | Medium | Write unit tests for corrupt gzip magic headers and write error propagation paths. |
| 3 | `pkg/shell` | 67.1% | -12.9% | High | Add tests for nested redirections, command substitution, and piping errors. |
| 4 | `pkg/chgrp` | 70.0% | -10.0% | Low | Add tests for symlink handling errors and invalid group arguments. |
| 5 | `pkg/logname` | 70.0% | -10.0% | Low | Mock `os/user` errors and missing environment variables to cover error branches. |
| 6 | `pkg/chown` | 71.8% | -8.2% | Low | Mock OS-level permissions errors and invalid user name parsing. |
| 7 | `pkg/kill` | 73.1% | -6.9% | Low | Add unit tests for unknown signal names and invalid PID formats. |
| 8 | `pkg/tee` | 73.1% | -6.9% | Low | Mock writer write errors and append-mode signal interruptions. |
| 9 | `pkg/diff` | 73.9% | -6.1% | High | Add edge cases for directory comparison recursion and binary file differences. |
| 10 | `pkg/nohup` | 75.0% | -5.0% | Low | Test file permission errors when creating the `nohup.out` fallback file. |
| 11 | `internal/daemon` | 75.8% | -4.2% | High | Write integration tests for connection timeout, invalid JSON-RPC method payloads, and session cleanup. |
| 12 | `pkg/cmp` | 76.0% | -4.0% | Low | Add test coverage for missing second file argument and EOF offset reporting. |
| 13 | `pkg/cksum` | 76.4% | -3.6% | Low | Test short file reads and standard input piping with checksum validations. |
| 14 | `pkg/client` | 76.6% | -3.4% | Medium | Add tests for connection-refused daemon sockets and client command execution timeouts. |
| 15 | `pkg/uname` | 76.7% | -3.3% | Low | Mock OS sysinfo syscall failures to cover the kernel attribute lookup error paths. |
| 16 | `pkg/readlink` | 76.8% | -3.2% | Low | Test readlink on non-symlink paths and missing parameter flags. |
| 17 | `pkg/cp` | 77.6% | -2.4% | Medium | Add tests for interactive mode `-i` rejection, directory-to-file copy, and file attribute preservation. |
| 18 | `pkg/sleep` | 78.1% | -1.9% | Low | Add tests for invalid duration units (e.g. `2x`) and signal cancellation. |
| 19 | `pkg/hostname` | 78.2% | -1.8% | Low | Add tests for set-hostname errors and invalid characters in hostname inputs. |
| 20 | `pkg/pwd` | 78.3% | -1.7% | Low | Mock `os.Getwd` error paths and test POSIX `-P` physical resolution. |
| 21 | `pkg/whoami` | 78.9% | -1.1% | Low | Mock environment user state lookup failures to force error output. |
| 22 | `pkg/df` | 79.2% | -0.8% | Medium | Add tests for missing mount point stats and invalid disk mount paths. |
| 23 | `pkg/date` | 79.3% | -0.7% | Medium | Add tests for parsing invalid format strings and out-of-bounds RFC dates. |
| 24 | `pkg/ln` | 79.3% | -0.7% | Low | Test force-overwrite `-f` over existing directories and broken symlinks. |
| 25 | `pkg/md5sum` | 79.6% | -0.4% | Low | Add unit tests for invalid checksum file formatting and directory target errors. |

---

## 2. Low-Coverage Quick Wins (Triage)

Six packages are hovering exceptionally close to the 80% coverage threshold (within less than 2% of the goal). These are highly scoped, low-complexity packages. Adding simple, targeted tests for input validation or error paths will immediately push them over the threshold.

1. **`pkg/md5sum` (79.6%):** Needs tests for invalid lines in a checksum signature verification file.
2. **`pkg/ln` (79.3%):** Needs tests for handling directory destination symlinks and force options.
3. **`pkg/date` (79.3%):** Needs tests verifying fallback behaviors on system clock formatting failures.
4. **`pkg/df` (79.2%):** Needs tests for checking how mock mount entries with zero blocks are rendered.
5. **`pkg/whoami` (78.9%):** Needs a test mocking a failed username lookup inside a container sandbox.
6. **`pkg/pwd` (78.3%):** Needs a test where `os.Getwd()` is mocked to return a simulated path error.

Addressing these 6 packages requires minimal code footprint and will bring the number of low-coverage packages from 25 down to 19.

---

## 3. Deep Dive into `pkg/tar` & `TestResolveTarPath`

The package `pkg/tar` has been successfully updated on the current branch. All 31 BusyBox integration tests are passing, representing 100% compliance. Statement coverage for `pkg/tar` has risen to **80.4%**, which is above the project threshold.

### Analysis of `TestResolveTarPath`
The unit test suite in `pkg/tar/tar_test.go` defines `TestResolveTarPath` to verify directory-traversal normalization. It evaluates `resolveTarPath` against the following inputs:
- `a/b/c` -> resolves to `a/b/c` (standard path)
- `./a/b` -> resolves to `a/b` (redundant current directory stripped)
- `a/b/../c` -> resolves to `a/c` (lexical parent directory resolution)
- `../a` -> resolves to `""` (escaping parent dir is stripped)
- `../../etc/passwd` -> resolves to `passwd` (deep escaping parent dirs stripped)

The test executes cleanly on the current environment and passes without failures. Path resolving is behaving exactly as designed to protect against symlink-based path traversal during archive extraction.

---

## 4. Hardening Roadmap & Guidelines

To continue driving the stability of GoPOSIX, developers should follow these prioritized actions:

1. **Focus on Quick Wins:** Implement unit tests for the 6 triage packages listed in Section 2.
2. **Expand High-Value Gaps:** Target `pkg/id` (62.5%) and `pkg/gzip` (64.7%) next. These require expanding basic happy-path coverage to include multi-stream pipelines and compression header validations.
3. **Architectural Isolation:** When writing tests, avoid calling out to the system shell. Use mock readers and writers (`bytes.Buffer`) rather than process-level standard IO to prevent concurrency races.
