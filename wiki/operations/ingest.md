# Ingest Workflow

## Goal

Absorb a repo change into the wiki without rediscovering the entire codebase.

## Procedure

1. Read `wiki/index.md` and the latest entries in `wiki/log.md`.
2. If `wiki-engine changed` returns no output **and** `wiki/log.md` has no prior ingest entries, this is a cold start — run the full onboarding survey instead of an incremental ingest (see Wiki Onboard prompt).
3. Inspect the changed files first.
4. Check for external knowledge files that belong in the wiki: `docs/`, `AGENTS.md`, `CONTRIBUTING.md`, `ARCHITECTURE.md`. If found and not yet migrated, add a migration step to this ingest.
5. Ignore repo-specific excluded paths from `.wikirc`.
6. Decide whether the change updates an existing page or needs a new page.
7. Update the relevant wiki page with the durable facts only.
8. Update `wiki/index.md` if page coverage changed.
9. Advance `wiki/phases.md` if phases were completed during this ingest.
10. Append an entry to `wiki/log.md`.

## Shell-First Inputs

```bash
wiki-engine changed
wiki-engine candidates
wiki-engine refresh
```

## Page Decision Rule

- Update an existing page when the change fits an existing concern.
- Create a new page when the change introduces a new subsystem, workflow, or recurring question.

## External Docs Migration Rule

If a project has existing documentation outside `wiki/` (e.g., `docs/bigPlan.md`, root `AGENTS.md`):
- Move durable content into a wiki page with a stable filename.
- Replace the original with a one-line stub redirecting to `wiki/<page>.md`, or delete it if fully superseded.
- Log the migration in `wiki/log.md`.

## Log Format

Use this exact heading pattern:

```md
## [YYYY-MM-DD] ingest | short summary
```

