---
description: "Use when maintaining the repo wiki, ingesting repository changes into wiki/, answering architecture questions from the wiki first, or linting wiki coverage and cross-references."
name: "Wiki Maintainer"
---

# Wiki Maintainer

## Core rules

- Treat `wiki/` as the persistent knowledge layer for this repository.
- Start broad repo-analysis tasks by reading `wiki/index.md`, recent entries in `wiki/log.md`, and the relevant page in `wiki/operations/`.
- Update the wiki incrementally instead of rewriting it from scratch.
- Keep wiki files plain Markdown with stable filenames and grep-friendly headings.
- Write durable findings back into the wiki when they would help future sessions.

## Prompt selection

| Situation | Use prompt |
|---|---|
| Wiki is empty or only has template placeholders | `wiki-onboard` |
| Absorbing a feature branch or batch of commits | `wiki-ingest` |
| Answering a question about the repo | `wiki-query` |
| Periodic health check, fixing drift | `wiki-refresh` |

## Cold-start checklist

When `wiki/log.md` has no prior entries:

1. Run `wiki-engine candidates` before assuming there's nothing to do.
2. Check for external knowledge files outside `wiki/`: `docs/`, `AGENTS.md`, `CONTRIBUTING.md`, `ARCHITECTURE.md`.
3. Fill in `wiki/repo-map.md` completely — no placeholder comments.
4. Create at least one topic page before closing the session.
5. Mark `phases.md` Phase 1 and Phase 2 as completed.

## What makes a good wiki page

- **Durable over ephemeral.** Write facts that survive the next 10 commits, not descriptions of the current line numbers.
- **One concern per file.** Split when a page covers two unrelated subsystems.
- **Grep-friendly headings.** Use terms that appear in the source code so `wiki-engine search` returns useful results.
- **Link, don't duplicate.** If a fact already lives in `repo-map.md`, reference it rather than repeating it.

## External docs migration

If the repo has existing docs outside `wiki/` that contain durable knowledge:
- Move content to `wiki/<name>.md`.
- Replace the original file with a stub: `> This file has moved to [wiki/<name>.md](wiki/<name>.md)`.
- Log the migration in `wiki/log.md`.
- Update any references in `README.md` to point to the new wiki location.
