---
description: "Bootstrap the wiki for a brand-new project, or recover a stale/empty wiki from scratch. Use when: the wiki is empty or only has template placeholders, there has been no prior ingest, or the repo has never had a wiki. NOT for incremental updates — use Wiki Ingest for that."
name: "Wiki Onboard"
argument-hint: "Optional: specific subsystems or files to prioritise..."
agent: "agent"
---

Run a full-project wiki onboarding. This is a **cold-start** survey — do not rely on `wiki-engine changed` having output.

## Required context

- Read [wiki/index.md](../../wiki/index.md) — check whether it still has template content.
- Read [wiki/phases.md](../../wiki/phases.md) — check which phases are still `not-started`.
- Skim the top-level directory to understand what kind of project this is before reading any source.

## Execution steps

### 1. Discover the repo shape

```bash
wiki-engine candidates
```

If candidates returns no output, fall back to surveying the repo manually:
- List top-level directories.
- Read the root `README.md`, `Makefile`, and any `package.json` / `go.mod` / `pyproject.toml`.
- Check for existing docs in common locations: `docs/`, `AGENTS.md`, `CONTRIBUTING.md`, `ARCHITECTURE.md`, `ADR/`, `notes/`.

### 2. Check for external knowledge to migrate

Before creating new pages, scan for files that already contain durable knowledge outside `wiki/`:

- `docs/` or `doc/` — planning docs, design decisions, lessons learned
- `AGENTS.md` — AI agent SOPs and architectural rules
- `CONTRIBUTING.md` — developer workflow rules
- `ARCHITECTURE.md`, `DESIGN.md`, or similar top-level docs

If found:
- Copy their durable content into appropriately-named wiki pages (e.g., `wiki/agents-guide.md`, `wiki/big-plan.md`).
- Replace the original file with a one-line stub pointing to its new wiki location, OR delete it if it is fully superseded.
- Note the migration in the log.

### 3. Populate wiki/repo-map.md

Fill in every section — do not leave placeholder comments:

- **Purpose** — one or two sentence description of what the project does.
- **High-Signal Areas** — the most important source directories and files, with a one-line role for each.
- **Generated Artifacts** — build outputs, caches, test fixtures.
- **Build and Run Path** — exact commands to build, test, and run. Copy from Makefile/README if available.
- **Ignored Paths** — list paths from `.wikirc` `ignore` array.

### 4. Create initial topic pages

Decide what the first wiki topic pages should be based on what you found. Common patterns:

| Page name | When to create |
|---|---|
| `architecture.md` | Any project with multiple layers or subsystems |
| `gameplay-systems.md` / `domain.md` | Domain-specific rules and data models |
| `bots.md` / `agents.md` | Projects with automation, bots, or AI integration |
| `api.md` | Projects with a public API |
| `data-model.md` | Projects with a significant schema |

Read only the source files needed to populate each page. Write durable facts only — not implementation details that change every PR.

### 5. Update wiki/index.md

Add a section for every new page created. Keep the index as the entry point — it should describe every page in one line.

### 6. Advance wiki/phases.md

After completing the above:
- Mark **Phase 1 (Populate repo map)** as `completed`.
- Mark **Phase 2 (First ingest cycle)** as `completed`.
- Add a **Phase 3** row if ongoing ingest makes sense (it almost always does):

  ```md
  | 3 | Ongoing ingest | in-progress | Run ingest after each meaningful commit batch |
  ```

### 7. Append to wiki/log.md

Use the required heading format:

```md
## [YYYY-MM-DD] ingest | initial full-repo wiki build
```

Include: what was added, what was migrated from external files, and anything that needs human review.

### 8. Run lint

```bash
wiki-engine lint
```

Fix any issues before finishing.

---

Finish by summarising what was created, what was migrated, and what still needs human review.
