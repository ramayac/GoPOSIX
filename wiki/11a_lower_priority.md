# Phase 11a — Lower Priority Improvements

> **Status:** In Progress (11a.2, 11a.4, 11a.5, 11a.6, 11a.8 complete) | **Depends on:** Phase 11 complete or in progress

---

## Context

These items improve quality, robustness, and production-readiness but are not blocking for the core value proposition. Address after Phase 11 priorities are shipped.

---

## 11a.1 — Compliance Test Suite Expansion (superseded)

**Decision:** The per-utility `test/compliance/` bash scripts and `make compliance` target have been removed. The BusyBox test suite (`make testsuite`, 479+ tests at 97.9% pass rate) provides broader, more standardized coverage. Expanding the BusyBox suite with additional test cases for KoreGo-specific features (e.g., `--json` output) is the preferred path going forward.

### Tasks

- [x] Remove `test/compliance/` scripts and `make compliance` target (superseded by BusyBox suite)
- [ ] Extend BusyBox test suite with `--json` output validation for key utilities
- [ ] Add missing BusyBox test cases for utilities not yet covered

---

## 11a.2 — Missing Unit Tests

**Current state:** 8 packages have no dedicated `_test.go` file: `client`, `cp`, `ln`, `mv`, `rmdir`, `yes`, `pkg/daemon`, and `pkg/common` (partial).

### Tasks

- [x] `pkg/cp` — test copy file, copy directory recursively, overwrite behavior (5 tests)
- [x] `pkg/mv` — test rename, cross-device move, overwrite (4 tests)
- [x] `pkg/ln` — test hard link, symlink creation, `-f` force (4 tests)
- [x] `pkg/rmdir` — test empty dir removal, non-empty rejection (4 tests)
- [x] `pkg/yes` — test output pattern, multi-word string (3 tests, also fixed bug: `fmt.Println` → `fmt.Fprintln(out, ...)`)
- [x] `pkg/daemon` — test daemon startup, socket creation, graceful shutdown (3 tests)
- [x] Enforce a minimum coverage gate in CI (suggest 70% per package if possible, with both positive and negative tests)
  - Added coverage step to CI workflow with overall threshold reporting. Per-package gate deferred (needs per-package threshold config).

---

## 11a.3 — Shell Interpreter Security Model

> **Note:** The sandbox design and implementation were completed in Phase 08
> ([08_hardening.md](08_hardening.md) 08.1). Testing (`interpreter_test.go`), env-var
> wiring (`KOREGO_SHELL_TIMEOUT`), and documentation (`docs/SECURITY.md`) are tracked
> in [12_road_to_gold.md](12_road_to_gold.md) (12.2). This section is superseded.

**Current state:** `internal/shell/interpreter.go` wraps `mvdan.cc/sh` with a hardcoded
30s timeout and a 128MB `LimitWriter` per stream. `SecurePath` confines file opens to
the session CWD. Code audit confirmed `KOREGO_SHELL_TIMEOUT` is not actually read from
the environment (env var is documented but not wired). No tests, no `docs/SECURITY.md`.

### Tasks (tracked in 12.2)

- [ ] Wire `KOREGO_SHELL_TIMEOUT` env var (currently hardcoded 30s)
- [ ] Write `internal/shell/interpreter_test.go` (timeout enforcement, path escape, resource limits)
- [ ] Write `docs/SECURITY.md` (trust model, accessible resources, limits, deployment posture)

---

## 11a.4 — CI Quality Gates

**Current state:** BusyBox baseline and image size are enforced as hard failures.
Coverage is **reported but informational** (warns at <50%, never fails the build).
Making coverage a hard gate is tracked in [12_road_to_gold.md](12_road_to_gold.md) (12.3).

| Check | Current | Target |
|-------|---------|--------|
| BusyBox suite | Hard fail if <479 passed | ✅ Enforced |
| Image size | Hard fail if >20MB | ✅ Enforced |
| Coverage | `::warning::` at <50%, exits 0 | Fail at 45% (stage 1) → 60% (stage 2) |
| Compliance tests | `test/compliance/` removed | BusyBox suite (479+ tests) |

### Tasks

- [x] Change BusyBox CI step to fail if pass count drops below 479 (current baseline)
- [x] Make image size gate a hard failure (was `::warning::`, now `::error::` with `exit 1`)
- [x] Add `go test -coverprofile` with a threshold check (coverage reported via `::warning::`; hard-fail gate deferred to 12.3)
- [x] Add compliance test step that runs all scripts from 11a.1 (step added, later removed — superseded by BusyBox suite)

---

## 11a.5 — Makefile Improvements

**Current state:** Several targets are missing or broken for non-Mac environments.

### Tasks

- [x] Add `make bench` target wired to `test/benchmark/` — already added during Phase 11
- [x] Fix `make cover` — `make cover-pct` already added during Phase 11 (prints per-package coverage %)
- [x] Add `make validate-schemas` target — already added during Phase 11
- [x] Add `make example-agent` target — already added during Phase 11
- [x] Document all targets in a `make help` target — already present with categorized target listing

---

## 11a.6 — Deployment Patterns

**Current state:** No deployment documentation beyond the basic Docker quickstart in README.

### Tasks

- [x] `docs/deploy/docker-compose.md` — daemon as a sidecar alongside an app container (with healthcheck, config, troubleshooting)
- [x] `docs/deploy/kubernetes.md` — sidecar, init container, and DaemonSet patterns with resource limits and probes
- [x] `docs/deploy/systemd.md` — unit file with socket activation, security hardening, and journalctl instructions
- [x] `examples/docker-compose.yml` — working example with daemon + Alpine-based smoke client

---

## 11a.7 — Release Pipeline Hardening

**Current state:** GoReleaser builds multi-arch binaries and pushes to GHCR, but no supply chain security measures are in place.

### Tasks

- [ ] Add SBOM generation to GoReleaser config (`sboms: true`)
- [ ] Add SLSA provenance attestation (GitHub Actions `slsa-framework/slsa-github-generator`)
- [ ] Add container image signing (Cosign)
- [ ] Add `trivy` or `grype` container scanning step in CI
- [ ] Auto-generate CHANGELOG from conventional commits on release

---

## 11a.8 — Clean Up `scratch.go`

**Current state:** `scratch.go` at the repo root contains a standalone Myers diff implementation with its own `main()` function. It is not imported anywhere and appears to be leftover scratch work.

### Tasks

- [x] Verify it is not referenced anywhere (`grep -r "scratch" .` returned no matches)
- [x] Delete `scratch.go` (removed)

---

## Milestone 11a

- [x] Compliance test approach changed: per-utility scripts removed in favor of BusyBox test suite (11a.1 — superseded)
- [x] 6 missing unit test files added (cp, mv, ln, rmdir, yes, daemon); coverage step in CI (11a.2)
- [x] Shell interpreter security model documented (11a.3 — completed via [12.2](12_road_to_gold.md): KOREGO_SHELL_TIMEOUT wired, interpreter_test.go with 10 tests, docs/SECURITY.md)
- [x] BusyBox baseline enforced; image size gate hard failure; coverage CI step added (informational only; hard-fail tracked in [12.3](12_road_to_gold.md)) (11a.4)
- [x] `make help`, `make bench`, `make validate-schemas`, `make example-agent`, `make cover-pct` all work (11a.5)
- [x] Three deployment patterns documented with a working docker-compose example (11a.6)
- [ ] Release pipeline hardened (11a.7 — tracked in [12.1](12_road_to_gold.md))
- [x] `scratch.go` deleted (11a.8)

**Summary:** 7 of 8 items complete (11a.3 resolved via 12.2; 11a.4 resolved via 12.4). Remaining: 11a.7 (release hardening, tracked in [12.1](12_road_to_gold.md)). 11a.1 superseded (compliance scripts removed, BusyBox suite is the path forward).
