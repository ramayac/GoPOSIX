# GoPOSIX — Open TODOs & Remaining Work

> **Last updated:** 2026-10-03 | **Utilities:** 115 | **Coverage:** 85.6% | **BusyBox:** 870/17/30 (98.1%) | **JSON-RPC Daemon:** 115/115 (100.0%)

This document serves as the live registry of remaining work, active plans, and known limitations in GoPOSIX.

---

> 📊 **Per-utility status** → **[wiki/test_coverage_matrix.md](test_coverage_matrix.md)**
> 🔍 **Command audit plan (IN PROGRESS)** → **[wiki/28_posix_command_audit.md](28_posix_command_audit.md)**
> 🛡️ **Hardening V results** → **[wiki/hardening.md](hardening.md)**
> ⚡ **Performance opportunities** → **[wiki/performance.md](performance.md)**
> ✅ **Completed changelog** → **[wiki/log.md](log.md)**
> 🗺️ **Phase history** → **[wiki/phases.md](phases.md)**

---

## 🟢 Deferred

### `awk` — 17 BusyBox failures + 8 skipped · coverage 90.0%

Blocked by upstream `goawk` v1.31.0 engine limitations: no bitwise ops, hex/octal constants, function arg parsing (4 tests), nested loop scoping, empty-paren handling, negative field access, continue/break edges, backslash-newline handling. *See:* [wiki/deferred.md](deferred.md).

### Coverage — 0 packages blocked ✅ RESOLVED (audit Phase 3, 2026-10-03)

All 12 packages are now ≥ 80% (project audit branch `audit/posix-commands`):
`whoami` 100.0%, `hostname` 98.2%, `diff` 89.5%, `gzip` 87.3%, `nohup` 80.9%,
`client` 83.7%, `internal/daemon` 82.4%, `chgrp` 83.3%, `logname` 80.0%, `shell` 90.2%,
`cp` 82.6%, `tee` 92.3%. The function-seam pattern (`var userCurrent = user.Current`,
`var osHostname = os.Hostname`) from PR #42 solved the syscall-mocking blockers.
See [wiki/28_posix_command_audit.md](28_posix_command_audit.md).

### `start-stop-daemon` signal parsing consolidation ✅ RESOLVED

`pkg/start-stop-daemon` now uses `common.ParseSignal` from [pkg/common/signal.go](../pkg/common/signal.go)
(audit Phase 2 / F5, 2026-10-03). The local 7-name `parseSignal` was deleted — full 31-signal
table with SIG-prefix tolerance, case-insensitivity, and numeric parsing.

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
