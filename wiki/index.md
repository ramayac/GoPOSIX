# Wiki Index

## Core

- [README.md](README.md) | Purpose, rules, and shell-first navigation.
- [usage.md](usage.md) | Usage guide: CLI, daemon, Docker Compose, Go SDK, recipes.
- [schema.md](schema.md) | Wiki structure contract (not to be confused with [json_schema.md](json_schema.md) for `--json` output schemas).
- [phases.md](phases.md) | Project roadmap, current state, and phase index.
- [repo-map.md](repo-map.md) | Current repo architecture and exclusions.
- [log.md](log.md) | Append-only timeline of wiki maintenance.
- [todos.md](todos.md) | Open TODOs, remaining BusyBox failures, and pending work.

## Architecture & Design

- [architecture.md](architecture.md) | System architecture, component flow, Docker images, package map.
- [performance.md](performance.md) | Performance benchmark commands, scale, categories, and optimization tracker.
- [security.md](security.md) | Security model, shell sandbox, deployment posture.
- [observability_exports.md](observability_exports.md) | Options for exposing daemon metrics to OS tools and external consumers.
- [self_upgrade.md](self_upgrade.md) | Self-upgrade (`--version`, `--upgrade`).

## SDK & API

- [sdk.md](sdk.md) | Go SDK guide — typed client, connection pooling, 60µs/call (primary interface).
- [rpc_api.md](rpc_api.md) | JSON-RPC typed method reference (signature catalog).
- [rpc_quickstart.md](rpc_quickstart.md) | JSON-RPC quickstart — raw protocol for non-Go clients.
- [json_schema.md](json_schema.md) | `--json` output envelope and per-utility schemas.
- [shell_integration.md](shell_integration.md) | CLI-to-daemon forwarding for shell users (socat, Python, Go helper).

## Test & Compliance

- [test_coverage_matrix.md](test_coverage_matrix.md) | Per-utility test status for all 115 utilities.
- [26_missing_tools.md](26_missing_tools.md) | Analysis of missing BusyBox tools based on test suite coverage (COMPLETED).
- [27_high_complexity_tools.md](27_high_complexity_tools.md) | High-complexity & privileged Tier 5 utilities (COMPLETED).
- [posix_faq.md](posix_faq.md) | POSIX compliance FAQ.

## Completed Phase Summaries

- [lessons_learned.md](lessons_learned.md) | Architectural lessons, gotchas, and validated patterns.
- [post_mvp.md](post_mvp.md) | Post-MVP: JSON gap fill, BusyBox regression fix, coverage drive, phases 15–18 utilities.
- [hardening.md](hardening.md) | Hardening II–V: flag audit, daemon-first pivot, compliance, coverage, tar audit.
- [performance.md](performance.md) | Benchmark results (Phase 19) + optimizations tracker (Phase 30, 12/30 done).
- [23_flags_rewrite.md](23_flags_rewrite.md) | Flags Rewrite: zero-allocation POSIX scanner (COMPLETED).
- [25_awesome_go_submission.md](25_awesome_go_submission.md) | Awesome-Go submission plan, checklists, and compliance validation.

## Deferred / Future

- [deferred.md](deferred.md) | Canonical registry of deferred architectural work and future phases.
- [observability_exports.md](observability_exports.md) | Multi-agent observability (deferred discussion — see Part 2).

## Operations

- [operations/ingest.md](operations/ingest.md) | How to absorb a repo change into the wiki.
- [operations/query.md](operations/query.md) | How to answer questions from the wiki first.
- [operations/lint.md](operations/lint.md) | How to health-check and repair wiki drift.

## Deploy

- [alpine_plan.md](alpine_plan.md) | Alpine integration blueprint (daemon mode, BusyBox override, tradeoffs).
- [deploy/docker-compose.md](deploy/docker-compose.md)
- [deploy/kubernetes.md](deploy/kubernetes.md)
- [deploy/systemd.md](deploy/systemd.md)
