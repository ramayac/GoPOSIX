# Phase 13 — Coverage & Hardening (Audit → Action)

> **Status:** In Progress | **Date:** 2026-05-16 | **Supersedes:** 13_code_audit.md, 15_coverage_ramp.md
>
> **Recent progress:**
> - Phase 14a (JSON gap fill — 8 utilities) **COMPLETED** (2026-05-15). [14a_json_gap_fill.md](14a_json_gap_fill.md)
> - Phase 14b (BusyBox regression fix — 79→3 failures) **COMPLETED** (2026-05-15). [14b_busybox_regression_fix.md](14b_busybox_regression_fix.md)
> - Phase 14c (JSON-RPC daemon coverage — 9/55→55/55 utilities, +46 tests) **COMPLETED** (2026-05-16). [14c_posix_json_gap.md](14c_posix_json_gap.md)
>
> **2026-05-16:** Stage 2 hardening underway on `feat/hardening-13-stage1-stage2`.
>
> **cat: 37.6% → 88.7%** ✅ (extracted `catRun()` injectable entry point, +18 CLI tests covering `-n`/`-b`/`-s`/`-e`/`-v`/`--json`/long flags/multi-file/stdin/file-not-found/empty input/bad flag)
>
> **2026-05-15:** Batch 1 (grep + head hardening) **COMPLETED** — overall 51.6% → **55.3%**.
> grep: 16.5% → **85.3%** (+context flags `-A`/`-B`/`-C`, `-H`/`-h`, `-R`, +29 tests).
> head: 29.0% → **94.1%** (3 bugs fixed, +22 tests). See commits on `feat/13-coverage`.

---

## Part A — Code Audit Summary

Code-level evidence gathered during the live-repository audit. Cross-referenced
against wiki claims and the [Gold roadmap](12_road_to_gold.md).

**Overall coverage at time of audit: 41.6%** (`go test -coverprofile` across `./pkg/...` `./internal/...`)

### Gap Summary

| # | Gap | Status |
|---|-----|--------|
| 13.0 | macOS build breakage (`uname`, `stat`, `client`) | ✅ Fixed |
| 13.1 | No supply chain security (SBOM, Cosign, SLSA, Trivy) | ✅ Fixed |
| 13.2 | Shell security model undocumented, untested | ✅ Fixed |
| 13.3 | Coverage gate was informational only | ✅ Fixed (enforced at ≥45%) |
| 13.4 | BusyBox CI/local discrepancy (testing system BusyBox, not KoreGo) | ✅ Fixed |
| 13.5 | `awk` not implemented (Platinum gate) | ⏳ Open — [07a_awk.md](07a_awk.md) |

Details for each gap are recorded in the [Gold roadmap](12_road_to_gold.md)
(12.0–12.5). This document focuses on the **action plan** going forward.

---

## Part B — Focus Areas

Post-Gold, work concentrates on five pillars:

| Pillar | Target | Primary Doc |
|--------|--------|-------------|
| **Coverage** | 75% overall (from ~50%) | This document |
| **POSIX** | 99%+ BusyBox pass rate, zero regressions | [posix_coverage.md](posix_coverage.md) |
| **Security** | Hardened shell, supply chain, SBOM, secured defaults | [08_hardening.md](08_hardening.md), [docs/SECURITY.md](../docs/SECURITY.md) |
| **Speed** | <1ms daemon latency, <15MB binary, <100ms CLI startup | This document |
| **Docker** | Usable `FROM scratch` image, smoke-tested, signed | [09_release_docs.md](09_release_docs.md) |

---

## Part C — Coverage Ramp: 50% → 75%

### Current State

| Metric | Current | Target Stage 1 | Target Stage 2 | Target Stage 3 |
|--------|---------|---------------|---------------|---------------|
| Overall coverage | **68.5%** | 60% | 68% | 75% |
| Packages at 0% | 1 (`cmd/korego`) | 0 | 0 | 0 |
| Packages at <10% | 1 (`pkg/daemon`) | 0 | 0 | 0 |
| Packages at <30% | 4 (`tail`, `touch`, `chmod`, `internal/daemon`) | 2 | 0 | 0 |
| Packages at ≥80% | 20 | 25 | 30 | 35 |

> Updated 2026-05-16 after Phase 14c completion. Coverage rose from 55.3% → 57.9%
> primarily from find + uniq bugfix tests and posix-json integration test daemon coverage
> in `internal/daemon` (19.1%, up from 3.3% audit baseline).

### Root Cause

Every package follows the same arch: a public `Run()` (library layer, testable with
`io.Reader`/`io.Writer`) and a private `run()` (CLI glue that calls
`common.ParseFlags()`, opens real files, binds `os.Stdin`/`os.Stdout`). Tests
overwhelmingly cover `Run()` but skip `run()` entirely.

```
grep:          354 total LOC, 240 in run() →  86.3%  ✅ (refactored: grepRun injectable)
head:          181 total LOC, 130 in run() →  94.1%  ✅ (headInput struct, 3 bugs fixed)
internal/daemon: 613 total LOC             →  19.1%  (up from 3.3%; posix-json tests)
tee:            66 total LOC,  44 in run() →  72.5%  (improved since audit; was 3.3%)
```

### Stage 1 — Critical Foundations (50% → 60%)

#### 1.1 `internal/daemon` (19.1% → 35%)

**Why first:** 613 LoC. Handles all RPC dispatch, Unix socket lifecycle, session
management, rate limiting, observability. This is the backbone of daemon mode.
Already improved from 3.3%→19.1% via posix-json integration tests (Phase 14c).

- [ ] `server_test.go` — `processRequest` error paths, `handleBatch` edge cases, graceful shutdown
- [ ] `session_test.go` — lifecycle: Create/Get/SetCwd/List/Destroy, TTL expiry
- [ ] `ratelimit_test.go` — verify existing, cover `Allow()` edge cases

**Est. gain:** +3.2% overall, ~400 test LOC.

#### 1.2 `cmd/korego` (0% → 50%)

- [ ] `main_test.go` — Extract `runMain(args []string) error`. Test: symlink dispatch,
  subcommand mode, `--list-commands`, `--help`, `--version`, missing subcommand.

**Est. gain:** +0.5% overall, ~80 test LOC.

#### 1.3 `pkg/client` (45% → 60%)

- [ ] `client_helpers_test.go` — Integration with real in-process Unix socket.
  `Cat()`, `Echo()`, `Pwd()`, `Stat()`, `Ls()`, `Ping()`, `Dial()` error path,
  `Close()`, `WithTimeout()`, `WithMaxRetries()`.

**Est. gain:** +1.5% overall, ~250 test LOC.

#### 1.4 `pkg/daemon` (6% → 40%)

- [ ] `daemon_test.go` — Socket creation, SIGTERM/SIGINT graceful shutdown,
  `--socket` flag path.

**Est. gain:** +0.3% overall, ~60 test LOC.

---

### Stage 2 — Utility CLI Layer (60% → 68%)

Close the `run()` gap via dispatch-call tests using `testdata/` fixtures.

**Pattern:**

```go
func TestRunViaDispatch(t *testing.T) {
    cmd := dispatch.Lookup("grep")
    var out bytes.Buffer
    exitCode := cmd.Run([]string{"-c", "Alice", "testdata/alice.txt"}, &out)
}
```

#### High-Impact Utilities

| Utility | Current | Target | Key tests |
|---------|---------|--------|-----------|
| `grep` | ~~16.5%~~ **86.3%** ✅ | 45% | `-e`, `-f`, `-v`, `-r`, `-c`, `-l`/`-L`, `-i`, `-w`, `-x`, `-A`/`-B`/`-C`, `-H`/`-h`, `-R` |
| `sed` | 49.1% | 65% | `s/foo/bar/`, `-n`, `d`, `-e`, `-f`, `-i`, address ranges, `y` |
| `cat` | ~~37.6%~~ **88.7%** ✅ | 55% | `-n`, `-b`, `-s`, `-v`, `-E`, `-A`, stdin `-` |
| `find` | 50.0% | 65% | `-name`, `-type`, `-size`, `-exec`, `-print`, `-delete`, `-maxdepth` |
| `tar` | 43.5% | 60% | `-c`, `-x`, `-t`, `-v`, `-f`, `-C` |
| `sort` | 58.0% | 70% | `-n`, `-r`, `-u`, `-k`, `-o`, `-t` |

#### Medium-Impact Utilities

| Utility | Current | Target | Key tests |
|---------|---------|--------|-----------|
| `ls` | 45.8% | 60% | `-l`, `-a`, `-R`, `-h`, `-1` |
| `wc` | 32.5% | 55% | `-l`, `-w`, `-c`, `-m`, multi-file, stdin |
| `head` | ~~29.0%~~ **94.1%** ✅ | 55% | `-n` pos/neg, `-c`, multi-file headers, `--json`, `--lines`/`--bytes` long flags |
| `tail` | 27.6% | 55% | `-n`, `-c`, `-f` (timeout), multi-file headers |
| `cp` | 49.6% | 65% | file→file, file→dir, dir→dir `-R`, `-a`, error: ENOENT |
| `rm` | 35.3% | 55% | single, `-r`, `-f`, `--no-preserve-root`, error: ENOENT |
| `stat` | 35.9% | 55% | `-c` format, `-f` filesystem, symlink deref |
| `touch` | 20.0% | 50% | create, update, `-r`, `-t`, error: bad format |
| `mv` | 31.8% | 50% | rename, cross-device, dir target, error: ENOENT |

#### Low-Hanging Fruit

| Utility | Current | Target | Key tests |
|---------|---------|--------|-----------|
| `tee` | 3.3% | 50% | stdin→stdout+file, `-a` append, multi-file |
| `pwd` | 30.4% | 60% | default, `-L`, `-P` |
| `mkdir` | 32.4% | 60% | single, `-p`, `-m` mode |
| `rmdir` | 44.4% | 65% | single, `-p`, error: not-empty |
| `readlink` | 34.4% | 55% | default, `-f`, `-e`, `-m` |
| `xargs` | 53.3% | 65% | `-n`, `-I`, `-t`, stdin piped |
| `gzip` | 54.8% | 65% | `-c`, `-d`, `-1`..`-9` |
| `printenv` | 44.4% | 60% | all vars, single var, missing var |
| `echo` | 56.9% | 70% | `-n`, `\\t`, `\\0NNN`, `\\xNN`, `\\c` |

---

### Stage 3 — Refactor & Harden (68% → 75%)

- [x] **`pkg/grep/`**: Extract `run(args, stdout, stderr io.Writer, stdin io.Reader) int` ✅ `grepRun()`
- [x] **`pkg/tar/`**: Extract create/extract/list into testable functions ✅ `createArchiveStream()`, `extractArchiveStream()`, `listArchiveStream()`
- [ ] **`pkg/sed/`**: Extract `compileScripts()` and `runEngine()` from file I/O ✅ `compileAst()` already extracted; engine is testable via temp files — refactoring deferred (low ROI, existing structure adequate)
- [ ] **`pkg/find/`**: Extract `parsePredicates()` as pure function ✅ existing `getMaxDepth()`/`buildExecArgs()` are pure; `run()` parses inline — refactoring deferred (no testability gap)
- [x] **Tooling**: Add `make cover-pkg` target; CI gate at 60% hard-fail ✅ `make cover-pkg` and `make cover-gate` added; `make ci` includes `cover-gate`
- [x] **Docs**: Coverage policy in AGENTS.md ✅ Section 4a added

---

## Part D — Speed Targets

| Metric | Current | Target | How |
|--------|---------|--------|-----|
| Binary size | ~15 MB | <12 MB | `-ldflags="-s -w"`, UPX, dead code elimination |
| Docker image | ~17 MB | <10 MB | `COPY --from=build /out/bin /bin` only |
| Daemon latency | <5ms | <1ms | Keep hot path allocations minimal; `sync.Pool` for buffers |
| CLI startup | ~10ms | <5ms | Avoid `init()` heavy lifting; lazy flag parsing |
| `go test -race` | Pass | Pass (maintain) | All tests must pass race detector |

---

## Part E — Task Summary

| Stage | Packages | New Tests | Est. LOC | Cumulative Coverage |
|-------|----------|-----------|----------|--------------------|
| **Current** | — | — | — | **57.9%** |
| ⚡ grep + head hardening | 2 | 51 | ~900 | 55.3% ✅ |
| 🌐 Phase 14c daemon tests | 46 | 46 | ~2,100 | 57.9% ✅ |
| 1.1 `internal/daemon` | 1 | ~20 | ~300 | 59.5% |
| 1.2 `cmd/korego` | 1 | ~8 | ~80 | 60.0% |
| 1.3 `pkg/client` | 1 | ~15 | ~250 | 61.5% |
| 1.4 `pkg/daemon` | 1 | ~4 | ~60 | 61.8% |
| — | — | — | — | **~60%** |
| 2.1 High-impact | 6 | ~50 | ~800 | 65.0% |
| 2.2 Medium-impact | 9 | ~35 | ~500 | 67.8% |
| 2.3 Low-hanging | 9 | ~20 | ~300 | 68.5% |
| — | — | — | — | **~68%** |
| 3.1 Refactors | 4 | ~30 | ~400 | 72.0% |
| 3.2 Tooling/Docs | — | — | — | — |
| — | — | — | — | **~75%** |

---

## Verification

```bash
make cover-pct          # 57.9% (current); target ≥60% (Stage 1), ≥68% (Stage 2), ≥75% (Stage 3)
make cover-pkg          # no package below 5% / 25% / 40% / 55%
make ci                 # coverage gate hard-fails below threshold
make testsuite          # 477 passed, 3 failed (date), 0 regressions
make smoke-docker       # docker run korego ls -la works
go test ./test/posix-json/...  # 55/55 utilities JSON-RPC daemon coverage (Phase 14c ✅)
```
