---
name: wiki
description: >
  Maintain a repo-local wiki for durable project knowledge.
  Use for ingesting repo changes, querying architecture, refreshing
  stale docs, onboarding cold-start projects, upgrading the wiki-engine
  CLI, monitoring drift, and linting wiki health.
  Works with the wiki-engine CLI for filesystem-level inspection.
license: MIT
compatibility: "Requires wiki-engine CLI (go install github.com/ramayac/go-wiki-engine/cmd/wiki-engine@latest)"
---

# Wiki Maintainer

Maintain the repository wiki — the persistent knowledge layer that survives
beyond a single chat session. This skill provides shell-first plumbing
commands and repeatable workflows for wiki operations.

## Quick Reference — wiki-engine CLI

```bash
wiki-engine context --active          # Active-page graph from index.md (skips legacy/deprecated)
wiki-engine context --active --sort=topo  # Hierarchical map: parents before children
wiki-engine --json context --active   # Structured map: nodes + edges (+ unlinked)
wiki-engine graph                     # Navigation tree from index.md (diamonds/cycles as ↰)
wiki-engine graph <page>              # Neighborhood: page + backlinks + outgoing links (+ references)
wiki-engine graph --strict            # Tree + diagnostics; exit 1 on orphans/graph issues
wiki-engine graph --dot               # Graphviz DOT export for visualization
wiki-engine graph --json              # Structured nodes/edges/unlinked/stats/issues
wiki-engine context --sort=chrono     # Recency map: recently-updated first
wiki-engine context                   # Snapshot: catalog with statuses, recent log, phase
wiki-engine context --summarize       # Snapshot with per-page previews (progressive disclosure)
wiki-engine list [--active]           # List wiki files (optionally active pages only)
wiki-engine search <term>             # Case-insensitive search across wiki
wiki-engine headings                  # All Markdown headings with file paths
wiki-engine log-tail [n]              # Last N log entries
wiki-engine changed                   # Non-wiki files changed in git diff
wiki-engine candidates                # Changed files worth ingesting (respects .wikirc ignore)
wiki-engine lint                      # Full wiki health check (front matter, links, structure, markers)
wiki-engine lint --check=<a,b> --skip=<c>  # Run selected checkers only
wiki-engine refresh                   # Combined snapshot: files + log + changed + candidates + lint
wiki-engine stats                     # File count, heading count, last-updated
wiki-engine relevant <query>          # Rank pages by relevance to a query
wiki-engine impact <file...>          # Which wiki pages mention changed files
wiki-engine watch [--once]            # Poll for changes + lint issues (interval from .wikirc)
wiki-engine diff <from> <to>          # Wiki changes between two git refs
```

## Page Lifecycle

Every wiki page has YAML front matter with a `status`:

```yaml
---
status: current          # planned | current | legacy | deprecated
description: "One-line summary of this page's purpose"
superseded_by: ""        # required when status is deprecated
references: [source:internal/engine/graph.go, issue:JIRA-42]  # optional typed cross-references
---
```

- Read only `current` and `planned` pages by default.
- Skip `legacy` and `deprecated` pages entirely (`context --active` filters them out).
- When replacing a page, set it to `deprecated` and point `superseded_by` at the new page.

## Workflows

### 1. Ingest Changes (`/wiki-ingest`)
Absorb repo changes into the wiki.
1. Run `wiki-engine context --active` and `wiki-engine candidates`.
2. Read only the changed source files relevant to the ingest.
3. Update or create active wiki pages with durable facts (with valid front matter).
4. Update `wiki/index.md` if page coverage changed.
5. Append a dated entry to `wiki/prologue/log.md`.
6. Run `wiki-engine lint`.

### 2. Query the Repo (`/wiki-query`)
Answer questions from the wiki first.
1. Map first: `wiki-engine graph` (human tree) or `wiki-engine context --active --sort=topo` (compact agent map).
2. Locate the topic: `wiki-engine search <term>` or `wiki-engine relevant <term>`.
3. Read along the graph's `->` links to related active pages.
4. Use source files only if the wiki lacks evidence.
5. File durable answers back into the wiki.

### 3. Refresh Wiki (`/wiki-refresh`)
Periodic maintenance cycle.
1. Run `wiki-engine refresh`.
2. If no ingest candidates, stop — no update needed.
3. Otherwise, update stale pages, index, and log.
4. Run `wiki-engine lint`.
5. Summarize what changed and what still needs review.

### 4. Onboard Cold Start (`/wiki-onboard`)
Bootstrap a wiki for a brand-new project or empty wiki.
1. Run `wiki-engine candidates` (fall back to manual repo survey).
2. Check for external docs outside `wiki/` to migrate.
3. Fill in `wiki/prologue/repo-map.md` completely.
4. Create initial topic pages (architecture, data-model, api, etc.).
5. Update `wiki/index.md` and `wiki/prologue/phases.md`.
6. Append an ingest entry to `wiki/prologue/log.md`.
7. Run `wiki-engine lint`.

### 5. Lint the Wiki (`/wiki-lint`)
Fix structural errors automatically.
1. Run `wiki-engine lint --json`.
2. Fix each issue category (front matter, index format, bare URLs, broken links,
   orphans, heading hierarchy, log format/chronology).
3. Re-run `wiki-engine lint` until clean.

### 6. Upgrade the CLI (`/wiki-upgrade`)
1. Run `wiki-engine upgrade`.
2. Run `wiki-engine sync-prompts` to refresh all instruction layers.
3. Verify with `wiki-engine lint` and `wiki-engine context --active`.

### 7. Watch for Drift (`/wiki-watch`)
1. Run `wiki-engine watch --once` before a work session.
2. For continuous monitoring, set `watch_interval` in `.wikirc` and run `wiki-engine watch`.
3. When candidates appear, run `wiki-engine impact <file>` and ingest the durable facts.

## Canonical Instructions

Full workflow definitions live in `.wiki-instructions/` — edit those files
to update prompts for all supported tools (GitHub Copilot, Claude Code, pi).
Tool directories contain symlinks back to the canonical files, so edits
propagate automatically. Run `wiki-engine sync-prompts` after upgrading the
wiki-engine binary.

## What Makes a Good Wiki Page

- **Durable over ephemeral.** Facts that survive 10+ commits.
- **One concern per file.** Split when a page covers two unrelated subsystems.
- **Grep-friendly headings.** Terms that appear in source code.
- **Link, don't duplicate.** Reference `repo-map.md` rather than repeating.
- **Declare references.** When a page documents source files, issues, or external repos, put them in front matter `references: [source:..., external:..., issue:...]` — they surface in `graph <page>` and drive `impact`.
- **Standard markdown links only.** No bare URLs, no HTML `<a>` tags, no `[[wiki-links]]`.
