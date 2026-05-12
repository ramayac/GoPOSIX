# Phase 11a — Lower Priority Improvements

> **Status:** Backlog | **Depends on:** Phase 11 complete or in progress

---

## Context

These items improve quality, robustness, and production-readiness but are not blocking for the core value proposition. Address after Phase 11 priorities are shipped.

---

## 11a.1 — Compliance Test Suite Expansion

**Current state:** Only 3 Bash compliance scripts exist (`test_ls.sh`, `test_cat.sh`, `test_basename_dirname.sh`). 46 utilities have no dedicated compliance tests. Regressions outside BusyBox coverage are invisible.

### Tasks

- [ ] Write compliance test scripts for all 49 utilities under `test/compliance/`
- [ ] Each script should: test the happy path, at least one edge case, and `--json` output validity
- [ ] Wire all scripts into `make compliance` (currently only runs 3)
- [ ] Add a CI step that fails on compliance test failure (no `continue-on-error`)

### Template

```bash
#!/usr/bin/env bash
# test/compliance/test_grep.sh
set -euo pipefail
KOREGO=${KOREGO:-./korego}

echo "hello world" | $KOREGO grep "hello"  # exit 0
echo "hello world" | $KOREGO grep "xyz" && exit 1 || true  # exit 1 expected
out=$($KOREGO grep --json "hello" <(echo "hello world"))
echo "$out" | jq -e '.matches | length > 0'
echo "PASS: grep"
```

---

## 11a.2 — Missing Unit Tests

**Current state:** 8 packages have no dedicated `_test.go` file: `client`, `cp`, `ln`, `mv`, `rmdir`, `yes`, `pkg/daemon`, and `pkg/common` (partial).

### Tasks

- [ ] `pkg/cp` — test copy file, copy directory recursively, overwrite behavior
- [ ] `pkg/mv` — test rename, cross-device move, overwrite
- [ ] `pkg/ln` — test hard link, symlink creation, `-f` force
- [ ] `pkg/rmdir` — test empty dir removal, non-empty rejection
- [ ] `pkg/yes` — test output pattern, `-n` limit (if implemented)
- [ ] `pkg/daemon` — test daemon startup, socket creation, graceful shutdown
- [ ] Enforce a minimum coverage gate in CI (suggest 70% per package)

---

## 11a.3 — Shell Interpreter Security Model

**Current state:** `internal/shell/interpreter.go` wraps `mvdan.cc/sh` with a 30s timeout and memory limit (per `wiki/07_agent_features.md`), but there are no tests for these limits and no documentation of the security boundaries.

### Tasks

- [ ] Write tests that verify timeout is enforced (e.g., `sleep 60` must be killed)
- [ ] Write tests that verify dangerous operations are blocked (if sandboxing is intended)
- [ ] Document the security model in `docs/SECURITY.md`:
  - What can a script access? (filesystem, env, network)
  - What are the resource limits?
  - What syscalls are blocked (if any)?
- [ ] Decide and document whether `korego.shell.exec` is "trusted input only" or "untrusted input safe"

---

## 11a.4 — CI Quality Gates

**Current state:** Several CI checks are informational and don't fail the build.

| Check | Current | Target |
|-------|---------|--------|
| BusyBox suite | `continue-on-error: true` | Fail on regression from baseline |
| Image size | Warn if >20MB | Fail if >20MB |
| Coverage | Not enforced | Fail if <70% on changed packages |
| Compliance tests | 3 scripts only | All 49 utilities |

### Tasks

- [ ] Change BusyBox CI step to fail if pass count drops below 479 (current baseline)
- [ ] Make image size gate a hard failure
- [ ] Add `go test -coverprofile` with a threshold check (`go-coverage-report` or similar)
- [ ] Add compliance test step that runs all scripts from 11a.1

---

## 11a.5 — Makefile Improvements

**Current state:** Several targets are missing or broken for non-Mac environments.

### Tasks

- [ ] Add `make bench` target wired to `test/benchmark/` (currently exists but not in Makefile)
- [ ] Fix `make cover` — currently opens HTML in browser; add `make cover-pct` that prints % only and is CI-friendly
- [ ] Add `make validate-schemas` target (prereq for Phase 11.1)
- [ ] Add `make example-agent` target (prereq for Phase 11.2)
- [ ] Document all targets in a `make help` target

---

## 11a.6 — Deployment Patterns

**Current state:** No deployment documentation beyond the basic Docker quickstart in README.

### Tasks

- [ ] `docs/deploy/docker-compose.md` — daemon as a sidecar alongside an app container
- [ ] `docs/deploy/kubernetes.md` — daemon as an init container or sidecar, with Unix socket shared via `emptyDir` volume
- [ ] `docs/deploy/systemd.md` — unit file for running the daemon on a Linux host
- [ ] `examples/docker-compose.yml` — working example from the docs

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

- [ ] Verify it is not referenced anywhere (`grep -r "scratch" .`)
- [ ] Delete `scratch.go`

---

## Milestone 11a

- [ ] All 49 utilities have compliance test scripts and CI runs them
- [ ] 8 missing unit test files added; coverage gate enforced in CI
- [ ] `make help`, `make bench`, `make validate-schemas` all work
- [ ] `docs/SECURITY.md` documents the shell interpreter threat model
- [ ] `scratch.go` deleted
- [ ] At least one deployment pattern documented with a working example
