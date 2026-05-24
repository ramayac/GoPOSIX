# Agent Context & Directives for GoPOSIX

**Hello AI Assistant!** you are working on **GoPOSIX**. This document provides the critical context, architectural invariants, and workflow rules required to contribute successfully to this project. 

## 1. Project Identity & Goal

GoPOSIX is a 100% Go-native, POSIX-compliant userland designed for **programmatic consumption** in containerized environments. It runs as a persistent JSON-RPC 2.0 daemon with a typed Go SDK (60µs per RPC call, 11× faster than BusyBox fork+exec). A multicall CLI binary (like BusyBox) is also available as a secondary interface.

GoPOSIX is designed for **programmatic consumption** in containerized environments:
1. Every utility supports structured machine-readable output via a `--json` flag.
2. It features a persistent JSON-RPC 2.0 daemon to avoid continuous process-spawning overhead for repeated operations.

### 1.1 Development loop

We follow a strict, systematic process for implementing the utilities one by one. The development loop is:

```
  [ CHECK ] ──> [ TEST ] ──> [ CODE ] ──> [ PASS ] ──> [ UPDATE ]
     │             │            │            │             │
  Inspect      Create our   Write Go     Run unit &    Mark utility
  BusyBox      own unit     logic &      BusyBox       implemented
  tests        tests        register     tests         in wiki, plan or coverage matrix.

```

Each new tool must pass the BusyBox test suite for the utility, have it's own *_test.go, registered in main.go and Makefile.
You must run for each new tool:
- make test
- make testsuite
- go vet
- go fmt
the tool must have json schema defined in schema.md and a json schema test in compliance/test_<tool>.sh

## 2. Strict Architectural Invariants

Whenever you write or modify code in this repository, you **MUST** adhere to the following rules:

- **No CGO:** The project must compile completely statically to run in a scratch container. Always use `CGO_ENABLED=0`.
- **Low Dependencies:** Avoid external Go modules unless absolutely necessary (e.g., a complex shell interpreter later on).
- **Unified Flag Parsing:** Use the custom POSIX-compliant parser in `pkg/common/flags.go` (`common.ParseFlags`). **Do not use the standard library `flag` package** or `pflag`. Our parser supports short flag grouping (`-laR`) and standard POSIX conventions.
- **Standardized Output:** Use the `common.Render()` and `common.RenderError()` functions in `pkg/common/output.go` to handle both standard text output and `--json` structured output. You must pass the `out io.Writer` provided in the `Run` function signature instead of using `os.Stdout`.
- **Multicall Dispatch:** Every utility lives in its own package under `pkg/` (e.g., `pkg/ls`, `pkg/echo`). Utilities register themselves automatically by calling `dispatch.Register()` in their `init()` function.

## 3. Component Structure

- `cmd/goposix/main.go`: The multicall entry point. Handles symlink invocation (e.g., `/bin/ls -> /bin/goposix`), subcommand invocation (`goposix ls`), and daemon mode (`goposix daemon`).
- `internal/dispatch/`: The command registry.
- `internal/daemon/`: The JSON-RPC 2.0 persistent daemon server.
- `pkg/client/`: The typed Go SDK for programmatic daemon access (60µs/call).
- `pkg/common/`: Foundation libraries (flags, JSON envelope, JSON-RPC types).
- `pkg/<utility>/`: Implementation of specific POSIX utilities (e.g., `pkg/cat/`, `pkg/ls/`).
- `test/compliance/`: Bash scripts that compare GoPOSIX's output and exit codes against the host OS (GNU/Linux) equivalents.
- `docker/`: Docker configurations. Consolidates daemon, CLI, debug, and Alpine Distro targets into a single unified multi-stage `Dockerfile` (using targets `daemon`, `cli`, `debug`, and `alpine-mvp`). Pre-built GoReleaser images use `Dockerfile.goreleaser` / `Dockerfile.goreleaser.daemon`.

## 4. Development Workflow

When implementing a new utility or feature, follow this checklist:

1. **Implement the Logic:** Write the utility in `pkg/<name>/<name>.go`.
2. **Library Layer vs CLI Layer:** Separate the core logic from the CLI parsing/printing so the core logic can be tested and reused easily by the JSON-RPC daemon.
3. **Unit Tests:** Write robust unit tests in `pkg/<name>/<name>_test.go` targeting > 80% coverage.
4. **Compliance Tests:** Add a test script in `test/compliance/test_<name>.sh`. Use `set -uo pipefail` (do NOT use `set -e`, as non-zero exit codes from utilities are expected and should be captured).
5. **Registration:** 
   - Add a blank import for the package in `cmd/goposix/main.go`.
   - Add the package to the `PKG_DIRS` variable in the `Makefile`.
   - Add the compliance script to the `compliance` target in the `Makefile`.
6. **Verification:**
   - Run `make all` to build and run unit tests.
   - Run `make compliance` to verify POSIX behavior against the system.
   - Run `make ci` to run the full pipeline including Docker builds.
7. **Documentation:** Update the corresponding Phase plan in the `wiki/` directory (e.g., check off the task list).
8. **Todos and Coverage Matrix:** Update and maintain`wiki/todos.md` and keep the `wiki/test_coverage_matrix.md` with the new utility's coverage percentage and any relevant notes.
9. **Wiki**: Refer to .wiki-instructions/wiki-maintainer.md for detailed instructions on how to update the wiki with new findings, architectural notes, and phase progress. Or use .wiki-instructions/query.md to ask specific questions about the wiki content.


## 4a. Coverage Policy

- **Gate:** `make ci` enforces a hard coverage gate at **≥80%** overall (see `COVERAGE_THRESHOLD` in Makefile). PRs that drop coverage below this threshold fail CI. Current overall coverage: **80.1%**. See [wiki/13_coverage_and_hardening.md](wiki/13_coverage_and_hardening.md) for full policy.
- **CLI Layer Testing:** The `run()` function (CLI glue) must be tested, not just the library-layer `Run()`. Extract an injectable entry point (e.g., `grepRun()`, `catRun()`) that accepts `io.Reader`/`io.Writer` instead of hardcoding `os.Stdin`/`os.Stdout`. See `pkg/cat/cat.go` for the canonical `catRun()` pattern.
- **Per-package:** Use `make cover-pkg` to audit per-package coverage. No package should be below 5%.
- **Before committing:** Always run `make testsuite` (BusyBox integration tests) in addition to `make test` (unit tests). The BusyBox suite catches cascading integration failures that unit tests miss.

## 5. Security & Safety

- **Daemon-First:** The default Docker image (`goposix:latest`) starts the persistent JSON-RPC daemon. CLI access is available as a secondary interface (`goposix:cli`). The Go SDK (`pkg/client/`) is the primary programmatic interface at 60µs/call.
- **Root Protection:** Utilities that perform destructive operations (like `rm`) must include guards against destroying the root filesystem (e.g., `rm -rf /` must be refused without `--no-preserve-root`).
- **BusyBox Test Suite:** 810 passed, 45 failed, 64 skipped (94.7% pass rate, 919 total tested). Failures: 16 in `awk` (goawk engine limitations), 22 in `bc` (precision/scale differences), 7 in `tar` (3 hardlink/symlink mode ordering, 3 symlink safety, 1 XZ). `rx` has 1 flaky test. Run `make testsuite` before every commit to prevent regressions.

## 6. Current State & Progression

For current status, remaining failures, deferred work, and active priorities, see [wiki/todos.md](wiki/todos.md).
For the full phase history and roadmap, see [wiki/phases.md](wiki/phases.md).

## 7. Docker & Containerization Insights

- **Go Version Alignment:** Always ensure the `golang` base image version in `docker/Dockerfile` matches or exceeds the `go` version specified in `go.mod`. Failing to do so will break the build during `go mod download`.
- **Debug Image Flexibility:** Use `CMD ["/bin/sh"]` instead of `ENTRYPOINT` in debug images. This allows `docker run -it goposix:debug sh` to work as expected, rather than passing `sh` as an argument to the `goposix` multicall binary.
- **Scratch Image Purity:** When generating symlinks in a multi-stage Docker build, do **not** `COPY --from=stage /bin/ /bin/`. This pulls in all host OS binaries (like Alpine's BusyBox). Instead, create a dedicated output directory (e.g., `/out/bin`) in the intermediate stage and copy only that to the final `scratch` image.
- **Testing Production:** Use `make smoke-docker` to verify the production image. Use `make docker-run CMD="ls -la"` for ad-hoc testing of specific utilities inside the minimal `scratch` environment.

## 8. BusyBox Test Suite Insights & Agent Learnings

While running and porting the BusyBox test suite to GoPOSIX, be aware of the following implicit assumptions the suite makes about the utilities it tests:

- **Formatting Rigidity:** Utilities like `wc` must not emit leading padding (e.g., `%7d`), as tests often compare raw string matches against expected output (e.g., `8185` vs `   8185`). 
- **Binary Data Parsing (`NUL` bytes):** The BusyBox test suite actively tests embedded `NUL` bytes (e.g., passing `he\0llo` to `sed` commands). Be careful when parsing text files or command arguments in Go. Do not use standard C-style `0` byte checks as an EOF marker or early-termination signal in parsers (like the `sed` AST builder), because literal `NUL` bytes are valid inputs.
- **Harness Dependencies (`echo -e`):** The `testing.sh` harness often relies on `echo` to generate binary payloads. If `ECHO="goposix echo"` is used, ensure `goposix echo` fully implements octal (`\0NNN`) and hexadecimal (`\xNN`) escapes. Otherwise, the tests will generate literal backslashes, leading to cascading false-positive failures in downstream tools like `sed` and `grep`.
- **Flag Pre-processing (`find`):** The custom `common.ParseFlags` expects double-dash long flags (`--name`). For tools that use single-dash long flags (like `find -name -exec {} \;`), an argument pre-processing step is required before passing arguments to the flag parser to ensure compatibility without breaking standard POSIX flag logic. `-exec` must be captured as a unit before the flag parser sees it to avoid treating it as bundled short flags (`-e -x -e -c`).
- **Symlink collision (`sh`):** While both `shell` and its alias `sh` are registered as dispatch commands in GoPOSIX, to prevent a shadowing `/bin/sh` symlink from breaking the BusyBox test harness (which runs test cases via `sh -x -e`), the `ListCommands` function in `internal/dispatch/dispatch.go` explicitly filters out `"sh"` from being printed via `--list-commands`. This prevents the symlink generator stage from creating a `sh -> goposix` symlink while still allowing `sh` to be invoked programmatically or as a subcommand.

**Always read the active Phase document before writing code!**
