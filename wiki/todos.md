# GoPOSIX — Open TODOs & Remaining Work

> **Last updated:** 2026-05-24 | **Utilities:** 115 | **Coverage:** 82.4% | **BusyBox:** 831/54/34 (90.4%) | **JSON-RPC Daemon:** 115/115 (100.0%)

This document serves as the live registry of remaining work, active plans, and known limitations in GoPOSIX.

---

## 📈 Current Project State

| Metric | Value |
|--------|-------|
| **Total Utilities Implemented** | **115** (all registered via `dispatch.Register`) |
| **Overall Statement Coverage** | **82.3%** (fully compliant with the `>=80%` CI gate) |
| **BusyBox Suite Passed / Failed / Skipped** | **831 / 54 / 34** (90.4% pass rate, 919 total) |
| **JSON-RPC Daemon Coverage** | **115/115** utilities with structured output tests |
| **Multicall Compatibility** | Complete dispatching via symlinks or direct subcommands |
| **CGO Status** | 100% CGO-free Go (`CGO_ENABLED=0`) |

---

## 🛠️ Active Phase: Phase 27 (High-Complexity Tier 5)

Phase 26 (Tiers 1–4) is **complete**. Phase 27 (Tier 5) is also **complete** (11 of 11 implemented). However, significant BusyBox test gaps remain (see below). The full breakdown is documented on the planning page:

👉 **[wiki/27_high_complexity_tools.md](27_high_complexity_tools.md)**

### Tier 5 Utilities (11 Utilities - COMPLETE ⚠️ with BusyBox gaps)
* **Compression & Archiving (2)**: ✅ `ar` (2 tests skipped), ✅ `cpio` (7 tests skipped)
* **Development & Hex (3)**: ⚠️ `rx` (1 flaky test), ✅ `hexdump`, ✅ `xxd`
* **Mathematics (2)**: ⚠️ `dc` (29 of 36 pass, 7 scale/macro failures), ❌ `bc` (22 failures)
* **Shell (1)**: ✅ `ash` (alias to existing native `shell` implementation)
* **System Admin & Hardware (3)**: ✅ `mdev` (12 skipped, needs root), ✅ `mkfs.minix`, ✅ `mount` (1 skipped, needs root)

---

## ❌ Known Limitations & Remaining Failures

### 1. `bc` — 22 BusyBox failures (precision and scale differences)

**Context**: 38 bc tests **pass** (control flow, basic arithmetic via `bc_add`/`bc_subtract`/`bc_divide`/`bc_boolean`/`bc_numbers*`, function definitions, comments, strings, print, read, ibase, length, comparison). The 22 failures are concentrated in precision-sensitive operations where `math/big` scale propagation diverges from BusyBox.

**Root cause**: GoPOSIX uses a global scale model inherited from `math/big.Rat`; BusyBox's bc tracks per-number scale. This architectural mismatch cascades into every operation where precision matters. Fixing it requires a scale-propagation redesign, not point fixes.

*Coverage*: 64.3% — lowest of any utility. *See*: [wiki/27_high_complexity_tools.md](27_high_complexity_tools.md).

#### 📋 Categorized TODO Items

##### 🔴 A. Trigonometric & Special Functions (5 failures) — *Series convergence at scale ≥ 20*

- [ ] **`bc_arctangent.bc`** — `a(x)` arctangent series. Fails at `scale=64` precision. BusyBox uses Machin-like formula; GoPOSIX likely uses simpler Taylor series that diverges at high scale.
- [ ] **`bc_cosine.bc`** — `c(x)` cosine. Same series convergence issue.
- [ ] **`bc_sine.bc`** — `s(x)` sine at `scale=20`. Series terms need more iterations for target precision.
- [ ] **`bc_pi.bc`** — `4*a(1)` pi calculation. Depends on arctangent precision; cascading failure from arctangent.
- [ ] **`bc_bessel.bc`** — `j(n,x)` Bessel J₀/J₁ functions. These are implemented as series over factorials; precision loss compounds with every term.

*Fix approach*: Increase series iteration count based on current `scale`. BusyBox uses `scale+10` guard digits internally. Or switch to Machin-formula for pi/arctan.

##### 🟠 B. Transcendental Functions (2 failures) — *e^x and ln precision*

- [ ] **`bc_exponent.bc`** — `e(x)` exponential. Taylor series for e^x; error grows with |x|.
- [ ] **`bc_log.bc`** — `l(x)` natural logarithm. Uses Newton's method or series; sensitive to initial guess and iteration count.

*Fix approach*: Same as trig — increase internal guard digits to `scale+10`. Use range reduction for large |x| in exponent.

##### 🟡 C. High-Precision Arithmetic (4 failures) — *Scale carrying through operations*

- [ ] **`bc_multiply.bc`** — Multiplication with fractional operands. Scale should be `min(a.scale+b.scale, global_scale)`. GoPOSIX likely truncates differently.
- [ ] **`bc_modulus.bc`** — Modulus with fractional operands. Per POSIX: `a%b = a - (a/b)*b`. Scale propagation through division then subtraction loses precision.
- [ ] **`bc_power.bc`** — `a^b` with integer exponents. `0^0=1`, `0^N=0` work (pass in basic tests) but fractional results diverge.
- [ ] **`bc_sqrt.bc`** — `sqrt(x)` at `scale=20`. Newton's method needs more iterations; `sqrt(4)` returning `1.999...` instead of `2.000...` is a guard-digit issue.

*Fix approach*: These are the most likely to benefit from a scale-propagation redesign. Each operation needs to compute result scale as `max(scale, min(a_scale, b_scale))` per POSIX semantics, then round.

##### 🟢 D. Variable / Array / Reference Scale Propagation (4 failures)

- [ ] **`bc_array.bc`** — Array element assignment and arithmetic. Scale is lost when storing/retrieving from array cells.
- [ ] **`bc_arrays.bc`** — Multi-dimensional array access and expression evaluation. Scale not preserved through index expressions.
- [ ] **`bc_references.bc`** — Array passed by reference to functions (`define printarray(a[], len)`). Scale of array elements lost across function boundary.
- [ ] **`bc_vars.bc`** — High-scale variable arithmetic (`scale=100`, `scale=10`). Variable assignment doesn't preserve operand scale; subsequent operations use wrong precision.

*Fix approach*: Each variable/array-cell must store its own scale alongside the `*big.Rat` value. This is the core architectural change needed — move from global scale to per-value scale.

##### 🔵 E. String & Decimal Formatting (5 failures)

- [ ] **`bc_decimal.bc`** — Decimal number formatting with leading/trailing zeros (`000.000`, `.00000`). Trailing zero stripping and leading-zero preservation differ from BusyBox.
- [ ] **`bc_strings.bc`** — String concatenation without separator. Multi-line string handling, escape sequences (`\n` as literal vs newline).
- [ ] **`bc_misc.bc`** — Mixed arithmetic with scientific notation (`1.-13`, backslash-continued expressions). Output format for negative results with fractional parts.
- [ ] **`bc_misc1.bc`** — Function return value formatting. `define x(x){return(x)}` — scale of return value vs caller's scale.
- [ ] **`bc_misc2.bc`** — String+number concatenation in function context. String output alongside numeric returns.

*Fix approach*: Most of these are output-formatting issues, not computation bugs. Compare raw `*big.Rat` values against expected, then fix the `String()`/`FloatString()` conversion. The decimal test is the best starting point — it's pure formatting with no math library dependency.

##### 🟣 F. Number Parsing & Printing (2 standalone failures)

- [ ] **`bc parsing of numbers`** — Parsing numbers with explicit ibase/obase. `ibase=16; FF` vs `ibase=16; 0FF` — leading-zero handling. Scientific notation (`1e5`). Negative numbers in non-decimal bases.
- [ ] **`bc printing of numbers`** — Printing numbers with obase. `obase=16; 255` should print `FF`, not `ff`. Zero-padding, uppercase hex digits, fractional parts in non-decimal bases.

*Fix approach*: These are the most isolated failures — pure lexer/printer issues. Start here for quick wins. Compare `goposix bc` vs `busybox bc` output character-by-character using `diff -u` for each test input.

---

#### 🎯 Recommended Attack Order

1. **Quick wins** (F): parsing & printing of numbers — isolated lexer/printer, no scale dependency. Estimate: 2–4 hours.
2. **Formatting** (E): decimal, strings, misc* — output rendering, not computation. Estimate: 4–8 hours.
3. **Arithmetic precision** (C): multiply, modulus, power, sqrt — small fixes to scale propagation per operation. Estimate: 8–12 hours.
4. **Per-value scale** (D): arrays, references, vars — requires architectural change to store scale per-value. **Blocker for trig/transcendental fixes**. Estimate: 16–24 hours.
5. **Series convergence** (A + B): trig, bessel, exponent, log — once per-value scale works, increase guard digits. Estimate: 8–12 hours.

### 2. `rx` — 1 flaky BusyBox test (XMODEM timing)
* The single `rx.tests` test passes most runs but fails intermittently. Likely a race condition or timeout in the XMODEM ACK/NAK handshake.
* *Status*: **Needs investigation** — flaky test indicates a non-deterministic timing bug. 72.4% coverage.
* *See*: [wiki/27_high_complexity_tools.md](27_high_complexity_tools.md).

### 3. `dc` — 7 BusyBox failures (scale propagation, string/macro, extended mode)

**Context**: 29 of 36 dc tests pass (80.6%). `FEATURE_DC_BIG` has been enabled in `runtest`. 6 bugs fixed in this session:
- ✅ Conditional execution direction (`><=!` now compares top vs second, matching BusyBox)
- ✅ Bracket string parsing (`\]` properly closes bracket)
- ✅ `x` command no longer pops non-string values
- ✅ Exit code always returns 0 (matching BusyBox convention)
- ✅ `-x` flag support (alias for `-e`)
- ✅ Per-number scale tracking (`fracDigits` propagated through `+`, `-`, `*`, `/`, `%`, `~`, `^`, `v`)

**7 remaining failures** fall into 3 categories:
1. **Scale propagation edge cases** (5): `dc_divide.dc`, `dc_divmod.dc`, `dc_modulus.dc`, `dc_multiply.dc`, `dc_power.dc` — last-digit precision differences through long operation chains. Same architectural limitation as `bc` (per-value vs global scale).
2. **String/macro interaction** (1): `dc_strings.dc` — recursive macro `[xz0<x]dsxx` with per-value scale interaction.
3. **Extended register mode** (1): `dcx_vars.dc` — multi-character register support in `-x` mode not yet implemented.

*Coverage*: 90.2%. *See*: [wiki/test_coverage_matrix.md](test_coverage_matrix.md).

### 4. `pidof` — All 4 BusyBox tests pass ✅
* `FEATURE_PIDOF_OMIT` enabled in `runtest`. The `-o` omit flag was already implemented — test passes immediately.

### 4. Compliance and Verification Updates
* **Resolved**: `realpath` and `readlink` — all BusyBox tests now pass ✅ (previously 3 failures each, now 0).
* **Resolved**: `cpio`, `pidof`, and `mkfs.minix` — all BusyBox tests pass 100% ✅.

### 5. Compliance tests — ✅ COMPLETE
* 28 `test/compliance/test_<name>.sh` scripts written for all Phase 26 Tier 4 and Phase 27 tools.
* 84 assertions, 0 failures. 1 test skipped (uncompress needs system `compress`).

### 6. JSON-RPC tests — 0 remaining gaps (115/115 tested)

---

## 📋 Backlog & Deferred Work

### 1. `awk` — 17 failures + 8 skipped (goawk v1.31.0 engine limitations)
* Bitwise operations, hex/octal constants, function argument parsing (4 tests), nested loop variable scoping, empty-paren handling, negative field access, continue/break edge cases, and backslash-newline handling are not supported by the underlying parsing engine.
* 8 tests also skipped: large integer, NUL printf, invalid for/colon syntax, missing delete arg, gcc build bug.
* *Status*: **Backlog / Deferred** (see [wiki/deferred.md](deferred.md) for full context).

---

### 2. Skipped BusyBox Tests — Full Categorized TODO List (34 total)

Every skipped test is listed below as an actionable checkbox, organized by root cause and difficulty.

---

#### 🔴 A. `dc` — Scale Propagation & Extended Mode (7 remaining, 29 resolved ✅)

*Context*: 22 of 29 previously-skipped dc tests now pass (80.6% pass rate). `FEATURE_DC_BIG` enabled in `runtest`. 6 bugs fixed (conditional direction, bracket parsing, `x` command, exit code, `-x` flag, per-number scale tracking). 7 remaining failures are scale/macro/extended-mode edge cases — see [Known Limitations §3](#3-dc--7-busybox-failures-scale-propagation-stringmacro-extended-mode) above.

*Resolved tests (now passing)*: x execute strings, x non-string no-op, x work with strings from a, p print invalid/trailing/single backslash strings, read, read string, `>a`/`>aeb` conditional, space/newline as register, Z length, dc_add, dc_boolean, dc_decimal, dc_misc, dc_modexp, dc_sqrt, dc_subtract. ✅

*Remaining tests (7)*:
- [ ] **dc dc_divide.dc** — division scale propagation
- [ ] **dc dc_divmod.dc** — divmod (`~`) scale propagation
- [ ] **dc dc_modulus.dc** — modulus scale propagation
- [ ] **dc dc_multiply.dc** — multiplication scale propagation
- [ ] **dc dc_power.dc** — power (`^`) scale propagation
- [ ] **dc dc_strings.dc** — string/macro recursive interaction
- [ ] **dc -x dcx_vars.dc** — extended register mode (`-x`)

---

#### 🟠 B. `mdev` — Root + Kernel Hotplug Required (13 skipped)

*Context*: All 13 mdev tests require root privileges and `/sys` kernel infrastructure. Cannot be tested in CI or user containers.*

*Fix*: These can only be validated manually on a real Linux system with `sudo`. Consider adding a `make test-mdev-root` target with `sudo` for manual verification.

##### Hotplug Events — 2 tests
- [ ] **mdev add /block/sda** — simulate block device hot-add
- [ ] **mdev deletes /block/sda** — simulate device removal

##### Rule Processing — 7 tests
- [ ] **mdev stops on first rule** — first-match-wins behavior
- [ ] **mdev does not stop on dash-rule** — `-` as no-op rule
- [ ] **mdev $ENVVAR=regex match** — environment variable substitution in rules
- [ ] **mdev regexp substring match + replace** — regex capture groups in rules
- [ ] **mdev #maj,min and no explicit uid** — default ownership from major/minor
- [ ] **mdev move/symlink rule '>bar/baz'** — symlink creation via `>`
- [ ] **mdev move/symlink rule '>bar/'** — symlink to directory

##### Move & Command Rules — 3 tests
- [ ] **mdev move rule '=bar/baz/fname'** — move/rename via `=`
- [ ] **mdev command** — external command execution on event
- [ ] **mdev move and command** — combined move + command

##### Edge Case — 1 test
- [ ] **move rule does not delete node with name == device_name** — same-name collision safety

---

#### 🟡 C. `tar` — Feature Gaps (10 skipped)

*Context*: 10 tar tests skipped — compression format support, symlink safety, hardlink mode, Pax extended headers.*

*Fix*: These require implementing specific tar features. Hardest: Pax UTF8 names (custom header parsing). Easiest: symlink extraction guards.

##### Compression Format Detection (auto-extract) — 2 tests
- [ ] **tar extract tgz** — auto-detect and extract `.tar.gz`
- [ ] **tar extract txz** — auto-detect and extract `.tar.xz`

##### Symlink Safety — 4 tests
- [ ] **tar does not extract into symlinks** — prevent symlink traversal attack
- [ ] **tar -k does not extract into symlinks** — `-k` (keep-old) + symlink safety
- [ ] **tar Symlink attack: create symlink and then write through it** — classic symlink attack guard
- [ ] **tar symlinks mode** — symlink permission/mode preservation

##### Hardlink Handling — 2 tests
- [ ] **tar hardlinks and repeated files** — hardlink detection and dedup
- [ ] **tar hardlinks mode** — hardlink permission preservation

##### Extended Attributes — 1 test
- [ ] **tar Pax-encoded UTF8 names and symlinks** — POSIX.1-2001 Pax extended headers for UTF8 filenames

##### Edge Case — 1 test
- [ ] **tar Empty file is not a tarball.tar.gz** — graceful rejection of empty/zero-byte files

---

#### 🟢 D. `awk` — Goawk Engine Limitations (8 skipped)

*Context*: Same root cause as the 17 awk failures — goawk v1.31.0 doesn't support these features.*

- [ ] **awk large integer** — integers exceeding int64 range
- [ ] **awk printf('%c') can output NUL** — NUL byte in printf %c
- [ ] **awk printf('%-10c') can output NUL** — left-justified NUL byte in printf
- [ ] **awk -e and ARGC** — `-e` program argument and ARGC tracking
- [ ] **awk handles invalid for loop** — graceful error for malformed for-loops
- [ ] **awk handles colon not preceded by ternary** — colon outside ternary context
- [ ] **awk errors on missing delete arg** — `delete` without array element argument
- [ ] **awk 'gcc build bug'** — regression test for a historical gcc bug

---

#### 🔵 E. `cpio` — POSIX Permission Features (7 skipped)

*Context*: cpio integration tests pass 100% for core functionality. These 7 skipped tests exercise suid/sgid preservation, uid/gid defaults, and zero-size hardlinks — POSIX permission features that require root or are not yet implemented.*

##### Ownership & Permission — 5 tests
- [ ] **cpio restores suid/sgid bits** — setuid/setgid permission preservation (needs root)
- [ ] **cpio uses by default uid/gid** — default ownership when no `-R` flag
- [ ] **cpio -R with create** — `-R owner` flag during archive creation
- [ ] **cpio -R with extract** — `-R owner` flag during extraction
- [ ] **cpio -p with absolute paths** — pass-through mode with absolute paths (safety concern)

##### Edge Cases — 2 tests
- [ ] **cpio extracts zero-sized hardlinks** — hardlinks to zero-byte files
- [ ] **cpio extracts zero-sized hardlinks 2** — variant of above

---

#### 🟣 F. `ar` — Archive Creation — ✅ 2 RESOLVED

- [x] **ar creates archives** — `ar -r -c` create new archive — ✅ PASSES
- [x] **ar replaces things in archives** — `ar -r` replace members in existing archive — ✅ PASSES

*Resolution*: Enabled `FEATURE_AR_CREATE` flag in `runtest`. GoPOSIX ar already supported `-r`/`-c` via `arReplace()` with `readArchive()` returning nil for new archives.

---

#### ⚪ G. `unzip` — Corrupted Archive Handling — ✅ 3 RESOLVED

- [x] **unzip (bad archive)** — graceful error on completely invalid zip — ✅ PASSES
- [x] **unzip (archive with corrupted lzma 1)** — LZMA corruption detection — ✅ PASSES
- [x] **unzip (archive with corrupted lzma 2)** — LZMA corruption detection variant — ✅ PASSES

*Resolution*: Added `scanCorruptedZip()` for local file header scanning in corrupted archives, BusyBox-compatible error messages ("corrupted data" + "inflate error"), `/`-prefix warning detection, and `sanitizeFilename()` for control-character replacement.

---

#### ⚪ H. `tree` — Directory Tree Display — ✅ 3 RESOLVED

- [x] **tree single file** — tree display of a single file — ✅ PASSES
- [x] **tree multiple directories** — tree display of multiple directory arguments — ✅ PASSES
- [x] **tree nested directories and files** — recursive tree display — ✅ PASSES

*Resolution*: Enabled `UNICODE_SUPPORT` flag in `runtest`. GoPOSIX tree already used Unicode box-drawing characters; all tree output matches BusyBox exactly.

---

#### ⚪ J. `pidof` — `-o` Omit Flag ✅ RESOLVED

- [x] **pidof -o init** — omit PID 1 (init) from results

*Resolution*: `FEATURE_PIDOF_OMIT` enabled in `runtest`. The `-o` flag was already implemented in `pidof.go` — test passes immediately. All 4 pidof tests pass 100%. ✅

---

#### ⚪ K. Root-Required — Can Only Test Manually (2 skipped)

- [ ] **mount (must be root to test this)** — all mount operations require `CAP_SYS_ADMIN`
- [ ] **makedevs (must be root to test this)** — device node creation requires root

*Fix*: These can never run in CI. Document manual test procedure: `sudo make test-mount` and `sudo make test-makedevs`.

---

### 3. Aliases
* **3 aliases** (`egrep`, `fgrep`, `gunzip`) share their parent's RPC method and are tested through `goposix.grep` / `goposix.gzip`.
* **Verdict:** every utility has a JSON-RPC test or a documented reason it can't be tested.

---

## 🚀 Historical Roadmaps & Archive

The sequence of completed milestones, architectural changes, and past iterations (Phases 01 to 25) are preserved for historical reference in the master phase ledger:

👉 **[wiki/phases.md](phases.md)**
