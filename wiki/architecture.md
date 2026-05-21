# System Architecture

GoPOSIX is a POSIX-compliant userland implemented as a single, statically-linked Go binary.
The primary interface is a persistent JSON-RPC 2.0 daemon with a typed Go SDK (60µs/call).
A multicall CLI binary is available as a secondary interface.

**Version:** v1.0.0 (Gold) | **Go:** 1.26 | **Binary:** ~10 MB fully static

## Core Design Principles

1. **Minimal Dependencies:** Only `mvdan.cc/sh/v3` (shell interpreter) and
   `golang.org/x/sys` (macOS/BSD compatibility). No other third-party libraries.
2. **Dual-Mode Execution:**
   - **CLI Mode:** Standard POSIX stdout/stderr, exit codes.
   - **JSON Mode:** `--json` flag or daemon invocation → structured JSON envelope output.
3. **Container-Native:** Runs as non-root user `goposix:1000` inside a `FROM scratch` Docker
   image. Compiles with `CGO_ENABLED=0` for full static linking.

## Performance

| Interface | Per-call latency | vs BusyBox (680µs fork+exec) |
|-----------|:---:|:---:|
| **Go SDK (persistent conn)** | **60µs** | **11× faster** |
| `socat` (per-call overhead) | 2,000µs | 3× slower |
| CLI cold start | 7,000µs | 10× slower |

Other wins: `grep` on 100MB file is 0.16s vs BusyBox 0.86s (5.4× faster, RE2 vs POSIX ERE).

## Component Flow

```
                         ┌─────────────────────────────┐
                         │  Go SDK Client (primary)     │
                         │  c.Ls(ctx, "/", nil)         │
                         │  60µs/call, typed methods    │
                         └──────────┬──────────────────┘
                                    │
                                    ▼
                  ┌─────────────────────────────────────┐
                  │  Programmatic Consumer / CLI User    │
                  └──────┬───────────────┬──────────────┘
                         │               │
                   Unix Socket     CLI invocation
                   (JSON-RPC)      (symlink/goposix <cmd>)
                         │               │
                         ▼               ▼
                  ┌────────────┐  ┌────────────────┐
                  │   daemon   │  │   multicall     │
                  │  (server)  │  │  dispatcher     │
                  └─────┬──────┘  └───────┬────────┘
                        │                 │
                        └────────┬────────┘
                                 │
                                 ▼
                        ┌────────────────┐
                        │ Command        │
                        │ Registry       │
                        │ (dispatch pkg) │
                        └───────┬────────┘
                                │
                    ┌───────────┼───────────┐
                    ▼           ▼           ▼
              ┌─────────┐ ┌─────────┐ ┌─────────┐
              │ pkg/ls  │ │ pkg/cat │ │ pkg/... │  (79 utilities)
              └────┬────┘ └────┬────┘ └────┬────┘
                   │           │           │
                   └───────────┼───────────┘
                               │
                               ▼
                      ┌────────────────┐
                      │ pkg/common     │
                      │ flags, output, │
                      │ security, json │
                      └────────────────┘
```

## Directory Structure

```
GoPOSIX/
├── cmd/goposix/          Main entry point: multicall dispatch + symlink handling
├── internal/
│   ├── dispatch/        Command registry (init() auto-registration)
│   ├── daemon/          JSON-RPC 2.0 persistent server (Unix socket, self-healing)
│   └── shell/           Sandboxed shell execution (mvdan.cc/sh, timeout, limits)
├── pkg/
│   ├── common/          Foundation: flags.go, compiled.go, output.go, security.go, json.go
│   ├── client/          Go JSON-RPC client (connection pool, retry, typed helpers)
│   ├── daemon/          Daemon bootstrap + CLI entry point
│   ├── shell/           Shell CLI wrapper
│   └── <utility>/       79 POSIX utility implementations (ls, cat, grep, sed, ...)
├── docker/              Dockerfiles
│   ├── Dockerfile       Default: daemon (FROM scratch, ~10 MB)
│   ├── Dockerfile.cli   CLI-only (FROM scratch, ~10 MB)
│   └── Dockerfile.debug Alpine + shell + strace for debugging
├── upgrade.go           Self-upgrade: GitHub release fetching, tar.gz extraction, atomic binary replacement
├── forwarder.go         Smart forwarding: CLI → daemon when socket available
├── test/                Integration tests
│   ├── benchmark/       GoPOSIX vs BusyBox performance benchmarks
│   └── busybox_testsuite/  Ported BusyBox test suite (552 tests)
├── testdata/            Shared test fixtures
├── wiki/                Architecture, security, RPC API, JSON schema, deploy guides
```

## Docker Images

| Image | Base | Size | Use case |
|-------|------|:---:|----------|
| `goposix:latest` | `FROM scratch` | ~10 MB | Default: daemon with JSON-RPC + HTTP metrics |
| `goposix:cli` | `FROM scratch` | ~10 MB | One-shot CLI invocations (`docker run --rm goposix:cli ls -la /`) |
| `goposix:debug` | `alpine:3.20` | ~28 MB | Shell, strace, file — interactive debugging |

Both production images use `# syntax=docker/dockerfile:1` + `COPY --chown=1000:1000`
to preserve directory ownership in `FROM scratch`. The daemon socket lives at
`/home/goposix/goposix.sock` (the only writable directory).

## Key Packages

| Package | Role |
|---------|------|
| `cmd/goposix` | Multicall entry. Detects symlink name (`/bin/ls → goposix`) or subcommand (`goposix ls`). |
| `internal/dispatch` | Registry where utilities self-register via `init()`. |
| `internal/daemon` | JSON-RPC 2.0 server over Unix socket. Dispatches to registered commands. |
| `internal/shell` | Sandbox for `shell.exec` RPC. Configurable timeout, output limits, path confinement. |
| `pkg/common` | Shared: POSIX flag parser (`ParseFlags`), JSON envelope output (`Render`/`RenderError`), path security guards. |
| `pkg/client` | Go SDK for JSON-RPC clients. Connection pooling, batch requests, exponential backoff, typed wrappers for every utility. |
| `pkg/<util>` | One package per POSIX utility. Library layer (testable `Run()`) + CLI layer (`run()`) wired via `init()` → dispatch. |

## Utilities Implemented (79)

`awk`, `basename`, `cat`, `chgrp`, `chmod`, `chown`, `cp`, `cut`, `date`, `df`, `diff`, `dirname`,
`du`, `echo`, `env`, `expr`, `find`, `grep`, `gzip`, `head`, `hostname`, `id`, `kill`,
`ln`, `ls`, `md5sum`, `mkdir`, `mv`, `printenv`, `printf`, `ps`, `pwd`, `readlink`, `rm`,
`rmdir`, `sed`, `sha256sum`, `sleep`, `sort`, `stat`, `tail`, `tar`, `tee`, `testcmd`,
`touch`, `tr`, `truefalse` (`true`/`false`), `uname`, `uniq`, `wc`, `whoami`, `xargs`, `yes`

## BusyBox Test Suite

Check [wiki/test_coverage_matrix.md](wiki/test_coverage_matrix.md) for the breakdown of unit test coverage, BusyBox test suite status, and JSON-RPC support for each utility.

## Phase History

All build phases (00–20, 22) are complete. See [phases.md](phases.md) for the full phase index.

## Related Documentation

- [phases.md](phases.md) — Project roadmap, current state, and phase index
- [security.md](security.md) — Security model, shell sandbox, deployment posture
- [rpc_api.md](rpc_api.md) — JSON-RPC client API reference (`pkg/client`)
- [json_schema.md](json_schema.md) — `--json` output envelope and per-utility schemas
- [usage.md](usage.md) — Usage guide: CLI, daemon, Docker Compose, Go SDK, recipes
- [self_upgrade.md](self_upgrade.md) — Self-upgrade (`--version`, `--upgrade`)
- [deferred.md](deferred.md) — Deferred and planned future work
- [todos.md](todos.md) — Open TODOs and remaining BusyBox failures
