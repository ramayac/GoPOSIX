# GoPOSIX

A Go-native, single-binary POSIX with 96.6% BusyBox test compatibility (591/612).

[![CI](https://github.com/ramayac/goposix/actions/workflows/ci.yml/badge.svg)](https://github.com/ramayac/goposix/actions/workflows/ci.yml)
[![go vet](https://img.shields.io/badge/go%20vet-passing-brightgreen)](https://github.com/ramayac/goposix/actions/workflows/ci.yml)
[![coverage](https://img.shields.io/badge/coverage-76.9%25-brightgreen)](https://github.com/ramayac/goposix/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/ramayac/goposix)](https://goreportcard.com/report/github.com/ramayac/goposix)
[![Go Version](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Docker](https://img.shields.io/badge/image-%3C10MB-blue?logo=docker)](https://github.com/ramayac/goposix/pkgs/container/goposix)

## Why?

Well, I wanted to do an experiment on [Harsness Engineering](https://walkinglabs.github.io/learn-harness-engineering/en/), and improve my "agentic development" skills, prompts, instructions and all that.
I did [LFS](https://www.linuxfromscratch.org/) in my early 20's and I had this weird itch of "do your own thing" but left it alone for my own sanity. Still the POSIX concepts remained in the back of my head.
Last year (2025) I started to learn Go-lang, and then LLMs got *really good* in December 2025. Good enough that I've been using it at work non stop since then.

During that time I got this notion that AI waste time formating output, so I started doing --json output in a lot of my work scripts and tools (to save some time for my robot friends).
Eventually all of these random ideas boiled to the conclusion that **I should** make a complete implementation of POSIX utilities in Go, with a JSON output and a Go SDK, and then benchmark it against BusyBox. It's the "natural conclusion" ... right?

Also deepseek-v4-pro had an very agressive [75% discount until 2026/05/31 15:59 UTC](https://api-docs.deepseek.com/quick_start/pricing), and I wanted to try [pi.dev](https://pi.dev) instead of Antigravity/ClaudeCode.

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

## Quickstart!

See [wiki/sdk.md](wiki/sdk.md) for the full Go SDK guide and [wiki/usage.md](wiki/usage.md) for CLI usage and Docker recipes.

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

I think there should be more... right?

## Performance Highlights

| Metric | GoPOSIX | BusyBox | Ratio |
|--------|:------:|:------:|:-----:|
| Per-call latency (Go SDK, persistent) | **60µs** | 680µs (fork+exec) | **11× faster** |
| `grep` on 100MB file | **0.16s** | 0.86s | **5.4× faster** (RE2 vs POSIX ERE) |
| Binary size | 10 MB | 800 KB | 12.5× larger |
| Cold start | 7ms | <1ms | Not bad, but not great |

See [Performance Quick Reference](wiki/performance.md) and [Benchmarking Plan](wiki/19_performance_benchmarking.md) for full details.

## Documentation (yes, we have docs and it's decent!)
- [Go SDK Guide](wiki/sdk.md) — typed client for all 78 utilities
- [RPC API Reference](wiki/rpc_api.md)
- [JSON-RPC Protocol](wiki/rpc_quickstart.md) — raw socket protocol for non-Go clients
- [Architecture](wiki/architecture.md)
- [Security Model](wiki/security.md)
- [JSON Schema](wiki/json_schema.md)
- [Test Coverage & Compliance Matrix](wiki/test_coverage_matrix.md)
- [POSIX FAQ](wiki/posix_faq.md)

## Quick Project Principles

- **Multicall Binary:** Single binary dispatched via symlink or subcommand (`goposix ls`).
- **Daemon-First:** The default image starts the persistent JSON-RPC daemon. Use the Go SDK for
  programmatic access (60µs/call). CLI is available as a secondary interface (`goposix:cli`).
- **No CGO:** Static compilation for `FROM scratch` containers (`CGO_ENABLED=0`).
- **Little Dependencies:** Only 3 external Go modules: `mvdan.cc/sh/v3` (shell interpreter),
  `golang.org/x/sys` (cross-platform syscalls), `golang.org/x/term` (terminal detection).
  No external libraries for flag parsing, output, or utility logic.
- **`--json` Only:** Structured output via `--json` long flag only — no short-form (`-j`) collision with POSIX flags (ouch!)
- **POSIX Flag Parsing:** Custom parser in `pkg/common/flags.go` with escape hatches for free-form utilities.

Does it work? damn right it does: [KoreGoOS](https://github.com/ramayac/KoreGoOS).
