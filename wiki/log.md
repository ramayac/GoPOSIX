# Wiki Log

Append-only timeline of wiki maintenance activity.

## [2026-05-18] implement | Phase 19 — Benchmark infrastructure (branch: `feat/performance`)

Implemented all benchmark infrastructure per `wiki/19_performance_benchmarking.md`:
- `test/benchmark/lib/harness.sh` — shared timing, stats, `scaled()` helper, markdown tables
- `test/benchmark/lib/report.sh` — `summary.md` + `narrative.md` report generator
- `test/benchmark/runner.sh` — master orchestrator (--all, --quick, --cat)
- `test/benchmark/Dockerfile.bench` — benchmark image (Alpine + GoPOSIX + BusyBox + tooling)
- 10 category scripts: cat_a_startup.sh through cat_j_agent_loop.sh
- Makefile: `bench-image`, `bench-all`, `bench-cat`, `bench-quick`, `bench-smoke/pu/stress`,
  `bench-report`, `bench-shell` targets + `SCALE` variable
- All 13 scripts pass `sh -n` syntax check

Updated: wiki/19_performance_benchmarking.md → IMPLEMENTING, wiki/phases.md → IMPLEMENTING

## [2026-05-18] plan | Phase 19 — Performance Benchmarking (GoPOSIX vs BusyBox)

Created comprehensive performance benchmarking plan (`wiki/19_performance_benchmarking.md`).
The plan defines 10 benchmark categories (A–J) comparing GoPOSIX against BusyBox v1.36.1
in identical Docker containers, with honest priors about where each tool wins:

- **BusyBox wins** on binary size (808 KB vs ~10 MB), single-invocation cold start, per-call RSS
- **GoPOSIX wins** on daemon amortized latency (5–100× for N≥50 calls), agent loop throughput
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
  07_agent_features, 08_hardening, 09_release_docs, 10_posix_framework, 10a_sed

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

## [2026-05-12] update | Document .goreleaser.yml file location convention

Added explanation to `09_release_docs.md` for why `.goreleaser.yml` lives at the repo
root rather than `.github/`: GoReleaser is a tool-level config, not a GitHub-specific
feature. The root is its conventional default location.

## [2026-05-12] annotate | Add source links to utility docs

Added `[pkg/<name>/](../pkg/<name>/)` links to every utility header in phase
pages (01, 03, 04, 06, 07). Also linked infrastructure packages in phases 00
and 05. All 55 utility packages now have clickable source links from their
wiki documentation.

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
