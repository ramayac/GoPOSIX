# GoPOSIX

A Go-native POSIX userland with a persistent JSON-RPC 2.0 daemon and a typed Go SDK.
GoPOSIX replaces GNU Coreutils in Docker containers, delivering **60µs per RPC call** —
**11× faster than BusyBox fork+exec (680µs)**. Every utility supports structured `--json`
output. CLI access is available as a secondary interface.

[![CI](https://github.com/ramayac/goposix/actions/workflows/ci.yml/badge.svg)](https://github.com/ramayac/goposix/actions/workflows/ci.yml)
[![go vet](https://img.shields.io/badge/go%20vet-passing-brightgreen)](https://github.com/ramayac/goposix/actions/workflows/ci.yml)
[![coverage](https://img.shields.io/badge/coverage-76.2%25-brightgreen)](https://github.com/ramayac/goposix/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/ramayac/goposix)](https://goreportcard.com/report/github.com/ramayac/goposix)
[![Go Version](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Docker](https://img.shields.io/badge/image-%3C10MB-blue?logo=docker)](https://github.com/ramayac/goposix/pkgs/container/goposix)

**Status: Gold.** All five Gold gaps resolved ([Phase 12](wiki/12_road_to_gold.md)). `awk` is the
Platinum gate ([Phase 07a](wiki/07a_awk.md)). 77 utilities, 548 BusyBox tests passing out of 552 tested (99.3%).

Key Features:
- **Persistent Daemon + Go SDK:** Start one container, call `c.Echo(ctx, "hi")` at 60µs/call.
  11× faster than BusyBox fork+exec for bulk operations ([Performance](wiki/performance.md)).
- **Machine-Readable by Default:** Every utility supports `--json` for structured output
  ([JSON Schema](wiki/json_schema.md)).
- **Portable Scripting:** Sandboxed shell interpreter via `mvdan.cc/sh` with configurable timeout
  and resource limits ([Security Model](wiki/security.md)).
- **High Compatibility:** 99.3% BusyBox test pass rate (548 of 552 tested).
- **CI Gate:** ≥70% overall code coverage enforced on every push (actual: 75.7%).

## Quickstart

### Daemon + Go SDK (recommended)

```bash
# Start the daemon.
./goposix daemon --socket /tmp/goposix.sock &
# Or in Docker:
docker run -d --name goposix ghcr.io/ramayac/goposix:latest
```

```go
package main

import (
    "context"
    "fmt"
    "github.com/ramayac/goposix/pkg/client"
)

func main() {
    c, _ := client.New("/tmp/goposix.sock")  // /var/run/goposix.sock in Docker
    defer c.Close()

    // List files as structured data.
    result, _ := c.Ls(context.Background(), "/", nil)
    for _, entry := range result.Entries {
        fmt.Printf("%s %7d %s\n", entry.Mode, entry.Size, entry.Name)
    }

    // Execute shell scripts.
    out, _ := c.ShellExec(context.Background(), "echo hello from goposix")
    fmt.Print(out.Stdout)
}
```

> **Performance:** 60µs per RPC call with persistent connection — 11× faster than BusyBox
> fork+exec. See [docs/SDK.md](docs/SDK.md) for the full Go SDK guide.

### CLI (secondary)

```bash
# One-shot CLI invocation.
docker pull ghcr.io/ramayac/goposix:cli
docker run --rm ghcr.io/ramayac/goposix:cli ls --json /
```

### Build & Test

```bash
make all          # vet + test + build
make test         # unit tests
make testsuite    # BusyBox integration tests (gates every commit)
make ci           # full pipeline (test + testsuite + coverage + docker)
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `GOPOSIX_SHELL_TIMEOUT` | `30s` | Shell execution timeout (Go duration format, e.g. `60s`, `5m`) |

## Performance Highlights

| Metric | GoPOSIX | BusyBox | Ratio |
|--------|:------:|:------:|:-----:|
| Per-call latency (Go SDK, persistent) | **60µs** | 680µs (fork+exec) | **11× faster** |
| `grep` on 100MB file | **0.16s** | 0.86s | **5.4× faster** (RE2 vs POSIX ERE) |
| Binary size | 10 MB | 800 KB | 12.5× larger |
| Cold start | 7ms | <1ms | Architecture tradeoff |

See [Performance Quick Reference](wiki/performance.md) and [Benchmarking Plan](wiki/19_performance_benchmarking.md) for full details.

## Documentation
- [Go SDK Guide](docs/SDK.md) — typed client for all 77 utilities
- [RPC API Reference](wiki/rpc_api.md)
- [JSON-RPC Quickstart](wiki/rpc_quickstart.md) — raw socket + Go SDK examples
- [Architecture](wiki/architecture.md)
- [Security Model](wiki/security.md)
- [JSON Schema](wiki/json_schema.md)
- [POSIX Coverage Matrix](wiki/posix_coverage.md)
- [Test Coverage Matrix](wiki/test_coverage_matrix.md)
- [POSIX FAQ](wiki/posix_faq.md)
- [Road to Gold](wiki/12_road_to_gold.md)

## Status

**77 POSIX utilities implemented** (100% of target scope excluding `awk`). Gold complete. `awk` deferred to Platinum.

For full details see the [POSIX Compliance Matrix](wiki/posix_coverage.md) and the
[Test Coverage Matrix](wiki/test_coverage_matrix.md) (per-utility breakdown across all suites).

**BusyBox Test Suite:** 548 passed, 4 failed, 10 skipped of 552 total tested (99.3%)

The 4 remaining failures: 3 `date` (Go TZ limitations + cosmetic error format) and 1 `fold`
(NUL handling — echo harness limitation). The 10 skipped tests require external compression tools
(bzip2, xz, uudecode).

## Project Principles

- **Daemon-First:** The default image starts the persistent JSON-RPC daemon. Use the Go SDK for
  programmatic access (60µs/call). CLI is available as a secondary interface (`goposix:cli`).
- **No CGO:** Static compilation for `FROM scratch` containers (`CGO_ENABLED=0`).
- **Near-Zero Dependencies:** Only 3 external Go modules: `mvdan.cc/sh/v3` (shell interpreter),
  `golang.org/x/sys` (cross-platform syscalls), `golang.org/x/term` (terminal detection).
  No external libraries for flag parsing, output, or utility logic.
- **Multicall Binary:** Single binary dispatched via symlink or subcommand (`goposix ls`).
- **`--json` Only:** Structured output via `--json` long flag only — no short-form (`-j`) collision with POSIX flags.
- **POSIX Flag Parsing:** Custom parser in `pkg/common/flags.go` with escape hatches for free-form utilities.
