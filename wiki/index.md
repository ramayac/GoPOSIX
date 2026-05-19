# Wiki Index

## Core

- [README.md](README.md) | Purpose, rules, and shell-first navigation.
- [schema.md](schema.md) | Required structure and maintenance contract.
- [phases.md](phases.md) | Project roadmap, current state, and phase index
- [test_coverage_matrix.md](test_coverage_matrix.md) | Per-utility test status for all 74 utilities.
- [repo-map.md](repo-map.md) | Current repo architecture and exclusions.
- [log.md](log.md) | Append-only timeline of wiki maintenance.

## Current State

- [test_coverage_matrix.md](test_coverage_matrix.md) | **Start here** — complete test status for all 74 utilities
- [todos.md](todos.md) | Open TODOs, remaining BusyBox failures, and session insights
- [12_road_to_gold.md](12_road_to_gold.md) | Gold certification — gap analysis and resolution log (COMPLETED)
- [13_coverage_and_hardening.md](13_coverage_and_hardening.md) | Coverage audit, hardening plan, speed targets (COMPLETED — 70.5%)

## Historical Phase Docs (retained for reference)

All phases 00–11 are complete. These documents describe the as-built implementation.

| File | Phase |
|------|-------|
| [00_foundation_libs.md](00_foundation_libs.md) | Foundation Libraries |
| [01_multicall_tier1.md](01_multicall_tier1.md) | Multicall + Tier 1 |
| [02_docker_ci.md](02_docker_ci.md) | Docker CI (maintained as living doc) |
| [03_filesystem_utils.md](03_filesystem_utils.md) | Filesystem Utils |
| [04_text_processing.md](04_text_processing.md) | Text Processing |
| [05_daemon_core.md](05_daemon_core.md) | Daemon Core |
| [06_system_utils.md](06_system_utils.md) | System Utils |
| [07_rpc_features.md](07_rpc_features.md) | RPC Features |
| [08_hardening.md](08_hardening.md) | Hardening |
| [09_release_docs.md](09_release_docs.md) | Release (maintained as living doc) |
| [10_posix_framework.md](10_posix_framework.md) | POSIX Framework |
| [10a_sed.md](10a_sed.md) | Sed Details |
| [11_post_mvp_priorities.md](11_post_mvp_priorities.md) | Post-MVP Priorities |
| [11a_lower_priority.md](11a_lower_priority.md) | Lower Priority |
| [11_lessons_learned.md](11_lessons_learned.md) | Lessons Learned |

## Post-MVP Fix Sessions (retained for reference)

| File | Description |
|------|-------------|
| [14a_json_gap_fill.md](14a_json_gap_fill.md) | JSON Gap Fill — 8 utilities added `--json` |
| [14b_busybox_regression_fix.md](14b_busybox_regression_fix.md) | BusyBox Regression Fix — 79→3 failures |
| [14c_posix_json_gap.md](14c_posix_json_gap.md) | JSON-RPC Coverage Gap — 55/55 utilities |

## Active Plans (Post-MVP — Branch: `feat/post-mvp`)

| File | Phase |
|------|-------|
| [15_post_mvp_tier1.md](15_post_mvp_tier1.md) | Phase 15 — Tier 1: `dd` + `od` (11 BusyBox tests) |
| [16_post_mvp_tier2.md](16_post_mvp_tier2.md) | Phase 16 — Tier 2: 9 text/stream utilities (43 BusyBox tests) |
| [17_post_mvp_tier3.md](17_post_mvp_tier3.md) | Phase 17 — Tier 3: 12 utilities without BusyBox coverage ✅ |
| [18_quality_fixes.md](18_quality_fixes.md) | Phase 18 — Quality Fixes: CI, `patch`, coverage, aliases |
| [19_performance_benchmarking.md](19_performance_benchmarking.md) | Phase 19 — Performance Benchmarking (DONE) |
| [20_hardening_ii.md](20_hardening_ii.md) | Phase 20 — Hardening II: flag audit, coverage, input safety ✅ |
| [22_hardening_iii.md](22_hardening_iii.md) | Phase 22 — Hardening III: Daemon-First Pivot ✅ |
| [23_multi_tenant_sandbox.md](23_multi_tenant_sandbox.md) | Phase 23 — Multi-Tenant Sandbox: Linux Experiment Security (DEFERRED) |
| [24_multi_agent_observability.md](24_multi_agent_observability.md) | Phase 24 — Multi-Agent Observability: audit trail, per-agent metrics, file tracking (PLANNING) |

## Deferred / Future

- [07a_awk.md](07a_awk.md) | Awk Implementation Plan (canonical; Platinum gate)
- [14_xml_output.md](14_xml_output.md) | XML Output Support design (not implemented)
- [23_multi_tenant_sandbox.md](23_multi_tenant_sandbox.md) | Multi-Tenant Sandbox — workspace isolation, command allowlist, subprocess sandboxing (deferred until untrusted-user use case emerges)
- [19_performance_benchmarking.md](19_performance_benchmarking.md) | Performance Benchmarking Plan — GoPOSIX vs BusyBox (DONE)
- [performance.md](performance.md) | Performance Quick Reference — commands, scale, categories, results
- [22_hardening_iii.md](22_hardening_iii.md) | Hardening III — Daemon-First Pivot ✅
## Design

- [goposixos.md](goposixos.md) | GoPOSIXOS design — historical snapshot (moved to separate repo); all GoPOSIX-side prep work is complete

## Reference

- [posix_coverage.md](posix_coverage.md) | POSIX Compliance Matrix (55 utilities)
- [posix_faq.md](posix_faq.md) | POSIX Compliance FAQ
- [self_upgrade.md](self_upgrade.md) | Self-upgrade (`--version`, `--upgrade`) — version comparison, GitHub release fetching, atomic binary replacement

## Operations

- [operations/ingest.md](operations/ingest.md) | How to absorb a repo change into the wiki.
- [operations/query.md](operations/query.md) | How to answer questions from the wiki first.
- [operations/lint.md](operations/lint.md) | How to health-check and repair wiki drift.
