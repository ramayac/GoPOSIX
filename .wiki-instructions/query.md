---
description: "Answer a repository question from the wiki first. Use when: querying architecture, repo workflows, exclusions, or durable decisions before scanning source files."
name: "Wiki Query"
argument-hint: "Ask a repository question..."
agent: "agent"
---

Answer the user's repository question from the wiki first.

## Required context

- Run `wiki-engine graph` for the human navigation map (ASCII tree from `index.md`), or `wiki-engine context --active --sort=topo` for the compact agent map (parents before children). Use `--sort=chrono` for recency, `wiki-engine --json context --active` for a structured nodes/edges map, and `wiki-engine graph <page>` to inspect one page's backlinks, outgoing links, and declared references.
- Search the wiki with `wiki-engine search <term>` or `wiki-engine relevant <term>`.
- Read only the active wiki pages needed to answer the question.
- Follow the guidelines in [wiki-maintainer.md](wiki-maintainer.md).
- If wiki-engine is not installed, read [wiki/index.md](../../wiki/index.md) and [wiki/log.md](../../wiki/prologue/log.md) instead.

## Execution steps

1. Map first, then search: run `wiki-engine graph` (or `wiki-engine context --active`), locate the topic with `wiki-engine search <term>` or `wiki-engine relevant <term>`, and follow the graph's `->` links to related pages.
2. Read only the active wiki pages needed to answer the question. Skip pages marked `deprecated` or `legacy`.
3. Use source files only if the active wiki pages lack enough evidence.
4. If the answer reveals a durable repo fact that is missing or stale in the wiki, update the relevant active page (ensuring it contains proper front matter, standard relative Markdown links, and cross-links to its related pages).
5. If durable wiki content changed, append a dated entry to [wiki/log.md](../../wiki/prologue/log.md) and run `wiki-engine lint`.

In the final response:

- Answer the question directly.
- State whether the answer came fully from the wiki or required widening to source files.
- Mention any wiki updates that were made.
