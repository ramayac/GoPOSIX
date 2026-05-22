# Phase 26 — Missing BusyBox Tools Analysis

> **Version:** 5.6 | **Date:** 2026-05-22 | **Tier:** GOLD | **Status:** COMPLETED
>
> **Analysis:** 35 Unimplemented Utilities | 53 Implemented & Tested Utilities | 29 Implemented Utilities Without BusyBox Tests

This document presents a comprehensive, systematic audit of the **BusyBox test suite** located in `test/busybox_testsuite` against the current registered commands and implemented packages in **GoPOSIX**. 

The goal of this analysis is to identify gaps in our POSIX userland implementation, catalog all missing utilities that possess existing test coverage in the BusyBox suite (such as `wget`), and lay the groundwork for targeted feature parity expansion in the `feat/missing-tools` branch.

---

## Strict Category H Implementation Workflow

We follow a strict, systematic process for implementing the utilities in **Category H** (General Utilities & Miscellaneous) one by one. The development loop is:

```
  [ CHECK ] ──> [ TEST ] ──> [ CODE ] ──> [ PASS ] ──> [ UPDATE ]
     │             │            │            │             │
  Inspect      Create our   Write Go     Run unit &    Mark utility
  BusyBox      own unit     logic &      BusyBox       implemented
  tests        tests        register     tests         in wiki
```

1. **CHECK**: Inspect the corresponding BusyBox test files in `test/busybox_testsuite` (e.g. `cal.tests` or the `which/` directory) to identify exact command flags, behaviors, and expected outputs.
2. **TEST**: Before writing any implementation code, create our own robust unit tests under `pkg/<utility>/<utility>_test.go` utilizing injectable standard streams. The tests must replicate the behaviors checked by BusyBox as well as standard edge cases.
3. **CODE**: Write the package code (`pkg/<utility>/<utility>.go`). Implement logic using pure Go, no CGO, using custom unified flag parsing (`common.ParseFlags`) and standardized output rendering (`common.Render` / `common.RenderError`). Register the subcommand in `init()` via `dispatch.Register()`.
4. **PASS**: Verify that all unit tests pass (enforcing ≥ 80% coverage on the new package) and the BusyBox integration tests also pass.
5. **UPDATE**: Update this document (`wiki/26_missing_tools.md`) to mark the completed utility as `[x] Implemented`, and update other wiki documentation if needed.

### Tier 1 Progress Tracker (ONE by ONE)
- `[ ]` **`which`**: Locates commands in the user's `PATH`.
- `[ ]` **`realpath`**: Resolves relative, absolute, and symlinked paths to absolute canonical paths.
- `[ ]` **`seq`**: Formatted loop printing numeric sequences.
- `[ ]` **`sha1sum` / `sha512sum`**: Computes and verifies SHA-1 and SHA-512 cryptographic digests.

---

## 1. High-Level Summary Matrix

Through programmatic cross-referencing of GoPOSIX's dispatch commands (`goposix --list-commands`) and BusyBox's test harness, we categorized all tested applets.

| Category | Count | Description |
| :--- | :---: | :--- |
| **Total BusyBox Test Suites Covered** | **88** | Active test directories or `.tests` scripts inside `test/busybox_testsuite/` |
| **Implemented & Tested** | **52** | Utilities implemented in `pkg/` and verified by BusyBox's test harness |
| **Unimplemented & Tested** | **36** | Utilities with active BusyBox tests that GoPOSIX does not yet implement |
| **Implemented Without BusyBox Tests** | **29** | GoPOSIX utilities that have custom tests but no upstream BusyBox suite files |

---

## 2. Unimplemented Utilities with Active BusyBox Tests

Below is the definitive catalog of the **36 utilities** that are tested in the BusyBox test suite but are not yet implemented in GoPOSIX. They have been grouped by functionality to assist in phased planning.

### A. Compression & Decompression (5 Utilities)
These tools are critical for file archiving and package management, complementing GoPOSIX's existing `gzip`/`gunzip` / `tar` support.
* **`bunzip2`** (Tested by `bunzip2.tests` & `bunzip2/` directory)
  * *Purpose*: Decompresses files created by `bzip2`.
  * *Feasibility*: Highly feasible using standard library or low-dependency pure-Go block decompression packages.
* **`bzcat`** (Tested by `bzcat.tests` & `bzcat/` directory)
  * *Purpose*: Decompresses `bzip2` files to standard output.
* **`unlzma`** (Tested by `unlzma.tests`)
  * *Purpose*: Decompresses files in `.lzma` format.
* **`uncompress`** (Tested by `uncompress.tests`)
  * *Purpose*: Restores files compressed by standard LZW `compress`.
* **`unzip`** (Tested by `unzip.tests`)
  * *Purpose*: Extracts files from `.zip` archives.

### B. Network Utilities (1 Utility)
* **`wget`** (Tested by `wget/` directory)
  * *Purpose*: Non-interactive network downloader.
  * *Feasibility*: Highly feasible using Go's `net/http` package.
  * > [!WARNING]
    > **Wget Internet Tests**: BusyBox tests like `wget-supports--P` and `wget-retrieves-google-index` attempt to query live endpoints (e.g., `http://www.google.com/`). When implementing this utility, these tests must handle offline environments gracefully or rely on mock HTTP servers to prevent CI breakage.

### C. Development, Hex & Binary Manipulation (5 Utilities)
* **`ar`** (Tested by `ar.tests`)
  * *Purpose*: Creates and maintains archive files (primarily static libraries).
* **`hexdump`** (Tested by `hexdump.tests`)
  * *Purpose*: Displays file contents in hexadecimal, decimal, octal, or ASCII.
* **`xxd`** (Tested by `xxd.tests`)
  * *Purpose*: Creates a hex dump of a given file or standard input.
* **`uuencode`** (Tested by `uuencode.tests`)
  * *Purpose*: Encodes binary files for transmission over 7-bit channels.
* **`rx`** (Tested by `rx.tests`)
  * *Purpose*: Receives files using the XMODEM protocol.

### D. Mathematics & Arithmetic (4 Utilities)
* **`bc`** (Tested by `bc.tests`)
  * *Purpose*: Arbitrary-precision calculator language.
  * *Feasibility*: High complexity. Requires a recursive descent parser and custom multi-precision arithmetic.
* **`dc`** (Tested by `dc.tests`)
  * *Purpose*: Reverse-Polish (stack-based) desk calculator.
* **`factor`** (Tested by `factor.tests`)
  * *Purpose*: Factorizes numbers into primes.
* **`seq`** (Tested by `seq.tests`)
  * *Purpose*: Prints a sequence of numbers (e.g., `1 1 10`). Highly requested and simple to write.

### E. Cryptographic Checksums (3 Utilities)
These tools supplement our existing `md5sum` and `sha256sum` commands.
* **`sha1sum`** (Tested by `sha1sum.tests`)
  * *Purpose*: Computes and verifies SHA-1 cryptographic digests.
* **`sha3sum`** (Tested by `sha3sum.tests`)
  * *Purpose*: Computes and verifies SHA-3 digests.
* **`sha512sum`** (Tested by `sha512sum.tests`)
  * *Purpose*: Computes and verifies SHA-512 digests.
  * *Feasibility*: Trivial to implement using Go's `crypto/sha1` and `golang.org/x/crypto/sha3`.

### F. Shell & Process Management (2 Utilities)
* **`ash`** (Tested by `ash.tests`)
  * *Purpose*: The Debian/BusyBox standard shell.
  * *Note*: GoPOSIX implements its shell via `pkg/shell` and `mvdan.cc/sh`, but is currently registered as `shell` and aliased to `sh`. The specific test suite `ash.tests` expects the command to respond to `ash`.
* **`pidof`** (Tested by `pidof.tests`)
  * *Purpose*: Finds the process ID of a running program by name.
  * *Feasibility*: Requires reading process trees (`/proc` on Linux).

### G. System Administration & Hardware (7 Utilities)
* **`cpio`** (Tested by `cpio.tests`)
  * *Purpose*: Copies files to and from archives.
* **`cryptpw`** (Tested by `cryptpw.tests`)
  * *Purpose*: Hashes passwords using standard Unix crypt algorithms.
* **`hostid`** (Tested by `hostid/` directory)
  * *Purpose*: Prints the numeric identifier for the current host.
* **`makedevs`** (Tested by `makedevs.tests`)
  * *Purpose*: Creates device blocks and nodes from a device table file.
* **`mdev`** (Tested by `mdev.tests`)
  * *Purpose*: BusyBox micro-udev device manager daemon.
* **`mkfs.minix`** (Tested by `mkfs.minix.tests`)
  * *Purpose*: Creates a Minix filesystem in a block device.
* **`mount`** (Tested by `mount.tests`)
  * *Purpose*: Mounts standard filesystems. Requires root privileges for full testing.

### H. General Utilities & Miscellaneous (9 Utilities)
* **`cal`** (Tested by `cal.tests`)
  * *Purpose*: Renders an ASCII calendar for a given month/year.
* **`realpath`** (Tested by `realpath.tests`)
  * *Purpose*: Resolves relative, absolute, and symlinked paths to absolute canonical paths.
  * *Feasibility*: Trivial in Go using `filepath.EvalSymlinks` and `filepath.Abs`.
* **`rev`** (Tested by `rev.tests`)
  * *Purpose*: Reverses the character order of lines in a file.
* **`start-stop-daemon`** (Tested by `start-stop-daemon.tests`)
  * *Purpose*: System V-style daemon management tool.
* **`taskset`** (Tested by `taskset.tests`)
  * *Purpose*: Sets or retrieves CPU affinity for processes.
* **`tree`** (Tested by `tree.tests`)
  * *Purpose*: Displays directories in an indented tree diagram.
* **`tsort`** (Tested by `tsort.tests`)
  * *Purpose*: Performs a topological sort on standard input.
* **`uptime`** (Tested by `uptime/` directory)
  * *Purpose*: Displays how long the system has been running, user count, and load averages.
* **`which`** (Tested by `which/` directory)
  * *Purpose*: Locates commands in the user's `PATH`.

---

## 3. Implemented Utilities Lacking BusyBox Test Coverage

The following **29 utilities** are successfully implemented in GoPOSIX but **do not** have corresponding test suites in `test/busybox_testsuite/`. These utilities rely solely on GoPOSIX's unit tests (`*_test.go`) and compliance scripts (`test/compliance/`):

1. `chgrp`
2. `chmod`
3. `chown`
4. `cksum`
5. `daemon` (custom)
6. `df`
7. `egrep` (aliased to grep)
8. `env`
9. `fgrep` (aliased to grep)
10. `join`
11. `kill`
12. `link`
13. `logger`
14. `logname`
15. `mkfifo`
16. `nice`
17. `nohup`
18. `printenv`
19. `ps`
20. `shell` / `sh`
21. `sleep`
22. `split`
23. `stat`
24. `tty`
25. `uname`
26. `unlink`
27. `who`
28. `whoami`
29. `yes`

---

## 4. Recommendations & Implementation Plan

For the new `feat/missing-tools` branch, we recommend implementing the missing utilities in tiers based on their implementation simplicity and value:

### Tier 1: Trivial & Quick Wins (4 Utilities)
These can be implemented in single-session tasks and quickly validated:
1. **`which`**: Inspects `os.Getenv("PATH")`, checks executable permissions.
2. **`realpath`**: Wrapped around standard library `filepath.EvalSymlinks`.
3. **`seq`**: Formatted loop printing numeric sequences.
4. **`sha1sum` / `sha512sum`**: Modeled directly after the existing `sha256sum` package.

### Tier 2: Mid-Level Complexity (3 Utilities)
1. **`wget`**: Core downloader leveraging Go's robust `net/http` stack, with mock server compliance testing.
2. **`uptime`**: Parses `/proc/uptime` or calls standard sysinfo on Linux.
3. **`rev`**: Line buffer reverser.

### Tier 3: Complex Utilities
* **`bc` / `dc` / `hexdump` / `xxd`**: Requires planning for dedicated parsing and output layouts.
