# KoreGo

A Go-native, single-binary POSIX userland (99.4% BusyBox test compatibility). KoreGo replaces
GNU Coreutils in Docker `FROM scratch` containers, featuring structured `--json` output in
every utility and a persistent JSON-RPC daemon to eliminate process-spawning overhead.

**Status: Gold.** All five Gold gaps resolved ([Phase 12](wiki/12_road_to_gold.md)). `awk` is the
Platinum gate ([Phase 07a](wiki/07a_awk.md)).

Key Features:
- **Machine-Readable by Default:** Every utility supports `--json` for structured output
  ([JSON Schema](docs/JSON_SCHEMA.md)). `--xml` is in progress ([Phase 14](wiki/14_xml_output.md)).
- **Low-Overhead Execution:** A persistent JSON-RPC 2.0 daemon with session management
  ([RPC API](docs/RPC_API.md)).
- **Portable Scripting:** Sandboxed shell interpreter via `mvdan.cc/sh` with configurable timeout
  and resource limits ([Security Model](docs/SECURITY.md)).
- **High Compatibility:** 99.4% BusyBox test pass rate (477/490 tests).
- **CI Gate:** ≥45% overall code coverage enforced on every push (actual: 70.5%).

## Quickstart

### Docker
```bash
docker pull ghcr.io/ramayac/korego:latest
docker run --rm ghcr.io/ramayac/korego:latest ls --json /
```

### Build from Source
```bash
make all
./korego --list-commands
```

### Run Tests
```bash
make test          # unit tests
make testsuite     # BusyBox integration tests (gates every commit)
make ci            # full pipeline (test + testsuite + coverage + docker)
```

### Start Daemon
```bash
./korego daemon --socket /tmp/korego.sock &
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `KOREGO_SHELL_TIMEOUT` | `30s` | Shell execution timeout (Go duration format, e.g. `60s`, `5m`) |

## Documentation
- [Architecture](docs/ARCHITECTURE.md)
- [JSON Schema](docs/JSON_SCHEMA.md)
- [RPC API](docs/RPC_API.md)
- [Agent Integration Guide](docs/AGENT_INTEGRATION.md)
- [Security Model](docs/SECURITY.md)
- [POSIX Coverage Matrix](wiki/posix_coverage.md)
- [POSIX FAQ](wiki/posix_faq.md)
- [Road to Gold](wiki/12_road_to_gold.md)

## Status

**56 POSIX utilities implemented** (100% of target scope excluding `awk`). Gold complete. `awk` deferred to Platinum.

**BusyBox Test Suite:** 477 passed, 3 failed, 10 skipped (99.4% effective pass rate)

All 3 remaining failures are in `date` (2 Go POSIX TZ limitations, 1 cosmetic error-format mismatch).
The 10 skipped tests require external tools (bzip2, xz, uudecode) or PAX extended header support.

### Implemented Utilities

| Category | Package | Utilities |
|----------|---------|-----------|
| Core & Env | 12 | echo, env, pwd, true, false, whoami, hostname, basename, dirname, yes, printenv, uname |
| Filesystem | 14 | ls, cat, mkdir, rmdir, rm, cp, mv, touch, ln, stat, readlink, chmod, chown, chgrp |
| Text Processing | 10 | head, tail, wc, sort, uniq, tr, cut, tee, grep, sed |
| System & Process | 9 | ps, kill, sleep, date, id, df, du, find, xargs |
| Pipeline / Format | 7 | printf, expr, test, md5sum, sha256sum, gunzip, gzip |
| Agent / Daemon | 4 | diff, tar, shell, daemon |

### Phase Status

| Phase | Status | Description |
|-------|--------|-------------|
| 00–10 | ✅ Complete | Foundation through POSIX Framework |
| 11–11a | ✅ Complete | Post-MVP: JSON schemas, client library, agent example |
| 12 | ✅ Complete | Road to Gold — all 5 gaps resolved |
| 13 | ✅ Complete | Coverage ramp (50%→70.5%) + hardening |
| 14 | ⏳ Deferred | XML output support ([plan](wiki/14_xml_output.md)) |
| Platinum | ⏳ Deferred | `awk` implementation ([plan](wiki/07a_awk.md)) |

## Project Principles

- **No CGO:** Static compilation for `FROM scratch` containers (`CGO_ENABLED=0`).
- **Zero Dependencies:** No external Go modules for flag parsing, output, or utility logic.
- **Multicall Binary:** Single binary dispatched via symlink or subcommand (`korego ls`).
- **`--json` Only:** Structured output via `--json` long flag only — no short-form collision with POSIX flags.
- **POSIX Flag Parsing:** Custom parser in `pkg/common/flags.go` with escape hatches for free-form utilities.
