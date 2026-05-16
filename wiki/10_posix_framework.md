# Phase 10 — POSIX Testing Framework

> **HISTORICAL — COMPLETED.** This phase is done. Document retained for reference.
>
> **Timeline:** Future | **Depends on:** Phase 09

---

## Goal

Integrate a formal POSIX testing framework to programmatically verify KoreGo's compliance against the IEEE Std 1003.1 spec, rather than relying solely on our internal bash compliance scripts.

## Rationale

Previously, `test/compliance/` bash scripts compared KoreGo's output to the host system's GNU Coreutils. Those scripts have been removed — the BusyBox test suite (`make testsuite`, 479+ tests at 97.9% pass rate) now serves as the primary external compliance check. GNU Coreutils often implements extensions beyond POSIX; the BusyBox suite provides a more appropriate baseline for a minimal multi-call binary.

## Framework Options

### 1. The Open Group POSIX Test Suite (VSTS)
- The official, historical test suite for POSIX compliance.
- **Pros:** The gold standard.
- **Cons:** Very old, hard to compile on modern systems, proprietary/licensed portions exist.

### 2. FreeBSD / NetBSD Test Suites (`kyua` / `atf`)
- BSD implementations are generally closer to POSIX than GNU. 
- **Pros:** Open source, robust, actively maintained.
- **Cons:** Some BSD-isms may still exist.

### 3. BusyBox Test Suite
- BusyBox has an extensive `testsuite/` directory that verifies utility behavior.
- **Pros:** Designed for minimal, multi-call binaries (very similar to KoreGo).
- **Cons:** Focuses on BusyBox behavior, which sometimes deviates from strict POSIX to save space.

### 4. Custom TAP-based Test Harness
- Build a Go-based or Bash-based Test Anything Protocol (TAP) runner that executes defined POSIX inputs and asserts outputs.
- **Pros:** Full control, easy to integrate into `go test`, can easily test JSON output alongside text output.
- **Cons:** High maintenance burden, requires transcribing POSIX spec into test cases.

## Selected Approach: Hybrid (BusyBox Test Suite + Custom TAP)

Given KoreGo's architecture, we will adapt the BusyBox test suite for baseline behavior and build a custom TAP-based runner to verify our specific `--json` outputs and JSON-RPC daemon interactions.

## Tasks

### 10.1 — Framework Selection and Porting
- [x] Investigate and fork the BusyBox `testsuite/` directory.
- [x] Adapt the runner to execute against the `korego` binary instead of `busybox`.
- [x] Disable tests for utilities we haven't implemented yet.

### 10.2 — Baseline Execution
- [x] Run the adapted suite.
- [x] Document all failures in [posix_coverage.md](posix_coverage.md) as known deviations.
- [x] Fix critical deviations (e.g. standard flags failing).

### 10.3 — JSON/RPC Testing Harness
- [x] Create a `test/posix-json/` directory.
- [x] Implement a Go-based TAP runner that runs POSIX commands via the JSON-RPC daemon.
- [x] Verify that the structured output semantics map correctly to POSIX expectations (e.g. exit codes, stderr vs stdout separation).

### 10.4 - Sed
- [x] Fix test/busybox_testsuite/sed.tests to pass all tests. (See [10a_sed.md](10a_sed.md) for detailed sub-tasks)

### 10.5 - busybox test step in github action
- [x] Add a step in github action to run the busybox test suite. Don't block the pipeline, just result of test that pass.

### 10.6 - Phase A: Core Utilities & Flag Parsing Test Resolution
- [x] Fixed `pkg/common/flags.go` to support concatenated values (e.g., `-s25`).
- [x] Fixed `pkg/xargs/xargs.go` to handle `-e` without a parameter.
- [x] Fixed `pkg/grep/grep.go` multiple `-e` and `-f` flags, `-f EMPTY_FILE` behavior, `-r` symlink traversal (trailing slash), and `-L` exit codes.
- [x] Fixed `pkg/find/find.go` trailing slash canonicalization by preprocessing single-dash long flags (`-name`, `-type`).
- [x] Fixed `pkg/sed/sed.go` missing labels validation and NUL bytes handling by removing the `0` EOF marker.
- [x] Fixed `pkg/echo/echo.go` to support `\0NNN` octal escapes and `\xNN` hex escapes, required by BusyBox's `sed.tests` harness.
- [x] Fixed `pkg/wc/wc.go` field formatting to align with POSIX (no leading spaces), which resolved the `tail -c +N` validation failures.

### 10.7 - Phase B: Filesystem & Diff Test Resolution
- [x] Rewrote `pkg/cp/cp.go` with `SymlinkMode` enum supporting `-d`, `-P`, `-L`, `-H` flags.
- [x] Fixed `cp -R` to preserve symlinks by default; `-L` dereferences all; `-H` dereferences command-line args only.
- [x] Fixed symlink copying to use `os.Readlink` + `os.Symlink` instead of file content copy.
- [x] Fixed `pkg/diff/diff.go` stdin reading (`-`) using `io.ReadAll(os.Stdin)`.
- [x] Implemented `diff -b` (ignore space change) via line normalization before O(ND) diff.
- [x] Implemented `diff -B` (ignore blank lines) via post-process script filtering; `differ` flag computed after filtering.
- [x] Fixed unified diff hunk range format: omit count if 1, use `start-1,0` if count is 0.
- [x] Added `\ No newline at end of file` marker after last `+`/`-` line when source file has no trailing newline.
- [x] Fixed `diff - -` (same stdin argument) to short-circuit as identical, exit 0.

> **Result:** BusyBox pass rate improved from 84% (351/413) to ~98% (405/413).

### 10.8 - Phase C: `tar` Advanced Features
- [x] `tar -X` exclude file flag (repeatable).
- [x] Stdin tarball detection and zeroed-block handling.
- [x] `tar tvf` verbose listing format matching BusyBox (owner/group, %10d size, date/time, symlink `->` notation).
- [x] `TZ` env var handling for date/time display (POSIX `UTC-2` = UTC+2).
- [x] Old-style flag preprocessing (`xvf` → `-x -v -f`).
- [x] `--overwrite` flag support.
- [x] `../` prefix stripping on extract with warning message.

### 10.9 - Phase D: `gzip` Compression Levels
- [x] Numeric compression levels (`-1` to `-9`).
- [x] `-` as stdin/stdout handling.

### 10.10 - Phase E: Round 1 — Head, Cat, Find Features (2026-05-02)
- [x] `head -n <negative>` — print all but last N lines.
- [x] `cat -v` / `cat -e` — show non-printing characters, `$` at EOL.
- [x] `cat -n` / `-b` tests enabled (already implemented).
- [x] `find -maxdepth N` — limit traversal depth.
- [x] `find -type f` test enabled (already implemented).

> **Result:** 10 skipped tests resolved (423 passed, 65 skipped).

### 10.11 - Phase F: Round 2 — Full Skipped Test Resolution (2026-05-02)
- [x] `xargs -n1` / `-n2` tests enabled (already working).
- [x] `xargs -I` replace-str flag (partial — edge cases remain).
- [x] `tr [:digit:]` / `[:xdigit:]` POSIX character classes with full class table.
- [x] `readlink -f` absolute path resolution, broken symlink handling.
- [x] `cut -DF` whitespace field splitting with correct field order.
- [x] `find -exec {} \;` and `-exec {} +` command execution.
- [x] `diff -ruN` recursive directory diff with `-r` and `-N` flags.
- [x] `md5sum -c` check mode tests enabled (already implemented).
- [x] `sort` key definition edge cases, non-default delimiters, ENDCHAR, stable+unique.
- [x] `tar --overwrite`, `../` stripping in archive creation.

> **Result:** 31 additional tests passing (454 passed, 19 failed, 15 skipped).
> **Remaining:** 10 sort advanced features, 3 diff edge cases, 2 tar, 2 xargs, 2 sort glibc.

## Milestone 10 — ✅ COMPLETE (2026-05-02)
- [x] External test suite integrated via `make testsuite` (BusyBox, 479+ tests).
- [x] CI pipeline runs the external test suite.
- [x] `posix_coverage.md` updated with programmatic results.
- [x] Phase C (`tar -X`, stdin) completed. See [todos.md](todos.md).
- [x] Phase D (`gzip` compression levels) completed. See [todos.md](todos.md).
- [x] Phase E (`head -n <negative>`, `cat -v/-e`, `find -maxdepth`) completed.
- [x] Phase F (Round 2 — full skipped test resolution) completed.
- [x] Tier 1 (xargs -I, sort -o/-z, grep NUL, md5sum, diff edge cases, head -c) completed.
- [x] Tier 2 (sort rewrite, diff dir fixes, tar ../ stripping, md5sum empty file) completed.

**Final Result:** 479 passed, 1 failed (umask-dependent), 10 skipped (external deps). 97.9% effective pass rate.
