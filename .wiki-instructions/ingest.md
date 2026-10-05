---
description: "Update the wiki from current repository changes. Use when: ingesting a feature branch, filing architecture updates, or converting diffs into durable wiki knowledge."
name: "Wiki Ingest"
argument-hint: "Optional diff range or focus area..."
agent: "agent"
---

Ingest the current repository changes into the wiki.

> **Cold-start check:** If `wiki-engine changed` returns no output AND `wiki/prologue/log.md` has no prior ingest entries, this is a first-time onboarding — switch to the **Wiki Onboard** prompt instead of continuing here.

## Required context

- Run `wiki-engine context --active` to get the current snapshot of active wiki pages (skipping `deprecated` or `legacy` pages).
- Read only the active pages from the catalog needed for this task.
- Follow the core wiki editing guidelines defined in [wiki-maintainer.md](wiki-maintainer.md).
- If wiki-engine is not installed, read [wiki/index.md](../../wiki/index.md) and [wiki/repo-map.md](../../wiki/prologue/repo-map.md) instead.

## Execution steps

1. Run `wiki-engine changed`.
2. Run `wiki-engine candidates`.
3. Ignore repo-specific excluded paths (configured in `.wikirc`).
4. Read only the changed source files that are relevant to the ingest.
5. Decide whether each durable fact belongs in an existing active wiki page or needs a new page.
6. When creating or modifying pages:
   - Ensure all new pages include a valid YAML front matter block (with `status: current` or `status: planned`, and a `description`).
   - If replacing or retiring an existing page, set its front matter `status` to `deprecated` and specify the replacing page path in `superseded_by: "target-page.md"`.
   - Update pages with durable facts only, using standard relative Markdown links (e.g., `[Text](file.md)`), and cross-link the page to its related pages.
7. Update [wiki/index.md](../../wiki/index.md) if coverage changed using standard relative Markdown links.
8. Append an ingest entry to [wiki/log.md](../../wiki/prologue/log.md) using the required heading format.
9. Run `wiki-engine lint` to verify wiki integrity and `wiki-engine context --active` to confirm the updated pages are connected in the graph.

Finish by summarizing what was added, what changed, what was deprecated (if any), and what still needs human review.
