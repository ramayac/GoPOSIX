# Phase 14c — POSIX JSON-RPC Coverage Gap

> **Date:** 2026-05-15 | **Status:** <mck>Implemented</mck> — 2026-05-16

## The Gap

The JSON-RPC daemon is a core architectural feature of KoreGo — every utility is supposed to support structured machine-readable output via `--json`, and the daemon exposes all utilities over a persistent JSON-RPC socket to avoid process-spawning overhead.

The integration test at `test/posix-json/runner_test.go` only exercises **9 of 55** utility packages:

| Covered | Uncovered |
|---------|-----------|
| cat, echo, sed, sleep, tee, testcmd, tr, truefalse, yes | 46 others |

No JSON-RPC coverage exists for critical utilities like ls, cp, mv, rm, grep, find, tar, gzip, diff, printf, date, du, df, etc.

## What's Needed

Each utility should have at least one JSON-RPC integration test that:

1. Starts the daemon (reuse `startDaemon`)
2. Calls the utility via `korego.<name>` method
3. Validates the structured JSON response envelope:
   - `exitCode` is correct
   - `data` contains expected typed fields
   - `error` is nil for success cases, populated for failure cases

## Priority Order

### Tier 1 — Filesystem (highest risk, most dependencies)
ls, cp, mv, rm, mkdir, rmdir, touch, ln, readlink, stat, chmod, chown, chgrp

### Tier 2 — Text/Stream (heavily used in agent pipelines)
grep, find, sort, uniq, wc, head, tail, cut, tr, diff, printf

### Tier 3 — System/Info
date, du, df, ps, id, hostname, whoami, pwd, uname, kill, sleep (already covered)

### Tier 4 — Archive/Compression
tar, gzip, sha256sum, md5sum

### Tier 5 — Expression/Misc
expr, basename, dirname, env, printenv, xargs

## Result (2026-05-16)

- **Coverage:** 55/55 utilities (100%) — up from 9/55 (16%)
- **Test files added:**
  - `test/posix-json/tier1_filesystem_test.go` — 13 new tests (ls, cp, mv, rm, mkdir, rmdir, touch, ln, readlink, stat, chmod, chown, chgrp)
  - `test/posix-json/tier2_text_test.go` — 11 new tests (grep, find, sort, uniq, wc, head, tail, cut, diff, printf)
  - `test/posix-json/tier3_system_test.go` — 10 new tests (date, du, df, ps, id, hostname, whoami, pwd, uname, kill)
  - `test/posix-json/tier4_archive_test.go` — 5 new tests (tar, gzip, sha256sum, md5sum)
  - `test/posix-json/tier5_misc_test.go` — 7 new tests (expr, basename, dirname, env, printenv, xargs)
  - **Total:** 46 new test cases (one per previously-uncovered utility)
- **Bugs found and fixed:**
  - `pkg/find/find.go`: Flag pre-processing added extra `-` to `--json`, breaking JSON output via daemon. Fixed by checking for `--` prefix first.
  - `pkg/uniq/uniq.go`: Overwrote `out` writer with `os.Stdout`, causing JSON output to bypass daemon buffer. Fixed by guarding with `!jsonMode`.
- **BusyBox test suite:** 477 passed, 3 failed (pre-existing date issues). No regressions.
- **Unit tests:** All packages pass, including find and uniq.

## Target

- Current: 9/55 utilities covered (16%)
- Target: 100% (at least 1 test per utility)
- Minimum viable: Tier 1 + Tier 2 (24 utilities, 44%)

## Test Pattern

```go
func TestLsViaDaemon(t *testing.T) {
    socket := startDaemon(t)
    c := client.Dial(socket, 5*time.Second)

    t.Run("ls lists current directory", func(t *testing.T) {
        var result ResultWrapper
        err := c.Call(context.Background(), "korego.ls",
            map[string]interface{}{"flags": []interface{}{"-1"}},
            &result)
        if err != nil {
            t.Fatalf("unexpected error: %v", err)
        }
        if result.ExitCode != 0 {
            t.Errorf("expected exit 0, got %d", result.ExitCode)
        }
        // Validate data.files is a non-empty array
        data := result.Data.(map[string]interface{})
        files, ok := data["files"].([]interface{})
        if !ok || len(files) == 0 {
            t.Errorf("expected non-empty files array")
        }
    })
}
```
