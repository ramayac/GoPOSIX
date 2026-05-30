# GoPOSIX — Open TODOs & Remaining Work

> **Last updated:** 2026-05-30 | **Utilities:** 115 | **Coverage:** 84.1% | **BusyBox:** 877/17/25 (98.1%) | **JSON-RPC Daemon:** 115/115 (100.0%)

This document serves as the live registry of remaining work, active plans, and known limitations in GoPOSIX.

---

> 📊 **Per-utility status** → **[wiki/test_coverage_matrix.md](test_coverage_matrix.md)**
> 🛡️ **Hardening V results** → **[wiki/31_hardening_v.md](31_hardening_v.md)**
> ⚡ **Performance opportunities** → **[wiki/30_performance_improvements.md](30_performance_improvements.md)**
> ✅ **Completed changelog** → **[wiki/log.md](log.md)**
> 🗺️ **Phase history** → **[wiki/phases.md](phases.md)**

---

## 🟢 Deferred

### `awk` — 17 BusyBox failures + 8 skipped · coverage 90.0%

Blocked by upstream `goawk` v1.31.0 engine limitations: no bitwise ops, hex/octal constants, function arg parsing (4 tests), nested loop scoping, empty-paren handling, negative field access, continue/break edges, backslash-newline handling. *See:* [wiki/deferred.md](deferred.md).

### Coverage — 13 packages blocked (hard-to-mock error paths)

| Tier | Packages | Blocker |
|------|----------|---------|
| Near 80% (78-79%) | `whoami`, `cp`, `tee`, `pwd`, `hostname` | Syscall error mocking (`user.Current()`, `os.Getwd()`) |
| Mid-range (73-77%) | `client`, `internal/daemon`, `nohup`, `diff` | Integration test infra (spawned daemon, file perms) |
| Hard (64-71%) | `chgrp`, `logname`, `shell`, `gzip` | Deep I/O + OS-level error injection |

12 packages pushed above 80% in Hardening V (25 → 13). Remaining 13 require interface-based mocking or integration harnesses. *See:* [wiki/31_hardening_v.md](31_hardening_v.md).

### Go-Alpine Coexistence Daemon Target

Planning. Docker target where `goposix` runs as a daemon alongside Alpine's native BusyBox/shell tools, serving JSON-RPC on a socket. *See:* [wiki/alpine_plan.md](alpine_plan.md).

---

## ⚪ Root-Required — Cannot Test in CI (23 skipped)

| Tool | Skipped | Reason |
|------|---------|--------|
| `mdev` | 13 | Requires root + `/sys` kernel infrastructure |
| `cpio` | 7 | suid/sgid preservation, uid/gid defaults, `-R` owner flag |
| `mount` | 1 | Requires `CAP_SYS_ADMIN` |
| `makedevs` | 1 | Device node creation requires root |
| `ash` | 1 | Needs interactive shell session |
| **Total** | **23** | |

(+2 awk deferred skips = 25 total skipped)
