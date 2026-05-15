# Wiki Index

## Core

- [README.md](README.md) | Purpose, rules, and shell-first navigation.
- [schema.md](schema.md) | Required structure and maintenance contract.
- [phases.md](phases.md) | Project roadmap and phase index.
- [repo-map.md](repo-map.md) | Current repo architecture and exclusions.
- [log.md](log.md) | Append-only timeline of wiki maintenance.

## Phases

- [00_foundation_libs.md](00_foundation_libs.md) | Phase 00 — Foundation Libraries (flag parser, JSON envelope, JSON-RPC types)
- [01_multicall_tier1.md](01_multicall_tier1.md) | Phase 01 — Multicall Dispatcher + Tier 1 Utilities
- [02_docker_ci.md](02_docker_ci.md) | Phase 02 — Docker Scratch Build + CI Pipeline
- [03_filesystem_utils.md](03_filesystem_utils.md) | Phase 03 — Tier 2: Filesystem Utilities
- [04_text_processing.md](04_text_processing.md) | Phase 04 — Tier 3: Text Processing Utilities
- [05_daemon_core.md](05_daemon_core.md) | Phase 05 — JSON-RPC Daemon Core
- [06_system_utils.md](06_system_utils.md) | Phase 06 — Tier 4: System & Process Utilities
- [07_agent_features.md](07_agent_features.md) | Phase 07 — Agent-Ready Features (sessions, shell, Tier 5)
- [07a_awk.md](07a_awk.md) | Phase 07a — Awk Implementation Plan (canonical awk document; Platinum gate)
- [08_hardening.md](08_hardening.md) | Phase 08 — Production Hardening & Security (sandbox design complete; tests/docs tracked in Phase 12)
- [09_release_docs.md](09_release_docs.md) | Phase 09 — Release Automation & Documentation
- [10_posix_framework.md](10_posix_framework.md) | Phase 10 — POSIX Testing Framework
- [10a_sed.md](10a_sed.md) | Phase 10a — Sed Implementation Details
- [11_post_mvp_priorities.md](11_post_mvp_priorities.md) | Phase 11 — Post-MVP Priorities (11.1–11.3 complete; 11.4 awk → 07a_awk.md)
- [11_lessons_learned.md](11_lessons_learned.md) | Phase 11 — Lessons Learned, Insights & Gotchas
- [11a_lower_priority.md](11a_lower_priority.md) | Phase 11a — Lower Priority Improvements (6/8 complete; shell security + coverage gate hardening → Phase 12)
- [12_road_to_gold.md](12_road_to_gold.md) | Phase 12 — Road to Gold (authoritative roadmap: 5/5 Gold gaps resolved)
- [13_coverage_and_hardening.md](13_coverage_and_hardening.md) | Phase 13 — Coverage & Hardening (audit findings + 50%→75% coverage ramp + speed targets)
- [14_xml_output.md](14_xml_output.md) | Phase 14 — XML Output Support (--xml flag for all 52 utilities + foundation)
- [14a_json_gap_fill.md](14a_json_gap_fill.md) | Phase 14a — JSON/XML Gap Fill (8 utilities missing --json get both flags)

## Design

## Reference

- [posix_coverage.md](posix_coverage.md) | POSIX Compliance Matrix (49 utilities)
- [posix_faq.md](posix_faq.md) | POSIX Compliance FAQ
- [todos.md](todos.md) | Open TODOs & Remaining Work

## Operations

- [operations/ingest.md](operations/ingest.md) | How to absorb a repo change into the wiki.
- [operations/query.md](operations/query.md) | How to answer questions from the wiki first.
- [operations/lint.md](operations/lint.md) | How to health-check and repair wiki drift.
