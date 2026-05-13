# KoreGo — Open TODOs & Remaining Work

> **Last updated:** 2026-05-13 | **BusyBox pass rate:** 83.5% real (409 passed, 71 failed, 10 skipped out of 490)
>
> **Note:** The previous 97.9% / 479 passed figure was inflated. Old-style BusyBox
> tests were running against the system BusyBox on CI, not KoreGo. Fixed in [12.4](12_road_to_gold.md).
> The current numbers reflect true KoreGo behavior. See [12_road_to_gold.md](12_road_to_gold.md) for Gold roadmap.

This is the living document tracking the final remaining test failures, known deviations,
and future improvements. See [10_posix_framework.md](10_posix_framework.md) for the full
Phase 10 implementation log.

---

## Final Summary

All planned implementation phases (00–10) are complete. The BusyBox test suite
is integrated into CI and runs on every push to `main`. All old-style tests now
route through KoreGo (not system BusyBox) — resolved 2026-05-13.

**Final numbers:**

| Metric | Value |
|--------|-------|
| Tests running | 490 |
| Passed | 409 |
| Failed | 71 |
| Skipped | 10 (all external-dependency-gated: bzip2/xz/uudecode/pax) |
| **Real pass rate** | **83.5%** |
| Target utilities implemented | 49/49 (100%) |

---

## Road to 99% — Implementation Plan

**Current state:** 479 passed, 1 failed, 10 skipped out of 490 total.
**99% target:** 485 passed (need +6, requires external deps for ~5).

### ✅ Tier 1: COMPLETED

| # | Test | Status |
|---|------|--------|
| 1-2 | `xargs -I` / `xargs argument line too long` | ✅ FIXED |
| 3 | `sort file in place` (`-o`) | ✅ FIXED |
| 4 | `sort -z outputs NUL terminated` | ✅ FIXED |
| 5-7 | `grep handles NUL` (×3) | ✅ FIXED |
| 8-9 | `md5sum` (×2) | ✅ FIXED (empty checksum file edge case) |
| 10-12 | `diff` edge cases (×3) | ✅ FIXED (dir diff, path normalization, -rN) |
| 13-14 | `head -c N` | ✅ FIXED |

### ✅ Tier 2: COMPLETED (Sort + Diff + Tar overhaul)

Sort was completely rewritten with `-o`, `-s`, `-z`, `-h`, `-M`, proper `-k` key spec parsing, numeric prefix parsing, global-vs-per-key reverse semantics, and stable tiebreaker handling.
- `sort file in place` ✅ PASSES
- `sort key doesn't strip leading blanks` ✅ PASSES
- `sort one key` ✅ PASSES
- `sort -u should consider field only` ✅ PASSES
- `sort with non-default leading delim 1-4` ✅ PASSES
- `sort with ENDCHAR` ✅ FIXED (startChar default to 1 when endChar set)
- `sort -h` ✅ PASSES
- `sort -k2,2M` ✅ FIXED (month sort)
- `sort key range with *` (×4) ✅ FIXED (key spec parsing + numeric prefix)
- `sort -sr` / `sort -s -u` ✅ FIXED (stable tiebreaker)
- `sort -z` ✅ FIXED (NUL terminator output)
- `glibc build sort unique` ✅ FIXED (multi-key unique dedup)

Diff directory fixes:
- Directory diff with raw path preservation ✅ FIXED
- `-rN` non-regular file messages ✅ FIXED
- Dir+file path resolution ✅ FIXED

Tar fixes:
- `../` member name stripping ✅ FIXED
- Directory trailing `/` in verbose listing ✅ FIXED
- md5sum/sha256sum `-c EMPTY` exit code ✅ FIXED

### Remaining Failures (1)

| Area | Count | Tests |
|------|-------|-------|
| `tar` | 1 | `writing into read-only dir` — umask-dependent (expects 644, gets 664 with umask 002) |
| `sort` | 0 | All clear! 🎉 |
| `diff` | 0 | All clear! 🎉 |
| `md5sum` | 0 | All clear! 🎉 |
| `xargs` | 0 | All clear! |
| `grep` | 0 | All clear! |

### Still Skipped (10)

All 10 are `tar` tests blocked by external dependencies:
- 7 tests need bzip2/gzip/xz/uudecode
- 2 tests need PAX/Unicode support
- 1 test needs `FEATURE_TAR_CREATE` hardlink detection

### Verdict

| Metric | Value |
|--------|-------|
| Starting point | 413 passed, 75 skipped |
| After tar fix | 413 passed, 0 failed, 75 skipped |
| After Round 1 | 423 passed, 0 failed, 65 skipped |
| After Round 2 | 454 passed, 19 failed, 15 skipped (94.5%) |
| After Tier 1 | 461 passed, 19 failed, 10 skipped (94.5%) |
| **After Tier 2 (2026-05-02)** | **479 passed, 1 failed, 10 skipped (97.9%)** |

98%+ is achievable. 99% requires bzip2/xz decompression support for remaining tar tests.

The single remaining failure (`tar writing into read-only dir`) is a umask-dependent test that
passes with umask 022 but fails with umask 002 due to expected file permissions (644 vs 664).

---

## Phase C: `tar` (7 failures) — **COMPLETED (2026-05-01)**

### C.1 — `-X` Exclude File Flag ✅
**File:** `pkg/tar/tar.go`

Implemented:
- Registered `-X` as a `FlagValue` type (repeatable) in flag spec.
- `readExcludeFile()` reads each `-X` file line-by-line into an exclusion set.
- `isExcluded()` checks archive entries against exclusion patterns (supports nested paths via prefix matching).
- Supports multiple `-X` flags via `flags.GetAll("X")`.

### C.2 — Stdin tarball (`-f -`) and zeroed-block detection ✅
**File:** `pkg/tar/tar.go`

Implemented:
- Old-style tar flag preprocessing (`xvf` → `-x -v -f`).
- `bufio.Reader.Peek(1)` to detect empty streams (0 bytes → "short read", exit 1).
- Two or more 512-byte zero blocks are treated as valid empty tarball (exit 0).
- Default to stdin when no `-f` specified for extract/list modes.
- Also added: `-O` flag (extract to stdout), `-C` archive path resolution fix, `busybox` alias in main dispatcher, and include-list normalization (stripping `./` prefix).

### C.3 — BusyBox Test Harness Integration ✅
**File:** `test/busybox_testsuite/runtest`, `cmd/korego/main.go`

Fixed:
- Added `busybox` symlink to korego in the test runner link directory.
- Added `LINKSDIR` to PATH in old-style test runner for `busybox` resolution.
- Added `busybox` alias in main dispatcher (`name == "busybox"` → subcommand mode).
- Exported `LINKSDIR` environment variable.

---

## Phase D: `gzip` (1 failure) — **COMPLETED (2026-05-01)**

### D.1 — Numeric Compression Levels (`-1` to `-9`) ✅
**File:** `pkg/gzip/gzip.go`

Implemented:
- Added `-1` through `-9` as boolean flags in flag spec.
- `getCompressionLevel()` maps detected flags to Go's `compress/flate` levels (1–9).
- `gzip.NewWriterLevel()` used instead of `gzip.NewWriter()` when level specified.
- Also added: `-` as stdin/stdout handling for gzip positional args.

---

## Phase E: Sort & Diff & Tar fixes — **COMPLETED (2026-05-02)**

### E.1 — Sort key spec parsing ✅
**File:** `pkg/sort/sort.go`

Implemented:
- Rewrote `parseKeySpec` to correctly parse `-k start,end[flags]` format.
- `IndexAny(rest, "nrMhb")` was finding flag chars before the `,` separator.
- New logic: find `,` first, then parse start and end field specs separately.
- Added `parseNumericPrefix` function (like C's strtod) for multi-field numeric keys.
- Global `-r` only reverses tiebreaker when key has numeric flag; per-key `r` reverses everything.
- `-s` stable disables last-resort full-line comparison.
- `-u` unique compares all keys (using `slices.Equal`), not just the first.
- ENDCHAR fix: default startChar to 1 when endChar is set but startChar is 0.
- Non-numeric values sort before numeric values in `-n` mode.

### E.2 — Diff directory fixes ✅
**File:** `pkg/diff/diff.go`

Implemented:
- `diffDirs` now preserves raw path arguments in diff headers (using `joinPreserving`).
- Dir+file path resolution: when one arg is a dir and the other a file, construct the file path inside the dir.
- `-rN` mode: check non-regular files BEFORE checking existence in either dir.
- Missing files with `-N` are treated as empty for diff purposes.

### E.3 — Tar `../` stripping ✅
**File:** `pkg/tar/tar.go`

Implemented:
- `resolveTarPath` function walks components left to right, maintaining a stack.
- `..` pops from stack; first empty-stack `..` forward-cancels next regular component.
- Subsequent empty-stack `..` are just added to strip prefix without forward-canceling.
- Walk uses resolved path instead of original target.
- Directory entries show trailing `/` in verbose listing (`doCreate`, `doExtract`, `doList`).

### E.4 — md5sum/sha256sum `-c` empty file ✅
**File:** `pkg/md5sum/md5sum.go`, `pkg/sha256sum/sha256sum.go`

Implemented:
- When checksum file has no valid checksum lines, exit with code 1 and print error.

---

## Known Deviations / Future Work

These are known differences from GNU/BusyBox behavior that are low-priority or by design:

| Utility | Deviation | Priority |
|---------|-----------|----------|
| `tar` | `writing into read-only dir` — umask-dependent (644 vs 664, passes with umask 022) | Low |
| `tar` | No pax headers, xz/bzip2 decompression (10 skipped tests) | Low |
| `grep` | No `-P` (Perl regex) — Go regexp ≠ PCRE | By design |
| `awk` | Not implemented (deferred post-MVP) | Deferred |
| `patch` | Not implemented | Deferred |

---

## BusyBox Skipped Tests (10 tests — external dependency gated)

These tests are skipped because they require external compression tools or features
not yet implemented in KoreGo.

| Test | Reason |
|------|--------|
| `tar Empty file is not a tarball.tar.gz` | Needs gunzip integration |
| `tar hardlinks and repeated files` | Hardlink creation (`FEATURE_TAR_CREATE`) |
| `tar hardlinks mode` | Hardlink mode preservation |
| `tar symlinks mode` | Symlink handling in archive |
| `tar extract tgz` | `.tgz` extraction (needs gzip) |
| `tar extract txz` | `.txz` extraction (needs xz, uudecode) |
| `tar does not extract into symlinks` | Symlink attack protection (needs bzip2) |
| `tar -k does not extract into symlinks` | Symlink attack with `-k` (needs bzip2) |
| `tar Pax-encoded UTF8 names and symlinks` | PAX/UTF-8 extended headers |
| `tar Symlink attack: …` | Symlink attack test (needs bzip2, uudecode) |

---

## CI vs Local Discrepancy Note — RESOLVED (2026-05-13)

The discrepancy was fixed by adding a global `busybox → korego` symlink in `runtest`.
All old-style `busybox <applet>` calls now resolve to KoreGo on both CI and local.
The true KoreGo BusyBox pass rate is now **409 passed, 71 failed, 10 skipped**.
See [12_road_to_gold.md](12_road_to_gold.md) (12.4) for the implementation details.

---

## Session Insights (2026-05-02 — Round 2)

### `sort` — `-k` key spec parsing
The `parseKeySpec` function must find the `,` separator between start and end fields BEFORE searching for flag characters (`nrMhb`). Otherwise `"2,3n"` is parsed as start field `"2,3"` instead of start=2, end=3 with flag `n`.

### `sort` — numeric prefix parsing for multi-field keys
When `-k M,Nn` is used with multi-field keys (e.g., `1\t010`), Go's `strconv.ParseFloat` fails because of embedded delimiters. Use a custom `parseNumericPrefix` function that extracts only the leading numeric prefix (like C's `strtod`), stopping at the first non-numeric character.

### `sort` — global `-r` vs per-key `r` semantics
Global `-r` with `-k M,Nn` only reverses the tiebreaker (last-resort full-line comparison), NOT the numeric key comparison. Per-key `r` (in `-k M,Nnr`) reverses the entire key comparison, including numeric. This distinction is crucial for BusyBox compatibility.

### `sort` — `-u` unique with multiple keys
When `-u` is active, uniqueness should be determined by comparing ALL sort keys (using `slices.Equal`), not just the first key. This affects tests like `glibc build sort unique`.

### `sort` — `-s` stable disables last-resort comparison
When `-s` is set, items with equal keys must preserve their original relative order without falling back to full-line comparison.

### `sort` — ENDCHAR default
When `-k start,end.char` specifies an end character but no start character, the start character defaults to 1 (beginning of field). The previous code skipped character trimming entirely when `startChar == 0`.

### `sort` — Non-numeric values sort BEFORE numeric in `-n` mode
In BusyBox, when using `-n` (numeric sort), non-numeric values sort before all numeric values. The previous code had this reversed.

### `diff` — raw path preservation in directory diff headers
When paths like `././//diff1` are passed to `diff -ur`, BusyBox preserves them in the output header. Use a `joinPreserving` function that doesn't clean the path, instead concatenating dir+"/"+rel with careful trailing-slash handling.

### `diff` — dir+file path resolution
When `-r` is set and one arg is a directory while the other is a file, look for the file's basename inside the directory: `dir + "/" + filepath.Base(file)`.

### `diff` — `-rN` non-regular file handling
With `-N`, missing files are treated as present (empty). Check for non-regular files BEFORE existence checks to emit the correct "is not a regular file" message rather than "Only in".

### `tar` — `../` component stripping
Walk path components left to right with a stack. `..` pops from stack. First empty-stack `..` forward-cancels the next regular component. Subsequent empty-stack `..` are just added to the strip prefix. Walk the resolved path, not the original.

### `md5sum` — empty checksum file
When `-c EMPTY` is called on an empty checksum file, GNU md5sum exits with code 1 and prints "no properly formatted checksum lines found". Track whether any valid lines were processed.
