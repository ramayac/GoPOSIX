# GoPOSIX

A Go-native, single-binary POSIX userland (99.3% BusyBox test compatibility). GoPOSIX replaces
GNU Coreutils in Docker `FROM scratch` containers, featuring structured `--json` output in
every utility and a persistent JSON-RPC daemon to eliminate process-spawning overhead.

**Status: Gold.** All five Gold gaps resolved ([Phase 12](wiki/12_road_to_gold.md)). `awk` is the
Platinum gate ([Phase 07a](wiki/07a_awk.md)). 77 utilities, 547/541 BusyBox tests passing (99.1%).

Key Features:
- **Machine-Readable by Default:** Every utility supports `--json` for structured output
  ([JSON Schema](docs/JSON_SCHEMA.md)). `--xml` is in progress ([Phase 14](wiki/14_xml_output.md)).
- **Low-Overhead Execution:** A persistent JSON-RPC 2.0 daemon with session management
  ([RPC API](docs/RPC_API.md)).
- **Portable Scripting:** Sandboxed shell interpreter via `mvdan.cc/sh` with configurable timeout
  and resource limits ([Security Model](docs/SECURITY.md)).
- **High Compatibility:** 99.1% BusyBox test pass rate (547/541 tests).
- **CI Gate:** ≥70% overall code coverage enforced on every push (actual: ~72%).

## Quickstart

### Docker
```bash
docker pull ghcr.io/ramayac/goposix:latest
docker run --rm ghcr.io/ramayac/goposix:latest ls --json /
```

### Build from Source
```bash
make all
./goposix --list-commands
```

### Run Tests
```bash
make test          # unit tests
make testsuite     # BusyBox integration tests (gates every commit)
make ci            # full pipeline (test + testsuite + coverage + docker)
```

### Start Daemon
```bash
./goposix daemon --socket /tmp/goposix.sock &
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `GOPOSIX_SHELL_TIMEOUT` | `30s` | Shell execution timeout (Go duration format, e.g. `60s`, `5m`) |

## Documentation
- [Architecture](docs/ARCHITECTURE.md)
- [JSON Schema](docs/JSON_SCHEMA.md)
- [RPC API](docs/RPC_API.md)
- [Agent Integration Guide](docs/AGENT_INTEGRATION.md)
- [Security Model](docs/SECURITY.md)
- [POSIX Coverage Matrix](wiki/posix_coverage.md)
- [Test Coverage Matrix](wiki/test_coverage_matrix.md)
- [POSIX FAQ](wiki/posix_faq.md)
- [Road to Gold](wiki/12_road_to_gold.md)

## Status

**77 POSIX utilities implemented** (100% of target scope excluding `awk`). Gold complete. `awk` deferred to Platinum.

For full details see the [POSIX Compliance Matrix](wiki/posix_coverage.md) and the
[Test Coverage Matrix](wiki/test_coverage_matrix.md) (per-utility breakdown across all suites).

**BusyBox Test Suite:** 548 passed, 4 failed, 10 skipped of 541 total (99.3%)

The 4 remaining failures: 3 `date` (Go TZ limitations + cosmetic error format) and 1 `fold`
(NUL handling — echo harness limitation). The 10 skipped tests require external compression tools
(bzip2, xz, uudecode).

## Project Principles

- **No CGO:** Static compilation for `FROM scratch` containers (`CGO_ENABLED=0`).
- **Zero Dependencies:** No external Go modules for flag parsing, output, or utility logic.
- **Multicall Binary:** Single binary dispatched via symlink or subcommand (`goposix ls`).
- **`--json` Only:** Structured output via `--json` long flag only — no short-form collision with POSIX flags.
- **POSIX Flag Parsing:** Custom parser in `pkg/common/flags.go` with escape hatches for free-form utilities.
