---
status: legacy
description: "Phase 25: Awesome-Go submission plan and checklist (merged)."
---

# Phase 25 — Awesome-Go Submission Plan & Checklist

> **Version:** 5.7 | **Date:** 2026-10-03 | **Tier:** GOLD | **Status:** MERGED

This document outlines the preparation, checklist validation, and exact content needed to submit **GoPOSIX** to the curated [awesome-go](https://github.com/avelino/awesome-go) repository.

---

## 1. Submission Metadata & Checklist

### Forge & Service Links
- **Forge Link (GitHub)**: `https://github.com/ramayac/goposix`
- **pkg.go.dev**: `https://pkg.go.dev/github.com/ramayac/goposix`
- **golangci-lint**: `https://golangci-lint.run/`
- **Coverage Service (Codecov)**: `https://app.codecov.io/gh/ramayac/goposix`

### Repository Requirements

| Requirement | Status | Verification / Action taken |
| :--- | :---: | :--- |
| **`go.mod` file & SemVer releases** | **PASS** | Validated `go.mod` exists; tags range from `v1.0.0` to `v1.0.14`. |
| **Open source license** | **PASS** | Added standard `LICENSE` (MIT) to the root directory. |
| **Documentation links** | **PASS** | Added `pkg.go.dev`, `golangci-lint`, and `codecov` badges directly to `README.md`. |
| **Grade A- or better on Go Report Card** | **PASS** | Fixed all 25 `staticcheck` static analysis warnings across all packages. |
| **Continuous Integration (CI)** | **PASS** | GitHub Actions pipeline configured (`ci.yml`) runs on every commit. |
| **CI runs and gates tests** | **PASS** | CI gates `make vet`, `make test`, `make cover-gate` (coverage ≥70%), and BusyBox Parity tests. |

---

## 2. Awesome-Go Pull Request Content

To submit GoPOSIX, create a pull request on the [avelino/awesome-go](https://github.com/avelino/awesome-go) repository.

## 3. Submission outcome (2026-10-03)

- PR [avelino/awesome-go#6345](https://github.com/avelino/awesome-go/pull/6345) merged 2026-10-03. Link text `[GoPOSIX]`, Command Line section, alphabetical spot.
- Review asked about the Codecov gap (78.2% shown vs ~84% local `go test -cover`). Root cause: Codecov counts source **lines**, Go counts **statements**, plus two untestable entry points (`cmd/goposix/main.go`, `test/benchmark/bench_client/main.go`) counted at 0%. Resolved in [PR #39](https://github.com/ramayac/GoPOSIX/pull/39): `codecov.yml` ignores + ~1,400 lines of tests → 80.07%, plus Trivy x/crypto bump (CVE-2026-56854).
