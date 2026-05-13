# Phase 13 — Code Audit: Validated Gap Analysis

> **Status:** Review | **Date:** 2026-05-13 | **Method:** Code-first exploration, then validated against wiki/12_road_to_gold.md

---

## Context

This document records code-level findings from a live-repository audit.
The execution plan originally in this document has been merged into
[12_road_to_gold.md](12_road_to_gold.md), which is the authoritative
source for the Gold roadmap. This document focuses on **code evidence**
and **discrepancies** between wiki claims and the actual codebase.

**Overall coverage at time of audit: 41.6%** (`go test -coverprofile` across `./pkg/...` `./internal/...`)

---

## Summary

| # | Gap | In Wiki? | Code Evidence | Effort |
|---|-----|----------|---------------|--------|
| 13.0 | macOS build breakage (`uname`, `stat`, `client`) | No | `syscall.Utsname`, `sys.Atim` undefined on darwin | 2h |
| 13.1 | No supply chain security (SBOM, Cosign, SLSA, Trivy) | Yes (12.1) | `.goreleaser.yml` has no `sboms:`; `release.yml` has no signing step | 1d |
| 13.2 | Shell security model undocumented, untested | Yes (12.2) | `interpreter_test.go` does not exist; `KOREGO_SHELL_TIMEOUT` hardcoded, not env-driven | 1d |
| 13.3 | Coverage gate is informational only; real coverage 41.6% | Yes (12.3) | `ci.yml` emits `::warning::`, exits 0; worst packages at 3–14% | 3d |
| 13.4 | BusyBox CI/local discrepancy | Yes (12.4) | `runtest` creates KoreGo symlink only for `tar`/`gzip`; all other old-style tests hit system BusyBox | 1–2d |
| 13.5 | `awk` not implemented (Platinum gate) | Yes (12.5) | `pkg/awk/` does not exist | 5–10d |

---

## 13.0 — macOS Build Breakage (NOT in wiki — discovered during audit)

**Severity: Blocks local development on macOS.**

Three packages fail to build on `GOOS=darwin`:

### `pkg/uname/uname.go`
```
pkg/uname/uname.go:50:16: undefined: syscall.Utsname
pkg/uname/uname.go:51:20: undefined: syscall.Uname
```
`syscall.Utsname` and `syscall.Uname()` are Linux-only. The Darwin equivalent is
`golang.org/x/sys/unix.Uname`. The file needs a `//go:build linux` tag with a Darwin
stub alongside it.

### `pkg/stat/stat.go:58`
```
pkg/stat/stat.go:58:28: sys.Atim undefined (type *syscall.Stat_t has no field or method Atim)
pkg/stat/stat.go:59:28: sys.Ctim undefined (type *syscall.Stat_t has no field or method Ctim)
```
Linux `syscall.Stat_t` uses `Atim`/`Ctim` (`syscall.Timespec`); Darwin uses `Atimespec`/`Ctimespec`.
Fix: build tags splitting `stat_linux.go` and `stat_darwin.go` for the `Sys()` cast.

### `pkg/client`
Build fails as a dependency of the above. Resolves once `uname` and `stat` are fixed.

**Fix template for `uname`:**
```
pkg/uname/
  uname.go          # shared types, flag spec, CLI layer
  uname_linux.go    # //go:build linux — syscall.Uname impl
  uname_darwin.go   # //go:build darwin — golang.org/x/sys/unix.Uname impl
```

---

## 13.1 — Supply Chain Security

**Confirmed: zero implementation.**

### `.goreleaser.yml` — no SBOM
Current file has `builds`, `dockers`, `docker_manifests`, `changelog`. No `sboms:` stanza.

Required addition:
```yaml
sboms:
  - artifacts: archive
  - artifacts: binary
```

### `.github/workflows/release.yml` — no signing, no provenance, no scanning
Current steps: `checkout` → `setup-go` → `setup-qemu` → `setup-buildx` → `docker/login-action` → `goreleaser`.

Missing:
- `sigstore/cosign-installer@v3` + `cosign sign` after image push
- `slsa-framework/slsa-github-generator` for SLSA Level 3 provenance
- `aquasecurity/trivy-action` in `ci.yml` after `make docker`

### `docs/SECURITY.md` — does not exist
No instructions for end users to verify artifact signatures or inspect SBOMs.

---

## 13.2 — Shell Security Model

**Confirmed: partial implementation, zero tests, no docs.**

### What exists (`internal/shell/interpreter.go`)
- `context.WithTimeout(context.Background(), 30*time.Second)` — timeout is enforced
- `common.LimitWriter{Limit: 128 * 1024 * 1024}` — 128 MB per stream cap
- `openHandler` calls `common.SecurePath(path, base)` — path confinement via cwd

### What is missing

**`KOREGO_SHELL_TIMEOUT` is hardcoded, not env-driven.** The wiki states it is configurable;
the code does not read any environment variable. The timeout is always 30 seconds.

Fix (single line):
```go
timeout := 30 * time.Second
if s := os.Getenv("KOREGO_SHELL_TIMEOUT"); s != "" {
    if d, err := time.ParseDuration(s); err == nil {
        timeout = d
    }
}
ctx, cancel := context.WithTimeout(context.Background(), timeout)
```

**`internal/shell/interpreter_test.go` does not exist.** The wiki requires:
- `TestTimeout` — `sleep 60` must error within the timeout
- `TestResourceLimits` — output-heavy script must be truncated by `LimitWriter`
- `TestPathEscape` — script attempting to open `../../../etc/passwd` must be blocked

**`docs/SECURITY.md` does not exist.** Operators have no written contract for safe deployment.
Required content: trust model, accessible resources (filesystem, env, network), enforced limits,
socket permission recommendations.

---

## 13.3 — Coverage Gate

**Confirmed: gate exits 0 regardless of coverage. Actual coverage is 41.6%.**

### The gate (`ci.yml` coverage step)
```bash
if (( $(echo "$COVERED < 50" | bc -l) )); then
  echo "::warning::Coverage ${COVERED}% is below 50% threshold"
fi
# ← no exit 1
```
A commit that drops coverage to 0% passes CI. The warning is invisible in the default
GitHub Actions UI unless the step is expanded.

### Actual per-package coverage

| Package | Coverage |
|---------|---------|
| `internal/daemon` | 3.3% |
| `pkg/tee` | 3.3% |
| `pkg/tail` | 10.3% |
| `pkg/head` | 11.2% |
| `pkg/grep` | 12.4% |
| `pkg/dirname` | 14.3% |
| `pkg/echo` | 20.8% |
| `pkg/touch` | 20.0% |
| `pkg/basename` | 23.8% |
| `pkg/sort` | 23.4% |
| `pkg/whoami` | 26.3% |
| `pkg/pwd` | 30.4% |
| `pkg/sed` | 31.6% |
| `pkg/rm` | 35.3% |
| ... | ... |
| `pkg/chown` | 89.7% |
| `pkg/printf` | 89.6% |
| `pkg/truefalse` | 100.0% |
| **Overall** | **41.6%** |

### Minimum fix (gate enforcement)
```bash
# ci.yml — change warning to hard failure
if (( $(echo "$COVERED < 60" | bc -l) )); then
  echo "::error::Coverage ${COVERED}% is below 60% threshold"
  exit 1
fi
```

Note: at 41.6% current, enforcing 60% immediately would break CI. A staged approach
(enforce 45% first, add tests to high-ROI packages, then raise to 60%) is detailed in
[12_road_to_gold.md](12_road_to_gold.md) (12.3).

### Highest-ROI packages to test first
`pkg/head`, `pkg/tail`, `pkg/grep`, `pkg/dirname`, `pkg/echo` — all have trivial
input/output contracts and can reach 70%+ with a few table-driven test cases each.

---

## 13.4 — BusyBox CI/Local Discrepancy

**Confirmed: most old-style tests run against system BusyBox on CI, not KoreGo.**

`test/busybox_testsuite/runtest` creates a `busybox → korego` symlink only inside the
`tar` and `gzip`/`gunzip` test tempdir blocks (lines ~22–36). All other old-style applet
dirs (`basename/`, `cat/`, `cp/`, `ln/`, `mv/`, `mkdir/`, `rm/`, etc.) rely on `busybox`
being found in `$PATH`, which on the CI Ubuntu runner resolves to `/usr/bin/busybox` —
the system binary, not KoreGo.

This means CI's "479 passed" count includes tests that prove system BusyBox works, not
that KoreGo's behavior matches. The true KoreGo pass rate for those cases is unknown.

### Preferred fix (Option A)
At the top of `runtest`, after `bindir` and `LINKSDIR` are set, prepend a block that
creates `busybox → korego` symlinks in a temp bin dir and prepends it to `PATH`:

```sh
# Route ALL 'busybox <applet>' calls to korego
BBDIR=$(mktemp -d)
ln -s "$bindir/korego" "$BBDIR/busybox"
export PATH="$BBDIR:$LINKSDIR:$PATH"
trap 'rm -rf "$BBDIR"' EXIT
```

### Fallback fix (Option B — CI only)
In `ci.yml`, before `make testsuite`:
```yaml
- name: Shadow system busybox with korego
  run: |
    sudo ln -sf "$PWD/korego" /usr/local/bin/busybox
```

---

## 13.5 — `awk` (Platinum Gate)

**Confirmed: `pkg/awk/` does not exist. No code at all.**

> **Full plan:** [07a_awk.md](07a_awk.md) — the canonical awk document. All task
> details and acceptance criteria live there.

---

## Verification Commands

See [12_road_to_gold.md](12_road_to_gold.md) for the full acceptance criteria and
verification commands for each gap. The audit-specific commands below validate the
code-level findings in this document:

```bash
# Confirm macOS build breakage
GOOS=darwin CGO_ENABLED=0 go build ./... 2>&1 | grep "undefined"

# Confirm KOREGO_SHELL_TIMEOUT is not read from env
grep -r "KOREGO_SHELL_TIMEOUT" internal/shell/

# Confirm coverage gate is warning-only
grep -A5 "COVERED" .github/workflows/ci.yml

# Confirm old-style tests hit system busybox
grep -r "busybox" test/busybox_testsuite/runtest | head -5
```
