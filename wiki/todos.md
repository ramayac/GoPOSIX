# GoPOSIX — Open TODOs & Remaining Work

> **Last updated:** 2026-05-22 | **BusyBox:** 679 pass / 20 fail / 22 skip | **Coverage:** 81.5% | **--json:** 81/92 (patch ✅, dd deferred)

## Active Plan: Coverage Improvement → 85%

> **Branch:** `feat/coverage-85` | **Current:** 81.5% | **Target:** 85%

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

### Phase C — Complete ✅ (80.9% → 81.5%)
Needs daemon integration test:

| Package | Uncovered | Effort | Notes |
|---------|:---:|:---:|---|
| `client_helpers` (30+ funcs) | **131 blocks → 5 remaining** | Medium | 27 new helper tests: Dirname, Hostname, Printf, Test, Whoami, Readlink, ID, Date, Uname, Env, Printenv, Sort, Cut, Uniq, Find, Mv, Cp, Ln, Rmdir, Chmod, Md5sum, Sha256sum, Df, Du, Ps, Xargs, Expr. Skipped: Chown/Chgrp (root), Gzip (type mismatch), Tar (cwd), Kill (PID). |

### Phase D — Complete ✅ (81.5% → 82.2%)

| Package | Tests Added | Coverage After | Key additions |
|---------|:---:|:---:|---|
| `sed` | +13 | 69.5% | a/i/c/q/n/N/D/P/T/w commands, SubNum (s/pat/repl/N), w-file, \\ delimiter, $ address, +N range |
| `tar` | +2 | 67.5% | Extract to stdout (-O), verbose listing (-t -v) |
| `cp` | +1 | 78.0% | Symlink copy |
| `date` | +4 | 73.5% | Last-week M.w.d eval, non-leap Julian, complex TZ with DST |
| `printf` | +3 | 79.5% | %c char, %- left-justify, %0 zero-pad float |

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
