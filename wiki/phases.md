# GoPOSIX — Project Roadmap & State

> **Version:** 7.0 | **Date:** 2026-05-30 | **Tier:** GOLD | **Branch:** `feat/hardening_v`
>
> **Status:** 115 utilities | 877 BusyBox passes (98.1%) | 84.1% coverage | 115/115 JSON-RPC
>
> ✅ All phases complete. 115 utilities implemented, JSON-RPC daemon at 100%, BusyBox at 98.1% (only 17 awk failures remain, deferred — upstream goawk engine limitations).

---

## Current State

| Metric | Value |
|--------|-------|
| BusyBox pass rate | 877 passed / 17 failed / 25 skipped (98.1%) |
| Overall test coverage | 84.1% |
| Utilities implemented | 115 |
| JSON-RPC daemon coverage | 115/115 utilities (100%) |
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
| **6 — Phase 26/27** | `which`, `realpath`, `seq`, `sha1sum`, `sha512sum`, `rev`, `uptime`, `wget`, `cal`, `hostid`, `factor`, `sha3sum`, `tree`, `tsort`, `pidof`, `bunzip2`, `bzcat`, `unlzma`, `uncompress`, `unzip`, `uuencode`, `uudecode`, `taskset`, `start-stop-daemon`, `cryptpw`, `makedevs`, `ar`, `cpio`, `ash`, `mount`, `mdev`, `dc`, `rx`, `hexdump`, `xxd`, `bc`, `mkfs.minix` | 26/27 ✅ |
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
| 00 | Foundation | completed |
| 01 | Tier 1 utilities | completed |
| 02 | Docker CI & `scratch` pipeline | completed |
| 03 | Filesystem utils (ls, cat, rm, cp, mv, ...) | completed |
| 04 | Text utils (grep, sed, sort, wc, ...) | completed |
| 05 | JSON-RPC daemon core | completed |
| 06 | System & process utils (ps, find, df, du, ...) | completed |
| 07 | Agent-ready features (diff, tar, shell) | completed |
| 08 | Security hardening | completed |
| 09 | Release & automation | completed |
| 10 | POSIX test framework + BusyBox suite | completed |
| 11 | Post-MVP cleanup, lessons learned | completed |
| 12 | Road to Gold — supply chain, macOS, coverage, BusyBox parity | completed |
| 13 | Coverage & hardening (76.7% reached) | completed |
| 14 | JSON gap fill, BusyBox regression fix, JSON-RPC daemon coverage | completed |
| 15–18 | Post-MVP utilities (dd, od, patch, expand, comm, paste, fold, sum, nl, cmp, strings, which, realpath, pidof, seq, cal, hostid, factor, uptime, wget, rev, tree, tsort, sha*) — consolidated into [wiki/post_mvp.md](post_mvp.md) | completed |
| 19 | Performance benchmarking | completed |
| 20 | Hardening II — flag audit, code cleanup, coverage, input safety | completed |
| 21 | Honest-takes audit | completed |
| 22 | Hardening III — Daemon-First Pivot | completed |
| 23 | Flags Rewrite — zero-allocation POSIX scanner | completed |
| 24 | Hardening IV Part 1 — 21 compliance gaps resolved, 6 remain | completed |
| 25 | Awesome-Go Prep & Daemon Stdin — checklists, Codecov & LICENSE, 25 lints fixed | completed |
| 07a | `awk` — Platinum gate (integrated goawk v1.31.0) | completed |
| 26 | Missing Tools Tier 1–4 — 26 utilities, 100% BusyBox compatibility | completed |
| 27 | High-Complexity Tier 5 — all 11 implemented | completed |
| 28 | High-Complexity Tier 5 planning | completed |
| 29 | High-Complexity Tier 5 execution | completed |
| 30 | Performance improvements (12/30 done) | completed |
| 31 | Hardening V — Coverage & Tar Compliance Audit (12 packages to 80%+, overall 84.1%) | completed |


---

## Active Work

See [todos.md](todos.md) for the canonical registry of remaining work, and [deferred.md](deferred.md) for planning-phase architectural work.
