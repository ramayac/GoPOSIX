# Wiki Schema

## Goal

The wiki is the persistent knowledge layer between the chat agent and the raw repository.

It should reduce repeated repo rediscovery by storing stable summaries, operating procedures, and durable answers in plain Markdown.

## Required Contract

Every repo that adopts this pattern should have at least these files:

- `wiki/README.md`
- `wiki/index.md`
- `wiki/log.md`
- `wiki/schema.md`
- `wiki/phases.md`
- `wiki/repo-map.md`
- `wiki/operations/ingest.md`
- `wiki/operations/query.md`
- `wiki/operations/lint.md`

## Read Order

1. Read `wiki/index.md`.
2. Read the latest entries in `wiki/log.md`.
3. Read the relevant operations page.
4. Read only the linked topic pages needed for the task.
5. Read source files only after the wiki has been consulted.

## Write Order

1. Update the topic page that changed.
2. Update `wiki/index.md` if a page was added or its role changed.
3. Append a dated entry to `wiki/log.md`.

## File Style

- Use plain Markdown.
- Prefer stable filenames over timestamped filenames, except for the log headings.
- Use grep-friendly headings and short lists.
- Prefer explicit relative links.
- Avoid generated JSON, vector indexes, or tool-specific metadata unless there is a clear need.

## Durable Knowledge Rules

- Put repeatable procedures in `wiki/operations/`.
- Put repo facts in `wiki/repo-map.md` or another topic page referenced by the index.
- Put longer-lived decisions or answers into the wiki instead of leaving them only in chat history.
- Keep the log append-only.

## Repo-Specific Exclusions

Each repo should document high-noise or user-authored areas that should not be routinely ingested in `.wikirc` under the `ignore` list.
