---
description: "Run the repo wiki maintenance cycle. Use when: refreshing wiki state, reviewing repo changes, updating wiki pages, or running a wiki health check."
name: "Wiki Refresh"
argument-hint: "Optional focus area, file, or subsystem..."
agent: "agent"
---

Run the repository wiki refresh workflow.

## Required context

- Read [wiki/index.md](../../wiki/index.md).
- Read recent entries in [wiki/log.md](../../wiki/log.md).
- Read [wiki/operations/ingest.md](../../wiki/operations/ingest.md), [wiki/operations/query.md](../../wiki/operations/query.md), and [wiki/operations/lint.md](../../wiki/operations/lint.md).
- Use [wiki/repo-map.md](../../wiki/repo-map.md) for repo-specific exclusions and architecture facts.

## Execution steps

1. Run `wiki-engine refresh`.
2. If it reports no ingest candidates, stop and explain that no wiki update is needed.
3. Review the output from `wiki-engine changed` and `wiki-engine candidates`.
4. If the repo changes require wiki maintenance, update the relevant pages under `wiki/`.
5. If a page is added or its role changes, update [wiki/index.md](../../wiki/index.md).
6. Append a dated entry to [wiki/log.md](../../wiki/log.md) using the log heading convention.
7. Run `wiki-engine lint`.
8. Summarize:
   - what changed in the wiki
   - which source files drove the change
   - any remaining gaps or follow-up questions

If the wiki does not need changes, say so explicitly and explain why.
