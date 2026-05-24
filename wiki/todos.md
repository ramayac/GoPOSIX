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
* 7 basic dc tests pass (stdin, argv, complex expressions). 29 tests behind `FEATURE_DC_BIG` are skipped: macro execution (`x`), string printing edge cases (`p`), conditional execution (`>a`, `>aeb`), register mechanics, length operations (`Z`).
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

### 1. `awk` — 17 failures (goawk v1.31.0 engine limitations)
* Bitwise operations, hex/octal constants, function argument parsing (4 tests), nested loop variable scoping, empty-paren handling, negative field access, continue/break edge cases, and backslash-newline handling are not supported by the underlying parsing engine.
* 8 tests also skipped: large integer, NUL printf, invalid for/colon syntax, missing delete arg, gcc build bug.
* *Status*: **Backlog / Deferred** (see [wiki/deferred.md](deferred.md) for full context).

### 2. Skipped Tests Summary (82 total)
* **Feature-gated skips (37)**: `dc` (29, `FEATURE_DC_BIG`), `cpio` (7, suid/sgid/uid/gid), `ar` (2, create/replace archives).
* **Root-required skips (14)**: `mdev` (12), `mount` (1), `makedevs` (1).
* **Other hard-constraint skips (11)**: `ash` (daemon flag conflict), `wget` (network), `cryptpw` (3, sha256/sha512 rounds), `tar` (9, symlink extraction, hardlinks mode, Pax UTF8), `unzip` (3, bad/corrupted archives), `pidof` (1, `-o init`).
* **Go engine skips (8)**: `awk` (large integer, NUL printf, invalid syntax forms).

### 3. Aliases
* **3 aliases** (`egrep`, `fgrep`, `gunzip`) share their parent's RPC method and are tested through `goposix.grep` / `goposix.gzip`.
* **Verdict:** every utility has a JSON-RPC test or a documented reason it can't be tested.

---

## 🚀 Historical Roadmaps & Archive

The sequence of completed milestones, architectural changes, and past iterations (Phases 01 to 25) are preserved for historical reference in the master phase ledger:

👉 **[wiki/phases.md](phases.md)**
