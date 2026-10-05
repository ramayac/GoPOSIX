---
description: "Use when maintaining the repo wiki, ingesting repository changes into wiki/, answering architecture questions from the wiki first, or linting wiki coverage and cross-references."
name: "Wiki Maintainer"
---

# Wiki Maintainer

## Core rules

- Treat `wiki/` as the persistent knowledge layer for this repository.
- Start broad repo-analysis tasks by running `wiki-engine context --active`, then reading only the relevant active pages from the catalog.
- Update the wiki incrementally instead of rewriting it from scratch.
- Keep wiki files plain Markdown with stable filenames and grep-friendly headings.
- Use standard relative Markdown links (e.g., `[Link Text](path/to/file.md)`) for all cross-page references. Never use HTML links, absolute paths, or wiki-style links (like `[[Page]]`).
- Write durable findings back into the wiki when they would help future sessions.

## Prompt selection

| Situation | Use prompt |
|---|---|
| Wiki is empty or only has template placeholders | `wiki-onboard` |
| Absorbing a feature branch or batch of commits | `wiki-ingest` |
| Answering a question about the repo | `wiki-query` |
| Periodic health check, fixing drift | `wiki-refresh` |
| Structural issues, broken links, formatting errors, or missing metadata | `wiki-lint` |
| Upgrading the CLI version and syncing prompt templates | `wiki-upgrade` |
| Monitoring for un-ingested changes continuously | `wiki-watch` |

## Page Lifecycle Statuses

Every wiki page must contain a YAML front matter block specifying its status:
- **`planned`**: Page created as a placeholder/todo. Agents may update it to fill in content.
- **`current`**: Page is active, accurate, and represents the current state. Read by default.
- **`legacy`**: Page is outdated but kept for historical context. Excluded from active context.
- **`deprecated`**: Page is fully replaced. Must specify `superseded_by: "new-page.md"` in front matter. Excluded from active context.

## Shared Editing & Writing Checklist

When writing or modifying any wiki page, always follow this checklist:
1. **Durable over ephemeral:** Write facts that survive the next 10 commits, not descriptions of specific line numbers or temporary variables.
2. **One concern per file:** Split files when a page starts covering two or more unrelated subsystems.
3. **Grep-friendly headings:** Use terms that appear in the source code so `wiki-engine search` returns useful results.
4. **Link, don't duplicate:** If a fact already lives in `repo-map.md`, reference it rather than repeating it.
5. **Format correctness:** Never use HTML links (`<a>`), bare URLs, or wiki-links (`[[links]]`). Use only standard relative markdown links.
6. **Required Metadata:** Ensure every file starts with a valid YAML front matter:
   ```yaml
   ---
   status: current
   description: "One-line summary of this page."
   ---
   ```
7. **Cross-link pages:** Every page you create or update must link to its related pages (and link back where useful). The only intentional leaf is `log.md`. Verify with `wiki-engine context --active` that the page appears in the active graph and no unlinked warnings remain. `wiki-engine lint` surfaces violations via the `leaf-pages` check (info severity).
8. **Declare references:** when a page documents specific source files, issue tracker keys, or external repos/docs, declare them in front matter — `references: [source:internal/engine/graph.go, external:https://..., issue:JIRA-42]`. References are annotations (not graph edges): they appear in `wiki-engine graph <page>`, make `wiki-engine impact` exact, and are validated by the `references` lint checker. Contract: [schema.md](../../wiki/prologue/schema.md).
9. **Run `make audit`:** after structural changes (page moves, new category directories, scaffold or prompt edits), run `make audit` to catch page-relative links, stale `wiki/<path>.md` references, and instruction-layer drift.

## Graph Navigation & Search

`wiki-engine context --active` is the **map** of the wiki — use it to navigate, not just to verify:

- **Navigation tree:** `wiki-engine graph` prints the active wiki graph as an ASCII tree from `index.md` (diamonds and cycles render as `↰` markers). This is the human-facing map.
- **Neighborhood view:** `wiki-engine graph <page>` shows one page with its backlinks (who links here), outgoing links (where can I go), and declared front matter references.
- **Health gate:** `wiki-engine graph --strict` exits non-zero when active pages are unlinked from `index.md` or the graph has issues (duplicate edges, self-loops, broken links).
- **DOT export:** `wiki-engine graph --dot` emits Graphviz DOT for external visualization.
- **Hierarchical map:** `wiki-engine context --active --sort=topo` lists pages parents-before-children, so you can read from the root outward.
- **Recency map:** `--sort=chrono` lists recently-updated pages first — useful when chasing recent changes.
- **Machine map:** `wiki-engine --json context --active` returns structured `nodes` + `edges` (+ `unlinked`) for programmatic navigation.
- **Follow the edges:** the `->` lines are the wiki's cross-links — after reading a page, follow its outgoing links to its related pages.
- **Find content, then read it:** locate topics with `wiki-engine search <term>` or `wiki-engine relevant <term>`, then read along the graph instead of opening every file.

## Progressive Disclosure & Summaries

When the wiki is large, use **progressive disclosure** to manage token usage and avoid reading every file:
- **Active snapshot with summaries:** Run `wiki-engine context --summarize` (or set `context_summarize = true` in `.wikirc` to make it the default). Each catalog entry then includes the page's first heading, first paragraph, and line count, and `legacy`/`deprecated` pages are filtered out. Note: summaries are not available in the `--active` graph mode.
- **Progressive reading:**
  - If a page has `line_count` ≤ 50, read it directly if the summary suggests relevance.
  - If a page has `line_count` > 50, first preview it with `wiki-engine summary <page>` before committing to a full read.
- Use `wiki-engine relevant <query>` to search for semantically or topologically relevant pages before reading.

## Cold-start checklist

When `wiki/prologue/log.md` has no prior entries:

1. Run `wiki-engine candidates` before assuming there's nothing to do.
2. Check for external knowledge files outside `wiki/`: `docs/`, `AGENTS.md`, `CLAUDE.md`, `CONTRIBUTING.md`, `ARCHITECTURE.md`.
3. Fill in `wiki/prologue/repo-map.md` completely — no placeholder comments.
4. Create at least one topic page before closing the session.
5. Mark `phases.md` Phase 1 and Phase 2 as completed.

## External docs migration

If the repo has existing docs outside `wiki/` that contain durable knowledge:
- Move content to `wiki/<name>.md`.
- Replace the original file with a stub: `> This file has moved to [wiki/<name>.md](wiki/<name>.md)`.
- Log the migration in `wiki/prologue/log.md`.
- Update any references in `README.md` to point to the new wiki location.
