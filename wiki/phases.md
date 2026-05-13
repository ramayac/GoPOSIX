# KoreGo — Development Roadmap

> **Version:** 2.1 | **Date:** 2026-05-02

---

## Analysis of the Original Plan

### Strengths

- Clear vision: Go-native POSIX userland with JSON-RPC for agentic runtimes.
- Multicall binary pattern (BusyBox-style) is proven and efficient.
- `FROM scratch` Docker image is the gold standard for minimal attack surface.
- Library-first design (`pkg/ls` returns structs) enables both CLI and RPC.

### Gaps Identified

| Gap | Risk | Fix |
|-----|------|-----|
| No POSIX flag parser up front | Blocks every utility — `-laR` must work | Move to Phase 00 as foundational |
| `--json` output format undefined | Each utility invents its own schema | Define a universal JSON envelope in Phase 00 |
| Only 4 utilities in Phase 2 | Far from "100% POSIX compliant" | Tier utilities (50+) across phases |
| No testing strategy per phase | "80% POSIX" is unmeasurable | Add concrete test commands & compliance suite |
| No CI/CD pipeline | Regressions go undetected | Add GitHub Actions in Phase 01 |
| No Phase 5 for hardening | Security & release not addressed | Add production hardening phase |
| Go `regexp` != POSIX BRE/ERE | `grep`/`sed` will silently differ | Document; may need custom BRE parser |
| No binary size / latency targets | No way to track perf regressions | Set targets: <15MB binary, <1ms daemon |
| Missing docs plan | JSON schemas, RPC API undocumented | Add 7 documentation deliverables |
| No `sed` mentioned at all | Core POSIX text tool missing | Add to Tier 3 |

### Architecture

```
korego binary (single static ELF, ~15MB)
├── Multicall Dispatch (os.Args[0] or subcommand)
├── CLI Wrappers (--json flag → JSON envelope)
├── Daemon Mode (JSON-RPC 2.0 over Unix socket)
└── pkg/ Libraries (return typed Go structs)
    ├── pkg/echo/     → EchoResult
    ├── pkg/ls/       → []FileInfo
    ├── pkg/grep/     → []MatchResult
    └── pkg/common/   → JSON-RPC types, flag parser, output helpers
```

### Utility Tiers

| Tier | Utilities | Phase |
|------|-----------|-------|
| **1 — Trivial** | `echo`, `true`, `false`, `yes`, `whoami`, `hostname`, `uname`, `pwd`, `printenv`, `env` | 01 |
| **2 — Filesystem** | `ls`, `cat`, `mkdir`, `rmdir`, `rm`, `cp`, `mv`, `touch`, `ln`, `stat`, `readlink`, `basename`, `dirname` | 03 |
| **3 — Text** | `head`, `tail`, `wc`, `sort`, `uniq`, `tr`, `cut`, `tee`, `grep`, `sed` | 04 |
| **4 — System** | `ps`, `kill`, `sleep`, `date`, `id`, `groups`, `chmod`, `chown`, `chgrp`, `df`, `du`, `find`, `xargs` | 06 |
| **5 — Advanced** | `awk` (Deferred), `tar`, `gzip`, `sha256sum`, `md5sum`, `diff`, `patch`, `test`/`[`, `printf`, `expr` | 07 |

### Directory Layout

```
korego/
├── cmd/korego/main.go          # Multicall entry point
├── pkg/
│   ├── common/                 # JSON-RPC types, flag parser, output helpers
│   ├── echo/                   # One package per utility
│   ├── ls/
│   └── ...
├── internal/
│   ├── dispatch/               # Command registry + routing
│   ├── daemon/                 # Unix socket server + RPC router
│   └── shell/                  # mvdan/sh integration
├── test/
│   ├── compliance/             # POSIX compliance tests
│   ├── integration/            # Docker-based E2E tests
│   └── benchmark/              # CLI vs daemon latency
├── docker/
│   ├── Dockerfile              # Multi-stage → scratch
│   └── Dockerfile.debug        # Multi-stage → alpine
├── docs/
│   ├── JSON_SCHEMA.md
│   ├── RPC_API.md
│   └── ...
├── go.mod
├── Makefile
└── README.md
```

### Technical Specs

| Spec | Value |
|------|-------|
| Language | Go 1.22+ (Pure Go, `CGO_ENABLED=0`) |
| Protocol | JSON-RPC 2.0 over Unix Domain Sockets |
| Base Image | `scratch` (prod), `alpine` (debug) |
| Key Dep | `mvdan.cc/sh/v3` (shell interpreter) |
| Binary Target | < 15 MB stripped |
| Image Target | < 20 MB |
| Daemon Latency | < 1ms trivial, < 5ms filesystem |

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
| [todos.md](todos.md) | Open TODOs, Remaining Failures & Session Insights | **LIVING DOC** |
| [10a_sed.md](10a_sed.md) | Sed Implementation Details | **COMPLETED** |
| [07a_awk.md](07a_awk.md) | Awk Deferral Analysis | **DEFERRED** |
| [posix_coverage.md](posix_coverage.md) | POSIX Compliance Matrix (49 utilities) | **COMPLETED** |
| [posix_faq.md](posix_faq.md) | POSIX Compliance FAQ | **COMPLETED** |
| [11_lessons_learned.md](11_lessons_learned.md) | Phase 11 Lessons Learned, Insights & Gotchas | **COMPLETED** |
| [11_post_mvp_priorities.md](11_post_mvp_priorities.md) | Post-MVP Priorities | **COMPLETED** |
| [11a_lower_priority.md](11a_lower_priority.md) | Lower Priority Improvements | **IN PROGRESS** |

---

## Risk Matrix

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| POSIX spec ambiguity | Med | High | Use GNU coreutils behavior as reference |
| `awk`/`sed` complexity | High | High | `awk` deferred to post-MVP, `sed` implemented incrementally |
| Binary size bloat | Med | Med | `-ldflags="-s -w"`, build tags |
| Daemon memory leaks | High | Med | `go test -race`, `pprof`, session TTLs |
| Shell interpreter security | High | Med | Sandbox: no network, restricted fs, timeouts |
| Go regexp ≠ POSIX BRE | Med | High | Document differences, custom BRE if needed |
