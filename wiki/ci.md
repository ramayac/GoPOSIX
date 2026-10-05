---
status: current
description: "CI pipeline: jobs, gates, and the test and Codecov wiring."
references: [source:.github/workflows/ci.yml, source:.github/workflows/lint.yml, source:test/testutil/rpc.go]
---

# CI Pipeline

Two GitHub Actions workflows gate every pull request. Both ignore Markdown-only
changes (`paths-ignore: '**.md'`).

| Workflow | File | Purpose |
|----------|------|---------|
| CI | `.github/workflows/ci.yml` | Vet, unit tests, coverage, Docker smoke, Trivy scan, BusyBox suite |
| golangci-lint | `.github/workflows/lint.yml` | Static analysis (`only-new-issues: true`) |

## CI job: Lint / Test / Docker smoke

The job runs these steps in order. A failure stops the job.

1. **Vet** — `make vet`.
2. **Unit tests** — `gotestsum` runs `go test` for all packages and writes `test-results.xml`.
3. **Coverage check** — `make cover-gate` enforces `COVERAGE_THRESHOLD` (80%) from the `Makefile`.
4. **Upload coverage** — `codecov/codecov-action@v5`.
5. **Upload test results** — the same action with `report_type: test_results`, `files: ./test-results.xml`, and `fail_ci_if_error: false`.
6. **Build binary** — `make build`.
7. **Binary size gate** — fails when `./goposix` exceeds 15 MB (15728640 bytes).
8. **Verify `--list-commands`** — the symlink generator depends on this output.
9. **Validate JSON schemas** — `make validate-schemas`.
10. **Build Docker image** — `make docker`.
11. **Smoke tests** — `make smoke` runs inside the container.
12. **Vulnerability scan** — Trivy fails on CRITICAL or HIGH findings.
13. **BusyBox Test Suite** — `make testsuite`; fails when the pass count drops below 750.

## Test harness

- Unit tests live next to each package as `*_test.go`.
- In-process daemon tests reach the daemon over JSON-RPC with the minimal client in
  `test/testutil` (`Dial`, `Call`, `Close`). It is one connection, serialized by a mutex.
- The JSON-RPC contract tests live in `test/posix-json/`, one file per tier.
- The benchmark client is `test/benchmark/bench_client`, built as `bench-rpc-client`.
  It measures a persistent JSON-RPC connection, not socat.

## Codecov notes

- `codecov/test-results-action` is deprecated. Use `codecov-action@v5` twice: once for
  coverage, once with `report_type: test_results`.
- `test-results.xml` comes from `gotestsum`. Without a JUnit file there is nothing for
  Codecov to ingest.
- Keep the test-results upload non-fatal. A Codecov outage must not fail CI.

## Known follow-ups

- `gotestsum` is installed with `@latest`. Pin the version for reproducible CI.
- GitHub warns that Node.js 20 actions are deprecated. Watch `actions/checkout`,
  `actions/setup-go`, and `golangci-lint-action` for newer majors.

---

## See Also

- [index.md](index.md) | Wiki index.
- [test_coverage_matrix.md](test_coverage_matrix.md) | Coverage and BusyBox status.
- [repo-map.md](repo-map.md) | Repo layout and build targets.
- [architecture.md](architecture.md) | Component flow and packages.
