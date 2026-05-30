# Lessons Learned

> **Permanent record** of insights, gotchas, and design decisions across all GoPOSIX development phases.
> **Last updated:** 2026-05-30 | **Coverage:** 84.1% | **BusyBox:** 877/17/25 (98.1%)

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

---

## Flag Parsing & CLI

### Never use `-j` as short flag for `--json`

It collides with `tar -j` (bzip2 in POSIX), and any utility where `-j` could be legitimate positional data. Use long-form `--json` only.

### Shared infrastructure needs escape hatches

`common.ParseFlags` applied uniformly to all utilities broke `echo`, `printf`, and `expr` — their positional args can start with `-`. One architectural mistake caused 40+ cascading test failures. Fix: free-form utilities must use manual flag parsing that stops at the first non-flag argument. `ParseFlags` offers a `StopAtFirstNonFlag` mode for this class of tool.

### Custom flag parsers break daemon `--json` prepend

The daemon prepends `--json` to every utility's args. Utilities with custom flag parsing that treat unknown flags as termination break when `--json` appears first. Fix: either make the custom parser recognize `--json`, or use a daemon-specific dispatch path.

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

---

## BusyBox Compatibility Notes

### BusyBox `dc` bugs found during implementation

1. **String parsing in `-f` vs `-e` mode**: BusyBox `dc` parses `\[` differently when reading from a file (`-f`) vs command-line (`-e`). In `-e` mode, `\[` is correctly treated as escaped bracket; in `-f` mode, it may cause premature string termination.

2. **Precision difference in chained division**: BusyBox `dc` produces a mathematically incorrect result for complex division chains due to floating-point rounding artifacts.

3. **Line wrapping at 69 chars**: BusyBox `dc` wraps long output lines at 69 content characters (with `\` at position 70). Our `wrapOutput()` must match this exact boundary.

4. **Per-number scale vs global scale**: BusyBox `dc` stores each number with its own internal scale. The `K` command pushes a value with scale 0, while division results use the global `k` scale.

5. **`0^0` and `0^(-n)` conventions**: BusyBox `dc` defines `0^0 = 1` and `0^(-n) = 0` for n > 0.

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

---

## Performance & Benchmarking

### Daemon-first: Go SDK is 11× faster than BusyBox fork+exec

Persistent Go SDK client achieves 60µs per RPC call vs 680µs for BusyBox fork+exec. Benchmark through the SDK, not socat — socat-per-call measures socat overhead, not daemon performance.

### Quick smoke before full benchmark

Run `make bench-quick SCALE=0.1` (~30s) before `make bench-all SCALE=1.0` (~8 min). Catches timing bugs, syntax errors, and daemon startup failures fast.

---

## Process & Workflow

### When an example exposes a pre-existing bug, note it but don't expand scope

The example can work around it. Fix the bug in a separate focused PR.

### Flag ordering matters for CLI utilities

Helper methods must match the CLI argument convention exactly. `append([]string{pattern}, flags...)` — pattern always comes first.

### Use `bash` as a fallback for tooling constraints

When a tool rejects creating new files, `bash` heredoc works reliably.
