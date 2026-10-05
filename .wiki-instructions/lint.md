---
description: "Run the wiki linter and automatically fix structural errors, broken links, formatting issues, or missing front matter."
name: "Wiki Lint"
argument-hint: "none"
agent: "agent"
---

Interpret wiki-engine lint output and fix all issues automatically.

## Required context

- Run `wiki-engine context --active` to get the current active wiki snapshot.
- Run `wiki-engine lint --json` to get the current list of linting issues in structured JSON format.

## Execution steps

1. Run `wiki-engine lint --json`.
2. If the output shows no issues (e.g. `ok` is `true` and the list is empty), report that the wiki is clean and exit.
3. For each returned issue, analyze the error:
   - **`front-matter`**: Add or correct YAML front matter blocks. Ensure every active page has `status: current` or `status: planned`, a description, and proper delimited formatting (`---` boundaries).
   - **`index-format`**: Fix any catalog entries in `wiki/index.md` that do not follow the `- [Title](path.md) | Description` format.
   - **`bare-urls`**: Wrap raw URLs in standard markdown links, and replace HTML `<a>` tags with markdown syntax.
   - **`cross-page-links` / `orphans`**: Fix broken links or add links to connect orphaned pages to the active graph. Also cross-link pages that only receive links — the only intentional leaf is `log.md`. Links are strictly page-relative: a page inside a subdirectory must use `../` to reach pages in other directories.
   - **`leaf-pages`**: An active page has no outgoing links — cross-link it to its related pages (`log.md` is the only intentional leaf).
   - **`heading-hierarchy` / `log-headings` / `log-chronology`**: Correct markdown heading levels, log entry formats, or reverse-chronological ordering.
   - Other checkers: resolve formatting, markers, phase consistency, or duplicate/stale content warnings.
4. Apply the necessary fixes to the files under `wiki/`.
5. Re-run `wiki-engine lint` (or `wiki-engine lint --json`) to verify that all issues have been successfully resolved, and run `wiki-engine context --active` to confirm the graph is connected.
6. If pages moved between directories or wiki files were renamed, run `make audit` (repo-wide reference audit) and fix any page-relative link or stale `wiki/<path>.md` prose reference it reports.
7. Summarize which issues were identified, the changes made to resolve them, and confirm that the linter now passes cleanly.
