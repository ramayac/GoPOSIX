---
description: "Update the wiki from current repository changes. Use when: ingesting a feature branch, filing architecture updates, or converting diffs into durable wiki knowledge."
name: "Wiki Ingest"
argument-hint: "Optional diff range or focus area..."
agent: "agent"
---

Ingest the current repository changes into the wiki.

> **Cold-start check:** If `wiki-engine changed` returns no output AND `wiki/log.md` has no prior ingest entries, this is a first-time onboarding — switch to the **Wiki Onboard** prompt instead of continuing here.

## Required context

- Read [wiki/index.md](../../wiki/index.md).
- Read recent entries in [wiki/log.md](../../wiki/log.md).
- Read [wiki/operations/ingest.md](../../wiki/operations/ingest.md).
- Read [wiki/repo-map.md](../../wiki/repo-map.md).

## Execution steps

1. Run `wiki-engine changed`.
2. Run `wiki-engine candidates`.
3. Ignore repo-specific excluded paths (configured in `.wikirc`).
4. Read only the changed source files that are relevant to the ingest.
5. Decide whether each durable fact belongs in an existing wiki page or needs a new page.
6. Update the relevant wiki pages with durable facts only.
7. Update [wiki/index.md](../../wiki/index.md) if coverage changed.
8. Append an ingest entry to [wiki/log.md](../../wiki/log.md) using the required heading format.
9. Run `wiki-engine lint`.

Finish by summarizing what was added, what changed, and what still needs human review.
