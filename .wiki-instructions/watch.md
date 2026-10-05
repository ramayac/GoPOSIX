---
description: "Monitor the repo for un-ingested changes. Use when: setting up continuous wiki maintenance, or before starting a work session to catch drift."
name: "Wiki Watch"
argument-hint: "none, or --once for a single check..."
agent: "agent"
---

Monitor the repository for changes that need wiki maintenance.

## Required context

- Run `wiki-engine context --active` to get the current snapshot of active wiki pages.
- Follow the guidelines in [wiki-maintainer.md](wiki-maintainer.md).
- Read [wiki/operations/ingest.md](../../wiki/operations/ingest.md).

## Execution steps

### Continuous mode

```bash
wiki-engine watch
```

This polls every N seconds (configured via `.wikirc` `watch_interval`). When changes are detected, it prints a summary of changed files, ingest candidates, and lint issues.

### One-shot mode

```bash
wiki-engine watch --once
```

Runs a single change + lint cycle and exits. Use this before starting work to check for un-ingested drift.

### Agent auto-ingest

When `wiki-engine watch` reports changed candidates:

1. Run `wiki-engine context --active`.
2. Run `wiki-engine candidates` to get the filtered list.
3. For each candidate, run `wiki-engine impact <file>` to see which wiki pages are affected.
4. Read the changed source files that are relevant.
5. Update active wiki pages with durable facts:
   - Ensure all new pages include a valid YAML front matter block (with `status: current` or `status: planned`, and a `description`).
   - If replacing or retiring an existing page, set its front matter `status` to `deprecated` and specify the replacing page path in `superseded_by: "target-page.md"`.
   - Update pages with durable facts only, using standard relative Markdown links (e.g., `[Text](file.md)`), and cross-link to related pages.
6. Append a dated entry to [wiki/log.md](../../wiki/prologue/log.md).
7. Run `wiki-engine lint`.

If no candidates have durable wiki impact, report "no wiki update needed" and move on.

## Config

Enable and tune in `.wikirc`:

```ini
watch_interval = 60   # seconds between polls (0 disables continuous watch)
stale_days = 30        # warn on pages whose last git commit is older than N days
```
