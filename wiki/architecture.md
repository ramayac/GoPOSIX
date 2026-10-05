---
status: current
description: "System architecture: component flow, key packages, Docker images."
---

# System Architecture

GoPOSIX is a POSIX-compliant userland implemented as a single, statically-linked Go binary.
The primary interface is a persistent JSON-RPC 2.0 daemon (60µs/call).
A multicall CLI binary is available as a secondary interface.

**Version:** see [releases](https://github.com/ramayac/goposix/releases) | **Go:** 1.26 | **Binary:** <12 MB fully static

## Core Design Principles

1. **Minimal Dependencies:** Twelve external Go modules — ten direct and two
   indirect. Direct: `mvdan.cc/sh/v3` (shell), `github.com/benhoyt/goawk`
   (`awk`), `github.com/blakesmith/ar` (`ar`), `github.com/cavaliergopher/cpio`
   (`cpio`), `github.com/hotei/dcompress` (`uncompress`),
   `github.com/sergeymakinen/go-crypt` and `github.com/tredoe/crypt` (`cryptpw`),
   `github.com/ulikunitz/xz` (`tar`, `unlzma`), `golang.org/x/sys` (syscalls),
   `golang.org/x/crypto` (`sha3sum`). Indirect: `github.com/hotei/mdr`,
   `golang.org/x/term`. No other third-party libraries.
2. **Dual-Mode Execution:**
   - **CLI Mode:** Standard POSIX stdout/stderr, exit codes.
   - **JSON Mode:** `--json` flag or daemon invocation → structured JSON envelope output.
3. **Container-Native:** Runs as non-root user `goposix:1000` inside a `FROM scratch` Docker
   image. Compiles with `CGO_ENABLED=0` for full static linking.

## Performance

| Interface | Per-call latency | vs BusyBox (680µs fork+exec) |
|-----------|:---:|:---:|
| **JSON-RPC (persistent conn)** | **60µs** | **11× faster** |
| `socat` (per-call overhead) | 2,000µs | 3× slower |
| CLI cold start | 7,000µs | 10× slower |

Other wins: `grep` on 100MB file is 0.16s vs BusyBox 0.86s (5.4× faster, RE2 vs POSIX ERE).

## Component Flow

```
                         ┌─────────────────────────────┐
                         │  JSON-RPC Client (primary)  │
                         │  {"method":"goposix.ls",...}│
                         │  60µs/call                  │
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
              │ pkg/ls  │ │ pkg/cat │ │ pkg/... │  (115 utilities)
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
│   ├── daemon/          Daemon bootstrap + CLI entry point
│   ├── shell/           Shell CLI wrapper
│   └── <utility>/       115 POSIX utility implementations
├── docker/              Dockerfiles
│   ├── Dockerfile       Unified multi-stage: daemon, cli, debug, alpine-mvp targets
│   ├── Dockerfile.goreleaser           Release build (GoReleaser)
│   ├── Dockerfile.goreleaser.daemon    Release daemon build (GoReleaser)
│   └── Dockerfile.openbox              OpenBox image
├── upgrade.go           Self-upgrade: GitHub release fetching, tar.gz extraction, atomic binary replacement
├── forwarder.go         Smart forwarding: CLI → daemon when socket available
├── test/                Tests and benchmarks
│   ├── benchmark/       GoPOSIX vs BusyBox performance benchmarks
│   ├── busybox_testsuite/  Ported BusyBox test suite
│   ├── compliance/      Shell scripts comparing against host utilities
│   ├── integration/     Daemon integration tests
│   ├── posix-json/      JSON-RPC contract tests per tier
│   ├── schemas/         JSON output schemas
│   └── testutil/        Minimal raw JSON-RPC test client
├── testdata/            Shared test fixtures
├── wiki/                Architecture, JSON-RPC protocol, JSON schema, deploy guides
```

## Docker Images

See [repo-map.md](repo-map.md) for the canonical Docker image catalog.

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
| `pkg/common` | Shared: POSIX flag parser (`ParseFlags`), JSON envelope output (`Render`/`RenderError`), path security guards, signal parsing (`ParseSignal`/`SignalName`/`SignalNames`). |
| `pkg/<util>` | One package per POSIX utility. Library layer (testable `Run()`) + CLI layer (`run()`) wired via `init()` → dispatch. |

## Utilities Implemented (115)

All 115 utilities are cataloged in the [test coverage matrix](test_coverage_matrix.md) with
per-utility unit coverage, BusyBox test status, and JSON-RPC daemon registration status.

## Phase History

All 31 phases are complete. See [phases.md](phases.md) for the full phase index and current state.

## Related Documentation

- [index.md](index.md) — Wiki index
- [repo-map.md](repo-map.md) — Current repo architecture and exclusions
- [phases.md](phases.md) — Project roadmap, current state, and phase index
- [test_coverage_matrix.md](test_coverage_matrix.md) — Per-utility coverage and BusyBox status
- [performance.md](performance.md) — Benchmark commands and results
- [security.md](security.md) — Security model, shell sandbox, deployment posture
- [rpc_quickstart.md](rpc_quickstart.md) — JSON-RPC protocol reference
- [json_schema.md](json_schema.md) — `--json` output envelope and per-utility schemas
- [usage.md](usage.md) — Usage guide: CLI, daemon, Docker Compose, JSON-RPC, recipes
- [self_upgrade.md](self_upgrade.md) — Self-upgrade (`--version`, `--upgrade`)
- [deferred.md](deferred.md) — Deferred and planned future work
- [todos.md](todos.md) — Open TODOs and remaining BusyBox failures
