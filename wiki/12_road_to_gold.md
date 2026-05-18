# Phase 12 — Road to Gold

> **Status:** COMPLETED — Gold Achieved | **Date:** 2026-05-13 | **Last verified:** 2026-05-16

---

## Context

GoPOSIX MVP and all post-MVP cleanup phases (00–11a) are complete or near-complete.
The project is assessed at **Strong Silver / borderline Gold**.

This phase documents the specific gaps preventing a Gold rating and the tasks required
to close them, ordered by impact. Completing items 12.0–12.4 achieves Gold. Item 12.5
(`awk`) pushes toward Platinum.

### Current Rating Summary

| Tier | Verdict |
|------|---------|
| Bronze | ✅ Cleared |
| Silver | ✅ Cleared |
| **Gold** | ✅ Cleared — 5/5 gaps resolved |
| Platinum | ❌ Requires awk + Gold first |

---

## Gap Analysis

| # | Gap | Risk | Impact |
|---|-----|------|--------|
| 12.0 | macOS build broken (`uname`, `stat`, `client`) | High — blocks local development on macOS | Dev velocity |
| 12.1 | No supply chain security (SBOM, Cosign, SLSA, scanning) | High — infra tooling supply chain attacks are common | Release trust |
| 12.2 | Shell security model undocumented, timeout hardcoded | High — tool accepts untrusted programmatic input | Adoption blocker |
| 12.3 | Coverage gate is informational (warns at 50%, never fails); actual coverage 41.6% | Medium — a patch can drop coverage to 0% and pass CI | Code quality |
| 12.4 | CI/local BusyBox test discrepancy | Medium — old-style tests pass via system BusyBox on CI, not GoPOSIX | Test reliability |
| 12.5 | `awk` not implemented | Low (MVP scope excluded it) — qualifies the "100% POSIX" claim | Compliance claim |

---

## 12.0 — macOS Build Breakage

**Why it matters:** Three packages fail to compile on `GOOS=darwin`, blocking local
development for anyone on macOS. This was discovered during the code audit ([13_coverage_and_hardening.md](13_coverage_and_hardening.md))
and is not captured in earlier phase docs.

**Current state:** `pkg/uname/uname.go` uses `syscall.Utsname`/`syscall.Uname()` (Linux-only).
`pkg/stat/stat.go` uses `sys.Atim`/`sys.Ctim` (Linux `syscall.Timespec`; Darwin uses
`Atimespec`/`Ctimespec`). `pkg/client` fails as a transitive dependency.

### Tasks

- [x] Add `//go:build linux` tag to `pkg/uname/uname.go`; create `pkg/uname/uname_darwin.go` using `golang.org/x/sys/unix.Uname`
- [x] Split `pkg/stat/stat.go` into `stat_linux.go` and `stat_darwin.go` for the `Sys()` cast difference
- [x] Verify: `GOOS=darwin CGO_ENABLED=0 go build ./...` exits 0

### Acceptance

```bash
GOOS=darwin CGO_ENABLED=0 go build ./...   # must exit 0
GOOS=linux CGO_ENABLED=0 go build ./...    # must still exit 0 (no regression)
```

---

## 12.1 — Supply Chain Security

**Why it matters:** GoPOSIX is distributed as a container image and binary. Without provenance
and signing, consumers cannot verify artifacts were built from the declared source. This is
increasingly table stakes for infrastructure tooling.

**Current state:** GoReleaser produces multi-arch binaries and pushes to GHCR. No SBOM,
no image signing, no vulnerability scanning, no provenance attestation.

### Tasks

- [x] Add SBOM generation to `.goreleaser.yml` (`sboms: artifacts: archive + binary`)
- [x] Add container image signing via Cosign (keyless/OIDC) in `release.yml`
- [x] Add SLSA Level 3 provenance via `slsa-framework/slsa-github-generator` in release workflow
- [x] Add vulnerability scanning (Trivy) in `ci.yml` after Docker build (CRITICAL,HIGH)
- [x] Update `docs/SECURITY.md` with artifact verification instructions + slsa-verifier

### Acceptance

```bash
# Verify a release image is signed
cosign verify ghcr.io/ramayac/goposix:latest --certificate-identity-regexp='.*' --certificate-oidc-issuer='https://token.actions.githubusercontent.com'

# Inspect SBOM
docker buildx imagetools inspect ghcr.io/ramayac/goposix:latest --format '{{ json .SBOM }}'
```

---

## 12.2 — Shell Security Model

**Why it matters:** `goposix.shell.exec` is the highest-risk surface in the entire codebase.
The daemon is designed for programmatic consumption — programs may pass partially
untrusted or model-generated shell scripts. Without a documented and tested security
contract, operators cannot safely expose the daemon.

**Current state:** `internal/shell/interpreter.go` wraps `mvdan.cc/sh` with a hardcoded 30s
timeout (the `GOPOSIX_SHELL_TIMEOUT` env var is documented but **not actually read by the code** —
found during code audit) and a `LimitWriter` cap of 128 MB per stream. `SecurePath` confines
file opens to the session CWD. No tests enforce these limits. No `docs/SECURITY.md` exists.

**Code audit finding:** The wiki previously stated `GOPOSIX_SHELL_TIMEOUT` was configurable,
but `interpreter.go` hardcodes `30*time.Second`. This needs to be wired up (see tasks).

### Tasks

- [x] Wire `GOPOSIX_SHELL_TIMEOUT` env var in `internal/shell/interpreter.go` (was hardcoded 30s)
- [x] Write `docs/SECURITY.md` defining the security model (trust level, accessible resources, limits, deployment posture, artifact verification)
- [x] Add tests in `internal/shell/interpreter_test.go` (10 tests: timeout enforcement ×2, path escape, path allow, env vars, stderr, exit codes, syntax error, output limits)
- [ ] Verify `GOPOSIX_SHELL_TIMEOUT` default (30s) is documented in README and `docs/RPC_API.md` — deferred
- [ ] Consider whether `shell.exec` should require an explicit session — deferred (design discussion)

### Acceptance

```bash
# Timeout is enforced
echo '{"jsonrpc":"2.0","method":"shell.exec","params":{"script":"sleep 60"},"id":1}' \
  | nc -U /tmp/goposix.sock
# → must return error within GOPOSIX_SHELL_TIMEOUT seconds, not hang

go test ./internal/shell/... -v -run TestTimeout
go test ./internal/shell/... -v -run TestResourceLimits
```

---

## 12.3 — Enforce Coverage Gate

**Why it matters:** The current CI coverage step is purely informational — it prints a
warning at <50% but never fails the build. A commit that drops coverage to 0% passes CI
undetected. This defeats the purpose of the gate.

**Current state:** `Makefile` enforces 70% threshold via `COVERAGE_THRESHOLD := 70`
(`make cover-gate` hard-fails below this). **Actual overall coverage is 70.5%** (up from
41.6%). All packages pass; no package below 5%.

All three stages complete. Coverage exceeds the 70% Gold target.

### Tasks

- [x] **Stage 1:** Change coverage check from `::warning::` to `exit 1` at 45% (current 50.0%)
- [x] **Stage 2a:** Add unit tests to highest-ROI packages (50.0% reached):
  - `pkg/head` (11.2% → 29.0%) — Run(), runBytes(), runNegative()
  - `pkg/tail` (10.3% → 27.6%) — Run() bytes mode, fromStart
  - `pkg/grep` (12.4% → 16.0%) — Run() regex, invert, line/word regexp
  - `pkg/cat` (21.8% → 37.6%) — visByte(), visLine(), Run() branches
  - `pkg/wc` (15.8% → 32.5%) — CountProper(), Count()
  - `pkg/sort` (23.4% → 58.0%) — parseHumanVal, parseNumericPrefix, parseMonth, extractKey, compareHuman
  - `pkg/echo` (20.8% → 56.9%) — processEscapes() edge cases
  - `pkg/uniq` — extractCompareKey, skipFields, checkChars
  - `pkg/cut` — parseList, inRange, fields/chars/bytes modes
  - `pkg/touch` — Run() multi-path, existing file
- [ ] **Stage 2b:** Continue adding tests to push coverage from 46.2% to 60%:
  - `pkg/grep` (16.0%) — more regex branch coverage
  - `pkg/sed` (31.6%) — Parse/runEngine branch coverage
  - `pkg/diff` (42.8%) — utility function coverage
  - `pkg/tee` (3.3%) — refactor to expose testable logic
  - `pkg/dirname` (14.3%) — test run() via dispatch
- [ ] **Stage 3:** Raise threshold to 60% once coverage exceeds it
- [ ] Add per-package coverage reporting to `make cover-pct` output
- [ ] Document the coverage policy in `AGENTS.md`

### Acceptance

```bash
# Fails CI if coverage drops below threshold
make cover-pct   # must show ≥60% overall
```

---

## 12.4 — Fix BusyBox CI / Local Discrepancy

**Why it matters:** Old-style BusyBox tests (in `test/busybox_testsuite/<applet>/`) call
`busybox <applet>` which resolves to the **system BusyBox** on CI (Ubuntu), not GoPOSIX.
This means CI passes those tests regardless of GoPOSIX's behavior — a silent blind spot.

**Current state:** RESOLVED. `runtest` now creates a global `busybox → goposix` symlink
in a temp directory (`BBDIR`) prepended to PATH before any tests run. All old-style
`busybox <applet>` calls resolve to GoPOSIX on both CI and local.

**New baseline:** The current GoPOSIX BusyBox pass rate is **477 passed, 3 failed, 10 skipped**
(99.4%). Early baseline after discrepancy fix was 409/71/10 (83.5%), resolved to 477/3/10
through fixes across 25+ utilities. The CI gate now enforces ≥477.

### Tasks

- [x] Audit which old-style tests exist — confirmed: per-applet dirs in `test/busybox_testsuite/`
- [x] Route ALL `busybox` calls to GoPOSIX via global `BBDIR` symlink (Option A)
- [x] Simplify old-style test PATH blocks (per-applet `case` removed; single shared PATH)
- [x] Update CI baseline from 479 to 409 (`ci.yml`)
- [x] Document the resolution in `todos.md` and close the discrepancy note

---

## 12.5 — `awk` Implementation (Platinum Gate)

> **Full plan:** [07a_awk.md](07a_awk.md) — the canonical awk document.
>
> `awk` is the last missing POSIX.2 utility. Without it, the "POSIX-compliant
> userland" claim carries a permanent asterisk. All task details, sub-phase
> breakdowns, and acceptance criteria live in 07a_awk.md. Completing this
> alongside 12.0–12.4 achieves **Platinum**.

---

## Milestone 12 — Gold Checklist

```
[x] 12.0 — macOS builds cleanly (GOOS=darwin go build ./... exits 0)
[x] 12.1 — SBOM + Cosign + SLSA + trivy in release/CI pipeline
[x] 12.2 — Shell security model documented + GOPOSIX_SHELL_TIMEOUT wired + tests passing
[x] 12.3 — Coverage gate hard-fails CI at 70% via `Makefile` (current: 70.5%) — see [coverage policy](13_coverage_and_hardening.md)
[x] 12.4 — CI/local BusyBox discrepancy resolved; baselines match
[ ] 12.5 — awk implemented, BusyBox awk tests pass (Platinum gate)
```

Completing 12.0–12.4 = **Gold** (5/5 done).
Completing 12.0–12.5 = **Platinum**.

---

## How to Verify

```bash
# macOS build
GOOS=darwin CGO_ENABLED=0 go build ./...   # must exit 0

# Supply chain
cosign verify ghcr.io/ramayac/goposix:latest \
  --certificate-identity-regexp='.*' \
  --certificate-oidc-issuer='https://token.actions.githubusercontent.com'
docker buildx imagetools inspect ghcr.io/ramayac/goposix:latest --format '{{ json .SBOM }}'

# Shell security
GOPOSIX_SHELL_TIMEOUT=5s go test ./internal/shell/... -v -run TestTimeout
go test ./internal/shell/... -v -run TestResourceLimits

# Coverage gate
make cover-pct   # ≥70% enforced; see coverage policy in wiki/13_coverage_and_hardening.md

# BusyBox parity
make testsuite   # same result locally and in CI

# awk (Platinum) — see 07a_awk.md for full criteria
echo "hello world" | ./goposix awk '{print $1}'
```
