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

### Proposed Category
We recommend adding GoPOSIX under the **"System"** or **"Command Line"** category. 

### Alphabetical Order Context
When editing `README.md` on the awesome-go repository, place the entry in **alphabetical order**. 
For example, under the **System** category:

```markdown
* [gops](https://github.com/google/gops) - A tool to list and diagnose Go processes currently running on your system.
* [GoPOSIX](https://github.com/ramayac/goposix) - 100% Go-native, POSIX-compliant userland and persistent JSON-RPC 2.0 daemon for containerized environments.
* [gosec](https://github.com/securego/gosec) - Security Checker for Go source code.
```

### Exact Markdown Line to Add
```markdown
* [GoPOSIX](https://github.com/ramayac/goposix) - 100% Go-native, POSIX-compliant userland and persistent JSON-RPC 2.0 daemon for containerized environments.
```

> [!NOTE]
> The description matches the quality standards: it is clear, concise, non-promotional, and **ends with a period**.

---

## 3. Pull Request Submission Checklist

When opening the PR on `awesome-go`, copy-paste and check off the following sections:

```markdown
- [x] Forge link (github.com, gitlab.com, etc): https://github.com/ramayac/goposix
- [x] pkg.go.dev: https://pkg.go.dev/github.com/ramayac/goposix
- [x] goreportcard.com: https://goreportcard.com/report/github.com/ramayac/goposix
- [x] Coverage service link (codecov, coveralls, etc.): https://app.codecov.io/gh/ramayac/goposix

## Pre-submission checklist

- [x] I have read the Contribution Guidelines
- [x] I have read the Quality Standards

## Repository requirements

- [x] The repo has a go.mod file and at least one SemVer release (vX.Y.Z).
- [x] The repo has an open source license.
- [x] The repo documentation has a pkg.go.dev link.
- [x] The repo documentation has a goreportcard link (grade A- or better).
- [x] The repo documentation has a coverage service link.

- [x] The repo has a continuous integration process (GitHub Actions, etc.).
- [x] CI runs tests that must pass before merging.

## Pull Request content

- [x] This PR adds/removes/changes only one package.
- [x] The package has been added in alphabetical order.
- [x] The link text is the exact project name.
- [x] The description is clear, concise, non-promotional, and ends with a period.
- [x] The link in README.md matches the forge link above.

## Category quality

- [x] The packages around my addition still meet the Quality Standards.
```
