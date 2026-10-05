---
status: current
description: "Open TODOs and remaining work."
references: [source:wiki/28_posix_command_audit.md, external:https://github.com/ramayac/GoPOSIX/pull/43]
---

# GoPOSIX — Open TODOs & Remaining Work

> **Last updated:** 2026-10-04 | **Utilities:** 115 | **Coverage:** 88.1% | **BusyBox:** 870/17/30 (98.1%) | **JSON-RPC Daemon:** 115/115 (100.0%)

This document serves as the live registry of remaining work, active plans, and known limitations in GoPOSIX.

---

> 📊 **Per-utility status** → **[wiki/test_coverage_matrix.md](test_coverage_matrix.md)**
> 🔍 **Command audit plan (ACTIVE — PR #47 in review)** → **[wiki/28_posix_command_audit.md](28_posix_command_audit.md)**
> 🛡️ **Hardening V results** → **[wiki/hardening.md](hardening.md)**
> ⚡ **Performance opportunities** → **[wiki/performance.md](performance.md)**
> ✅ **Completed changelog** → **[wiki/log.md](log.md)**
> 🗺️ **Phase history** → **[wiki/phases.md](phases.md)**

---

## 🟡 Phase 28 — POSIX Command Audit (active)

All numbered findings are resolved. PR #47 (draft, `audit/whatsleft`) closes F16, P1, P2, P4
and is in review. The per-tool deep audit continues after it merges.

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

### ✅ Done — 5d JSON Changes (PR #46, `audit/5d-json`)

| Finding | Result |
|---------|--------|
| F12 | 37 new schemas in [test/schemas/](../test/schemas/) with golden fixtures. `dd` and `daemon` are documented exemptions. `make validate-schemas` reports 0 skipped (115 passed since PR #47 added the `gunzip` schema). |
| F15 | `shell` parses `--json` in inline, file, and pipe modes; the daemon test asserts the envelope. `dd` remains the documented exception. |
| Daemon tests | New [test/posix-json/tier9_json_contract_test.go](../test/posix-json/tier9_json_contract_test.go) covers `bc`, `mount`, `hexdump`, `makedevs`, `mdev`, `mkfs.minix`, `wget`, `xxd`, `rx`, `shell`. |
| JSON error paths | `wget`, `which`, `seq`, `pidof`, `mdev`, `rx`, `shell` now honour `--json` on usage errors (see §5d.1 of the plan). |
| P7 | `gen_golden.sh` fixed (`set -u` bug, `%b` escapes, absolute paths) and extended for all 38 new fixtures. `pkg/who` now emits `users: []` instead of `null`. |

### ✅ Done — open findings F16, P1, P2, P4 (PR #47, `audit/whatsleft`, in review)

| Finding | Result |
|---------|--------|
| F16 | JSON stdout modes embed the payload as base64 `content` inside the envelope (F7 decompress core, `pkg/gzip`, `cpio -o` without `-F`). Text mode unchanged. Schemas for the six tools plus the new `gunzip` schema gained the optional `content` field. |
| P1 | `logger.Run` takes the injected `errOut io.Writer`; the package-global `stderrWriter` and the swap logic are deleted. Coverage 98.5%. |
| P2 | All 115 coverage cells in [wiki/test_coverage_matrix.md](test_coverage_matrix.md) refreshed from a `go test -cover` run (84 rows changed). Overall coverage 88.1%. |
| P4 | The harness builds the applet links in a per-run `mktemp -d` (removed on exit) and uses `.tmpdir.$applet.$$`. The tracked `runtest-tempdir-links/` tree is deleted from the repo. Verified with concurrent same-applet runs: zero failures. |

### ▶️ Next — Phase 4: deep audit of the XL/L commands

One PR per command, ordered by the `PreAudit` score:
`bc` (5.50) → `sed` (3.75) → `date` (3.75) → `tar` (3.50) → `dc` (3.25) → `diff` (2.25).
`printf` is done (`KEEP ✅`). See plan §6 in
[wiki/28_posix_command_audit.md](28_posix_command_audit.md).

### ⏳ Open audit items

| Item | What |
|------|------|
| PR #47 | Draft in review — closes F16, P1, P2, P4. Merge after review. |
| Phase 4 | Deep audit of 6 commands: `bc`, `sed`, `date`, `tar`, `dc`, `diff`. |
| Phase 5 | 13 open verdicts: `grep` (REFACTOR), `patch`, `hexdump`, `start-stop-daemon`, `unzip`, `uudecode`, `wget`, `xxd`, `sort`, `uuencode`, `taskset`, `rx`, `xargs`. |

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
