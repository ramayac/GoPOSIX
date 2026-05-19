# Contributing to GoPOSIX

GoPOSIX is a Go-native POSIX userland. Contributions must adhere to the architectural invariants
documented in [AGENTS.md](AGENTS.md). Please read that file before contributing.

## Quick Start

```bash
make all        # vet + test + build
make test       # unit tests (REQUIRED before every commit)
make testsuite  # BusyBox integration tests (REQUIRED before every commit)
make ci         # full pipeline (test + testsuite + coverage + docker)
```

## Adding a New Utility

Follow this checklist when implementing a new POSIX utility:

1. **Implement the Logic:** Write the utility in `pkg/<name>/<name>.go`.
2. **Separate Library from CLI:** Core logic in an exported `Run()` function; CLI
   parsing and output in a `run()` function. This enables both unit testing and
   JSON-RPC daemon reuse.
3. **Use the Common Library:**
   - Flag parsing: `common.ParseFlags()` in `pkg/common/flags.go`
   - Output: `common.Render()` / `common.RenderError()` in `pkg/common/output.go`
   - Always use the injected `out io.Writer` — never write directly to `os.Stdout`
   - `--json` must use `{Long: "json", Type: common.FlagBool}` — **no short `-j`**
4. **Unit Tests:** Write tests in `pkg/<name>/<name>_test.go` targeting ≥70% coverage.
5. **BusyBox Tests:** Add a compliance test in `test/busybox_testsuite/<name>/`.
6. **Registration:**
   - Add a blank import in `cmd/goposix/main.go`
   - Add the package to `PKG_DIRS` in the `Makefile`
7. **Documentation:** Update the relevant Phase plan in `wiki/`.
8. **Verify:** Run `make all && make testsuite && make ci`.

## Coverage Policy

- **Gate:** CI enforces ≥70% overall coverage (`COVERAGE_THRESHOLD` in Makefile).
- **Current coverage:** 75.7% (as of 2026-05-18).
- **Per-package:** No package should be below 60%. Target ≥70% for all.
- **SDK Coverage:** The Go SDK client (`pkg/client/`) has typed methods for all 77 utilities.
  When adding new utilities, add the corresponding typed method to the SDK and test coverage
  for it in `pkg/client/client_test.go`.
- Check per-package coverage: `make cover-pkg`

## BusyBox Test Suite

The BusyBox test suite is the regression gate — it chains utilities together and
catches cascading failures that unit tests miss. Run it before every commit:

```bash
make testsuite
```

Current baseline: **548 passed, 4 failed, 10 skipped** (99.3%). The 4 failures are
in `date` (3, Go TZ limitations) and `fold` (1, NUL handling). PRs must not
introduce new failures.

## Code Style

- `gofmt` compliance required (check with `make fmt-check`)
- Pass `go vet` (`make vet`)
- Zero CGO (`CGO_ENABLED=0` — the binary must be fully static)
- Prefer the standard library; external dependencies need strong justification

## Commit Guidelines

- Commits should be atomic and self-contained
- Run `make test && make testsuite` before committing
- Reference Phase/issue numbers in commit messages where applicable

## Security

Destructive utilities (`rm`, etc.) must include root filesystem protection.
See [wiki/security.md](wiki/security.md) for the full security model.

## Questions?

Check the [wiki/](wiki/) directory for phase plans and the
[test_coverage_matrix.md](wiki/test_coverage_matrix.md) for per-utility test status.
