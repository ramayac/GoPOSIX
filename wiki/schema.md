---
status: current
description: "Wiki structure contract."
references: [external:https://github.com/ramayac/go-wiki-engine]
---

# Wiki Schema (Structure Contract)

> **Not to be confused with [json_schema.md](json_schema.md)**, which documents the `--json` output schemas for each utility. This page describes the wiki's own structural requirements.

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

## Front Matter (required)

Every page must start with a YAML front matter block:

```yaml
---
status: current
description: "One-line summary of this page."
---
```

- `status` is required. Allowed values: `planned`, `current`, `legacy`, `deprecated`.
- `description` is recommended. It appears in `wiki-engine graph` and `context`.
- `deprecated` pages must also set `superseded_by: "target-page.md"`.
- Declare source files, external URLs, and issue keys:

  ```yaml
  references: [source:internal/daemon/server.go, external:https://example.com, issue:JIRA-42]
  ```

`legacy` and `deprecated` pages leave the active graph. They are excluded from
`context --active`, from the `orphans` check, and from the `leaf-pages` check.

## Lint Gates

`wiki-engine lint` gates on warn and error findings (`fail_severity` defaults to
`warn`). Info findings print but pass. Fix these before you close a task:

| Check | Rule |
|-------|------|
| `front-matter` | Every page needs `status`; `description` is recommended. |
| `cross-page-links` | Every `.md` link must resolve, page-relative. |
| `external-links` | Non-`.md` links must resolve relative to the page or the repo root. |
| `index-format` | Every `index.md` bullet link needs a `\| description`. |
| `index-links` | Every `index.md` link must resolve. |
| `orphans` | Every active page must be linked from `index.md`. |
| `leaf-pages` | Every active page needs an outgoing `.md` link (`log.md` is the only leaf). |
| `log-chronology` | Log headings must be in descending date order. |
| `phase-consistency` | A phase row requires the previous phase row, marked `completed`. |
| `heading-hierarchy` | No skipped heading levels; one `h1` per page. |
| `stale-content` | A page older than `stale_days` (default 30) is flagged. |

Run `wiki-engine graph --strict` to fail on unlinked pages and graph issues.

## Durable Knowledge Rules

- Put repeatable procedures in `wiki/operations/`.
- Put repo facts in `wiki/repo-map.md` or another topic page referenced by the index.
- Put longer-lived decisions or answers into the wiki instead of leaving them only in chat history.
- Keep the log append-only.

## Repo-Specific Exclusions

Each repo should document high-noise or user-authored areas that should not be routinely ingested in `.wikirc` under the `ignore` list.
