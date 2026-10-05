---
description: "Run the repo wiki maintenance cycle. Use when: refreshing wiki state, reviewing repo changes, updating wiki pages, or running a wiki health check."
name: "Wiki Refresh"
argument-hint: "Optional focus area, file, or subsystem..."
agent: "agent"
---

Run the repository wiki refresh workflow.

## Required context

- Run `wiki-engine context --active` to get the current snapshot of active wiki pages.
- Use `wiki-engine refresh` for the full maintenance report.
- Follow the guidelines in [wiki-maintainer.md](wiki-maintainer.md).
- If wiki-engine is not installed, read [wiki/index.md](../../wiki/index.md) and [wiki/repo-map.md](../../wiki/prologue/repo-map.md).

## Execution steps

1. Run `wiki-engine refresh`.
2. If it reports no ingest candidates, stop and explain that no wiki update is needed.
3. Review the output from `wiki-engine changed` and `wiki-engine candidates`.
4. If the repo changes require wiki maintenance, update the relevant active pages under `wiki/` (creating them with proper YAML front matter if new). Cross-link updated pages to their related pages.
5. If a page is added, its status changes, or its role changes, update [wiki/index.md](../../wiki/index.md) using standard relative Markdown links.
6. Append a dated entry to [wiki/log.md](../../wiki/prologue/log.md) using the log heading convention.
7. Run `wiki-engine lint`.
8. Summarize:
   - what changed in the wiki
   - which source files drove the change
   - any remaining gaps or follow-up questions

If the wiki does not need changes, say so explicitly and explain why.
