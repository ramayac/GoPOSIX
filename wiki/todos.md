# GoPOSIX — Open TODOs & Remaining Work

> **Last updated:** 2026-05-24 | **Utilities:** 115 | **Coverage:** 82.3% | **BusyBox:** 798/39/82 (95.3%) | **JSON-RPC Daemon:** 115/115 (100.0%)

This document serves as the live registry of remaining work, active plans, and known limitations in GoPOSIX.

---

## 📈 Current Project State

| Metric | Value |
|--------|-------|
| **Total Utilities Implemented** | **115** (all registered via `dispatch.Register`) |
| **Overall Statement Coverage** | **82.3%** (fully compliant with the `>=80%` CI gate) |
| **BusyBox Suite Passed / Failed / Skipped** | **798 / 39 / 82** (95.3% pass rate, 919 total) |
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
* **Mathematics (2)**: ⚠️ `dc` (29 of 36 tests skipped — `FEATURE_DC_BIG` not enabled), ❌ `bc` (22 failures)
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

### 3. `dc` — 29 of 36 BusyBox tests skipped
* 7 basic dc tests pass (stdin, argv, complex expressions). 29 tests behind `FEATURE_DC_BIG` are skipped. Full categorized TODO list below in [📋 Backlog — Skipped Tests](#-backlog--skipped-tests).
* *Status*: Feature flag `FEATURE_DC_BIG` needs to be enabled and validated against BusyBox expectations.

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

### 2. Skipped BusyBox Tests — Full Categorized TODO List (82 total)

Every skipped test is listed below as an actionable checkbox, organized by root cause and difficulty.

---

#### 🔴 A. `dc` — `FEATURE_DC_BIG` Flag Not Enabled (29 skipped)

*Context*: All 29 dc tests are gated behind `optional FEATURE_DC_BIG` in `dc.tests`. The flag is not in `OPTIONFLAGS` in `runtest`. GoPOSIX's dc implementation has 90.3% coverage — these features likely work but are untested against BusyBox expectations.*

*Fix*: Enable `FEATURE_DC_BIG` in `OPTIONFLAGS` (in `runtest`), run the suite, fix any failures, then leave flag enabled permanently.

##### Macro Execution (`x`) — 3 tests
- [ ] **dc: x should execute strings** — `[40 2 +] x f` should produce `42`
- [ ] **dc: x should not execute or pop non-strings** — `42 x f` should produce `42` (no-op)
- [ ] **dc: x should work with strings created from a** — `42 112 a x` — ascii-to-string then execute

##### String Printing Edge Cases (`p`) — 4 tests
- [ ] **dc: p should print invalid escapes** — backslash sequences in printed strings
- [ ] **dc: p should print trailing backslashes** — strings ending with `\`
- [ ] **dc: p should parse/print single backslashes** — single `\` in strings
- [ ] **dc: p should print single backslash strings** — literal backslash output

##### Conditional Execution (`>a`, `>aeb`) — 3 tests
- [ ] **dc '>a' (conditional execute string) 1** — `>a` register conditional
- [ ] **dc '>a' (conditional execute string) 2** — second variant
- [ ] **dc '>aeb' (conditional execute string with else)** — if-then-else conditional

##### Script-Based Tests (dc_*.dc) — 11 tests
- [ ] **dc dc_add.dc** — addition script
- [ ] **dc dc_subtract.dc** — subtraction script
- [ ] **dc dc_multiply.dc** — multiplication script
- [ ] **dc dc_divide.dc** — division script
- [ ] **dc dc_modulus.dc** — modulus script
- [ ] **dc dc_divmod.dc** — divmod (`~`) script
- [ ] **dc dc_power.dc** — power (`^`) script
- [ ] **dc dc_sqrt.dc** — sqrt (`v`) script
- [ ] **dc dc_boolean.dc** — boolean/comparison script
- [ ] **dc dc_decimal.dc** — decimal/fractional script
- [ ] **dc dc_modexp.dc** — modular exponentiation (`|`) script

##### Register & Stack Mechanics — 5 tests
- [ ] **dc dc_misc.dc** — miscellaneous operations
- [ ] **dc dc_strings.dc** — string manipulation
- [ ] **dc -x dcx_vars.dc** — variable and register operations with `-x`
- [ ] **dc space can be a register** — whitespace as register name
- [ ] **dc newline can be a register** — newline as register name

##### I/O Operations — 3 tests
- [ ] **dc read** — `?` read from stdin
- [ ] **dc read string** — reading string input
- [ ] **dc Z (length) for numbers** — `Z` command for number length

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

#### 🟣 F. `ar` — Archive Creation Not Yet Wired (2 skipped)

- [ ] **ar creates archives** — `ar -r -c` create new archive
- [ ] **ar replaces things in archives** — `ar -r` replace members in existing archive

*Fix*: The `blakesmith/ar` library supports full read/write. These are likely skipped because the `ar` command dispatcher doesn't surface the create/replace flags to the test harness. Check flag parsing and `-r`/`-c` wiring.

---

#### ⚪ G. `cryptpw` — SHA-256/512 with Rounds (4 skipped)

- [ ] **cryptpw sha256** — SHA-256 password hashing (`$5$` prefix)
- [ ] **cryptpw sha256 rounds=99999** — SHA-256 with custom rounds parameter
- [ ] **cryptpw sha512** — SHA-512 password hashing (`$6$` prefix)
- [ ] **cryptpw sha512 rounds=99999** — SHA-512 with custom rounds parameter

*Fix*: Go's `crypto/sha256` and `crypto/sha512` in stdlib. Implement SHA-crypt modular crypt format (PHC string format with `$5$`/`$6$` prefix and `rounds=` parameter).

---

#### ⚪ H. `unzip` — Corrupted Archive Handling (3 skipped)

- [ ] **unzip (bad archive)** — graceful error on completely invalid zip
- [ ] **unzip (archive with corrupted lzma 1)** — LZMA corruption detection
- [ ] **unzip (archive with corrupted lzma 2)** — LZMA corruption detection variant

*Fix*: GoPOSIX unzip already handles valid archives. Add error-path tests for corrupted data — likely just need to ensure `archive/zip` errors are surfaced as non-zero exit codes.

---

#### ⚪ I. `tree` — Directory Tree Display (3 skipped)

- [ ] **tree single file** — tree display of a single file
- [ ] **tree multiple directories** — tree display of multiple directory arguments
- [ ] **tree nested directories and files** — recursive tree display

*Fix*: `tree` is registered in dispatch but the BusyBox test harness may not be discovering it. Check `--list-commands` output for `tree`, verify symlink is created in `LINKSDIR`.

---

#### ⚪ J. `pidof` — `-o` Omit Flag (1 skipped)

- [ ] **pidof -o init** — omit PID 1 (init) from results

*Fix*: Core `pidof` passes BusyBox tests. The `-o` omit flag is a minor feature addition — parse `-o` argument, filter matching PIDs from results.

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
