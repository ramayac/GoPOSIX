# GoPOSIX — Project Roadmap & State

> **Version:** 5.6 | **Date:** 2026-05-20 | **Tier:** GOLD | **Branch:** `main`
>
> **Status:** 77 utilities | 548 BusyBox passes (99.3%) | 76.7% coverage | 77/77 JSON-RPC
>
> ✅ Phase 24 (Hardening IV Part 1) — 21 compliance gaps resolved (branch `feat/hardening4-part1`): injectable streams for 11 utilities, resolved sed regex issues, 13+ POSIX `date` format specifiers, `grep` binary detection, `ls -d` flag, `PreProcess` flag hook, 5 more resolved. 6 HIGH-priority items remain (H1–H6).

---

## Current State

| Metric | Value |
|--------|-------|
| BusyBox pass rate | 548 passed / 4 failed / 10 skipped (99.3%) |
| Overall test coverage | 76.7% |
| Utilities implemented | 77 |
| JSON-RPC daemon coverage | 77/77 utilities |
| Daemon unit coverage | 64.6% |
| Supply chain security | SBOM + Cosign + SLSA L3 + Trivy |
| Shell security model | Documented + tested (GOPOSIX_SHELL_TIMEOUT, SecurePath, 128MB limits) |

### Architecture

```
goposix binary (single static ELF, <12MB)
├── Multicall Dispatch (os.Args[0] or subcommand)
├── CLI Wrappers (--json flag → JSON envelope)
├── Daemon Mode (JSON-RPC 2.0 over Unix socket)
└── pkg/ Libraries (return typed Go structs)
    ├── pkg/echo/     → EchoResult
    ├── pkg/ls/       → []FileInfo
    ├── pkg/grep/     → []MatchResult
    └── pkg/common/   → JSON-RPC types, flag parser, output helpers
```

### Utility Tiers (All Complete)

| Tier | Utilities | Phase |
|------|-----------|-------|
| **1 — Trivial** | `echo`, `true`, `false`, `yes`, `whoami`, `hostname`, `uname`, `pwd`, `printenv`, `env` | 01 ✅ |
| **2 — Filesystem** | `ls`, `cat`, `mkdir`, `rmdir`, `rm`, `cp`, `mv`, `touch`, `ln`, `stat`, `readlink`, `basename`, `dirname` | 03 ✅ |
| **3 — Text** | `head`, `tail`, `wc`, `sort`, `uniq`, `tr`, `cut`, `tee`, `grep`, `sed` | 04 ✅ |
| **4 — System** | `ps`, `kill`, `sleep`, `date`, `id`, `groups`, `chmod`, `chown`, `chgrp`, `df`, `du`, `find`, `xargs` | 06 ✅ |
| **5 — Advanced** | `tar`, `gzip`, `sha256sum`, `md5sum`, `diff`, `patch`, `test`/`[`, `printf`, `expr` | 07 ✅ |
| **Post-MVP** | `dd`, `od`, `patch` (`egrep`, `fgrep` aliases) | 15/18 ✅ |
| **Platinum** | `awk` | 07a ✅ |

### Technical Specs

| Spec | Value |
|------|-------|
| Language | Go 1.22+ (Pure Go, `CGO_ENABLED=0`) |
| Protocol | JSON-RPC 2.0 over Unix Domain Sockets |
| Base Image | `scratch` (prod), `alpine` (debug) |
| Key Dep | `mvdan.cc/sh/v3` (shell interpreter) |
| Binary Target | < 12 MB stripped |
| Image Target | < 10 MB |
| Daemon Latency | < 1ms |

---

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
| 13 | Coverage & hardening (76.7% reached) | ✅ |
| 14a-c | JSON gap fill, BusyBox regression fix, JSON-RPC daemon coverage | ✅ |
| 15–18 | Post-MVP utilities, quality fixes | ✅ |
| 19 | Performance benchmarking | ✅ |
| 20 | Hardening II — flag audit, code cleanup, coverage, input safety | ✅ |
| 22 | Hardening III — Daemon-First Pivot | ✅ |
| 23 | Flags Rewrite — zero-allocation POSIX scanner | ✅ |
| 24 | Hardening IV Part 1 — 21 compliance gaps resolved, 6 remain | 🟡 |
| 07a | `awk` — Platinum gate (integrated goawk v1.31.0) | ✅ |

---

## Active Work

| # | Item | Doc |
|---|------|-----|
| 🔴 | Hardening IV — 6 remaining HIGH-priority gaps (H1–H6) | [24_hardening_iv.md](24_hardening_iv.md) — branch `feat/hardening4-part1` |
| ✅ | Hardening IV Part 1 — 21 gaps resolved (injectable streams, date specifiers, grep binary detection, ls -d, PreProcess hook, +15 more) | [24_hardening_iv.md](24_hardening_iv.md) |
| 🔧 | Observability exports (A–D done, E external) | [observability_exports.md](observability_exports.md) — branch `feat/observability` |
| — | Multi-agent observability | [observability_exports.md#part-2](observability_exports.md) — deferred discussion |
| — | Ongoing maintenance | [todos.md](todos.md), [deferred.md](deferred.md) |
