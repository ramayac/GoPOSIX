# Phase 12 — Road to Gold

> **Status:** Planning | **Date:** 2026-05-13 | **Depends on:** All Phases 00–11 complete

---

## Context

KoreGo MVP and all post-MVP cleanup phases (00–11a) are complete or near-complete.
The project is assessed at **Strong Silver / borderline Gold**.

This phase documents the specific gaps preventing a Gold rating and the tasks required
to close them, ordered by impact. Completing items 12.0–12.4 achieves Gold. Item 12.5
(`awk`) pushes toward Platinum.

### Current Rating Summary

| Tier | Verdict |
|------|---------|
| Bronze | ✅ Cleared |
| Silver | ✅ Cleared |
| **Gold** | ⚠️ In Progress — 3/5 gaps resolved (12.1 supply chain + 12.3 coverage remaining) |
| Platinum | ❌ Requires awk + Gold first |

---

## Gap Analysis

| # | Gap | Risk | Impact |
|---|-----|------|--------|
| 12.0 | macOS build broken (`uname`, `stat`, `client`) | High — blocks local development on macOS | Dev velocity |
| 12.1 | No supply chain security (SBOM, Cosign, SLSA, scanning) | High — infra tooling supply chain attacks are common | Release trust |
| 12.2 | Shell security model undocumented, timeout hardcoded | High — tool explicitly targets AI agents with untrusted input | Adoption blocker |
| 12.3 | Coverage gate is informational (warns at 50%, never fails); actual coverage 41.6% | Medium — a patch can drop coverage to 0% and pass CI | Code quality |
| 12.4 | CI/local BusyBox test discrepancy | Medium — old-style tests pass via system BusyBox on CI, not KoreGo | Test reliability |
| 12.5 | `awk` not implemented | Low (MVP scope excluded it) — qualifies the "100% POSIX" claim | Compliance claim |

---

## 12.0 — macOS Build Breakage

**Why it matters:** Three packages fail to compile on `GOOS=darwin`, blocking local
development for anyone on macOS. This was discovered during the code audit ([13_code_audit.md](13_code_audit.md))
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

**Why it matters:** KoreGo is distributed as a container image and binary. Without provenance
and signing, consumers cannot verify artifacts were built from the declared source. This is
increasingly table stakes for infrastructure tooling.

**Current state:** GoReleaser produces multi-arch binaries and pushes to GHCR. No SBOM,
no image signing, no vulnerability scanning, no provenance attestation.

### Tasks

- [ ] Add SBOM generation to `.goreleaser.yml`:
  ```yaml
  sboms:
    - artifacts: archive
    - artifacts: binary
  ```
- [ ] Add container image signing via Cosign in `.github/workflows/release.yml`:
  ```yaml
  - uses: sigstore/cosign-installer@v3
  - run: cosign sign --yes ghcr.io/ramayac/korego:${{ github.ref_name }}
  ```
- [ ] Add SLSA Level 3 provenance via `slsa-framework/slsa-github-generator` in release workflow
- [ ] Add vulnerability scanning step in CI (`ci.yml`) after Docker build:
  ```yaml
  - uses: aquasecurity/trivy-action@master
    with:
      image-ref: korego:dev
      exit-code: 1
      severity: CRITICAL,HIGH
  ```
- [ ] Update `docs/SECURITY.md` with artifact verification instructions for end users

### Acceptance

```bash
# Verify a release image is signed
cosign verify ghcr.io/ramayac/korego:latest --certificate-identity-regexp='.*' --certificate-oidc-issuer='https://token.actions.githubusercontent.com'

# Inspect SBOM
docker buildx imagetools inspect ghcr.io/ramayac/korego:latest --format '{{ json .SBOM }}'
```

---

## 12.2 — Shell Security Model

**Why it matters:** `korego.shell.exec` is the highest-risk surface in the entire codebase.
The daemon is explicitly positioned for AI agent use — agents may pass partially
untrusted or model-generated shell scripts. Without a documented and tested security
contract, operators cannot safely expose the daemon.

**Current state:** `internal/shell/interpreter.go` wraps `mvdan.cc/sh` with a hardcoded 30s
timeout (the `KOREGO_SHELL_TIMEOUT` env var is documented but **not actually read by the code** —
found during code audit) and a `LimitWriter` cap of 128 MB per stream. `SecurePath` confines
file opens to the session CWD. No tests enforce these limits. No `docs/SECURITY.md` exists.

**Code audit finding:** The wiki previously stated `KOREGO_SHELL_TIMEOUT` was configurable,
but `interpreter.go` hardcodes `30*time.Second`. This needs to be wired up (see tasks).

### Tasks

- [x] Wire `KOREGO_SHELL_TIMEOUT` env var in `internal/shell/interpreter.go` (was hardcoded 30s)
- [x] Write `docs/SECURITY.md` defining the security model (trust level, accessible resources, limits, deployment posture, artifact verification)
- [x] Add tests in `internal/shell/interpreter_test.go` (10 tests: timeout enforcement ×2, path escape, path allow, env vars, stderr, exit codes, syntax error, output limits)
- [ ] Verify `KOREGO_SHELL_TIMEOUT` default (30s) is documented in README and `docs/RPC_API.md` — deferred
- [ ] Consider whether `shell.exec` should require an explicit session — deferred (design discussion)

### Acceptance

```bash
# Timeout is enforced
echo '{"jsonrpc":"2.0","method":"shell.exec","params":{"script":"sleep 60"},"id":1}' \
  | nc -U /tmp/korego.sock
# → must return error within KOREGO_SHELL_TIMEOUT seconds, not hang

go test ./internal/shell/... -v -run TestTimeout
go test ./internal/shell/... -v -run TestResourceLimits
```

---

## 12.3 — Enforce Coverage Gate

**Why it matters:** The current CI coverage step is purely informational — it prints a
warning at <50% but never fails the build. A commit that drops coverage to 0% passes CI
undetected. This defeats the purpose of the gate.

**Current state:** `.github/workflows/ci.yml` — coverage step uses `::warning::` and exits 0.
**Actual overall coverage is 41.6%** (audit measured via `go test -coverprofile` across
`./pkg/...` `./internal/...`). Worst packages: `internal/daemon` 3.3%, `pkg/tee` 3.3%,
`pkg/tail` 10.3%, `pkg/head` 11.2%, `pkg/grep` 12.4%. Best: `pkg/truefalse` 100%,
`pkg/printf` 89.6%, `pkg/chown` 89.7%.

Enforcing 60% immediately would break CI. A staged approach is needed.

### Tasks

- [ ] **Stage 1:** Change coverage check from `::warning::` to `exit 1` at 45% (current 41.6% → need ~3% more):
  ```bash
  if (( $(echo "$COVERED < 45" | bc -l) )); then
    echo "::error::Coverage ${COVERED}% is below 45% threshold"
    exit 1
  fi
  ```
- [ ] **Stage 2:** Add unit tests to highest-ROI packages (trivial input/output contracts):
  - `pkg/head` (11.2%) — table-driven tests, can reach 70%+ quickly
  - `pkg/tail` (10.3%) — same pattern
  - `pkg/grep` (12.4%) — pattern matching tests
  - `pkg/dirname` (14.3%) — simple path manipulation
  - `pkg/echo` (20.8%) — basic output tests
  - `pkg/tee` (3.3%) — I/O multiplexing tests
- [ ] **Stage 3:** Raise threshold to 60% once coverage exceeds it
- [ ] Add per-package coverage reporting to `make cover-pct` output so regressions are visible
- [ ] Document the coverage policy in `AGENTS.md` so contributors know the bar

### Acceptance

```bash
# Fails CI if coverage drops below threshold
make cover-pct   # must show ≥60% overall
```

---

## 12.4 — Fix BusyBox CI / Local Discrepancy

**Why it matters:** Old-style BusyBox tests (in `test/busybox_testsuite/<applet>/`) call
`busybox <applet>` which resolves to the **system BusyBox** on CI (Ubuntu), not KoreGo.
This means CI passes those tests regardless of KoreGo's behavior — a silent blind spot.

**Current state:** RESOLVED. `runtest` now creates a global `busybox → korego` symlink
in a temp directory (`BBDIR`) prepended to PATH before any tests run. All old-style
`busybox <applet>` calls resolve to KoreGo on both CI and local.

**New baseline:** The true KoreGo BusyBox pass rate is **409 passed, 71 failed, 10 skipped**
(83.5% real). The previous 479/97.9% was inflated — ~70 old-style test passes were
actually testing system BusyBox on CI, not KoreGo. The CI gate now enforces ≥409.

### Tasks

- [x] Audit which old-style tests exist — confirmed: per-applet dirs in `test/busybox_testsuite/`
- [x] Route ALL `busybox` calls to KoreGo via global `BBDIR` symlink (Option A)
- [x] Simplify old-style test PATH blocks (per-applet `case` removed; single shared PATH)
- [x] Update CI baseline from 479 to 409 (`ci.yml`)
- [ ] Document the resolution in `todos.md` and close the discrepancy note

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
[ ] 12.1 — SBOM + Cosign + SLSA + trivy in release/CI pipeline
[x] 12.2 — Shell security model documented + KOREGO_SHELL_TIMEOUT wired + tests passing
[ ] 12.3 — Coverage gate hard-fails CI (staged: 45% → 60%)
[x] 12.4 — CI/local BusyBox discrepancy resolved; baselines match
[ ] 12.5 — awk implemented, BusyBox awk tests pass (Platinum gate)
```

Completing 12.0–12.4 = **Gold** (3/5 done, 12.1 + 12.3 remaining).
Completing 12.0–12.5 = **Platinum**.

---

## How to Verify

```bash
# macOS build
GOOS=darwin CGO_ENABLED=0 go build ./...   # must exit 0

# Supply chain
cosign verify ghcr.io/ramayac/korego:latest \
  --certificate-identity-regexp='.*' \
  --certificate-oidc-issuer='https://token.actions.githubusercontent.com'
docker buildx imagetools inspect ghcr.io/ramayac/korego:latest --format '{{ json .SBOM }}'

# Shell security
KOREGO_SHELL_TIMEOUT=5s go test ./internal/shell/... -v -run TestTimeout
go test ./internal/shell/... -v -run TestResourceLimits

# Coverage gate
make cover-pct   # ≥60% required (staged: 45% first, then 60%)

# BusyBox parity
make testsuite   # same result locally and in CI

# awk (Platinum) — see 07a_awk.md for full criteria
echo "hello world" | ./korego awk '{print $1}'
```
