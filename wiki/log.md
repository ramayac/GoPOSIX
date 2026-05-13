# Wiki Log

Append-only timeline of wiki maintenance activity.

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
code audit finding (KOREGO_SHELL_TIMEOUT not env-driven).

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

Wired KOREGO_SHELL_TIMEOUT env var in internal/shell/interpreter.go (was hardcoded
30s). Created internal/shell/interpreter_test.go with 10 tests: TestExecBasic,
TestTimeout (×2 env var scenarios), TestOutputWithinLimits, TestPathEscape,
TestPathEscapeBlocked (via shell redirection), TestEnvVarInjection, TestStderrCapture,
TestNonZeroExit, TestSyntaxError. Created docs/SECURITY.md: trust model, accessible
resources, resource limits, RPC-level protections, deployment posture, artifact
verification.

## [2026-05-13] implement | 12.4 — Fix BusyBox CI/local discrepancy

Old-style BusyBox tests called 'busybox <applet>' which resolved to system BusyBox
on CI (Ubuntu), not KoreGo — inflating the pass rate from 83.5% real to 97.9% fake.
Fixed test/busybox_testsuite/runtest: added global BBDIR temp directory with
busybox→korego symlink prepended to PATH. Removed per-applet case block (tar/gzip).
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
