# System Architecture

GoPOSIX is a POSIX-compliant userland implemented as a single, statically-linked Go binary.
It functions as both a traditional CLI tool (multicall binary) and as a persistent
JSON-RPC 2.0 daemon for AI agent backends.

**Version:** v1.0.0 (Gold) | **Go:** 1.26 | **Binary:** ~10 MB fully static

## Core Design Principles

1. **Minimal Dependencies:** Only `mvdan.cc/sh/v3` (shell interpreter) and
   `golang.org/x/sys` (macOS/BSD compatibility). No other third-party libraries.
2. **Dual-Mode Execution:**
   - **CLI Mode:** Standard POSIX stdout/stderr, exit codes.
   - **JSON Mode:** `--json` flag or daemon invocation → structured JSON envelope output.
3. **Container-Native:** Runs as non-root user `goposix:1000` inside a `FROM scratch` Docker
   image. Compiles with `CGO_ENABLED=0` for full static linking.

## Component Flow

```
                  ┌──────────────────────────────────┐
                  │          AI Agent / User          │
                  └──────┬───────────────┬───────────┘
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
              │ pkg/ls  │ │ pkg/cat │ │ pkg/... │  (40+ utilities)
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

Agent clients can also use the Go client library (pkg/client) for typed
JSON-RPC calls, connection pooling, and retry logic.
```

## Directory Structure

```
GoPOSIX/
├── cmd/goposix/          Main entry point: multicall dispatch + symlink handling
├── internal/
│   ├── dispatch/        Command registry (init() auto-registration)
│   ├── daemon/          JSON-RPC 2.0 persistent server (Unix socket)
│   └── shell/           Sandboxed shell execution (mvdan.cc/sh, timeout, limits)
├── pkg/
│   ├── common/          Foundation: flags.go, output.go, security.go, json.go
│   ├── client/          Go JSON-RPC client (connection pool, retry, typed helpers)
│   ├── daemon/          Daemon bootstrap + CLI entry point
│   ├── shell/           Shell CLI wrapper
│   └── <utility>/       40+ POSIX utility implementations (ls, cat, grep, sed, ...)
├── docker/              Dockerfiles (scratch production, alpine debug)
├── test/                Integration tests
│   └── busybox_testsuite/  Ported BusyBox test suite (552 tests)
├── testdata/            Shared test fixtures
├── docs/                Architecture, SECURITY, RPC_API, JSON_SCHEMA, AGENT_INTEGRATION
├── wiki/                Phase plans, checklists, lessons learned
└── examples/            Agent integration examples
```

## Key Packages

| Package | Role |
|---------|------|
| `cmd/goposix` | Multicall entry. Detects symlink name (`/bin/ls → goposix`) or subcommand (`goposix ls`). |
| `internal/dispatch` | Registry where utilities self-register via `init()`. |
| `internal/daemon` | JSON-RPC 2.0 server over Unix socket. Dispatches to registered commands. |
| `internal/shell` | Sandbox for `shell.exec` RPC. Configurable timeout, output limits, path confinement. |
| `pkg/common` | Shared: POSIX flag parser (`ParseFlags`), JSON envelope output (`Render`/`RenderError`), path security guards. |
| `pkg/client` | Go SDK for agents. Connection pooling, batch requests, exponential backoff, typed wrappers for every utility. |
| `pkg/<util>` | One package per POSIX utility. Library layer (testable `Run()`) + CLI layer (`run()`) wired via `init()` → dispatch. |

## Utilities Implemented (40+)

`basename`, `cat`, `chgrp`, `chmod`, `chown`, `cp`, `cut`, `date`, `df`, `diff`, `dirname`,
`du`, `echo`, `env`, `expr`, `find`, `grep`, `gzip`, `head`, `hostname`, `id`, `kill`,
`ln`, `ls`, `md5sum`, `mkdir`, `mv`, `printenv`, `printf`, `ps`, `pwd`, `readlink`, `rm`,
`rmdir`, `sed`, `sha256sum`, `sleep`, `sort`, `stat`, `tail`, `tar`, `tee`, `testcmd`,
`touch`, `tr`, `truefalse` (`true`/`false`), `uname`, `uniq`, `wc`, `whoami`, `xargs`, `yes`

## BusyBox Test Suite

**477 passed, 3 failed, 10 skipped** (99.4% pass rate). The 3 remaining failures are all in the
`date` utility — 2 Go POSIX timezone limitations and 1 cosmetic error-format mismatch.
Run `make testsuite` before every commit to prevent regressions.

## Phase History

| Phase | Scope | Status |
|-------|-------|--------|
| 00–01 | Foundation + Tier 1 utilities | ✅ |
| 02 | Docker CI & `scratch` pipeline | ✅ |
| 03 | Filesystem utils (ls, cat, rm, cp, mv, ...) | ✅ |
| 04 | Text utils (grep, sed, sort, wc, ...) | ✅ |
| 05 | JSON-RPC daemon core | ✅ |
| 06 | System & process utils (ps, find, df, du, ...) | ✅ |
| 07 | Agent-ready features (diff, tar, shell) | ✅ |
| 08 | Security hardening | ✅ |
| 09 | Release & automation | ✅ |
| 10 | POSIX test framework + BusyBox suite | ✅ |
| 11 | Post-MVP cleanup, lessons learned | ✅ |
| 12 | Road to Gold — supply chain, macOS, coverage, BusyBox parity | ✅ |
| 13 | Coverage & hardening (75.7% coverage reached) | ✅ |
| 14a-c | JSON gap fill, BusyBox regression fix, JSON-RPC daemon coverage | ✅ |
| 15–19 | Post-MVP tiers 1–3, quality fixes, performance benchmarks | ✅ |
| 20 | Hardening II — flag audit, code cleanup, doc fixes, coverage | 🔄 |
| — | `awk` (Platinum gate) | ⬜ |

## Related Documentation

- [SECURITY.md](SECURITY.md) — Security model, shell sandbox, deployment posture
- [RPC_API.md](RPC_API.md) — JSON-RPC client API reference (`pkg/client`)
- [JSON_SCHEMA.md](JSON_SCHEMA.md) — `--json` output envelope and per-utility schemas
- [AGENT_INTEGRATION.md](AGENT_INTEGRATION.md) — How to use GoPOSIX as an AI agent backend
