# GoPOSIX — Open TODOs & Remaining Work

> **Last updated:** 2026-05-24 | **Utilities:** 115 | **Coverage:** 82.4% | **BusyBox:** 840/54/25 (91.4%) | **JSON-RPC Daemon:** 115/115 (100.0%)

This document serves as the live registry of remaining work, active plans, and known limitations in GoPOSIX.

---

> 📊 **Current project state** (coverage, BusyBox stats, per-utility status) → **[wiki/test_coverage_matrix.md](test_coverage_matrix.md)**

---

## 🛠️ Open Issues by Tool

### 🔴 Highest Priority — Active Development

#### `dc` — 7 failures (29/36 pass, 80.6%) · coverage 90.2%

**Context**: `FEATURE_DC_BIG` enabled. 22 previously-skipped tests now pass from prior session fixes (conditional direction, bracket parsing, `x` command, exit code, `-x` flag, per-number scale tracking). 7 failures remain:

| # | Test | Category | Difficulty | Symptoms |
|---|------|----------|------------|----------|
| 1 | `dc_strings.dc` | String/macro | 🟡 Medium | Stack overflow — recursive macro `[xz0<x]dsxx` infinite-recurses when leftover stack strings contain `[` that triggers nested string pushes |
| 2 | `dc_modulus.dc` | Scale propagation | 🔴 Hard | `%` operator uses integer truncation instead of scale-aware division; in 0k mode produces wrong values + trailing-zero formatting |
| 3 | `dc_divmod.dc` | Scale propagation | 🟡 Medium | `~` divmod integer formatting in 0k mode (`.000…` instead of bare ints); uses same broken `%` as modulus |
| 4 | `dc_power.dc` | Formatting | 🟢 Easy | `0` formatted as `.00000000000000000000` and `-0` as `-.000…` instead of bare `0`; also some last-digit precision diffs |
| 5 | `dc_multiply.dc` | Formatting | 🟢 Easy | Zero formatting (same root cause as power); 1 last-digit precision diff |
| 6 | `dc_divide.dc` | Scale propagation | 🟢 Easy | 1 last-digit precision diff (line 32 of 32) |
| 7 | `dcx_vars.dc` | Extended mode | 🟡 Medium | Multi-character register names (`s xotj`, `l yotp`) not supported — only single-rune registers implemented |

**Recommended attack order**: #1 (crash fix) → #4+#5 (formatRat zero/integer cleanup — fixes most diffs in one change) → #2+#3 (scale-aware modulus/divmod) → #7 (string registers) → #6 (last digit quirks)

---

#### `bc` — 22 failures (59/81 pass, 72.8%) · coverage 64.3%

**Root cause**: Global scale model (`math/big.Rat`) vs BusyBox's per-number scale tracking. Architectural mismatch cascades through all precision-sensitive operations.

| # | Group | Count | Difficulty | Description |
|---|-------|-------|------------|-------------|
| 1 | Number parsing/printing | 2 | 🟢 Easy | `ibase=16; FF` parsing, `obase=16` uppercase hex output. Pure lexer/printer — no scale dependency. |
| 2 | String & decimal formatting | 5 | 🟢 Easy | Leading/trailing zeros, string concat, scientific notation, function return formatting. Output rendering only. |
| 3 | High-precision arithmetic | 4 | 🟡 Medium | Multiply/modulus/power/sqrt scale propagation through operations. Per-operation fixes. |
| 4 | Per-value scale (vars/arrays/refs) | 4 | 🔴 Hard | Scale lost when storing in arrays/variables, across function boundaries. Requires architectural redesign. |
| 5 | Series convergence (trig/bessel/exp/log) | 7 | 🔴 Hard | `s(x)`, `c(x)`, `a(x)`, `e(x)`, `l(x)`, `j(n,x)`, `4*a(1)` — need `scale+10` guard digits. **Blocked by group 4** (per-value scale). |

**Recommended attack order**: 1 → 2 → 3 → 4 → 5 (groups 4-5 require the architectural per-value scale change first).

---

### 🟡 Medium Priority — Well-Defined, Scoped

#### `tar` — 7 failures + 10 skipped (24/31 pass, 77.4%) · coverage 74.8%

| # | Type | Count | Difficulty | Description |
|---|------|-------|------------|-------------|
| 1 | Failures | 3 | 🟡 Medium | Hardlink/symlink mode ordering — permission bits applied in wrong order |
| 2 | Failures | 3 | 🟡 Medium | Symlink safety — no traversal-attack guard during extraction |
| 3 | Failures | 1 | 🟡 Medium | XZ compression auto-detect (`.tar.xz`) not implemented |
| 4 | Skipped | 2 | 🟡 Medium | Auto-detect `.tar.gz`/`.tar.xz` on extract |
| 5 | Skipped | 4 | 🟡 Medium | Symlink safety guards (extraction into symlinks, `-k` mode, symlink attack) |
| 6 | Skipped | 2 | 🟡 Medium | Hardlink detection/dedup + mode preservation |
| 7 | Skipped | 1 | 🟡 Medium | Pax-encoded UTF8 filenames and symlinks (extended headers) |
| 8 | Skipped | 1 | 🟢 Easy | Graceful rejection of empty `.tar.gz` files |

---

#### `rx` — 1 flaky test · coverage 72.4%

- [ ] **XMODEM timing race** — intermittent pass/fail in `rx.tests`. Likely ACK/NAK handshake timeout or race condition.
- *Estimate*: 2-4 hours investigation.

---

### 🟢 Low Priority / Deferred

#### `awk` — 17 failures + 8 skipped · coverage 90.0%

**Status**: Deferred. Root cause is the `goawk` v1.31.0 engine — upstream doesn't support bitwise ops, hex/octal constants, function arg parsing (4 tests), nested loop scoping, empty-paren handling, negative field access, continue/break edge cases, and backslash-newline handling.
- 8 additional tests skipped (large integer, NUL printf, invalid for/colon syntax, missing delete arg, gcc build bug).
- *See*: [wiki/deferred.md](deferred.md).

---

### ⚪ Root-Required — Cannot Test in CI (23 skipped)

Tests that need `CAP_SYS_ADMIN`, kernel hotplug, or interactive shell — can only be validated manually. The other 2 of 25 total skipped are in the `awk` deferred section above.

| Tool | Skipped | Reason |
|------|---------|--------|
| `mdev` | 13 | Requires root + `/sys` kernel infrastructure (hotplug events, rule processing, move/command rules) |
| `cpio` | 7 | suid/sgid preservation, uid/gid defaults, `-R` owner flag, absolute path safety, zero-size hardlinks |
| `mount` | 1 | Requires `CAP_SYS_ADMIN` |
| `makedevs` | 1 | Device node creation requires root |
| `ash` | 1 | Needs interactive shell session |
| **Total** | **23** | |

---

> ✅ **Completed & resolved** (all phases, per-utility changelog) → **[wiki/log.md](log.md)**

---

## 🚀 Historical Roadmaps & Archive

The sequence of completed milestones, architectural changes, and past iterations (Phases 01 to 27) are preserved in the master phase ledger:

👉 **[wiki/phases.md](phases.md)**
