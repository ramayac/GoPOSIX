# GoPOSIX — Project Roadmap & State

> **Version:** 5.4 | **Date:** 2026-05-17 | **Tier:** GOLD | **Branch:** `main`
>
> **Status:** 77 utilities | 548 BusyBox passes (99.3%) | ~72% coverage | 59/77 JSON-RPC
>
> ✅ Phase 15 COMPLETED — dd + od (2 I/O utilities, 10 BusyBox tests)
> ✅ Phase 16 COMPLETED — 9 Tier 2 utilities
> ✅ Phase 17 COMPLETED — 12 Tier 3 utility stubs
> ✅ Phase 18 COMPLETED — CI, patch, egrep/fgrep, coverage (daemon 64.6%, diff 57.1%, client 55.4%)
> 📋 Phase 19 IMPLEMENTING — Performance Benchmarking (GoPOSIX vs BusyBox)

---

## Current State

All build phases (00–10) and post-MVP cleanups (11–14c) are **COMPLETED**. The project is at **Gold** tier.

| Metric | Value |
|--------|-------|
| BusyBox pass rate | 548 passed / 4 failed / 10 skipped (99.3%) |
| Overall test coverage | 70.4% |
| Utilities implemented | 77 |
| JSON-RPC daemon coverage | 59/77 utilities (77%) |
| Daemon unit coverage | 64.6% (from 35.9%) |
| Supply chain security | SBOM + Cosign + SLSA L3 + Trivy |
| Shell security model | Documented + tested (GOPOSIX_SHELL_TIMEOUT, SecurePath, 128MB limits) |

### Architecture

```
goposix binary (single static ELF, <12MB)
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
| **Post-MVP** | `dd`, `od`, `patch` (`egrep`, `fgrep` aliases) | 15/18 ✅ |
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
| [12_road_to_gold.md](12_road_to_gold.md) | Road to Gold — Gap analysis + resolution plan (12.0–12.4) | **GOLD ACHIEVED** |
| [13_coverage_and_hardening.md](13_coverage_and_hardening.md) | Coverage & Hardening — Audit findings + ramp plan | **COMPLETED** (70.5% reached, exceeds 60% target) |
| [14_xml_output.md](14_xml_output.md) | XML Output Support — `--xml` flag design (not implemented) | **DEFERRED** |
| [14a_json_gap_fill.md](14a_json_gap_fill.md) | JSON Gap Fill — 8 utilities now support `--json` | **COMPLETED** |
| [14b_busybox_regression_fix.md](14b_busybox_regression_fix.md) | BusyBox Regression Fix — 79→3 failures | **COMPLETED** |
| [14c_posix_json_gap.md](14c_posix_json_gap.md) | JSON-RPC Coverage Gap — 55/55 utilities tested via daemon | **COMPLETED** |
| [15_post_mvp_tier1.md](15_post_mvp_tier1.md) | Phase 15 — Tier 1: `dd` + `od` (10 BusyBox tests) | **COMPLETED** |
| [16_post_mvp_tier2.md](16_post_mvp_tier2.md) | Phase 16 — Tier 2: 9 text/stream utilities (43 BusyBox tests) | **COMPLETED** |
| [17_post_mvp_tier3.md](17_post_mvp_tier3.md) | Phase 17 — Tier 3: 12 utilities without BusyBox coverage | **COMPLETED** |
| [18_quality_fixes.md](18_quality_fixes.md) | Phase 18 — Quality Fixes: CI, `patch`, coverage, aliases | **COMPLETED** |
| [19_performance_benchmarking.md](19_performance_benchmarking.md) | Phase 19 — Performance Benchmarking (GoPOSIX vs BusyBox) | **IMPLEMENTING** |
| [todos.md](todos.md) | Open TODOs, Remaining Failures & Session Insights | **LIVING DOC** |

---

## Active Work

| # | Item | Doc |
|---|------|-----|
| — | Performance benchmarking vs BusyBox | [19_performance_benchmarking.md](19_performance_benchmarking.md) — implementing (`feat/performance`) |
| — | `awk` implementation (Platinum gate) | [07a_awk.md](07a_awk.md) — deferred |
| — | Low-coverage hardening (split, tty — all >60% now) | [todos.md](todos.md) |
| — | XML output (`--xml`) | [14_xml_output.md](14_xml_output.md) — deferred |
| — | Ongoing maintenance | [todos.md](todos.md) — living doc |

## Historical Phase Docs

These phase-plan documents describe completed work and are retained for reference only:

| File | Phase |
|------|-------|
| [00_foundation_libs.md](00_foundation_libs.md) | Phase 00 — Foundation Libraries |
| [01_multicall_tier1.md](01_multicall_tier1.md) | Phase 01 — Multicall + Tier 1 |
| [02_docker_ci.md](02_docker_ci.md) | Phase 02 — Docker CI (maintained as living doc) |
| [03_filesystem_utils.md](03_filesystem_utils.md) | Phase 03 — Filesystem Utils |
| [04_text_processing.md](04_text_processing.md) | Phase 04 — Text Processing |
| [05_daemon_core.md](05_daemon_core.md) | Phase 05 — Daemon Core |
| [06_system_utils.md](06_system_utils.md) | Phase 06 — System Utils |
| [07_agent_features.md](07_agent_features.md) | Phase 07 — Agent Features |
| [08_hardening.md](08_hardening.md) | Phase 08 — Hardening |
| [09_release_docs.md](09_release_docs.md) | Phase 09 — Release (maintained as living doc) |
| [10_posix_framework.md](10_posix_framework.md) | Phase 10 — POSIX Framework |
| [10a_sed.md](10a_sed.md) | Phase 10a — Sed Details |
| [11_post_mvp_priorities.md](11_post_mvp_priorities.md) | Phase 11 — Post-MVP Priorities |
| [11a_lower_priority.md](11a_lower_priority.md) | Phase 11a — Lower Priority |
| [11_lessons_learned.md](11_lessons_learned.md) | Phase 11 — Lessons Learned |

## Risk Register

| Risk | Impact | Status |
|------|--------|--------|
| `awk` complexity | High | Deferred ([07a_awk.md](07a_awk.md)) |
| Go regexp ≠ POSIX BRE | Med | Documented — by design |
| Binary size bloat | Med | Mitigated — `-ldflags="-s -w"`, ~10MB |
