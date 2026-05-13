# Wiki

This directory is the repository-local knowledge base.

The wiki is the working memory for repo analysis, architectural summaries, operating procedures, and durable answers that should survive beyond a single chat session. It is meant to be updated incrementally instead of regenerated from zero.

## Rules

- Read the wiki before doing broad repo analysis.
- Keep files plain Markdown with stable filenames and grep-friendly headings.
- Update the wiki when the repo meaningfully changes.
- Append dated entries to `log.md` for ingest, query, and lint activity.
- Treat repo source files as the underlying evidence.

## Required Files

- `index.md` tracks the catalog.
- `log.md` tracks chronology.
- `schema.md` defines the contract.
- `phases.md` tracks rollout.
- `repo-map.md` records the current repo model.
- `operations/` holds repeatable workflows.

## Shell-First Navigation

```bash
wiki-engine list
wiki-engine headings
wiki-engine log-tail
wiki-engine search <term>
```

## Update Loop

1. Read `index.md`.
2. Read the last few entries in `log.md`.
3. Read the relevant page under `operations/`.
4. Read only the topic pages needed for the task.
5. Read source files when the wiki lacks detail or needs verification.
6. Write durable findings back into the wiki.
