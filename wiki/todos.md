# GoPOSIX — Open TODOs & Remaining Work

> **Last updated:** 2026-05-23 | **Utilities:** 110 | **Coverage:** 83.8% | **JSON-RPC Daemon:** 110/110 (100%)

This document serves as the live registry of remaining work, active plans, and known limitations in GoPOSIX.

---

## 📈 Current Project State

| Metric | Value |
|--------|-------|
| **Total Utilities Implemented** | **110** (all registered via `dispatch.Register`) |
| **Overall Statement Coverage** | **83.8%** (fully compliant with the `>=80%` CI gate) |
| **JSON-RPC Daemon Coverage** | **110/110** utilities supported |
| **Multicall Compatibility** | Complete dispatching via symlinks or direct subcommands |
| **CGO Status** | 100% CGO-free Go (`CGO_ENABLED=0`) |

---

## 🛠️ Active Phase: Phase 27 (High-Complexity Tier 5)

With the complete success of Tiers 1–4, the active focus shifts to the final remaining BusyBox-tested utilities. These are cataloged in detail on the dedicated Phase 27 planning page:

👉 **[wiki/27_high_complexity_tools.md](27_high_complexity_tools.md)**

### Tier 5 Utilities (11 Utilities - PLANNING ⏳)
* **Compression & Archiving (2)**: `ar`, `cpio`
* **Development & Hex (3)**: `hexdump`, `xxd`, `rx`
* **Mathematics (2)**: `bc`, `dc`
* **Shell (1)**: `ash` (alias to existing native `shell` implementation)
* **System Admin & Hardware (3)**: `mdev`, `mkfs.minix`, `mount`

---

## ❌ Known Limitations & Remaining Failures

### 1. `awk` — 17 failures (goawk v1.31.0 engine limitations)
* Hex/Octal constants, scoped nested variables, scoped scopes, and bitwise operations are not supported by the underlying parsing engine.
* *Status*: **Deferred** (see [wiki/deferred.md](deferred.md) for full context).

### 2. `realpath` — 3 failures (symlink environment resolving differences)
* Canonical path resolution differences on non-existent symlink endpoints under specific sandbox/CWD structures.
* *Status*: **Accepted** (POSIX-compliant baseline is achieved).

---

## 🚀 Historical Roadmaps & Archive

The sequence of completed milestones, architectural changes, and past iterations (Phases 01 to 25) are preserved for historical reference in the master phase ledger:

👉 **[wiki/phases.md](phases.md)**
