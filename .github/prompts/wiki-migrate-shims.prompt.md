---
description: "Migrate custom AGENTS.md / CLAUDE.md content to the wiki, then replace both files with standard redirect shims. Use when wiki-engine sync-prompts reports that these files already exist with custom content."
name: "Wiki Migrate Shims"
agent: "agent"
---

Migrate any custom AI-agent instructions from `AGENTS.md` and `CLAUDE.md` into the wiki, then replace both files with standard redirect shims.

## Required context

- Read `AGENTS.md` in the repo root (if it exists).
- Read `CLAUDE.md` in the repo root (if it exists).
- Read [wiki/index.md](../../wiki/index.md).
- Read [wiki/README.md](../../wiki/README.md).

## Execution steps

1. Read `AGENTS.md` and `CLAUDE.md`.

2. **Check if they are already shims.** A file is already a shim if its only substantive content is a redirect pointing to `wiki/index.md` (or the configured wiki dir). If both files are already shims, stop here and report no action needed.

3. For each file that is NOT already a shim, identify durable content that is not yet in the wiki:
   - Coding conventions or team workflow → add to [wiki/README.md](../../wiki/README.md).
   - Architecture or component notes → add to [wiki/repo-map.md](../../wiki/repo-map.md).
   - Broad AI guidance (how the agent should behave in this repo) → add a dedicated **AI Agent Guidance** section in [wiki/README.md](../../wiki/README.md).
   - Skip generic boilerplate that adds no project-specific value.

4. Replace `AGENTS.md` with the standard shim (adjust `wiki/` path if the repo uses a custom wiki dir name from `.wikirc`):

   ```markdown
   # AI Agent Instructions

   This project uses a structured wiki for all documentation and agent context.

   Start here: **[wiki/index.md](wiki/index.md)**

   The wiki covers architecture, conventions, active phases, and the project change log.
   To update or query the wiki, use the `/wiki-ingest`, `/wiki-query`, or `/wiki-refresh`
   Copilot slash commands (installed in `.github/prompts/`).
   ```

5. Replace `CLAUDE.md` with the same shim content, changing the heading to `# Claude Instructions`.

6. Append a dated entry to [wiki/log.md](../../wiki/log.md) describing what was migrated (be specific about which facts moved to which wiki page).

7. Run `wiki-engine lint`.

Finish by summarising: what content was migrated and where, what was redundant or skipped, and confirming both shim files were written.
