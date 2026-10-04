# GoPOSIX — Open TODOs & Remaining Work

> **Last updated:** 2026-10-03 | **Utilities:** 115 | **Coverage:** 87.6% | **BusyBox:** 871/16/30 (98.2%) | **JSON-RPC Daemon:** 115/115 (100.0%)

This document serves as the live registry of remaining work, active plans, and known limitations in GoPOSIX.

---

> 📊 **Per-utility status** → **[wiki/test_coverage_matrix.md](test_coverage_matrix.md)**
> 🔍 **Command audit plan (PAUSED — see section below)** → **[wiki/28_posix_command_audit.md](28_posix_command_audit.md)**
> 🛡️ **Hardening V results** → **[wiki/hardening.md](hardening.md)**
> ⚡ **Performance opportunities** → **[wiki/performance.md](performance.md)**
> ✅ **Completed changelog** → **[wiki/log.md](log.md)**
> 🗺️ **Phase history** → **[wiki/phases.md](phases.md)**

---

## 🟡 Phase 28 — POSIX Command Audit (paused 2026-10-03)

Work stopped after the repository-level phases. The per-tool deep audit continues later.

### ✅ Done — phases 0–3, all findings F1–F7 resolved

Delivered in [PR #43](https://github.com/ramayac/GoPOSIX/pull/43) (`audit/posix-commands`, 13 commits, awaiting review):

| Finding | Result |
|---------|--------|
| F1 | All 53 commands use injected writers. Daemon separates stdout/stderr buffers and returns a `stderr` field in JSON-RPC responses. |
| F2 | Shared digest core in [pkg/common/digest.go](../pkg/common/digest.go). The five sum tools dropped from 1,485 to 348 LOC. |
| F3 | Every package ≥ 80% coverage. |
| F5 | `start-stop-daemon` uses `common.ParseSignal` (31-signal table). |
| F6 | `bc` `NewInterpreter` returns an error. No `panic` in the math-lib path. |
| F7 | Shared decompression core in [pkg/common/decompress.go](../pkg/common/decompress.go). The four tools dropped from 950 to ~230 LOC. |

Verification at stop: 100% patch coverage, overall coverage 87.6%, BusyBox 871/16/30 (17 awk failures deferred, no other regressions).

### ▶️ Next when work resumes — Phase 4: deep audit of 7 XL/L commands

The plan preflight corrected the XL/L scope to 7 commands (non-test LOC ≥ 700):
`bc` (score 5.50), `sed` (3.75), `printf`, `date`, `tar`, `dc`, `diff`.
Work one PR per command, ordered by the `PreAudit` score in
[wiki/28_posix_command_audit.md](28_posix_command_audit.md). After that: Phase 5 sweeps the M/S commands.

### ⏳ Open audit items

| Item | What |
|------|------|
| F8 | Hand-rolled parser helpers duplicated across `bc`, `expr`, `sed`, `testcmd`. Move to a shared expression core in `pkg/common`. |
| P1 | `logger` keeps a package-global `stderrWriter`. Replace with an injected writer (daemon safety, audit item 10). |
| P2 | [wiki/test_coverage_matrix.md](test_coverage_matrix.md) is stale (e.g. `chmod` 68.3% vs 92.7% measured). Refresh from the tree. |
| P4 | Two `make testsuite` runs in the same checkout corrupt each other (shared `runtest-tempdir-links`). Re-run an applet alone before reporting a regression. |
| README | Add the Phase 28 link after PR #43 merges. |

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
