# GoPOSIX — Open TODOs & Remaining Work

> **Last updated:** 2026-05-30 | **Utilities:** 115 | **Coverage:** 83.6% | **BusyBox:** 870/24/25 (97.3%) | **JSON-RPC Daemon:** 115/115 (100.0%)

This document serves as the live registry of remaining work, active plans, and known limitations in GoPOSIX.

---

> 📊 **Current project state** (coverage, BusyBox stats, per-utility status) → **[wiki/test_coverage_matrix.md](test_coverage_matrix.md)**

---

## 🛠️ Open Issues by Tool

### 🔴 Highest Priority — Active Development


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
