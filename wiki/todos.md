# GoPOSIX — Open TODOs & Remaining Work

> **Last updated:** 2026-05-24 | **Utilities:** 115 | **Coverage:** 82.3% | **JSON-RPC Daemon:** 115/115 (100.0%)

This document serves as the live registry of remaining work, active plans, and known limitations in GoPOSIX.

---

## 📈 Current Project State

| Metric | Value |
|--------|-------|
| **Total Utilities Implemented** | **115** (all registered via `dispatch.Register`) |
| **Overall Statement Coverage** | **82.3%** (fully compliant with the `>=80%` CI gate) |
| **JSON-RPC Daemon Coverage** | **115/115** utilities with structured output tests |
| **Multicall Compatibility** | Complete dispatching via symlinks or direct subcommands |
| **CGO Status** | 100% CGO-free Go (`CGO_ENABLED=0`) |

---

## 🛠️ Active Phase: Phase 27 (High-Complexity Tier 5)

Phase 26 (Tiers 1–4) is **complete**. Phase 27 (Tier 5) is also **complete** (11 of 11 implemented). The full breakdown is documented on the planning page:

👉 **[wiki/27_high_complexity_tools.md](27_high_complexity_tools.md)**

### Tier 5 Utilities (11 Utilities - COMPLETE ✅)
* **Compression & Archiving (2)**: ✅ `ar`, ✅ `cpio`
* **Development & Hex (3)**: ✅ `rx`, ✅ `hexdump`, ✅ `xxd`
* **Mathematics (2)**: ✅ `dc`, ✅ `bc`
* **Shell (1)**: ✅ `ash` (alias to existing native `shell` implementation)
* **System Admin & Hardware (3)**: ✅ `mdev`, ✅ `mkfs.minix`, ✅ `mount`

## ❌ Known Limitations & Remaining Failures

### 1. `bc` — 22 failures (precision and scale differences)
* Multi-precision scale and formatting quirks in decimal/fractional division under specific ibase/obase configurations.
* *Status*: **Accepted** (core arithmetic is fully functional and correct).

### 2. `realpath` — 3 failures (non-existent link resolution differences)
* Minor directory structure resolution discrepancies on non-existent symlinks.
* *Status*: **Accepted** (standard realpath resolution operates correctly).

### 3. Compliance and Verification Updates
* **Resolved & Verified**: `cpio`, `pidof`, and `mkfs.minix` integration test suites now **pass 100%** after implementing standard-block counting wrapper streams, argv[0]-only process boundaries, and exact Minix `.badblocks` directory packing.

### 5. Compliance tests — ✅ COMPLETE
* 28 `test/compliance/test_<name>.sh` scripts written for all Phase 26 Tier 4 and Phase 27 tools.
* 84 assertions, 0 failures. 1 test skipped (uncompress needs system `compress`).

### 6. JSON-RPC tests — 0 remaining gaps (115/115 tested)

---

## 📋 Backlog & Deferred Work

### 1. `awk` — 16 failures (goawk v1.31.0 engine limitations)
* Hex/Octal constants, scoped nested variables, scoped scopes, and bitwise operations are not supported by the underlying parsing engine.
* *Status*: **Backlog / Deferred** (see [wiki/deferred.md](deferred.md) for full context).
* **6 skipped** for hard constraints:
  - `ash` — shell's custom flag parser conflicts with daemon's `--json` auto-prepend
  - `wget` — requires live network connectivity
  - `daemon` — cannot run a daemon inside the daemon process
  - `mount` — requires root privileges for most operations
  - `mdev` — requires root + kernel hotplug infrastructure
  - `makedevs` — requires root for device node creation
* **3 aliases** (`egrep`, `fgrep`, `gunzip`) share their parent's RPC method and are tested through `goposix.grep` / `goposix.gzip`.
* **Verdict:** every utility has a JSON-RPC test or a documented reason it can't be tested.

---

## 🚀 Historical Roadmaps & Archive

The sequence of completed milestones, architectural changes, and past iterations (Phases 01 to 25) are preserved for historical reference in the master phase ledger:

👉 **[wiki/phases.md](phases.md)**
