# Phase 25 — Awesome-Go Submission Plan & Checklist

> **Version:** 5.6 | **Date:** 2026-05-22 | **Tier:** GOLD | **Status:** READY FOR SUBMISSION

This document outlines the preparation, checklist validation, and exact content needed to submit **GoPOSIX** to the curated [awesome-go](https://github.com/avelino/awesome-go) repository.

---

## 1. Submission Metadata & Checklist

### Forge & Service Links
- **Forge Link (GitHub)**: `https://github.com/ramayac/goposix`
- **pkg.go.dev**: `https://pkg.go.dev/github.com/ramayac/goposix`
- **goreportcard.com**: `https://goreportcard.com/report/github.com/ramayac/goposix`
- **Coverage Service (Codecov)**: `https://app.codecov.io/gh/ramayac/goposix`

### Repository Requirements

| Requirement | Status | Verification / Action taken |
| :--- | :---: | :--- |
| **`go.mod` file & SemVer releases** | **PASS** | Validated `go.mod` exists; tags range from `v1.0.0` to `v1.0.14`. |
| **Open source license** | **PASS** | Added standard `LICENSE` (MIT) to the root directory. |
| **Documentation links** | **PASS** | Added `pkg.go.dev`, `goreportcard`, and `codecov` badges directly to `README.md`. |
| **Grade A- or better on Go Report Card** | **PASS** | Fixed all 25 `staticcheck` static analysis warnings across all packages. |
| **Continuous Integration (CI)** | **PASS** | GitHub Actions pipeline configured (`ci.yml`) runs on every commit. |
| **CI runs and gates tests** | **PASS** | CI gates `make vet`, `make test`, `make cover-gate` (coverage ≥70%), and BusyBox Parity tests. |

---

## 2. Awesome-Go Pull Request Content

To submit GoPOSIX, create a pull request on the [avelino/awesome-go](https://github.com/avelino/awesome-go) repository.
