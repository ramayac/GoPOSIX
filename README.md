# GoPOSIX

A Go-native, single-binary POSIX userland with dozens of tools. Runs as a persistent JSON-RPC daemon or multicall CLI, with a typed Go SDK and structured `--json` output on every utility.

[![CI](https://github.com/ramayac/goposix/actions/workflows/ci.yml/badge.svg)](https://github.com/ramayac/goposix/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/ramayac/goposix.svg)](https://pkg.go.dev/github.com/ramayac/goposix)
[![Go Report Card](https://goreportcard.com/badge/github.com/ramayac/goposix)](https://goreportcard.com/report/github.com/ramayac/goposix)
[![codecov](https://codecov.io/gh/ramayac/goposix/graph/badge.svg)](https://codecov.io/gh/ramayac/goposix)
[![Go Version](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Docker](https://img.shields.io/badge/image-%3C10MB-blue?logo=docker)](https://github.com/ramayac/goposix/pkgs/container/goposix)


## Why?

Well, I wanted to do an experiment on [Harsness Engineering](https://walkinglabs.github.io/learn-harness-engineering/en/), and improve my "agentic development" skills, prompts, instructions and all that.
I did [LFS](https://www.linuxfromscratch.org/) in my early 20's and I had this weird itch of "do your own thing" but left it alone for my own sanity. Still the POSIX concepts remained in the back of my head.
Last year (2025) I started to learn Go-lang, and then LLMs got *really good* in December 2025. Good enough that I've been using it at work non stop since then.

During that time I got this notion that AI waste time formating output, so I started doing --json output in a lot of my work scripts and tools (to save some time for my robot friends).
Eventually all of these random ideas boiled to the conclusion that **I should** make a complete implementation of POSIX utilities in Go, with a JSON output and a Go SDK, and then benchmark it against BusyBox. It's the "natural conclusion" ... right?

Also deepseek-v4-pro had an very agressive [75% discount](https://api-docs.deepseek.com/quick_start/pricing)!, and I wanted to try [pi.dev](https://pi.dev) instead of Antigravity/ClaudeCode (I ended up using `agy` for some auditing).

All things kind of aligned in the last month so here we are now.

I'm not the first to start something like this, there is [cugo](https://github.com/jcmdln/cugo) and [go-posix](https://github.com/nirenjan/go-posix), but sadly they seem to be abandoned, and no wonder! A project like this is a huge undertaking, its probably a year of solid work for 1 human, that being said, took about 3 weeks to do with AI, with the proper "harness" and "agentic development" approach, that's really something.

Anyway the project got into a point that _I'm happy_ with the results, that's enough for me ☑️.

## Honest and Obvious Recognitions

I want to be very clear about this:
>  The only reason this works is that there's a brutally thorough, existing corpus of tests to validate against. The AI iterated until it passed. Without BusyBox's tests, this project is just random hallucinated code. **The test suite is the real hero**.

- So check and support [BusyBox](https://busybox.net/) project and take a look at its amazing [test suite](https://github.com/brgl/busybox/blob/master/testsuite/runtest), it's a masterpiece of thoroughness and coverage, and it made this project possible.
- Also [Mvdan Shell](https://github.com/mvdan/sh), it really saved my butt. Absolutely brilliant.
- And [goawk](https://github.com/benhoyt/goawk), which I used for the `awk` implementation, another big save!

Finally: let's not kid ourselves, this project is 90% wiring the AI to do the heavy lifting, 10% is steering it in the right direction, the fact that I was able to "solo dev" this with an LLM, reproducing close to 99% of BusyBox's behavior in a completely different language shows that POSIX utilities are, at their core, text transformers with very well-defined contracts (do one thing and do it well).

## Quickstart

See **[wiki/sdk.md](wiki/sdk.md)** for the full Go SDK guide and **[wiki/usage.md](wiki/usage.md)** for CLI usage and Docker recipes.

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
| `GOPOSIX_SOCKET` | `/var/run/goposix.sock` | Daemon UNIX socket path for CLI forwarding and client SDK connections |
| `GOPOSIX_DEBUG` | (empty) | Set to `1` to enable verbose JSON-RPC request/response debug logging to stderr |
| `GOPOSIX_SHELL_TIMEOUT` | `30s` | Shell execution timeout (Go duration format, e.g. `60s`, `5m`) |
| `GOPOSIX_MAX_REQUEST_SIZE` | `1048576` (1MB) | Max JSON-RPC request size in bytes |
| `GOPOSIX_RATE_LIMIT` | `100` | Max JSON-RPC requests/sec per connection |
| `GOPOSIX_SHUTDOWN_TIMEOUT` | `5s` | Graceful shutdown drain timeout |

#### Standard POSIX Environment Variables

| Variable | Description |
|----------|-------------|
| `TZ` | Standard timezone rule parsed dynamically by `date` and `tar` to format and project timestamps |
| `LOGNAME` | Current login username retrieved by `logname` |
| `PWD` | Logical working directory used by `readlink` to resolve symlinks component-by-component |

## Daemon Stdin

The JSON-RPC daemon accepts a `stdin` field in request params, enabling stdin-consuming utilities (grep, sed, sort, wc, tr, head, tail, cut, tee, uniq, and 30+ others) to receive input directly through the Go SDK without temp files.

```go
// Pass stdin through the daemon
c.Grep(ctx, []string{"foo"}, client.WithStdin("line1\nline2\nfoo\n"))
c.Wc(ctx, []string{"-l"}, client.WithStdin("line1\nline2\nline3\n"))
```

## Performance

| Metric | GoPOSIX | BusyBox |
|--------|:------:|:------:|
| Per-call latency (Go SDK, persistent) | **~60µs** | ~680µs (fork+exec) |
| Large-file grep | **significantly faster** | baseline |
| Binary size | ~10 MB | ~800 KB |
| Cold start | ~7ms | <1ms |

> Numbers above are approximate. For reproducible benchmarks with scale factors and full methodology, see **[wiki/performance.md](wiki/performance.md)**.

## Documentation

- [Go SDK Guide](wiki/sdk.md) — typed client for all utilities
- [RPC API Reference](wiki/rpc_api.md)
- [JSON-RPC Protocol](wiki/rpc_quickstart.md) — raw socket protocol for non-Go clients
- [Architecture](wiki/architecture.md)
- [Security Model](wiki/security.md)
- [JSON Schema](wiki/json_schema.md) — `--json` output schemas for every utility
- [Test Coverage & Compliance Matrix](wiki/test_coverage_matrix.md)
- [POSIX FAQ](wiki/posix_faq.md)
- [Performance Quick Reference](wiki/performance.md)

## Quick Project Principles

- **Multicall Binary:** Single binary dispatched via symlink or subcommand (`goposix ls`).
- **Daemon-First:** The default image starts the persistent JSON-RPC daemon. Use the Go SDK for programmatic access. CLI is available as a secondary interface (`goposix:cli`).
- **No CGO:** Static compilation for `FROM scratch` containers (`CGO_ENABLED=0`).
- **Little Dependencies:** Only 3 external Go modules: `mvdan.cc/sh/v3` (shell interpreter), `golang.org/x/sys` (cross-platform syscalls), `golang.org/x/term` (terminal detection). No external libraries for flag parsing, output, or utility logic.
- **`--json` Only:** Structured output via `--json` long flag only — no short-form (`-j`) collision with POSIX flags.
- **POSIX Flag Parsing:** Custom parser in `pkg/common/flags.go` with escape hatches for free-form utilities (echo, printf, expr).

Does it work? Yes — see how GoPOSIX replaces BusyBox in Alpine: **[docker/Dockerfile](docker/Dockerfile)** (target: `alpine-mvp`).
