# Hardening Phases — Consolidated

> **Last updated:** 2026-05-30 | **Overall coverage:** 84.1% | **BusyBox:** 877/17/25 (98.1%)

GoPOSIX hardening phases II through V. Each phase targeted a specific architectural or quality dimension.

---

## Phase 20 — Hardening II (Post-Gold Audit)

**Date:** 2026-05-18 | **Coverage:** 76.7% | **BusyBox:** 548/4/10

Full-architecture audit resolving all CRITICAL and HIGH items. Score: 87 → 95/100.

**Key outcomes:**
- Flag audit across all utilities — identified and fixed short-flag collisions (`-j` for `--json` removed)
- Input safety: NUL byte handling in `sed` and `grep` parsers, POSIX-compliant flag parsing escape hatches
- Code cleanup: removed dead code, standardized error handling patterns
- Coverage gate established at ≥80%

---

## Phase 22 — Hardening III (Daemon-First Pivot)

**Date:** 2026-05-18

Benchmark-driven architecture pivot. Discovery: Go SDK with persistent connection achieves **60µs per RPC call — 11× faster than BusyBox fork+exec**. The old socat-per-call approach was 3× slower than BusyBox.

**Key outcomes:**
- Rebranded project messaging: "daemon-first" with CLI as secondary interface
- Daemon benchmark infrastructure (`make bench-quick`, `make bench-all`)
- Documented that daemon benchmarking through socat measures socat overhead, not daemon performance
- Established Go SDK (`pkg/client/`) as the primary programmatic interface
- Removed all socat-based forwarding; CLI forwarding through forwarder.go
- Daemon stdin support via `dispatch.Command.Run` signature expansion

---

## Phase 24 — Hardening IV (Architecture, Security & Compliance)

**Date:** 2026-05-21 | **Score:** 100/100 | **Gaps resolved:** 27

Comprehensive compliance gap audit. All 27 findings resolved.

**Key outcomes:**
- Shell redirect fix: `>&` and `<&` operators in shell interpreter
- Phase 25 Daemon stdin: expanded `dispatch.Command.Run` signature to include `stdin io.Reader`, plumbed through 40+ utilities
- Stderr refactor: replaced hardcoded `os.Stderr` with injectable `io.Writer` in all packages
- Injectable streams pattern: `run()` / `catRun()` pattern for testable CLI layer
- CWD signature refactoring (deferred): serialized shell execution via `execMu sync.Mutex`
- Date specifiers, grep binary detection, `ls -d`, PreProcess hook, +15 more compliance fixes
- BusyBox test suite integration established as per-commit gate

---

## Phase 31 — Hardening V (Coverage & Tar Compliance Audit)

**Date:** 2026-05-30 | **Overall:** 83.7% → 84.1% | **Under 80%:** 25 → 13

**Key outcomes:**
- **tar BusyBox:** 7 failures resolved → **31/31 (100%)** — symlink safety, hardlink dedup, XZ auto-detect
- **tar coverage:** 72.4% → 80.4%
- **12 packages** pushed above 80% individual coverage:
  - `id` (62.5% → 94.6%), `chown` (71.8% → 92.3%), `kill` (73.1% → 92.3%)
  - `sleep` (78.1% → 87.5%), `df` (79.2% → 87.5%), `uname` (76.7% → 86.0%)
  - `cksum` (76.4% → 85.5%), `md5sum` (79.6% → 84.7%), `ln` (79.3% → 82.8%)
  - `cmp` (76.0% → 82.3%), `readlink` (76.8% → 81.2%), `date` (79.3% → 81.0%)
- **13 packages deferred** — blocked by hard-to-mock syscall/I/O error paths
- **ls fix:** file-before-directory argument ordering, lowercase `l` for symlink mode character

---

## Remaining

| Area | Status | Reference |
|------|--------|-----------|
| 13 coverage packages | Deferred (syscall mocking needed) | `wiki/todos.md` |
| 17 awk BusyBox failures | Deferred (upstream goawk limits) | `wiki/deferred.md` |
| 18 performance optimizations | 12/30 done | `wiki/30_performance_improvements.md` |
| Daemon pipeline composition | Planning | `wiki/deferred.md` |
| Alpine daemon target | Planning | `wiki/alpine_plan.md` |
