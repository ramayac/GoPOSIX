# Wiki Log

> **Note:** References to "agent," "agentic," or "AI agent" in historical entries below predate the Phase 21 honest-takes audit (2026-05-18). The project's positioning has been corrected to "programmatic consumer" / "JSON-RPC client."

Append-only timeline of wiki maintenance activity.

## [2026-05-30] feat | tar: all 7 BusyBox failures resolved — 31/31 (100%), coverage 80.4%, XZ (branch `feat/tar-fixes`)

Resolved all 7 remaining tar BusyBox test failures across 4 areas:

- **Symlink safety (3 tests):** Pre-scan archive for entry name conflicts.
  `isSymlinkSafe()` checks if a symlink target escapes the extraction root;
  `hasConflict()` determines if other archive entries depend on the symlink.
  Dangerous symlinks are refused only when dependents exist (non-conflicting
  absolute symlinks like CA cert bundles are allowed). Added `hasDangerousParent()`
  to track child entries of refused symlinks.

- **Hardlink dedup for symlinks (1 test):** `createArchiveStream` now checks
  `seenInodes` for symlink entries, producing `TypeLink` hardlink entries for
  duplicate inodes instead of new `TypeSymlink` entries. Tests: `tar symlinks mode`.

- **ls output fixes (2 tests):** Reordered `ls.Run()` to process non-directory
  args before directory args (GNU/POSIX convention). Fixed symlink mode character
  from Go's uppercase `L` to lowercase `l`. Fixed error message format to strip
  Go's `lstat` prefix and capitalize (matches BusyBox/GNU `ls`). Tests: `tar
  hardlinks and repeated files`, `tar hardlinks mode`.

- **XZ compression auto-detect (1 test):** Added XZ magic bytes detection
  (`0xFD 7zXZ 0x00`) in both `doExtract` and `doList`. Uses `github.com/ulikunitz/xz`
  v0.5.15 for LZMA2 decompression. Test: `tar extract txz`.

**Coverage:** tar 72.4% → 80.4% (+11 new tests: XZ compress/extract/list,
symlink edge cases, extract to stdout, overwrite, include list, gzip list,
verbose create, stdin list, resolveTarPath, JSON list, error paths).
Added `test/compliance/test_tar.sh` (5 JSON schema assertions).

**BusyBox:** 873→877 pass, 21→17 fail (all awk, deferred), 97.7%→98.1%.
Overall coverage: 83.4%→83.7%.

Updated: `pkg/tar/tar.go`, `pkg/tar/tar_test.go`, `pkg/ls/ls.go`,
`test/compliance/test_tar.sh`, `go.mod`, `go.sum`, `wiki/todos.md`,
`wiki/test_coverage_matrix.md`, `README.md`, `wiki/log.md`.

## [2026-05-28] implement | Resolved `rx` XMODEM flakiness and buffered `hexdump` input

Traced the intermittent `rx` BusyBox test failure to GoPOSIX's `hexdump` utility prematurely flushing partial pipe reads instead of buffering blocks. Extended `rx` unit tests and coverage to 86.2%.

**Changes implemented:**
- **Hexdump input buffering**: Replaced standard `input.Read()` in `pkg/hexdump/hexdump.go` with `io.ReadFull()`. Programmed `hexdump` to accumulate complete blocks up to the defined `blockSize` (e.g. 16 bytes for canonical `-C` mode) before printing, matching POSIX/GNU conventions and preventing lines from being split on partial reads.
- **Graceful EOF handling**: Configured `hexdump` to safely handle `io.ErrUnexpectedEOF` to format and print final partial blocks before gracefully breaking the loop.
- **XMODEM test hardening**: Expanded `pkg/rx/rx_test.go` with comprehensive test coverage including duplicate blocks, invalid inverse block numbers, unexpected block numbers, loops cancellation, write errors, and handshake failures. Raised statement coverage from 72.4% to 86.2%.
- **Verification**: Verified 100% stable pass rate on `runtest rx` (20/20 loop iterations pass).

## [2026-05-28] implement | Hardened `dc` utility, resolved all remaining BusyBox failures

Resolved all remaining 7 BusyBox test suite failures for the `dc` (desk calculator) utility. Added unit tests reaching 87.8% statement coverage.

**Changes implemented:**
- **Recursive stack overflow crash resolved**: Fixed `Z` (length) and `S` (store stack) commands to correctly pop values off the main stack, preventing infinite recursion during recursive macro execution (fixes `dc_strings.dc` failure).
- **Scale-aware modulus and divmod**: Refactored `%` and `~` operators to be scale-aware under BusyBox conventions (`a - (a / b) * b` evaluated under current scale), resolving scale propagation discrepancies in 0k mode (fixes `dc_modulus.dc` and `dc_divmod.dc` failures).
- **Decimal truncation**: Implemented scale-aware decimal truncation helper `truncateRat` after arithmetic operations (`*`, `/`, `%`, `~`, `^`, `v`) to avoid cascading high-precision fractions and match BusyBox's math engine.
- **Formatting cleanup**: Aligned mathematical zero formatting under BusyBox conventions to format zero as bare `"0"` regardless of global/number scale, except when pushed from explicit non-zero literals (fixes zero-value precision in `dc_power.dc` and `dc_multiply.dc`).
- **Extended registers (`-x` / `extendedReg`)**: Implemented full support for multi-character extended register names (e.g. `s xotj`, `l yotp`). Added identifier character matching (`isIdentChar`) and register parser (`parseRegName`) supporting space and command-terminated extended register names (fixes `dcx_vars.dc` failure).
- **Unit tests**: Added comprehensive unit tests in `pkg/dc/dc_test.go` covering extended registers and runtime flags, with package coverage reaching 87.8%.
- **Verification**: Verified 100% compliance rate on the BusyBox test suite (`runtest dc`) with all 36/36 tests passing, and 100% passing rate on GoPOSIX unit tests.

## [2026-05-22] cleanup | Wiki dedup + README de-staling (branch `docs/wiki-cleanup-2`)

**Duplicate elimination:** Deleted `.wiki-instructions/wiki-maintainer.instructions.md`
(identical to `wiki-maintainer.md` — both 2273 bytes, same content). Kept
`wiki-maintainer.md` as canonical.

**Alpine consolidation:** Replaced `wiki/alpine_integration.md` (marketing-style
welcome page) with a stub redirecting to `wiki/alpine_plan.md` (technical
blueprint). The integration page had ~80% overlap with the plan.

**README de-staling (14 hardcoded values removed):**
- Replaced "92+ tools" / "97.1% (679/699)" with descriptive language.
- Removed the performance table with hardcoded µs/seconds/sizes — replaced
  with approximate ranges and a link to `wiki/performance.md` for reproducible
  benchmarks.
- Stripped phase references ("Phase 25") from Daemon Stdin section.
- Kept all CI badges (they auto-update).
- Trimmed "Why?" section without losing the project's voice.

**Clarity:** Added disambiguation note to `wiki/schema.md` (wiki structure
contract vs. `wiki/json_schema.md` output schemas). Added Alpine links to
`wiki/index.md`.

Updated: `README.md`, `wiki/alpine_integration.md`, `wiki/index.md`,
`wiki/schema.md`, `wiki/log.md`. Deleted:
`.wiki-instructions/wiki-maintainer.instructions.md`.

## [2026-05-22] ingest | Phase 26 complete + Phase 27 partial (branch `feat/missing-tools-tier4`)

Absorbed 31 new utilities (Phase 26 Tiers 1-4: 26 tools; Phase 27 Tier 5: 5 tools).

**Files ingested:**
- `wiki/26_missing_tools.md` — marked Tier 4 complete, updated counts
- `wiki/27_high_complexity_tools.md` — marked 5 implemented: `ar`, `cpio`, `ash`, `mount`, `mdev`
- `wiki/test_coverage_matrix.md` — added Tier 8 section (16 tools), updated to 115 utils, 729/19/53 BusyBox, 106/115 JSON-RPC, 82.9% coverage
- `wiki/todos.md` — updated project state, added cpio/pidof limitations, 28 compliance tests missing
- `wiki/phases.md` — added Phase 26 (✅) and Phase 27 (🔨), bumped version to 6.0
- `wiki/index.md` — added link to `27_high_complexity_tools.md`
- `wiki/11_lessons_learned.md` — added Phase 26/27 lessons

**Code changes absorbed:**
- `test/posix-json/tier8_phase26_27_test.go` — 30 new daemon tests (24 running, 6 skipped)
- `cmd/goposix/main_test.go` — added `ash` to cmdPkgMapping (CI fix)
- `pkg/cpio/cpio.go` — fixed stdout leak in JSON list mode (was printing filenames before JSON, corrupting daemon output)
- `test/compliance/test_ar.sh`, `test/compliance/test_ash.sh` — fixed path/exit code bugs
- 31 new `pkg/<tool>/` packages at >= 80% coverage

**BusyBox delta:** +50 pass (679→729), -1 fail (20→19), +31 skip (22→53). 2 new cpio failures (block count from cavaliergopher/cpio).

**JSON-RPC delta:** +31 tests (82→106/115), 9 remaining gaps (6 hard-skipped: ash `--json` conflict, wget network, daemon recursive, mount/mdev/makedevs root).

## [2026-05-21] cleanup | Prune todos.md — removed all resolved sections

Stripped `wiki/todos.md` of resolved cruft (~130 lines → ~40 lines). Removed:
Hardening IV (resolved), Phase 25 Daemon Stdin (resolved), date/fold
failures (resolved), JSON-RPC daemon gaps (resolved), CWD signature
refactoring (resolved). Kept: BusyBox awk failures (17, goawk-limited),
Alpine daemon mode (planning), deferred.md link.

Updated: `wiki/todos.md`, `wiki/log.md`.

## [2026-05-21] plan | Alpine daemon mode analysis + CI binary-size gate

Documented what it takes to run GoPOSIX as a daemon inside the Alpine MVP
image. Two changes needed: entrypoint swap (shell → daemon) + user setup
(addgroup/adduser, or use root + /tmp socket). Discussed BusyBox override
tradeoff: keep it for pure-Go experiments, drop it for practical Alpine +
GoPOSIX JSON-RPC coexistence. Added to `alpine_plan.md` and `todos.md`.

Removed the image-size gate from `.github/workflows/ci.yml` (it was building
the wrong target — `alpine-mvp` at 26 MB, not `daemon` at 10 MB). Replaced
with a binary-size gate (<15 MB) that checks the compiled `goposix` binary
directly. Reordered Dockerfile targets so `daemon` is last (default).

Updated: `wiki/alpine_plan.md`, `wiki/todos.md`, `wiki/log.md`,
`.github/workflows/ci.yml`, `docker/Dockerfile`.

## [2026-05-20] sync | JSON schema gap fill — 30 new schemas, doc updated, patch --json

Audited `wiki/json_schema.md` against `test/schemas/` and `pkg/`:
- Found the doc claimed 42 utilities support `--json` — actual count is 76.
- The doc's "without --json" list (sleep, tee, tr, yes, sed, true, false) was
  stale: all 7 had gained `--json` flags since the doc was last updated.
- 23 additional utilities had `--json` support with no schema file and no doc
  table entry: awk, cksum, cmp, comm, expand, fold, join, link, logger, logname,
  mkfifo, nice, nl, nohup, od, paste, split, strings, sum, tty, unexpand, unlink, who.

Created 30 new JSON Schema (draft-07) files in `test/schemas/` covering all
missing utilities. Updated `wiki/json_schema.md`: count 42→76, 46→76 table rows,
replaced stale "without --json" list with "Only dd does not yet support --json."
Cross-check: 76 doc rows ↔ 76 schema files ↔ 76 `--json` flags.

**`patch --json` implemented same session:** Added `--json` flag to flag spec,
wired `common.Render`/`RenderError` into `patchRun()` for success, hunk-failure
(data preserved), and parse/IO-error paths. 4 new CLI tests (JSON success, hunk
failure, stdin error, bad patch file). Coverage: 78.0%, race-clean. `dd` deferred
(manual operand parsing makes flag injection non-trivial; ~30–60 min).

Created: 30 `test/schemas/*.schema.json` files (awk, cksum, cmp, comm, expand,
false, fold, join, link, logger, logname, mkfifo, nice, nl, nohup, od, paste,
sed, sleep, split, strings, sum, tee, tr, true, tty, unexpand, unlink, who, yes).
Updated: `pkg/patch/patch.go`, `pkg/patch/patch_test.go`, `wiki/json_schema.md`,
`wiki/todos.md`, `wiki/log.md`.

## [2026-05-20] fix | H6 — Shell os.Chdir() thread-safety + Makefile test-race target (branch: `feat/hardeing-iv-partii`)

Resolved H6 (shell os.Chdir not thread-safe):
- Added `execMu sync.Mutex` serializing all `Exec()` calls. Process CWD saved at
  entry and restored on exit — `cd` side-effects no longer leak between sequential
  calls. The daemon tracks CWD per-session via the `cwd` parameter, not via process
  state, so restoring after each Exec is correct.
- Updated 3 tests to verify new behavior: `TestCdAndPwd` (CWD restored after),
  `TestCdPersistsAcrossExecCalls` (cd does NOT leak without explicit cwd),
  `TestCdWithExplicitCwd` (CWD restored after explicit cwd).
- Added `TestConcurrentShellExec`: 5 goroutines × 100 cd+pwd iterations, passes
  under `-race` with zero failures.
- Added `make test-race` target to Makefile — runs all unit tests with Go race
  detector. Documented in help output and Makefile comments (~10x slower, dev-only).
- Documented future work in interpreter.go: eliminate `os.Chdir` entirely by
  threading a CWD parameter through `dispatch.Command.Run`.
- Updated `wiki/24_hardening_iv.md`: H6 → RESOLVED. Score: 3 remaining / 24 resolved.

Updated: `internal/shell/interpreter.go`, `internal/shell/interpreter_test.go`,
`Makefile`, `wiki/24_hardening_iv.md`, `wiki/log.md`, `wiki/todos.md`.

## [2026-05-20] fix | H3 — Session data race (branch: `feat/hardeing-iv-partii`)

Resolved H3 (session data race on concurrent access):
- `Get()` and `List()` now return deep copies via `Session.copy()` — CWD string
  copied, Env map cloned before mutex release. All callers receive independent
  snapshots with no shared references.
- Fixed pre-existing `close of closed channel` panic in `SessionManager.Stop()`
  — made idempotent with select/default pattern.
- Added `session_test.go` with `TestSessionManagerConcurrentRace` — 4 data races
  detected by `go test -race` before fix: string field write vs read, map
  write vs read (×2), and realistic SetCwd races. Zero races after fix.
  All daemon tests pass under `-race`.
- Updated `wiki/24_hardening_iv.md`: H3 → RESOLVED. Score: 4 remaining / 23 resolved.

Updated: `internal/daemon/session.go`, `internal/daemon/session_test.go`,
`wiki/24_hardening_iv.md`, `wiki/log.md`.
Created: `internal/daemon/session_test.go`.

## [2026-05-20] fix | H2 — SecurePath symlink resolution (branch: `feat/hardeing-iv-partii`)

Resolved H2 (SecurePath does not resolve symlinks):
- Added `resolveSymlinks()` helper in `pkg/common/security.go` that calls
  `filepath.EvalSymlinks` on the target path. For paths that do not exist yet
  (new file creation), walks up to the deepest existing parent, resolves its
  symlinks, and appends the non-existent tail — catching escape symlinks in
  intermediate directories.
- `SecurePath` now resolves symlinks on the base directory before the prefix
  comparison, and on the target path for the traversal check.
- Added `TestSecurePathSymlinks` with 10 test cases covering: normal paths,
  symlinks inside base, symlinks outside base, non-existent files through escape
  symlinks, deep non-existent paths, and resolved-path verification.
- Updated `wiki/security.md`: symlinks limitation → symlinks resolved.
- Updated `wiki/24_hardening_iv.md`: H2 → RESOLVED. Score: 5 remaining / 22 resolved.

Updated: `pkg/common/security.go`, `pkg/common/security_test.go`,
`wiki/security.md`, `wiki/24_hardening_iv.md`, `wiki/log.md`.

## [2026-05-20] ingest | Hardening IV Part 1 — injectable streams, compliance gap resolution (branch: `feat/hardening4-part1`)

Three commits merged into `feat/hardening4-part1` for PR #21:

**Commit `4bf739d` — Code readability & maintainability:** 57 files (+1237/-431).
Formatting/spacing cleanup. Resolved sed regex and state management issues.
Implemented 13+ missing `date` POSIX format specifiers (`%j`, `%p`, `%r`, `%u`, `%V`,
`%W`, `%U`, `%n`, `%t`, `%D`, `%F`, `%R`, `%w`, `%k`, `%l`). Added grep binary file
detection (NUL scan + `-a`/`--text` override). Enhanced JSON output in `sleep`,
`strings`, `sum`. Hardened CLI test error handling across packages.

**Commit `3f5130f` — Custom error output & input streams:** 17 files (+263/-149).
Extracted injectable `*_Run()` entry points for 11 utilities (`gzip`, `head`, `ls`,
`sed`, `sort`, `tar`, `tee`, `tr`, `uniq`, `xargs`, `cut`, `find`) — accepting
`errW io.Writer` + `stdin io.Reader` (partially addresses H4). Added `PreProcess`
hook to `common.FlagSpec` for `tar`/`find` argument preprocessing (resolves M8).
Implemented `ls -d` / `--directory` flag (resolves M2).

**Commit `5eeefae` — Compliance gap status update:** Wiki only (+7/-7).
Marked H7, M2, M8 as RESOLVED. Header updated from "9 remaining, 18 resolved"
to "6 remaining, 21 resolved."

**Wiki maintenance:** Fixed stale summary table and recommended fix order in
`wiki/24_hardening_iv.md` (table showed 18/9, now shows 21/6). Updated
`wiki/phases.md` to add Hardening IV Part 1 to active work. Fixed log entry.

Updated: `wiki/24_hardening_iv.md`, `wiki/phases.md`, `wiki/log.md`.

## [2026-05-19] implement | Observability exports — Options A, B, C, D (branch: `feat/observability`)

Implemented four observability options on a shared feature branch:

**Option A — Thread naming:** `internal/daemon/threadname_linux.go` (and darwin stub).
Worker goroutines locked to OS threads and named `goposix/wrk-NN` via
`unix.Prctl(PR_SET_NAME)`. Visible in `htop` / `top -H`.

**Option B — Process name in `ps`:** `internal/daemon/proctitle_linux.go` (and darwin stub).
Original argv+env memory area discovered at init via `unsafe.StringData`, overwritten
with live status every 5s. Title: `goposix daemon [W:3/4 S:12 C:500K]` in `ps aux`.

**Option C — Go runtime stats in Prometheus `/metrics`:** Extended `handleMetrics` in
`internal/daemon/observability.go` with 11 new OpenMetrics series: goroutines,
gomaxprocs, num_cpu, heap_alloc_bytes, heap_sys_bytes, stack_inuse_bytes, mallocs_total,
frees_total, total_alloc_bytes, num_gc_cycles, gc_pause_ns. Pure stdlib, ~50 lines.

**Option D — JSON `/status` endpoint:** Added `handleStatus` handler with full
`StatusSnapshot` struct covering pid, uptime, goroutines, heap, GC, workers, sessions,
per-method aggregates, and per-session details. Registered at `/status` in the HTTP mux.
Registered `/status` in the HTTP mux.

**Shutdown fix (pre-existing bug):** `Server.Stop()` now tracks active connections in
`conns map[net.Conn]struct{}` and closes them before `connWG.Wait()`, preventing
hang when `handleConn` is blocked in `dec.Decode()`.

**Wiki consolidation:** Merged `wiki/24_multi_agent_observability.md` into
`wiki/observability_exports.md` (Part 1: Infrastructure Exports, Part 2: Multi-Agent
Observability). Deleted standalone 24 page. Option E (`gotop`) marked as EXTERNAL
(separate repo). Phase 24 marked DEFERRED DISCUSSION.

Updated: `wiki/observability_exports.md`, `wiki/index.md`, `wiki/log.md`,
`internal/daemon/observability.go`, `internal/daemon/server.go`,
`internal/daemon/session.go`.
Created: `internal/daemon/threadname_linux.go`, `internal/daemon/threadname_darwin.go`,
`internal/daemon/proctitle_linux.go`, `internal/daemon/proctitle_darwin.go`.
Deleted: `wiki/24_multi_agent_observability.md`.

## [2026-05-19] plan | Daemon observability exports (`wiki/observability_exports.md`)

Created a comprehensive options doc covering 7 approaches (A–G) for exposing daemon
internals (goroutines, memory, sessions, per-method throughput) to OS tools and
external consumers. Currently PLANNING — no implementation.

Recommended path (if approved):
- Phase 1: Go runtime stats in Prometheus `/metrics` (30 min)
- Phase 2: JSON `/status` endpoint (50 min)
- Phase 3: `gotop` TUI via `pkg/gotop/` (1–3 hr)
- Phase 4: Process name in `ps` (15 min, bonus)

Rejected/deferred: thread naming (fragile), cgroups v2 (privilege barrier), eBPF (overkill).

## [2026-05-19] migrate | Move docs/ → wiki/ (branch: `docs/cleanup`)

Moved the last two remaining docs/ files into wiki/:
- `docs/SDK.md` → `wiki/sdk.md`
- `docs/SHELL_INTEGRATION.md` → `wiki/shell_integration.md`

Original files replaced with stubs. Cross-references updated in:
- `README.md`
- `wiki/rpc_quickstart.md` (3 links → internal `sdk.md`)
- `wiki/repo-map.md` (docs/ entry updated)
- `wiki/performance.md` (fixed broken `../docs/ARCHITECTURE.md` → `architecture.md`)
- `wiki/index.md` (SDK & API section, added `sdk.md` and `shell_integration.md`)

Historical references in `wiki/22_hardening_iii.md` left as-is (they document
Phase 22 milestones when the docs/ path was accurate).

## [2026-05-19] cleanup | Wiki consolidation — 48 files → 26 files (branch: `feat/clean-up-wiki`)

Comprehensive wiki cleanup to reduce bloat and improve navigability:

- **Tier 1 — Deleted 17 historical phase docs (00–10a, 11/11a/12/13):** Removed detailed
  implementation checklists for phases completed months ago. Phase summaries live in
  `phases.md`. Architectural knowledge lives in `architecture.md`.
- **Tier 2 — Merged 10 files into 3:** `14a+14b+14c` → `14_post_mvp_fixes.md`,
  `15+16+17+18` → `15_post_mvp_utilities.md`, `posix_coverage.md` → merged into
  `test_coverage_matrix.md`.
- **Tier 3 — Deduplicated living docs:** `index.md` → pure link index (was status-table-heavy).
  `phases.md` → removed architecture diagram (→ `architecture.md`), risk register (stale),
  giant phase-files table (redundant). `todos.md` → removed current-state table
  (→ `phases.md`). `architecture.md` → removed phase history table (→ `phases.md`).
- **Tier 4 — Trimmed heavy docs:** `19_performance_benchmarking.md` 771→200 lines.
  `20_hardening_ii.md` 684→250 lines. Keep results, trim planning rationale.
- **Tier 5 — Deferred reorg:** Created `deferred.md` consolidating all deferred items.
  Folded `14_xml_output.md` and `23_multi_tenant_sandbox.md` into it. Kept canonical
  designs: `07a_awk.md`, `24_multi_agent_observability.md`.

Deleted: goposixos.md, 00_foundation_libs.md–10a_sed.md, 11_post_mvp_priorities.md,
11a_lower_priority.md, 12_road_to_gold.md, 13_coverage_and_hardening.md,
14a_json_gap_fill.md, 14b_busybox_regression_fix.md, 14c_posix_json_gap.md,
14_xml_output.md, 15_post_mvp_tier1.md, 16_post_mvp_tier2.md, 17_post_mvp_tier3.md,
18_quality_fixes.md, 23_multi_tenant_sandbox.md, posix_coverage.md.

Created: deferred.md, 14_post_mvp_fixes.md, 15_post_mvp_utilities.md.
Updated: index.md, phases.md, todos.md, architecture.md, test_coverage_matrix.md,
19_performance_benchmarking.md, 20_hardening_ii.md, log.md.

Result: 48 files → 26 files, ~60% less text, zero information loss.

---

## [2026-05-19] ingest | Self-upgrade, Docker image shrink, Phase 23 deferred, Phase 24 planned, examples removed (branch: `feat/upgrade`)

Self-upgrade implemented:
- `goposix --version` (prints injected version), `goposix --upgrade` (GitHub releases API,
  tar.gz extraction, atomic binary replacement) — `upgrade.go`, `upgrade_test.go`
- Zero external dependencies, stdlib only (`net/http`, `archive/tar`, `compress/gzip`)
- Wired into dispatcher in `goposix.go` alongside `--help`, `--version`, `--list-commands`

Docker image size fix (28 MB → 9.7 MB):
- Switched daemon Dockerfile from `FROM alpine:3.20` + `strace file` to `FROM scratch`
- Added `# syntax=docker/dockerfile:1` + `COPY --chown=1000:1000` for directory ownership
- Daemon server now calls `os.MkdirAll(filepath.Dir(socketPath), 0700)` before binding
- Socket path: `/home/goposix/goposix.sock` (writable scratch directory via `--chown`)
- Removed `docker/Dockerfile.daemon` (Dockerfile IS the daemon now)
- Created `docker/Dockerfile.cli` for CLI-only image
- Updated `Makefile` daemon-image target, bench-daemon socket path

Phase restructuring:
- Phase 23 (Multi-Tenant Sandbox): DEFERRED — workspace isolation breaks shared-repo multi-agent model
- Phase 24 (Multi-Agent Observability): PLANNING — agent-aware sessions, audit trail, per-agent metrics
- Both documented with rationale in `wiki/23_multi_tenant_sandbox.md`, `wiki/24_multi_agent_observability.md`

Examples removed — `examples/docker-compose.yml` (broken: scratch has no sh/nc, stale socket path),
`examples/rpc_client/main.go` (stale, doesn't use Go SDK). Replaced by `wiki/usage.md`:
- CLI mode, daemon mode, Docker Compose, Go SDK, raw JSON-RPC, smart forwarding, recipes

Updated: upgrade.go, upgrade_test.go, goposix.go, internal/daemon/server.go,
docker/Dockerfile, docker/Dockerfile.cli, docker/Dockerfile.goreleaser, Makefile,
wiki/23_multi_tenant_sandbox.md, wiki/24_multi_agent_observability.md,
wiki/self_upgrade.md, wiki/usage.md, wiki/architecture.md, wiki/index.md, wiki/log.md

## [2026-05-19] doc | Filled `wiki/repo-map.md`

Populated the repo-map template with: top-level file roles, high-signal directory
map (cmd, pkg, internal, docker, test, wiki, docs), generated artifacts, build/run
commands (from Makefile), .wikirc ignored paths, and the 8 architectural invariants
(from AGENTS.md).

## [2026-05-19] ingest | Phase 22 complete + Phase 23 plan + benchmark results (branch: `feat/hardening-iii`)

Phase 22 (Daemon-First Pivot) completed:
- `docker/Dockerfile` → daemon default, `docker/Dockerfile.cli` → CLI-only
- GoReleaser builds daemon as primary (`Dockerfile.goreleaser.daemon`), CLI as secondary
- README: SDK quickstart first, benchmark numbers (60µs/call, 10.9×, 5.1× grep)
- `docs/SDK.md` — comprehensive Go SDK guide with typed method reference
- M5 forwarder (`forwarder.go`) exists but not yet wired into `main.go` (deferred)

Benchmark suite hardened (Phase 19):
- Nanosecond timing (Cat A), xargs-P4 parallel ops (Cat B/D), Go SDK bench_client
- Cat F: 3-mode comparison (socat vs Go SDK vs BusyBox) — SDK is 10.9× faster
- Cat J: Go SDK rpc-loop mode — 5 typed calls/iter, 2.1× faster than BusyBox
- Data-driven findings in all 10 categories, report.sh fixed for cat/ subdirectory
- Rate limiter raised 100→100K req/s in `internal/daemon/server.go`
- `wiki/19_performance_benchmarking.md` §6: actual measured matrix replaces predictions

Phase 23 (Multi-Tenant Sandbox) planned: 6-step roadmap for per-session filesystem
isolation, command allowlists, audit trails, resource quotas, subprocess jailing.

Updated: wiki/19_performance_benchmarking.md, wiki/22_hardening_iii.md,
wiki/23_multi_tenant_sandbox.md, wiki/phases.md, wiki/index.md,
.github/prompts/continue.prompt.md (new reusable session prompt)

## [2026-05-18] doc | Performance Quick Reference (`wiki/performance.md`)

Created a standalone quick-reference page for the performance benchmarking system.
Covers: all commands, scale factor tiers with numeric mappings, category key with
short/full/friendly names, expected results (priors), output file layout, architecture
diagram in ASCII, adding-new-category guide, and troubleshooting table. Linked from
index.md and phases.md.

## [2026-05-18] implement | Phase 19 — Benchmark infrastructure (branch: `feat/performance`)

Implemented all benchmark infrastructure per `wiki/19_performance_benchmarking.md`:
- `test/benchmark/lib/harness.sh` — shared timing, stats, `scaled()` helper, markdown tables
- `test/benchmark/lib/report.sh` — `summary.md` + `narrative.md` report generator
- `test/benchmark/runner.sh` — master orchestrator (--all, --quick, --cat)
- `test/benchmark/Dockerfile.bench` — benchmark image (Alpine + GoPOSIX + BusyBox + tooling)
- 10 category scripts: cat_a_startup.sh through cat_j_rpc_loop.sh
- Makefile: `bench-image`, `bench-all`, `bench-cat`, `bench-quick`, `bench-smoke/pu/stress`,
  `bench-report`, `bench-shell` targets + `SCALE` variable
- All 13 scripts pass `sh -n` syntax check

Updated: wiki/19_performance_benchmarking.md → IMPLEMENTING, wiki/phases.md → IMPLEMENTING

## [2026-05-18] plan | Phase 19 — Performance Benchmarking (GoPOSIX vs BusyBox)

Created comprehensive performance benchmarking plan (`wiki/19_performance_benchmarking.md`).
The plan defines 10 benchmark categories (A–J) comparing GoPOSIX against BusyBox v1.36.1
in identical Docker containers, with honest priors about where each tool wins:

- **BusyBox wins** on binary size (808 KB vs ~10 MB), single-invocation cold start, per-call RSS
- **GoPOSIX wins** on daemon amortized latency (5–100× for N≥50 calls), RPC task loop throughput
- **Fair fight** on text I/O throughput, bulk filesystem ops (both bottleneck on kernel VFS)

Plan includes Dockerfile.bench design, harness library spec (`lib/harness.sh`), Makefile
targets (`bench-image`, `bench-all`, `bench-quick`, `bench-cat`), CI integration blueprint,
and predicted result matrix with confidence ratings. ~20h estimated implementation effort.

Updated: wiki/index.md, wiki/phases.md (v5.4 → v5.5, Phase 19 added)

## [2026-05-17] fix | fold Unicode + low-coverage utilities — BusyBox 548, diff 71%, join 77%, paste 77%

Fixed `fold -sw66 with unicode input` BusyBox failure by rewriting foldLine to count
runes (not bytes) when not in `-b` mode. The 2 fold failures reduced to 1 (NUL handling
remains, but root cause is echo harness `\0` escape limitation, not fold itself).

Coverage surge on low-coverage utilities:
- **diff**: 57.1% → 71.0% (+13.9%). Added recursive dir diff (-r), -N new file,
  identical dirs tests. `diffDirs` now covered.
- **join**: 49.0% → 76.8% (+27.8%). Added CLI run() tests with temp files.
- **paste**: 46.2% → 76.9% (+30.7%). Added CLI tests for basic, serialize, delimiter.
- **split**: 45.2% (unchanged — CLI tests require CWD-sensitive setup).
- **tty**: 54.3% (unchanged — terminal-only paths).
- **who**: 54.5% (unchanged — utmp-dependent).

**BusyBox: 548 PASS (99.3%)** — fold Unicode fixed, now 4 failures (3 date + 1 fold NUL).

Updated: README.md, wiki/todos.md, wiki/test_coverage_matrix.md, .github/workflows/ci.yml.

## [2026-05-17] complete | Phase 18 finished — coverage ramp + docs sweep

Completed all remaining Phase 18 coverage work:
- **internal/daemon**: 35.9% → 64.6% (+28.7%). Added Start/Stop integration tests
  over Unix sockets, handleConn/handleSingleAsync via real connections, batch,
  notifications, invalid JSON, unknown method, and ping/echo end-to-end.
- **pkg/diff**: 54.8% → 57.1% (+2.3%). Added -w (ignoreAllSpace/stripAllSpace),
  -B (ignoreBlankLines), empty files, CRLF, binary data tests.
- **pkg/client**: 54.1% → 55.4% (+1.3%). Added rpcError.Error(), CloseTwice,
  context cancellation, Stat helper, helper coverage.

**Phase 18 is now COMPLETED.** All milestones checked off.

Comprehensive wiki sweep:
- `todos.md` — cleared completed items, restructured as pending-only living doc
- `phases.md` — v5.3, all phases marked COMPLETE, coverage numbers final
- `18_quality_fixes.md` — status → COMPLETED, final milestone + verify
- `posix_coverage.md` — 77 utilities, 99.1% pass rate
- `test_coverage_matrix.md` — updated daemon/diff/client numbers
- `README.md` — coverage gate, utility count, pass rate confirmed

**Final metrics:** 77 utilities, 547/541 BusyBox (99.1%), 85 test packages,
daemon 64.6%, 70.4% overall coverage.

## [2026-05-17] feature | Phase 15 + 18.1–18.4 — dd, od, patch, CI, egrep/fgrep

Implemented `dd` (6 BusyBox tests) and `od` (4 BusyBox tests) per Phase 15 spec.
`od` supports `-b`, `-c`, `-x`, `-f`, `-t`, `-N`, `--json`, `--traditional`.
Implemented `patch` (11 BusyBox tests) with unified diff parser, fuzzy context
matching, reverse/ignore-applied logic, `-p` strip, `-R`, `-N` flags.

CI fixes: coverage gate → `make cover-gate` (70%), BusyBox baseline 409→547.
Added `egrep`/`fgrep` dispatch aliases in pkg/grep.

Coverage ramp: internal/daemon 35.9%→51.5% (+15.6%, 20 new tests covering
WorkerPool, writeError, processRequest edge cases, batch handling, session
lifecycle, metrics, concurrent stress). pkg/diff +4 edge case tests.
pkg/client +3 helper tests.

**Metrics:** 77 utilities, 547/541 BusyBox (99.1%), 85 test packages.

Updated: README.md, wiki/phases.md, wiki/todos.md, wiki/15_post_mvp_tier1.md,
wiki/18_quality_fixes.md, wiki/test_coverage_matrix.md, .github/workflows/ci.yml,
test/busybox_testsuite/runtest.

## [2026-05-16] cleanup | Documentation sweep — stale numbers, historical markers, link consolidation

Comprehensive wiki+docs cleanup post-v1.0 Gold release:

**Stale numbers fixed:**
- `README.md`: 96.2%→99.4% pass rate, 53→56 utilities, 45%→70% coverage gate
- `docs/ARCHITECTURE.md`: 409→477 BusyBox passed, added Phases 13/14a-c to history
- `wiki/12_road_to_gold.md`: 45%→70% coverage gate, 409→477 BusyBox baseline
- `wiki/11a_lower_priority.md`: 45%→70% coverage gate, linked to canonical coverage page
- `wiki/11_post_mvp_priorities.md`: 45%→70% gate, 3/4→COMPLETED status

**Historical markers added to completed phase docs (00-10):**
- Added "HISTORICAL — COMPLETED" banner to: 00_foundation_libs, 01_multicall_tier1,
  03_filesystem_utils, 04_text_processing, 05_daemon_core, 06_system_utils,
  07_rpc_features, 08_hardening, 09_release_docs, 10_posix_framework, 10a_sed

**Status lines corrected:**
- `12_road_to_gold.md`: "Planning" → "COMPLETED — Gold Achieved"
- `13_coverage_and_hardening.md`: "In Progress" → "COMPLETED (70.5%)"
- `14_xml_output.md`: "Planning" → "DEFERRED"
- `11_post_mvp_priorities.md`: "3/4 complete" → "COMPLETED"
- `phases.md`: v3.0→v4.0, Gold status, current metrics, condensed layout

**Coverage consolidation:**
- `wiki/13_coverage_and_hardening.md` is now the **canonical coverage policy page**
- Added CI gate section (70% enforced via Makefile `COVERAGE_THRESHOLD`)
- All other docs link to it instead of duplicating stale numbers

**Index reorganization:**
- `wiki/index.md`: regrouped into Current State / Historical / Post-MVP Fix Sessions /
  Deferred / Reference — clear hierarchy for readers
- `wiki/todos.md`: removed stale `tar writing into read-only dir` failure (tar.tests
  sets umask 022, test always passes in-suite)

**Cross-references verified:** no broken internal links.

## [2026-05-16] design | GoPOSIXOS — bootable distro design document

Created `wiki/goposixos.md` — comprehensive design for a separate project
that imports GoPOSIX as a Go module and builds a bootable Linux distro.
Architecture: Linux kernel + initramfs containing a single multicall binary
(goposix 56 utilities + ~25 boot/system utilities). Boot process: PID 1 init
→ /etc/rc (goposix shell interpreter) → getty on /dev/console.

Three tiers of new utilities: boot-critical (init, mount, umount, mknod,
reboot, poweroff, halt, ~400 LOC), usable system (getty, login, passwd,
ifconfig, route, dhclient, ping, dmesg, ~600 LOC), real distro (modprobe,
syslogd, crond, fsck, mkfs, fdisk, ~800 LOC).

Design decisions: separate repo (layer boundary, independent cadence,
different test profiles), goposix shell for /etc/rc (already has resource
limits + path confinement), devtmpfs over static /dev, no package manager
(build pipeline IS the update mechanism).

Three milestones: M0 proof-of-concept (QEMU boots, 1-2 days), M1 usable
system (multi-user + networking + BusyBox gate, 3-5 days), M2 real distro
(persistent storage + fsck + releases, 1-2 weeks).

Named GoPOSIXOS for the inevitable Goose mascot.

Added to wiki/index.md Design section.

## [2026-05-16] feature | Public multicall API — goposix.Main() + goposix.Run()

Extracted the dispatch entry point from `cmd/goposix/main.go` into a public
`goposix.go` at the module root (`package goposix`). Downstream projects can now
import GoPOSIX as a library and build custom multicall binaries:

```go
package main
import (
    "os"
    "github.com/ramayac/goposix"
    _ "github.com/ramayac/goposix/pkg/ls"
    _ "github.com/ramayac/koreboot/pkg/init"  // custom utilities
)
func main() {
    goposix.WellKnownNames = append(goposix.WellKnownNames, "koreboot")
    os.Exit(goposix.Main())
}
```

API surface:
- `goposix.Version` — set via ldflags `-X github.com/ramayac/goposix.Version=...`
- `goposix.WellKnownNames` — binary names that trigger subcommand dispatch
- `goposix.Main()` → `goposix.Run(os.Args)` — dispatch entry point

Updated `cmd/goposix/main.go` (now 14 lines of logic + blank imports).
Updated LDFLAGS in Makefile, docker/Dockerfile, docker/Dockerfile.debug,
and .goreleaser.yml: `-X main.Version=...` → `-X github.com/ramayac/goposix.Version=...`.
Updated wiki/02_docker_ci.md LDFLAGS reference.

## [2026-05-16] maintain | 02 — Docker docs refreshed to current state

Updated `wiki/02_docker_ci.md` from original Phase 02 plan document to reflect
current as-built state. Changes: Go version 1.22→1.26, LDFLAGS updated (two
Version variables: pkg/common + main), `/out/bin/` staging directory doc,
system tzdata source, multi-arch buildx docs, CI pipeline steps (coverage
gate, Trivy scan, BusyBox baseline 409→477), release pipeline with supply
chain security. Added design-decision rationale table. Marked as COMPLETED /
MAINTAINED.

Updated `AGENTS.md` section 5 BusyBox numbers: 479/1/10 (stale pre-12.4 fix)
→ 477/3/10 (current). Noted all 3 failures are date-specific.

## [2026-05-15] fix | 14b + 14c — BusyBox regression fix + JSON-RPC coverage gap

### 14b — BusyBox Regression Fix

Ran `make testsuite` and found 79 failures. Root cause: `common.ParseFlags`
applied uniformly to all utilities, crashing on arguments starting with `-`
in free-form tools (echo, printf, expr). Two-phase fix session resolved 76
failures across 25+ utilities. 3 remain (all date: 2 Go TZ limitations,
1 cosmetic BusyBox error-format mismatch).

Key lessons:
- Shared infra needs escape hatches (stop-at-first-nonflag mode for ParseFlags)
- Never use `-j` short for `--json` (collides with tar, free-form data)
- BusyBox suite gates every commit (catches cascading failures)
- `devID` formatted a pointer instead of dereferencing Dev:Ino

Added ~75 hardening unit tests across 17 packages.
Updated wiki/14b_busybox_regression_fix.md with full details.

### 14c — JSON-RPC Coverage Gap

Audited test/posix-json/runner_test.go: only 9 of 55 utility packages
tested via the JSON-RPC daemon path. Created wiki/14c_posix_json_gap.md
with priority-ordered fix plan (5 tiers). Target: 100% coverage.

Updated wiki/phases.md, wiki/index.md with 14b/14c entries.

## [2026-05-15] polish | README — refreshed description, note --xml is not yet live

Updated README.md first paragraph and Key Features. Promoted --json structured
output but correctly notes --xml is in progress (Phase 14, PLANNING) rather
than claiming it as implemented. Fixed utility table: Agent row corrected from
(5) to (7). Docker quickstart kept at --json only.

Wiki phases.md and index.md already reflect XML Phase 14; no structural
changes needed there.

## [2026-05-15] plan | 14 + 14a — XML output support + gap fill

Created wiki/14_xml_output.md — plan to add `--xml` structured output to all 52
registered GoPOSIX utilities, consistent with the existing `--json` / `-j` system.
XML envelope mirrors JSON envelope: <goposix> with command/version/schemaVersion/
exitCode attrs, <data> innerxml payload, <error> block. Uses `encoding/xml` from
stdlib. Five phases: foundation (output.go XMLElement), Core batch (18 utilities),
Remaining batch (26), Gap-fill batch (12 including the 8 missing --json), and
Integration (test/posix-xml/ mirroring test/posix-json/). No short form `-x` —
reserved for future POSIX flags.

Created wiki/14a_json_gap_fill.md — detailed implementation plan for the 8
utilities that currently lack `--json` structured output: echo (manual parsing),
testcmd (strips --json before parse), sed, tee, tr, sleep, truefalse, yes.
Each gets a typed Result struct, FlagSpec integration, both --json and --xml
flags, and tests. Updated wiki/phases.md and wiki/index.md.

## [2026-05-15] consolidate | Remove rejected/future phases, merge audit + coverage ramp

Removed wiki/14_agent_architecture.md (ReAct agent — rejected) and
wiki/16_mcp_server.md (MCP design — out of scope). Merged
wiki/13_code_audit.md + wiki/15_coverage_ramp.md into a single page:
wiki/13_coverage_and_hardening.md — audit findings plus 3-stage coverage
ramp (50%→75%) plus speed targets.

Updated wiki/phases.md (v3.0): removed original plan analysis (phases 00–10
are complete), focused on five post-MVP pillars: Coverage, POSIX, Security,
Speed, Docker. Updated wiki/index.md to match.

## [2026-05-13] design | 16 — MCP Server design (replaces Phase 14)

Created `wiki/16_mcp_server.md` — design for exposing GoPOSIX as an MCP (Model
Context Protocol) server. External agents (Claude Desktop, Claude Code, Cursor,
Continue) drive GoPOSIX as their sandboxed Linux environment via stdio or HTTP/SSE
transport. Six curated tools: shell.exec, file.read, file.write, file.edit,
file.list, workspace.set. Reuses existing shell interpreter, session manager,
worker pool, and SecurePath. No new external dependencies — MCP uses JSON-RPC 2.0.

Rejected `wiki/14_agent_architecture.md` — GoPOSIX's natural role is as a tool
backend for external agents, not as an agent itself. Building an LLM agent inside
GoPOSIX duplicates what external agents already do well. Phase 16 is a protocol
adapter (~600 lines) vs Phase 14's full agent engine (~3000+ lines).

Updated wiki/index.md, wiki/phases.md.

## [2026-05-13] plan | 15 — Coverage Ramp plan (50% → 75%)

Created `wiki/15_coverage_ramp.md` — 3-stage plan targeting 75% overall coverage.
Stage 1 targets `internal/daemon` (3.3%), `cmd/goposix` (0%), `pkg/client` (44.9%),
and `pkg/daemon` (5.9%) to reach 60%. Stage 2 closes the `run()` gap across 24
utilities via dispatch-call tests with `testdata/` fixtures to reach 68%. Stage 3
refactors `run()` signatures to accept interfaces, pushing to 75%. Includes per-package
coverage targets, test fixture conventions, and verification commands.

## [2026-05-13] implement | 12.3b — Coverage push 46.2% → 50.0%

Added/enhanced unit tests across 15+ packages: sed (31.6%→49.1% — parser/compiler/
BRE tests), diff (42.8%→52.9% — Diff, normalizeSpace, filterBlankLines, edge cases),
grep (16.0%→16.5% — regex/invert/line-regex bounds), find (buildExecArgs),
dirname/basename/hostname/whoami/uname (run() CLI tests), dispatch (ListAll empty).

Overall coverage: **41.6% → 50.0%** (+8.4%). All 57 packages pass. CI enforces ≥45%.

## [2026-05-13] implement | 12.3 — Coverage gate enforcement + unit test expansion

Changed CI coverage step from `::warning::` (exits 0) to hard failure at 45% (`exit 1`).
Added/enhanced unit tests across 9 packages: head (11.2%→29.0%), tail (10.3%→27.6%),
grep (12.4%→16.0%), cat (21.8%→37.6%), wc (15.8%→32.5%), echo (20.8%→56.9%),
sort (23.4%→58.0%), uniq (37.2%→higher), cut (56.2%→higher), touch (20.0%→higher).

Overall coverage: **41.6% → 46.2%**. All 57 packages pass. CI enforces ≥45%.

Updated wiki: 11a_lower_priority.md (status, CI gate, remaining work table),
11_post_mvp_priorities.md (status, remaining work table).

## [2026-05-13] milestone | Promote agent architecture to Phase 14

Renamed `wiki/agent_architecture.md` → `wiki/14_agent_architecture.md`. Added to
wiki index and phases.md as Phase 14 (status: DESIGN). Updated ARCHITECTURE.md to
acknowledge pkg/agent and go-git dependency as planned additions.

## [2026-05-13] design | Agent Architecture design document

Created `wiki/agent_architecture.md` — a detailed design for an autonomous coding
agent compiled into the GoPOSIX binary. Covers: ReAct agent loop, go-git integration,
LLM provider interface (OpenAI / Anthropic / local), CLI + JSON-RPC dual interface,
workspace management, security model, Docker compose integration, and state machine.
No code changes; design-only phase. (Later promoted to Phase 14.)

## [2026-05-13] consolidate | Merge redundant wiki content, resolve inconsistencies

Consolidated awk content: 07a_awk.md is now the canonical awk document. Removed
duplicated task lists from 11_post_mvp_priorities.md (11.4), 12_road_to_gold.md
(12.5), and 13_code_audit.md (13.5) — all now point to 07a_awk.md with short
descriptions. Added cross-cutting deliverables (schema, BusyBox, posix_coverage,
README update) to 07a_awk.md.

Resolved shell security contradictions: 08_hardening.md claimed sandbox complete
but code had hardcoded timeout and no tests/docs. Added cross-references between
08 (design done), 11a.3 (deferred), and 12.2 (remaining work). Updated 12.2 with
code audit finding (GOPOSIX_SHELL_TIMEOUT not env-driven).

Resolved coverage gate inconsistency: 11a.4 claimed gate was done but 12.3/13.3
showed it's informational only (warns, exits 0). Clarified 11a.4 as "step added,
informational only; hard-fail tracked in 12.3."

Merged 13_code_audit.md execution plan into 12_road_to_gold.md. Added macOS build
breakage (13.0) as 12.0 in the gap analysis. 13 now focuses on code evidence and
wiki discrepancies, not task planning.

Updated 8 wiki files: index.md, phases.md (v2.2), 07a_awk.md, 11_post_mvp_priorities.md,
11a_lower_priority.md, 12_road_to_gold.md, 13_code_audit.md, 08_hardening.md.

Gold formula: 12.0–12.4 = Gold, +12.5 = Platinum.

## [2026-05-13] implement | 12.0 — macOS build fix

Split pkg/uname/uname.go and pkg/stat/stat.go into platform-specific files:
- uname_linux.go (syscall.Uname, [65]int8 fields)
- uname_darwin.go (unix.Uname, [256]byte fields, separate bytesToString helper)
- stat_linux.go (sys.Atim/sys.Ctim from syscall.Timespec)
- stat_darwin.go (sys.Atimespec/sys.Ctimespec)

Verified: GOOS=darwin CGO_ENABLED=0 go build ./... exits 0. Full test suite passes.

## [2026-05-13] implement | 12.2 — Shell security model

Wired GOPOSIX_SHELL_TIMEOUT env var in internal/shell/interpreter.go (was hardcoded
30s). Created internal/shell/interpreter_test.go with 10 tests: TestExecBasic,
TestTimeout (×2 env var scenarios), TestOutputWithinLimits, TestPathEscape,
TestPathEscapeBlocked (via shell redirection), TestEnvVarInjection, TestStderrCapture,
TestNonZeroExit, TestSyntaxError. Created docs/SECURITY.md: trust model, accessible
resources, resource limits, RPC-level protections, deployment posture, artifact
verification.

## [2026-05-13] implement | 12.4 — Fix BusyBox CI/local discrepancy

Old-style BusyBox tests called 'busybox <applet>' which resolved to system BusyBox
on CI (Ubuntu), not GoPOSIX — inflating the pass rate from 83.5% real to 97.9% fake.
Fixed test/busybox_testsuite/runtest: added global BBDIR temp directory with
busybox→goposix symlink prepended to PATH. Removed per-applet case block (tar/gzip).
Simplified old-style test PATH to a single shared block. Updated CI baseline in
.github/workflows/ci.yml from 479 to 409. Updated todos.md discrepancy note to
RESOLVED. True baseline: 409 passed, 71 failed, 10 skipped (83.5%).

## [2026-05-13] implement | 12.1 — Supply chain security

Added sboms: stanza to .goreleaser.yml (archive + binary). Added Cosign keyless
signing (OIDC) to release.yml with id-token: write permission. Added SLSA Level 3
provenance job via slsa-framework/slsa-github-generator@v2.1.0. Added Trivy
vulnerability scan to ci.yml (CRITICAL,HIGH severity, exits 1). Updated
docs/SECURITY.md artifact verification section with cosign verify, SBOM inspect,
and slsa-verifier commands. Updated 09_release_docs.md with supply chain section
(09.2b) and milestone items.

## [2026-05-13] update | Phase 11a complete, Gold 4/5 resolved

11a_lower_priority.md milestone: 8/8 complete (11a.3 → 12.2, 11a.4 → 12.4,
11a.7 → 12.1). 12_road_to_gold.md: 4/5 Gold gaps resolved, only 12.3 (coverage
gate) remains. 13_code_audit.md: 4/6 fixed. phases.md status updated. All
deferred work from 11a now resolved via Phase 12.

## [2026-05-12] update | Document .goreleaser.yml file location convention

Added explanation to `09_release_docs.md` for why `.goreleaser.yml` lives at the repo
root rather than `.github/`: GoReleaser is a tool-level config, not a GitHub-specific
feature. The root is its conventional default location.

## [2026-05-12] annotate | Add source links to utility docs

Added `[pkg/<name>/]` source links to every utility header in phase
pages (01, 03, 04, 06, 07). Also linked infrastructure packages in phases 00
and 05. All 55 utility packages now have clickable source links from their
wiki documentation.


