---
status: current
description: "How to health-check and repair wiki drift."
references: [external:https://github.com/ramayac/go-wiki-engine, source:wiki/schema.md]
---

# Lint Workflow

## Goal

Keep the wiki coherent, linked, and current.

## Checks

`wiki-engine lint` runs a set of checkers. The gate fails on `warn` and `error`
findings (`fail_severity` defaults to `warn`); `info` findings print but pass.

| Check | Rule | Severity |
|-------|------|----------|
| `required-files` | Canonical pages exist (`index.md`, `log.md`, `schema.md`, `phases.md`, `repo-map.md`, `operations/*`). | error |
| `index-links` | Every `index.md` link resolves. | error |
| `cross-page-links` | Every `.md` link resolves, page-relative. | error |
| `index-format` | Every `index.md` bullet link has a `\| description` part. | warn |
| `orphans` | Every active page is linked from `index.md`. | warn |
| `front-matter` | Every page has `status`; `description` recommended; `superseded_by` required when `deprecated`. | warn |
| `external-links` | Non-`.md` links resolve relative to the page or the repo root. | warn |
| `references` | Front matter `references` are well formed (`source:`, `external:`, `issue:`). | warn |
| `log-chronology` | Log headings are in descending date order. | warn |
| `log-headings` | Log headings follow `## [YYYY-MM-DD] kind \| summary`. | warn |
| `phase-consistency` | A phase row requires the previous phase row, marked `completed`. | warn |
| `stale-content` | A page older than `stale_days` (default 30) is flagged; severity rises when the repo has active changes. | warn |
| `markers` | Flags `TODO`/`TBD`/`UNKNOWN` outside code blocks. | warn |
| `duplicate-content` | Flags near-duplicate paragraphs across pages. | warn |
| `bare-urls` | Flags bare URLs outside code and front matter. | warn |
| `markdown-format` | Flags malformed links and wiki-style `[[links]]`. | warn |
| `heading-hierarchy` | Flags skipped heading levels; flags multiple `h1`. | warn / info |
| `leaf-pages` | Every active page needs an outgoing `.md` link (`log.md` is the only leaf). | info |

The metadata contract lives in [schema.md](../schema.md). Use
`wiki-engine lint --check=<a,b>` or `--skip=<a,b>` to run a subset.

## Shell-First Checks

```bash
wiki-engine lint
wiki-engine lint --skip=stale-content,leaf-pages
wiki-engine graph --strict
wiki-engine list
wiki-engine search "TODO:"
```

## Repair Order

1. Fix stale or incorrect topic pages.
2. Fix `wiki/index.md` links or summaries.
3. Append a log entry if the lint changed durable content.

## Log Format

Use this exact heading pattern:

```md
## [YYYY-MM-DD] lint | short summary
```

---

## See Also

- [index.md](../index.md) | Wiki index.
- [ingest.md](ingest.md) | Ingest workflow.
- [query.md](query.md) | Query workflow.
