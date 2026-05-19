# Phase 20 — Hardening II (Post-Gold Audit)

> **Status:** COMPLETED | **Date:** 2026-05-18 | **Final coverage:** 76.7% | **BusyBox:** 548/4/10
>
> Full-architecture audit. Resolved all CRITICAL and HIGH items. Score: 87 → 95/100.

---

## Issue Catalog

### 🔴 20.1 — Short flag `-j` used for `--json` (51 utilities) ✅ FIXED

**Violated:** AGENTS.md proscription against `-j` short flag. Real collisions: `tar -j` (bzip2).

**Fix:** Removed `Short: "j"` from 51 FlagDefs, replaced `flags.Has("j")` → `flags.Has("json")` in 33 run() functions. ~155 LOC, 85 files.

### 🔴 20.2 — Production debug code in sed.go ✅ FIXED

Debug trap (`if three { ... }`) active in production. 7 lines deleted.

### 🔴 20.3 — Output injection via `fmt.Printf`/`fmt.Println` ✅ FIXED

`whoami`, `uname`, `stat`, `basename`, `dirname` wrote to `os.Stdout` instead of injected `out io.Writer`. Fixed to `fmt.Fprintf(out, ...)`. `ls` `printLong()` now accepts `out io.Writer`.

### 🔴 20.4 — `Run()` hardcodes `os.Stdout` ✅ FIXED

Added `RunWithWriter()` to `goposix.go` — `Run()` delegates to it. Enables proper output capture in tests.

### 🔴 20.5 — `rm` no `--no-preserve-root` guard ✅ FIXED

Implemented POSIX `--no-preserve-root` flag. `rm -rf /` is refused without it.

### 🟡 20.6–20.12 — Various medium/low issues ✅ FIXED

Documentation drift (ARCHITECTURE.md frozen at Phase 10), missing CONTRIBUTING.md,
cover-gate race condition, shell sandbox fallback decision, `x/sys` dep documented,
`--json` Only wording in README, multi-arch build fragility. All resolved.

### 🟡 20.13 — 17 packages below 70% unit coverage ✅ PARTIALLY FIXED

9 packages brought above 70% gate. 8 remain (hard-to-test paths: net.Dial, terminal I/O, complex parsers). Overall coverage: 75.7% → 76.7%.

### 🟡 20.14 — JSON-RPC daemon test gaps ✅ 3 OF 4 FIXED

Added daemon integration tests for `truefalse`, `tee`, `testcmd`. `daemon` self-test remains (circular dependency).

---

## Results

| Phase | Scope | Status |
|-------|-------|--------|
| 20a | Flag fix: remove `-j` from 51 utilities | ✅ |
| 20b | Code cleanup: debug code, `--no-preserve-root`, output injection | ✅ |
| 20c | Coverage hardening: 9 packages above 70%, 3 daemon tests added | ✅ |
| 20d | Documentation: fix drift, add CONTRIBUTING.md, fix cover-gate | ✅ |
| 20e | Input safety: buffer limits on grep, sort, head, tail | ✅ |

**Verification:** `make test` passes. `make testsuite` 548/4 (unchanged). Zero `Short: "j"` remaining in pkg/.
