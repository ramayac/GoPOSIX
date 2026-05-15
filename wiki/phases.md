# KoreGo — Development Roadmap

> **Version:** 3.0 | **Date:** 2026-05-15

---

## Post-MVP Focus (Five Pillars)

All build phases (00–10) are **COMPLETED**. The project now targets:

| Pillar | Goal | Primary Doc |
|--------|------|-------------|
| **Coverage** | 75% overall test coverage | [13_coverage_and_hardening.md](13_coverage_and_hardening.md) |
| **POSIX** | 99%+ BusyBox pass rate, zero regressions | [posix_coverage.md](posix_coverage.md), [10_posix_framework.md](10_posix_framework.md) |
| **Security** | Hardened shell, SBOM, Cosign/SLSA, secured defaults | [08_hardening.md](08_hardening.md), [docs/SECURITY.md](../docs/SECURITY.md) |
| **Speed** | <1ms daemon, <12MB binary, <10MB image, <5ms CLI | [13_coverage_and_hardening.md](13_coverage_and_hardening.md) |
| **Docker** | Usable, signed `FROM scratch` image, smoke-tested | [09_release_docs.md](09_release_docs.md) |

### Architecture

```
korego binary (single static ELF, <12MB)
├── Multicall Dispatch (os.Args[0] or subcommand)
├── CLI Wrappers (--json flag → JSON envelope)
├── Daemon Mode (JSON-RPC 2.0 over Unix socket)
└── pkg/ Libraries (return typed Go structs)
    ├── pkg/echo/     → EchoResult
    ├── pkg/ls/       → []FileInfo
    ├── pkg/grep/     → []MatchResult
    └── pkg/common/   → JSON-RPC types, flag parser, output helpers
```

### Utility Tiers (All Complete)

| Tier | Utilities | Phase |
|------|-----------|-------|
| **1 — Trivial** | `echo`, `true`, `false`, `yes`, `whoami`, `hostname`, `uname`, `pwd`, `printenv`, `env` | 01 ✅ |
| **2 — Filesystem** | `ls`, `cat`, `mkdir`, `rmdir`, `rm`, `cp`, `mv`, `touch`, `ln`, `stat`, `readlink`, `basename`, `dirname` | 03 ✅ |
| **3 — Text** | `head`, `tail`, `wc`, `sort`, `uniq`, `tr`, `cut`, `tee`, `grep`, `sed` | 04 ✅ |
| **4 — System** | `ps`, `kill`, `sleep`, `date`, `id`, `groups`, `chmod`, `chown`, `chgrp`, `df`, `du`, `find`, `xargs` | 06 ✅ |
| **5 — Advanced** | `tar`, `gzip`, `sha256sum`, `md5sum`, `diff`, `patch`, `test`/`[`, `printf`, `expr` | 07 ✅ |
| **Platinum** | `awk` | 07a ⏳ |

### Technical Specs

| Spec | Value |
|------|-------|
| Language | Go 1.22+ (Pure Go, `CGO_ENABLED=0`) |
| Protocol | JSON-RPC 2.0 over Unix Domain Sockets |
| Base Image | `scratch` (prod), `alpine` (debug) |
| Key Dep | `mvdan.cc/sh/v3` (shell interpreter) |
| Binary Target | < 12 MB stripped |
| Image Target | < 10 MB |
| Daemon Latency | < 1ms |

---

## Phase Files Index

| File | Title | Status |
|------|-------|--------|
| [00_foundation_libs.md](00_foundation_libs.md) | Foundation Libraries (flag parser, JSON envelope, JSON-RPC types) | **COMPLETED** |
| [01_multicall_tier1.md](01_multicall_tier1.md) | Multicall Dispatcher + Tier 1 Utilities | **COMPLETED** |
| [02_docker_ci.md](02_docker_ci.md) | Docker Scratch Build + CI Pipeline | **COMPLETED** |
| [03_filesystem_utils.md](03_filesystem_utils.md) | Tier 2 — Filesystem Utilities | **COMPLETED** |
| [04_text_processing.md](04_text_processing.md) | Tier 3 — Text Processing Utilities | **COMPLETED** |
| [05_daemon_core.md](05_daemon_core.md) | JSON-RPC Daemon — Core Server | **COMPLETED** |
| [06_system_utils.md](06_system_utils.md) | Tier 4 — System & Process Utilities | **COMPLETED** |
| [07_agent_features.md](07_agent_features.md) | Agent-Ready Features (sessions, shell, Tier 5) | **COMPLETED** |
| [08_hardening.md](08_hardening.md) | Production Hardening & Security | **COMPLETED** |
| [09_release_docs.md](09_release_docs.md) | Release Automation & Documentation | **COMPLETED** |
| [10_posix_framework.md](10_posix_framework.md) | POSIX Testing Framework Integration | **COMPLETED** |
| [10a_sed.md](10a_sed.md) | Sed Implementation Details | **COMPLETED** |
| [07a_awk.md](07a_awk.md) | Awk Implementation Plan (canonical; Platinum gate) | **DEFERRED** |
| [posix_coverage.md](posix_coverage.md) | POSIX Compliance Matrix (49 utilities) | **COMPLETED** |
| [posix_faq.md](posix_faq.md) | POSIX Compliance FAQ | **COMPLETED** |
| [11_lessons_learned.md](11_lessons_learned.md) | Phase 11 Lessons Learned, Insights & Gotchas | **COMPLETED** |
| [11_post_mvp_priorities.md](11_post_mvp_priorities.md) | Post-MVP Priorities (11.1–11.3 complete; 11.4 awk → 07a) | **COMPLETED** |
| [11a_lower_priority.md](11a_lower_priority.md) | Lower Priority Improvements (8/8 complete) | **COMPLETED** |
| [12_road_to_gold.md](12_road_to_gold.md) | Road to Gold (5/5 Gold gaps resolved) | **GOLD ACHIEVED** |
| [13_coverage_and_hardening.md](13_coverage_and_hardening.md) | Coverage & Hardening — Audit findings + ramp plan (50%→75%) | **IN PROGRESS** |
| [todos.md](todos.md) | Open TODOs, Remaining Failures & Session Insights | **LIVING DOC** |

---

## Risk Matrix

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| POSIX spec ambiguity | Med | High | Use GNU coreutils behavior as reference |
| `awk` complexity | High | Med | Deferred to post-MVP (see 07a_awk.md) |
| Binary size bloat | Med | Med | `-ldflags="-s -w"`, strip, UPX |
| Daemon memory leaks | High | Med | `go test -race`, `pprof`, session TTLs |
| Shell interpreter security | High | Med | Sandbox: no network, restricted fs, timeouts |
| Go regexp ≠ POSIX BRE | Med | High | Document differences, custom BRE if needed |
