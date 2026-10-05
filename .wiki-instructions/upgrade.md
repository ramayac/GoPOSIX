---
description: "Upgrade the wiki-engine CLI binary, sync all prompt layers, and verify wiki integrity."
name: "Wiki Upgrade"
argument-hint: "none"
agent: "agent"
---

Upgrade the wiki-engine CLI and synchronize all instructions, then verify wiki integrity and lint health.

## Required context

- Run `wiki-engine context --active` to get the current snapshot of active pages.

## Execution steps

1. Run `wiki-engine upgrade`.
2. Run `wiki-engine sync-prompts` in the repository root to synchronize all local prompt files and symlinks.
3. Verify that the linter passes cleanly:
   ```bash
   wiki-engine lint --json
   ```
4. If the linter reports any issues (such as missing front matter or invalid formatting on newly synced templates/files), fix them or run the `/wiki-lint` workflow.
5. Confirm that the active wiki graph is healthy:
   ```bash
   wiki-engine context --active
   ```

Finish by summarizing that the upgrade is complete, confirming the new version, and confirming that the linter and active graph are healthy.
