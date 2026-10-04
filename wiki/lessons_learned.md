# Lessons Learned

> **Permanent record** of insights, gotchas, and design decisions across all GoPOSIX development phases.
> **Last updated:** 2026-10-04 | **Coverage:** 85.6% | **BusyBox:** 870/17/30 (98.1%)

---

## Architecture & Design

### Generic `callUtility[T]` eliminated 42× boilerplate

Rather than writing bespoke JSON unmarshaling for every utility helper, a single Go generic function handles all of them:

```go
func callUtility[T any](c *Client, ctx context.Context, method string, params interface{}) (*T, error)
```

Each of 42 helpers is now 3–4 lines. Reusable for future utilities.

### Connection pool semaphore pattern

Using a buffered channel (`chan struct{}`) as a semaphore for connection pooling is clean and idiomatic. `select` on `ctx.Done()` vs. `p.sem` gives correct context propagation for free.

### Batch operations deliberately skip retry

Batch requests are not retried because partial success/failure is ambiguous — some requests may have succeeded on the server before the connection dropped.

### Schemas are self-contained (envelope + data in one file)

Each schema file includes both the envelope structure AND the utility-specific `data` shape. Zero-config validation (`ajv validate -s test/schemas/ls.schema.json -d golden/ls.json`). Tradeoff: some duplication across schema files.

### `schemaVersion` field is forward-looking

The `"schemaVersion": "1.0"` field in every JSON envelope allows consumers to detect and adapt to future schema changes. Major changes increment the integer; minor additions increment the decimal.

### In JSON mode, stdout must carry only the envelope

Cat-style and stream tools (`bzcat`, `gzip -c`, `bunzip2/unlzma/uncompress -c`, `cpio -o`) write the raw payload to stdout **before** the envelope when `--json` is set. The daemon cannot parse the mixed output, and the golden fixture must be trimmed to the trailing JSON line. This is audit finding F16 — the fix belongs to the decompress/archive cores, not the CLI glue.

**F16 resolution (audit/whatsleft):** in JSON stdout mode, capture the payload in a bounded buffer (50 MB, the daemon response cap) and embed it as base64 `content` inside the envelope (`DecompFileInfo.Content`, `GzipStat.Content`, `CpioResult.Content`). Text mode stays byte-for-byte unchanged. The optional schema field keeps the shape backwards-compatible. Tradeoff accepted: base64 inflates the payload ~33% and the cap bounds it — machine consumers decode the field, humans pipe without `--json`.

### Shell scripts need a JSON wrapper, not per-command JSON

`shell` runs arbitrary programs, so per-command JSON shapes are impossible. The F15 fix wraps the whole script result: `data = {exitCode, stdout, stderr}`. The script's own exit code travels in `data.exitCode` **and** as the process exit code (the daemon returns the process code with the parsed envelope data). Same pattern as `expr`, `nice`, and `nohup` embedding child exit codes.

### Package-global writers break the daemon contract

`logger` swapped a package-global `stderrWriter` per call — process-local mutable state that races under concurrent JSON-RPC calls (P1). The fix follows the `catRun()` pattern: thread the writer through the library function (`Run(..., errOut io.Writer)`) and delete the global. Any utility that needs output outside the envelope must take it as a parameter.

### Test-harness temp dirs must be unique per run

Two `make testsuite` runs shared `runtest-tempdir-links/` and `.tmpdir.$applet`; each run `rm -rf`'d what the other was using, producing spurious 92-failure runs (P4). Fix: `mktemp -d` per run with an EXIT trap, and PID-suffixed per-testcase dirs. Delete tracked harness-symlink trees from the repo so the harness never mutates tracked files.

---

## Flag Parsing & CLI

### Never use `-j` as short flag for `--json`

It collides with `tar -j` (bzip2 in POSIX), and any utility where `-j` could be legitimate positional data. Use long-form `--json` only.

### Shared infrastructure needs escape hatches

`common.ParseFlags` applied uniformly to all utilities broke `echo`, `printf`, and `expr` — their positional args can start with `-`. One architectural mistake caused 40+ cascading test failures. Fix: free-form utilities must use manual flag parsing that stops at the first non-flag argument. `ParseFlags` offers a `StopAtFirstNonFlag` mode for this class of tool.

### Custom flag parsers break daemon `--json` prepend

The daemon prepends `--json` to every utility's args. Utilities with custom flag parsing that treat unknown flags as termination break when `--json` appears first. Fix: either make the custom parser recognize `--json`, or use a daemon-specific dispatch path.

### Dash-form signal flags need preprocessing before `ParseFlags`

`kill -TERM`, `-15`, or `-9` look like bundled short flags to the unified parser (`-T -E -R -M`), producing "unknown flag" errors. POSIX signal syntax must be rewritten before parsing: scan args and convert dash-form signals (`-TERM`, `-15`) into `-s <signal>` pairs, passing known flags through unchanged. Same preprocessing pattern as `find -exec`: capture non-flag units before the parser sees them.

---

## Testing & CI

### BusyBox test suite gates every commit

The BusyBox suite chains utilities: echo creates files → diff compares → ls lists → find verifies. A bug in one utility silently breaks downstream tests in completely different utilities. Run `make testsuite` before every commit to prevent regressions.

### Integration tests catch cascading failures

When shared infrastructure changes (like `common.ParseFlags`), unit tests pass but BusyBox integration tests fail across dozens of utilities. The two suites catch different failure modes. Always run both.

### `SecurePath` blocks absolute paths when session CWD ≠ `/`

When session CWD is `/tmp` and a utility call references `/etc/hosts`, the resolved path `/tmp/etc/hosts` doesn't exist and `SecurePath` rejects it as traversal. This is a deliberate security feature — session-based access restricts all file operations to the session's working directory.

### Compliance tests need careful variable scoping

Don't use relative paths (`./goposix`) in compliance tests — they break when the test `cd`s to a tempdir. Use configurable variables (`GOPOSIX_AR=${GOPOSIX_AR:-goposix}`). Never use `|| true` after exit — it masks exit codes.

### Golden fixture generation can't be fully automated

Each utility has edge cases in its `--json` output (stdout leakage, flag name inconsistencies, data dependencies). Manual verification of each fixture is essential.

### Never register `sh` in the multicall binary

The BusyBox test harness auto-generates symlinks for every command returned by `--list-commands`. If `sh` is registered, a `sh -> goposix` symlink shadows the system `/bin/sh`, causing ALL tests to fail. Only register `shell`. The `--list-commands` output is consumed by tooling that creates real filesystem symlinks.

### Vacuous tests pass for the wrong reason

`TestKillInvalidSignal` asserted only a non-zero exit code; `-s BOGUS` failed flag parsing, so the test passed while the feature did not exist. When adding tests for new flags, assert the positive behavior (the feature working), not just error exits. Tests that "pass" via parse errors are the smell to look for.

### Codecov line coverage: kill dead closures in JSON-mode `Render`

`common.Render` never calls its text callback in JSON mode, so passing `func() {}` leaves a permanent "partial" line in Codecov. Pass `nil` instead — every such call site is guarded by `if jsonMode`. This is also why Codecov shows lower numbers than `go test -cover`: line-based vs statement-based counting, plus dead callbacks and unreachable defensive branches.

### Function seams make defensive error branches testable

`os.Getwd`/`os.Stat` error paths cannot be triggered without races or root-only filesystem states. Package-level function vars (`var osGetwd = os.Getwd`) with test overrides exercise each branch deterministically — the same injectable-entry-point pattern as `catRun`. Use for hard-to-mock syscall error paths instead of skipping coverage.

### Codecov patch coverage: test both branches of every new error path

The 5d JSON work added `if jsonMode { RenderError } else { plain text }` blocks. The first test pass covered only the JSON branches and Codecov reported 76% patch coverage with 18 missing lines. Adding plain-text counterparts (`run([]string{}, ...)` alongside `run([]string{"--json"}, ...)`) and erroring-reader seams (`type errorReader struct{}`) closed all of them. When a new branch has two modes, write both tests immediately.

### A package-var seam is enough for hardcoded system paths

`pkg/who` hardcoded `/var/run/utmp` and `/run/utmp`, so the no-utmp return path was unreachable in tests. Moving the path list to a package var (`var utmpPaths = ...`) let a test swap in a nonexistent path and cover the `users: []` fix. Cheaper than full dependency injection and consistent with the function-seam pattern.

---

## Go-Specific Gotchas

### DevID pointer trap

`fmt.Sprintf("%v", fi.Sys())` formats a **pointer address**, not the struct value. Two `Lstat` calls return different pointer addresses even for the same file, making inode-based hard link tracking silently broken. Always dereference: `st.Dev:st.Ino`.

### Go's `init()` registration pattern — tests must import every utility they exercise

When new helpers are added to tests, the tests fail with "Method not found" because those packages weren't imported via blank imports. A missing import produces a runtime error, not a compile error. Always check the import list when adding new helper tests.

### Don't write to stdout before JSON in daemon mode

When a utility runs in JSON mode under the daemon, its stdout is captured as the response. Any non-JSON output written to stdout before the JSON envelope corrupts the response. Guard text output with `&& !jsonMode`.

### Go's `compress/lzw` ≠ Unix compress format

Go's `compress/lzw` package produces the LZW algorithm output but doesn't include the Unix `.Z` file header/magic bytes. Use system `compress` or embed pre-computed `.Z` files in test data.

### Uncommitted files can break `NewServer` signatures everywhere

An untracked work-in-progress file that changes a shared function signature breaks the entire build. Always check `git status` for stray uncommitted files before starting work.

### Adding `context.Context` to a public API is a breaking change

Grep the entire repo for callers before committing. Even test code in other packages can break.

### `os.Getwd` honors a valid `$PWD` — never trust it for physical paths

On Linux, Go's `os.Getwd` returns the `$PWD` environment value when it stats to the current directory (the logical path through symlinks), falling back to `getcwd` otherwise. Code that needs the physical path must resolve symlinks explicitly with `filepath.EvalSymlinks(dir)`. This bit `pwd` when the repo was reached through `/home/ramayac/git -> /mnt/plex_media/git`.

---

## BusyBox Compatibility Notes

### BusyBox `dc` bugs found during implementation

1. **String parsing in `-f` vs `-e` mode**: BusyBox `dc` parses `\[` differently when reading from a file (`-f`) vs command-line (`-e`). In `-e` mode, `\[` is correctly treated as escaped bracket; in `-f` mode, it may cause premature string termination.

2. **Precision difference in chained division**: BusyBox `dc` produces a mathematically incorrect result for complex division chains due to floating-point rounding artifacts.

3. **Line wrapping at 69 chars**: BusyBox `dc` wraps long output lines at 69 content characters (with `\` at position 70). Our `wrapOutput()` must match this exact boundary.

4. **Per-number scale vs global scale**: BusyBox `dc` stores each number with its own internal scale. The `K` command pushes a value with scale 0, while division results use the global `k` scale.

5. **`0^0` and `0^(-n)` conventions**: BusyBox `dc` defines `0^0 = 1` and `0^(-n) = 0` for n > 0.

### `pwd` defaults to the physical path; `-L` is logical

BusyBox and coreutils print the resolved (physical) path by default; `-L` prints `$PWD` only when it is absolute and names the current directory (POSIX `-L` semantics). A `pwd` that defaults to logical output breaks `realpath.tests` when the repo is reached through a symlink: the harness's `which pwd` resolves to the goposix `pwd` symlink (LINKSDIR precedes system PATH), and the test assumes external `pwd` prints physical — expected becomes logical while `realpath` prints physical. Root cause of 3 suite failures through `/home/ramayac/git`.

---

## Shell & Subprocess

### Shell interpreter arg-passing bug (pre-existing)

The shell interpreter (`mvdan.cc/sh`) includes the command name as `args[0]` when dispatching to GoPOSIX utilities, causing argument misalignment. Note it but don't expand scope to fix it.

### Shell aliases must be listed in `cmdPkgMapping`

When a package registers multiple command names (e.g., `shell`, `sh`, `ash`), the test `TestListCommandsMatchesPkgDir` checks every registered command maps to a `pkg/` directory. Always add aliases to the mapping.

---

## Schema & Validation

### `ajv-cli` via `npx` — zero install, runs everywhere

JSON Schema validation uses `npx ajv-cli validate -s <schema> -d <golden>`. No package.json, no global install, no CI caching needed.

### JSON Schema draft-07 was the correct target

Draft-07 has the broadest tooling support: `ajv`, Python `jsonschema`, every major language. Newer drafts have spotty CLI support.

### Moving schemas to `test/schemas/` was the right call

Schemas are test artifacts — they validate golden fixtures in CI — not documentation. Clear separation of concerns.

### Schema file names must match dispatch names, not source dirs

`mkfs_minix` registers as `mkfs.minix`, so its schema is `mkfs.minix.schema.json`. The audit plan called this the "name trap" (`testcmd` → `test.schema.json`, `truefalse` → `true`/`false`). `validate_schemas.sh` derives the utility name from the file name, so a mismatch silently skips validation.

### JSON emitters must never output `null` for an empty collection

`pkg/who` returned `Users: nil` when no utmp file existed, so `--json` emitted `"users": null`. The schema required an array, so the fixture could not validate. Rule: empty collections serialize as `[]` (non-nil slice), never `null`.

### Error envelopes go to stderr; the daemon parses stdout only

`common.RenderError` writes the envelope to the passed writer, and flag/usage errors use `stderr`. Over JSON-RPC the daemon parses the stdout buffer only, so a stderr error envelope becomes `{exitCode, data: null, stderr: "<envelope json>"}` — a successful JSON-RPC result, not a JSON-RPC error. Success envelopes always go to stdout. In JSON mode stdout must carry **only** the envelope.

### The envelope `exitCode` can differ from the process exit code

`common.Render` hardcodes the envelope `exitCode: 0`; some commands (e.g. `uncompress`, `makedevs`) render a success envelope but still return exit 1 when a per-file operation failed. The daemon returns the **process** exit code plus the parsed envelope `data`. Consumers must not assume the two agree.

### Golden fixtures close the schema-gap silently

`validate_schemas.sh` reports `SKIP` for schemas without fixtures. The pre-audit wiki claimed 77 schemas, but 31 of them skipped validation because no fixture existed. After regenerating fixtures, `make validate-schemas` reports 114 passed, 0 failed, 0 skipped — the skip count is part of the definition of done.

### `gen_golden.sh` — bash `local a="$1" b="$a"` expands before assignment

In a single `local` declaration, the second assignment sees the **old** value of the first variable. With `set -u` this aborts the script on the first call. Declare locals first, then assign on separate lines. The pre-existing script had never run cleanly because of this.

---

## Performance & Benchmarking

### Benchmark through the SDK, not socat

Socat-per-call measures socat process overhead, not daemon performance. The Go SDK with a persistent connection is the only valid way to benchmark (see [performance.md](performance.md) for numbers).

### Quick smoke before full benchmark

Run `make bench-quick SCALE=0.1` (~30s) before `make bench-all SCALE=1.0` (~8 min). Catches 90% of timing bugs and daemon startup failures before the 8-minute run wastes time. See [performance.md](performance.md) for the full benchmark reference.

---

## Process & Workflow

### When an example exposes a pre-existing bug, note it but don't expand scope

The example can work around it. Fix the bug in a separate focused PR.

### Flag ordering matters for CLI utilities

Helper methods must match the CLI argument convention exactly. `append([]string{pattern}, flags...)` — pattern always comes first.

### Use `bash` as a fallback for tooling constraints

When a tool rejects creating new files, `bash` heredoc works reliably.
