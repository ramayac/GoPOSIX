# Phase 07 — Agent-Ready Features + Tier 5

> **HISTORICAL — COMPLETED.** This phase is done. Document retained for reference.
>
> **Timeline:** Week 9–10 | **Depends on:** Phase 05, 06

---

## Goal

Add stateful RPC sessions, embedded shell interpreter, and the final tier of advanced utilities.

## Tasks

### 07.1 — Stateful Sessions (`internal/daemon/session.go`)

Each agent connection can create a session that remembers state across RPC calls.

```json
// Create session
{"jsonrpc":"2.0","method":"korego.session.create","id":1}
→ {"result":{"sessionId":"abc-123","cwd":"/"}}

// Set working directory
{"jsonrpc":"2.0","method":"korego.session.setCwd","params":{"sessionId":"abc-123","path":"/tmp"},"id":2}

// Run command in session context
{"jsonrpc":"2.0","method":"korego.ls","params":{"sessionId":"abc-123","path":"."},"id":3}
// ls runs relative to /tmp because of session cwd
```

- [x] Session stores: `cwd`, `env` vars, command history
- [x] Session TTL: auto-expire after 30min idle (configurable)
- [x] `korego.session.list` — list active sessions
- [x] `korego.session.destroy` — cleanup

### 07.2 — Shell Interpreter (`internal/shell/interpreter.go`)

Embed `mvdan.cc/sh/v3` to execute shell scripts via RPC.

```json
{"jsonrpc":"2.0","method":"korego.shell.exec",
 "params":{"script":"ls -la | grep txt | wc -l", "sessionId":"abc-123"},"id":4}
→ {"result":{"stdout":"3\n","stderr":"","exitCode":0}}
```

- [x] Builtins: KoreGo utilities are registered as shell builtins (no fork/exec)
- [x] Pipes: `ls | grep | wc` uses Go channels, not OS pipes
- [x] Environment: inherits from session env vars
- [x] Safety: execution timeout (default 30s), memory limit

### 07.3 — Structured Logging

All daemon operations log as structured JSON to stderr/file.

```json
{"time":"2026-04-24T20:00:00Z","level":"info","method":"korego.ls","sessionId":"abc-123","durationMs":2.1}
```

- [x] Use `log/slog` from stdlib
- [x] Log levels: debug, info, warn, error
- [x] Fields: timestamp, level, method, session_id, duration_ms, error

### 07.4 — Tier 5 Utilities

Implemented in order of complexity, each building on patterns established by earlier utilities.

> **Note:** `awk` has been moved to a dedicated document — see [07a_awk.md](07a_awk.md).

---

#### 07.4.1 — `printf` ([pkg/printf/](../pkg/printf/)) — Formatted Output ✅ Complete

**Complexity:** Low — pure output, no filesystem interaction.

Implement POSIX `printf` with format specifiers (`%s`, `%d`, `%x`, `%o`, `%f`, `\n`, `\t`, `\\`, `\0NNN`).

- [x] Custom format engine (not just `fmt.Sprintf` wrapper)
- [x] Support `%s`, `%d`, `%i`, `%x`, `%X`, `%o`, `%f`, `%e`, `%g`, `%c` specifiers with width/precision
- [x] Support escape sequences `\n`, `\t`, `\r`, `\\`, `\a`, `\b`, `\f`, `\v`, `\0NNN` (octal)
- [x] Cycle through args if more args than specifiers
- [x] `--json` output: `{"output": "..."}`
- [x] Library layer (`Format`) separated from CLI layer
- [x] Unit tests (24 tests)

---

#### 07.4.2 — `expr` ([pkg/expr/](../pkg/expr/)) — Integer Arithmetic & String Ops ✅ Complete

**Complexity:** Low-Medium — recursive-descent expression parser, no I/O.

Implement POSIX `expr` supporting arithmetic (`+`, `-`, `*`, `/`, `%`), comparison (`<`, `<=`, `=`, `!=`, `>=`, `>`), logical (`|`, `&`), string matching (`:` / `match`), and `substr`, `index`, `length`.

- [x] Recursive-descent parser with correct operator precedence
- [x] Arithmetic: `+`, `-`, `*`, `/`, `%` with division-by-zero guard
- [x] Comparison operators (`<`, `<=`, `=`, `!=`, `>=`, `>`) — integer and string
- [x] Logical operators (`|`, `&`)
- [x] String operations: `match`, `substr`, `index`, `length`
- [x] `:` infix operator (regex match)
- [x] Parenthesized sub-expressions
- [x] Exit code: `0` (true/nonzero), `1` (false/zero), `2` (error)
- [x] `--json` output: `{"result": "...", "exitCode": N}`
- [x] Library layer (`Eval`) separated from CLI layer
- [x] Unit tests (23 tests)

---

#### 07.4.3 — `test` / `[` ([pkg/testcmd/](../pkg/testcmd/)) — Conditional Expressions ✅ Complete

**Complexity:** Medium — must handle file tests, string tests, integer comparisons, and compound expressions.

Implement POSIX `test` (and `[` symlink form) for shell conditional evaluation.

- [x] File tests: `-e`, `-f`, `-d`, `-s`, `-r`, `-w`, `-x`, `-L`, `-h`, `-b`, `-c`, `-p`, `-S`, `-g`, `-u`, `-k`
- [x] String tests: `-z`, `-n`, `=`, `==`, `!=`
- [x] Integer comparisons: `-eq`, `-ne`, `-lt`, `-le`, `-gt`, `-ge`
- [x] Logical operators: `!`, `-a`, `-o`, `(`, `)`
- [x] `[` mode: validate closing `]` argument
- [x] Exit code only (no stdout), `--json` → `{"result": true/false}`
- [x] Library layer (`Evaluate`) separated from CLI layer
- [x] Unit tests (24 tests)

---

#### 07.4.4 — `sha256sum` / `md5sum` ([pkg/sha256sum/](../pkg/sha256sum/), [pkg/md5sum/](../pkg/md5sum/)) — Cryptographic Hashing ✅ Complete

**Complexity:** Medium — streaming file I/O with `crypto/sha256` and `crypto/md5`.

Hash one or more files, output in standard `HASH  FILENAME` format.

**`sha256sum`:**
- [x] Stream file contents through `sha256.New()` via `io.Copy` (no full-file buffering)
- [x] Multi-file hashing with per-file error handling
- [x] Support `-c` / `--check` mode (verify hashes from a checksum file)
- [x] Handle stdin via `-` argument
- [x] `--json` output: `[{"file": "f", "hash": "abc...", "algorithm": "sha256"}]`
- [x] Library layer (`HashFile`) separated from CLI layer
- [x] Unit tests (9 tests)

**`md5sum`:**
- [x] Full implementation mirroring `sha256sum` with `crypto/md5`
- [x] Supports same flags: `-c`, `--json`, stdin
- [x] Library layer (`HashFile`) separated from CLI layer
- [x] Unit tests (8 tests)

---

#### 07.4.5 — `diff` ([pkg/diff/](../pkg/diff/)) — File Comparison ✅ Complete

**Complexity:** Medium-High — Myers diff algorithm, hunk formatting.

Compare two files line by line, producing unified diff output.

- [x] Read two files, compare with `bytes.Equal`
- [x] Exit code: `0` (identical), `1` (different), `2` (error)
- [x] Basic `--json` output (`{"differ": true/false}`)
- [x] Implement Myers diff algorithm (or equivalent LCS-based)
- [x] Unified diff format with `---`/`+++` headers and `@@` hunk markers
- [x] Context lines (default 3, configurable via `-U N`)
- [x] `--json` structured hunks: `{"files": [...], "hunks": [...]}`
- [x] Unit tests (5 tests)

---

#### 07.4.6 — `gzip` / `gunzip` ([pkg/gzip/](../pkg/gzip/)) — Compression ✅ Complete

**Complexity:** Medium-High — streaming compression with `compress/gzip`, file replacement semantics.

Compress and decompress files using gzip format.

- [x] `gzip FILE` → creates `FILE.gz`, removes original
- [x] `gunzip FILE.gz` → restores `FILE`, removes `.gz`
- [x] `-d` / `--decompress` — gzip delegates to gunzip
- [x] Stdin/stdout piping when no file argument
- [x] `-c` / `--stdout` — write to stdout, keep original
- [x] `-k` / `--keep` — keep original file
- [x] `-f` / `--force` — overwrite existing output
- [x] `--json` output with size/ratio stats
- [x] Unit tests (5 tests)

---

#### 07.4.7 — `tar` ([pkg/tar/](../pkg/tar/)) — Archive Create/Extract ✅ Complete

**Complexity:** High — recursive directory traversal, multiple archive formats, gzip integration.

Create and extract tar archives with optional gzip compression.

- [x] `-c` — create archive (working, uses `filepath.Walk` + `tar.Writer`)
- [x] `-f FILE` — specify archive filename
- [x] `-z` — gzip compression on create (via `compress/gzip`)
- [x] `-v` — verbose listing during create
- [x] Preserve permissions and timestamps (via `tar.FileInfoHeader`)
- [x] `-x` — extract archive
- [x] `-t` — list archive contents
- [x] `-C DIR` — change to directory before operating
- [x] `--json` output: `[{"name": "...", "size": N, "mode": "..."}]`
- [x] Unit tests (5 tests)

### 07.5 — Benchmarking Suite

```
test/benchmark/
├── bench_cli_test.go      # Cold-start CLI latency per utility
├── bench_daemon_test.go   # Warm daemon RPC latency per utility
└── bench_batch_test.go    # Batch throughput (requests/sec)
```

Targets:
- CLI cold start: < 10ms (Go runtime init)
- Daemon echo: < 1ms
- Daemon ls: < 5ms
- Batch throughput: > 1000 req/sec

## Milestone 07

- [x] Agent creates session, sets cwd, runs relative commands
- [x] `korego.shell.exec "ls | grep go | wc -l"` returns structured result
- [x] Daemon latency < 1ms for `echo`, < 5ms for `ls`
- [x] Sessions auto-expire after TTL
- [x] `sha256sum --json file` returns hash
- [x] `tar -czf archive.tar.gz dir/` creates archive
- [x] Makefile target so the user can get an interactive shell in the docker image. (docker run -it korego)

## How to Verify

```bash
# Session workflow
echo '{"jsonrpc":"2.0","method":"korego.session.create","id":1}' | socat - UNIX-CONNECT:/tmp/korego.sock
# → get sessionId, then use it in subsequent calls

# Shell exec
echo '{"jsonrpc":"2.0","method":"korego.shell.exec","params":{"script":"echo hello | tr a-z A-Z"},"id":2}' \
  | socat - UNIX-CONNECT:/tmp/korego.sock

# Benchmarks
go test -bench=. -benchmem ./test/benchmark/
```
