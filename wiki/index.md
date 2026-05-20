# Wiki Index

## Core

- [README.md](README.md) | Purpose, rules, and shell-first navigation.
- [usage.md](usage.md) | Usage guide: CLI, daemon, Docker Compose, Go SDK, recipes.
- [schema.md](schema.md) | Required structure and maintenance contract.
- [phases.md](phases.md) | Project roadmap, current state, and phase index.
- [repo-map.md](repo-map.md) | Current repo architecture and exclusions.
- [log.md](log.md) | Append-only timeline of wiki maintenance.
- [todos.md](todos.md) | Open TODOs, remaining BusyBox failures, and pending work.

## Architecture & Design

- [architecture.md](architecture.md) | System architecture, component flow, Docker images, package map.
- [performance.md](performance.md) | Performance quick reference — commands, scale, categories, results.
- [security.md](security.md) | Security model, shell sandbox, deployment posture.
- [self_upgrade.md](self_upgrade.md) | Self-upgrade (`--version`, `--upgrade`).

## SDK & API

- [sdk.md](sdk.md) | Go SDK guide — typed client, connection pooling, 60µs/call (primary interface).
- [rpc_api.md](rpc_api.md) | JSON-RPC client API reference (`pkg/client`).
- [rpc_quickstart.md](rpc_quickstart.md) | JSON-RPC quickstart — raw protocol for non-Go clients.
- [json_schema.md](json_schema.md) | `--json` output envelope and per-utility schemas.
- [shell_integration.md](shell_integration.md) | CLI-to-daemon forwarding for shell users (socat, Python, Go helper).

## Test & Compliance

- [test_coverage_matrix.md](test_coverage_matrix.md) | Per-utility test status for all 77 utilities.
- [posix_faq.md](posix_faq.md) | POSIX compliance FAQ.

## Completed Phase Summaries

- [11_lessons_learned.md](11_lessons_learned.md) | Architectural lessons, gotchas, and validated patterns.
- [14_post_mvp_fixes.md](14_post_mvp_fixes.md) | JSON gap fill, BusyBox regression fix, JSON-RPC daemon coverage.
- [15_post_mvp_utilities.md](15_post_mvp_utilities.md) | Post-MVP utilities: `dd`, `od`, text tools, stubs, quality fixes.
- [19_performance_benchmarking.md](19_performance_benchmarking.md) | Benchmark results: GoPOSIX vs BusyBox (DONE).
- [20_hardening_ii.md](20_hardening_ii.md) | Hardening II: flag audit, code cleanup, coverage, input safety.
- [22_hardening_iii.md](22_hardening_iii.md) | Hardening III: Daemon-First Pivot.
- [23_flags_rewrite.md](23_flags_rewrite.md) | Flags Rewrite: zero-allocation POSIX scanner (COMPLETED).

## Deferred / Future

- [deferred.md](deferred.md) | Summary of all deferred and planned future work.
- [07a_awk.md](07a_awk.md) | Awk implementation plan (canonical; Platinum gate).
- [24_multi_agent_observability.md](24_multi_agent_observability.md) | Multi-agent observability (PLANNING).

## Operations

- [operations/ingest.md](operations/ingest.md) | How to absorb a repo change into the wiki.
- [operations/query.md](operations/query.md) | How to answer questions from the wiki first.
- [operations/lint.md](operations/lint.md) | How to health-check and repair wiki drift.

## Deploy

- [deploy/docker-compose.md](deploy/docker-compose.md)
- [deploy/kubernetes.md](deploy/kubernetes.md)
- [deploy/systemd.md](deploy/systemd.md)
