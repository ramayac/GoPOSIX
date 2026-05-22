# GoPOSIX — Open TODOs & Remaining Work

> **Last updated:** 2026-05-22 | **BusyBox:** 679 pass / 20 fail / 22 skip | **Coverage:** 77.9% | **--json:** 81/92 (patch ✅, dd deferred)

## Active Plan: Coverage Improvement (Phase A & B → 85%)

> **Branch:** `feat/coverage-85` | **Current:** 80.9% | **Target:** 85%

### Phase A — Complete ✅ (80.1% → 80.7%)
Low-effort, pure-computation or simple test patterns:

| Package | Tests Before | Tests After | Key additions |
|---------|:---:|:---:|---|
| `xargs` | 2 | **12** | `-n` max-args, `-t` trace, `-I` replace-str, `-E` eof-str, `-s` max-chars, default echo, bad flag, JSON, command failure, no-input-runs-once |
| `paste` | 13 | **18** | JSON mode, file-not-found, bad flag, stdin default, trailing backslash in delimiters |
| `join` | 8 | **13** | `-t` custom delimiter, JSON, `-1`/`-2` field spec, stdin, bad flag |
| `tr` | 5 | **11** | `-s` squeeze, `-c` complement, CLI bad flag, missing operand |
| `hostname` | 5 | **8** | `-d` domain, JSON, bad flag |

### Phase B — Complete ✅ (80.7% → 80.9%)
Mid-effort, needs temp files/dirs:

| Package | Key additions |
|---------|--------------|
| `mkdir` | `-m` mode flag, missing operand, JSON |
| `mv` | `-t` target-dir, missing operand, JSON |
| `cp` | recursive directory copy (`-r`) |
| `diff` | recursive non-regular file, missing-only-in-one, both missing |
| `comm` | `-1`/`-2`/`-3` suppress columns, `--total` flag, JSON, bad flag |

### Phase C — Future (target: ~83-84%)
Needs daemon integration test:

| Package | Uncovered | Effort | Notes |
|---------|:---:|:---:|---|
| `client_helpers` (30+ funcs) | **131 blocks** | Medium | All follow same `callUtility` template. Test daemon already exists in `forwarder_test.go:startTestDaemon()`. Batch-test all helpers against it. |

### Phase D — Future (target: ~85%)
Complex but impactful:

| Function | Coverage | Uncovered | Effort | Notes |
|----------|:---:|:---:|:---:|---|
| `sed/execFlat` | 36.7% | 25 blocks | High | Massive switch statement for `P`/`D`/`n`/`N`/`q`/`h`/`H`/`g`/`G`/`x`/`y`/`:`/`b`/`t` |
| `tar/doExtract+doList` | 43-59% | 20 blocks | Medium | Extract to stdout, overwrite, exclude, verbose listing |
| `cp/copyDir` | 53.1% | 12 blocks | Medium | Symlink-preserving, nested dirs |
| `date/eval+parsePOSIXTZ` | 52-66% | 15 blocks | Medium | More M.w.d format variations, Julian day rollover |
| `printf` | 79.0% | ~10 blocks | Low | Remaining escape sequences in format strings |

### Skipped (platform-specific / needs subprocess)

| Function | Why |
|----------|-----|
| `main()` | `os.Exit()` |
| `setProcTitle` | Modifies argv memory, Linux-only |
| `RunDaemon` | Full daemon lifecycle |
| `interactive()` | REPL with `os.Stdin` |
| `ttyname()` | `IoctlGetTermios` syscall |

## Remaining Failures (20)

### `awk` — 17 failures (goawk v1.31.0 limitations)

| # | Test | Root Cause |
|---|------|------------|
| 1 | `awk bitwise op` | goawk doesn't implement bitwise operators |
| 2 | `awk properly handles undefined function` | goawk parse error on undefined functions |
| 3 | `awk unused function args are evaluated` | goawk evaluation order difference |
| 4 | `awk hex const 1` | goawk doesn't support hex constants |
| 5 | `awk hex const 2` | Same |
| 6 | `awk oct const` | goawk doesn't support octal constants |
| 7 | `awk handles non-existing file correctly` | goawk error handling difference |
| 8 | `awk nested loops with the same variable` | goawk scoping difference |
| 9–12 | `awk func arg parsing 1–4` | goawk function argument parsing |
| 13 | `awk handles empty ()` | goawk empty arg list handling |
| 14 | `awk break` | goawk break statement |
| 15 | `awk continue` | goawk continue statement |
| 16 | `awk negative field access` | goawk negative field access |
| 17 | `awk backslash+newline` | goawk line continuation handling |

### `realpath` — 3 failures (canonical path resolution limits in symlinked workspace)

| # | Test | Root Cause |
|---|------|------------|
| 18 | `realpath on non-existent local file 1` | Path canonicalization behavior difference on non-existent paths |
| 19 | `realpath on link to non-existent file 1` | Path canonicalization behavior difference on non-existent paths |
| 20 | `realpath on link to non-existent file 3` | Path canonicalization behavior difference on non-existent paths |

## Planned & Deferred Work

All active planning phases, deferred architectural enhancements, completed transitions, and engine limitations are consolidated in a single central registry:

👉 **[wiki/deferred.md](deferred.md)**

Refer to that document for full details.

### Alpine Daemon Mode

| # | Item | Status |
|---|------|--------|
| — | Daemon-in-Alpine: `alpine-mvp` image runs CLI-only (shell). Adding daemon mode requires entrypoint change + user setup + BusyBox override decision. | PLANNING — see [alpine_plan.md § Daemon Mode](alpine_plan.md#daemon-mode-in-alpine) |
