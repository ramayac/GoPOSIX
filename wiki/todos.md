# GoPOSIX — Open TODOs & Remaining Work

> **Last updated:** 2026-05-22 | **Utilities:** 115 | **Coverage:** 82.9% | **JSON-RPC Daemon:** 82/115 (71.3%)

This document serves as the live registry of remaining work, active plans, and known limitations in GoPOSIX.

---

## 📈 Current Project State

| Metric | Value |
|--------|-------|
| **Total Utilities Implemented** | **115** (all registered via `dispatch.Register`) |
| **Overall Statement Coverage** | **82.9%** (fully compliant with the `>=80%` CI gate) |
| **JSON-RPC Daemon Coverage** | **82/115** utilities with structured output tests |
| **Multicall Compatibility** | Complete dispatching via symlinks or direct subcommands |
| **CGO Status** | 100% CGO-free Go (`CGO_ENABLED=0`) |

---

## 🛠️ Active Phase: Phase 27 (High-Complexity Tier 5)

Phase 26 (Tiers 1–4) is **complete**. Phase 27 has 5 of 11 implemented. The remaining 6 are cataloged in detail on the dedicated Phase 27 planning page:

👉 **[wiki/27_high_complexity_tools.md](27_high_complexity_tools.md)**

### Tier 5 Utilities (11 Utilities - IN PROGRESS 🔨)
* **Compression & Archiving (2)**: ✅ `ar`, ✅ `cpio`
* **Development & Hex (3)**: ⏳ `hexdump`, ⏳ `xxd`, ⏳ `rx`
* **Mathematics (2)**: ⏳ `bc`, ⏳ `dc`
* **Shell (1)**: ✅ `ash` (alias to existing native `shell` implementation)
* **System Admin & Hardware (3)**: ✅ `mdev`, ⏳ `mkfs.minix`, ✅ `mount`

---

## ❌ Known Limitations & Remaining Failures

### 1. `awk` — 16 failures (goawk v1.31.0 engine limitations)
* Hex/Octal constants, scoped nested variables, scoped scopes, and bitwise operations are not supported by the underlying parsing engine.
* *Status*: **Deferred** (see [wiki/deferred.md](deferred.md) for full context).

### 2. `cpio` — 2 failures (cavaliergopher/cpio library limitation)
* Block count output not emitted by `-t` or `-i` operations; `cavaliergopher/cpio` reader doesn't track block counts.
* *Status*: **Accepted** (low-impact output formatting; core functionality correct).

### 3. `pidof` — 1 failure (exit code mismatch)
* BusyBox test expects specific exit code behavior when no matching process is found.
* *Status*: **Needs investigation**.

### 4. Compliance tests missing for 28 utilities
* Only `ar`, `cpio`, and `ash` have `test/compliance/test_<name>.sh` scripts.
* All Phase 26 Tier 4 and Phase 27 tools need compliance tests.

### 5. JSON-RPC tests missing for 33 utilities
* All Tier 8 (16 tools) plus 17 legacy tools lack structured output tests in `test/posix-json/`.

---

## 🚀 Historical Roadmaps & Archive

The sequence of completed milestones, architectural changes, and past iterations (Phases 01 to 25) are preserved for historical reference in the master phase ledger:

👉 **[wiki/phases.md](phases.md)**
