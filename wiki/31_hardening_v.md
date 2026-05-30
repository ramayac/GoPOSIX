# Phase 31 — Hardening V (Coverage & Tar Compliance Audit)

> **Status:** ✅ COMPLETED | **Date:** 2026-05-30 | **Branch:** `feat/hardening_v`
>
> **Final results:**
> 1. Overall coverage: **83.7% → 84.1%**
> 2. **12 packages** pushed above 80% individual threshold.
> 3. Under-80% packages: **25 → 13** (all remaining blocked by hard-to-mock syscall/I/O error paths).
> 4. Tar: all 31 BusyBox tests pass (100%), coverage at 80.4%.

---

## Resolved: 12 Packages Crossed 80%

| # | Package | Before | After | Δ | Method |
|---|---------|:------:|:-----:|:--:|--------|
| 1 | `id` | 62.5% | **94.6%** | +32.1 | 8 flag combination tests (-u, -g, -G, -n) |
| 2 | `chown` | 71.8% | **92.3%** | +20.5 | Recursive flag, user:group format, invalid user |
| 3 | `kill` | 73.1% | **92.3%** | +19.2 | Invalid signal, invalid PID, signal by name |
| 4 | `sleep` | 78.1% | **87.5%** | +9.4 | Invalid duration, invalid suffix, missing arg |
| 5 | `df` | 79.2% | **87.5%** | +8.3 | Invalid mount path |
| 6 | `uname` | 76.7% | **86.0%** | +9.3 | All-flag combination test (-s, -n, -r, -m, -a) |
| 7 | `cksum` | 76.4% | **85.5%** | +9.1 | Stdin piped, multiple files |
| 8 | `md5sum` | 79.6% | **84.7%** | +5.1 | Invalid checksum file line |
| 9 | `ln` | 79.3% | **82.8%** | +3.5 | Force flag, target-is-directory |
| 10 | `cmp` | 76.0% | **82.3%** | +6.3 | -l verbose, stdin dash, stdin vs file |
| 11 | `readlink` | 76.8% | **81.2%** | +4.4 | -f canonicalize missing path |
| 12 | `date` | 79.3% | **81.0%** | +1.7 | POSIX TZ parsing, timestamp edge cases |

---

## Deferred: 13 Packages Blocked (by difficulty tier)

### Near threshold (78-79%) — syscall error mocking needed

| Package | Coverage | Gap | Blocker |
|---------|:------:|:----:|---------|
| `whoami` | 78.9% | 1.1% | `user.Current()` error unmockable |
| `cp` | 78.9% | 1.1% | File copy error paths |
| `tee` | 78.8% | 1.2% | Writer write-error injection |
| `pwd` | 78.3% | 1.7% | `os.Getwd()` error unmockable |
| `hostname` | 78.2% | 1.8% | Set-hostname syscall error |

### Mid-range (73-77%) — integration/I/O mocking needed

| Package | Coverage | Gap | Blocker |
|---------|:------:|:----:|---------|
| `client` | 76.6% | 3.4% | Needs running daemon for connection-refused/timeout paths |
| `internal/daemon` | 75.8% | 4.2% | Integration tests for connection lifecycle |
| `nohup` | 75.0% | 5.0% | File permission error injection |
| `diff` | 73.9% | 6.1% | Directory comparison, binary diff edges |

### Hard (64-71%) — deep I/O + syscall mocking

| Package | Coverage | Gap | Blocker |
|---------|:------:|:----:|---------|
| `chgrp` | 70.0% | 10.0% | OS-level permission errors |
| `logname` | 70.0% | 10.0% | `os/user` error mocking |
| `shell` | 67.1% | 12.9% | Complex subprocess error injection |
| `gzip` | 64.7% | 15.3% | Corrupt header detection, write-error propagation |

---

## Tar Deep Dive (Completed)

All 31 BusyBox integration tests pass (100% compliance). Tar coverage at **80.4%**. `resolveTarPath` correctly normalizes directory traversal (handles `.`, `..`, leading `./`, deep `../../` escape paths). Symlink safety with pre-scan conflict detection prevents traversal attacks.

---

## Hardening Guidelines

1. **Quick wins exhausted.** All remaining sub-80% packages are blocked by hard-to-mock syscall error paths or require integration test infrastructure.
2. **Architectural isolation.** Use `bytes.Buffer` for I/O, avoid process-level side effects.
3. **Future work.** The 13 deferred packages need either interface-based mocking (e.g., filesystem abstraction) or integration test harnesses (spawned daemon, temp filesystem with controlled permissions).
