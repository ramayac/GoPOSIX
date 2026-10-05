# GoPOSIX

A Go-native, single-binary POSIX userland with 115 tools. Runs as a persistent JSON-RPC daemon or multicall CLI, with ~98.2% BusyBox test compatibility (871 of 917 tests pass).

[![CI](https://github.com/ramayac/goposix/actions/workflows/ci.yml/badge.svg)](https://github.com/ramayac/goposix/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/ramayac/goposix.svg)](https://pkg.go.dev/github.com/ramayac/goposix)
[![golangci-lint](https://github.com/ramayac/goposix/actions/workflows/lint.yml/badge.svg)](https://golangci-lint.run/)
[![codecov](https://codecov.io/gh/ramayac/goposix/graph/badge.svg)](https://codecov.io/gh/ramayac/goposix)
[![Go Version](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Docker](https://img.shields.io/badge/image-%3C10MB-blue?logo=docker)](https://github.com/ramayac/goposix/pkgs/container/goposix)

## Does it work?

Yes! yes it does! see how GoPOSIX replaces BusyBox in Alpine here: **[docker/Dockerfile](docker/Dockerfile)** (target: `alpine-mvp`).

### Why?

Check out **[HISTORY.md](HISTORY.md)** for the story behind GoPOSIX, the projects that made it possible, and the reasons why it exists.

## Quickstart

See **[wiki/rpc_quickstart.md](wiki/rpc_quickstart.md)** for the JSON-RPC protocol and **[wiki/usage.md](wiki/usage.md)** for CLI usage and Docker recipes.

### CLI (secondary)

```bash
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

#### Daemon & CLI Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `GOPOSIX_SOCKET` | `/var/run/goposix.sock` | Daemon UNIX socket path for CLI forwarding and JSON-RPC client connections |
| `GOPOSIX_DEBUG` | (empty) | Set to `1` to enable verbose JSON-RPC request/response debug logging to stderr |
| `GOPOSIX_SHELL_TIMEOUT` | `30s` | Shell execution timeout (Go duration format, e.g. `60s`, `5m`) |
| `GOPOSIX_MAX_REQUEST_SIZE` | `1048576` (1MB) | Max JSON-RPC request size in bytes |
| `GOPOSIX_RATE_LIMIT` | `100` | Max JSON-RPC requests/sec per connection |
| `GOPOSIX_SHUTDOWN_TIMEOUT` | `5s` | Graceful shutdown drain timeout |
| `GOPOSIX_LS_CACHE_TTL` | `30s` | Time-to-Live for `ls` owner/group name string translation caching (Go duration format, e.g., `30s`, `1m`) |


#### Standard POSIX Environment Variables

| Variable | Description |
|----------|-------------|
| `TZ` | Standard timezone rule parsed dynamically by `date` and `tar` to format and project timestamps |
| `LOGNAME` | Current login username retrieved by `logname` |
| `PWD` | Logical working directory used by `readlink` to resolve symlinks component-by-component |

## Daemon Stdin

The JSON-RPC daemon accepts a `stdin` field in request params, enabling stdin-consuming utilities (grep, sed, sort, wc, tr, head, tail, cut, tee, uniq, and 30+ others) to receive input directly through JSON-RPC without temp files.

```json
{"jsonrpc":"2.0","method":"goposix.grep","params":{"flags":["foo"],"stdin":"line1\nline2\nfoo\n"},"id":1}
{"jsonrpc":"2.0","method":"goposix.wc","params":{"flags":["-l"],"stdin":"line1\nline2\nline3\n"},"id":2}
```

Every command response also includes a `stderr` field with human-readable error text. `rawOutput` mode returns `stdout` and `stderr` as raw text.

## Performance

| Metric | GoPOSIX | BusyBox |
|--------|:------:|:------:|
| Per-call latency (JSON-RPC, persistent) | **~60µs** | ~680µs (fork+exec) |
| Large-file grep | **significantly faster** | baseline |
| Binary size | ~10 MB | ~800 KB |
| Cold start | ~7ms | <1ms |

> Numbers above are approximate. For reproducible benchmarks with scale factors and full methodology, see **[wiki/performance.md](wiki/performance.md)**.

## Documentation

- [JSON-RPC Protocol](wiki/rpc_quickstart.md) — socket protocol for clients
- [Architecture](wiki/architecture.md)
- [Security Model](wiki/security.md)
- [JSON Schema](wiki/json_schema.md) — `--json` output schemas for every utility
- [Test Coverage & Compliance Matrix](wiki/test_coverage_matrix.md) — 871/917 (98.2%) BusyBox pass rate
- [POSIX Command Audit Plan](wiki/28_posix_command_audit.md) — Phase 28 audit plan, matrix, and remaining work
- [POSIX FAQ](wiki/posix_faq.md)
- [Performance Quick Reference](wiki/performance.md)
- [History](HISTORY.md) — why GoPOSIX started and the projects that made it possible

## Quick Project Principles

- **Multicall Binary:** Single binary dispatched via symlink or subcommand (`goposix ls`).
- **Daemon-First:** The default image starts the persistent JSON-RPC daemon. Use the JSON-RPC API for programmatic access. CLI is available as a secondary interface (`goposix:cli`).
- **No CGO:** Static compilation for `FROM scratch` containers (`CGO_ENABLED=0`).
- **Little Dependencies:** 12 external Go modules — 10 direct and 2 indirect. Direct: `mvdan.cc/sh/v3` (shell), `github.com/benhoyt/goawk` (`awk`), `github.com/blakesmith/ar` (`ar`), `github.com/cavaliergopher/cpio` (`cpio`), `github.com/hotei/dcompress` (`uncompress`), `github.com/sergeymakinen/go-crypt` and `github.com/tredoe/crypt` (`cryptpw`), `github.com/ulikunitz/xz` (`tar`, `unlzma`), `golang.org/x/sys` (syscalls), `golang.org/x/crypto` (`sha3sum`). Indirect: `github.com/hotei/mdr`, `golang.org/x/term`. No external libraries for flag parsing, output, or utility logic.
- **`--json` Only:** Structured output via `--json` long flag only — no short-form (`-j`) collision with POSIX flags.
- **POSIX Flag Parsing:** Custom parser in `pkg/common/flags.go` with escape hatches for free-form utilities (echo, printf, expr).

