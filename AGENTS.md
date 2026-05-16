# Agent Context & Directives for KoreGo

**Hello AI Assistant!** you are working on **KoreGo**. This document provides the critical context, architectural invariants, and workflow rules required to contribute successfully to this project. 

## 1. Project Identity & Goal

KoreGo is a 100% Go-native, POSIX-compliant userland designed to run inside a Docker `FROM scratch` container. It serves as a modern replacement for GNU Coreutils CLI tools by compiling down to a single multicall binary (like BusyBox).

Crucially, KoreGo is designed for **Agentic Runtimes**:
1. Every utility supports structured machine-readable output via a `--json` flag.
2. It will eventually feature a persistent JSON-RPC daemon to avoid continuous process-spawning overhead.

## 2. Strict Architectural Invariants

Whenever you write or modify code in this repository, you **MUST** adhere to the following rules:

- **No CGO:** The project must compile completely statically to run in a scratch container. Always use `CGO_ENABLED=0`.
- **Zero Dependencies:** Avoid external Go modules unless absolutely necessary (e.g., a complex shell interpreter later on). Do not use external libraries for flag parsing, colors, or utility logic.
- **Unified Flag Parsing:** Use the custom POSIX-compliant parser in `pkg/common/flags.go` (`common.ParseFlags`). **Do not use the standard library `flag` package** or `pflag`. Our parser supports short flag grouping (`-laR`) and standard POSIX conventions.
- **Standardized Output:** Use the `common.Render()` and `common.RenderError()` functions in `pkg/common/output.go` to handle both standard text output and `--json` structured output. You must pass the `out io.Writer` provided in the `Run` function signature instead of using `os.Stdout`.
- **Multicall Dispatch:** Every utility lives in its own package under `pkg/` (e.g., `pkg/ls`, `pkg/echo`). Utilities register themselves automatically by calling `dispatch.Register()` in their `init()` function.

## 3. Component Structure

- `cmd/korego/main.go`: The multicall entry point. Handles symlink invocation (e.g., `/bin/ls -> /bin/korego`) and subcommand invocation (`korego ls`).
- `internal/dispatch/`: The command registry.
- `pkg/common/`: Foundation libraries (flags, JSON envelope, JSON-RPC types).
- `pkg/<utility>/`: Implementation of specific POSIX utilities (e.g., `pkg/cat/`, `pkg/ls/`).
- `test/compliance/`: Bash scripts that compare KoreGo's output and exit codes against the host OS (GNU/Linux) equivalents.
- `docker/`: Dockerfiles for the production `scratch` image and the `alpine` debug image.

## 4. Development Workflow

When implementing a new utility or feature, follow this checklist:

1. **Implement the Logic:** Write the utility in `pkg/<name>/<name>.go`.
2. **Library Layer vs CLI Layer:** Separate the core logic from the CLI parsing/printing so the core logic can be tested and reused easily by the JSON-RPC daemon.
3. **Unit Tests:** Write robust unit tests in `pkg/<name>/<name>_test.go` targeting > 80% coverage.
4. **Compliance Tests:** Add a test script in `test/compliance/test_<name>.sh`. Use `set -uo pipefail` (do NOT use `set -e`, as non-zero exit codes from utilities are expected and should be captured).
5. **Registration:** 
   - Add a blank import for the package in `cmd/korego/main.go`.
   - Add the package to the `PKG_DIRS` variable in the `Makefile`.
   - Add the compliance script to the `compliance` target in the `Makefile`.
6. **Verification:**
   - Run `make all` to build and run unit tests.
   - Run `make compliance` to verify POSIX behavior against the system.
   - Run `make ci` to run the full pipeline including Docker builds.
7. **Documentation:** Update the corresponding Phase plan in the `wiki/` directory (e.g., check off the task list).

## 5. Security & Safety

- **Root Protection:** Utilities that perform destructive operations (like `rm`) must include guards against destroying the root filesystem (e.g., `rm -rf /` must be refused without `--no-preserve-root`).
- **BusyBox Test Suite:** 477 passed, 3 failed, 10 skipped (99.4% pass rate). The 3 remaining failures are all in the `date` utility (2 Go POSIX TZ limitations, 1 cosmetic error-format mismatch). Run `make testsuite` before every commit to prevent regressions.

## 6. Current State & Progression

Refer to the Phase documents in `wiki/` (e.g., `wiki/plan_updated.md`) to understand the current task.
- **Phase 00 & 01:** Foundation & Tier 1 (echo, true, false, env, pwd, etc.) — **COMPLETED**
- **Phase 02:** Docker CI & Scratch pipeline — **COMPLETED**
- **Phase 03:** Filesystem Utils (ls, cat, rm, cp, mv, etc.) — **COMPLETED**
- **Phase 04:** Text Utils (grep, sed, sort, wc, etc.) — **COMPLETED**
- **Phase 05:** JSON-RPC Daemon Core — **COMPLETED**
- **Phase 06:** System & Process Utils (ps, find, df, du, etc.) — **COMPLETED**
- **Phase 07:** Agent-Ready Features (diff, tar, shell) — **COMPLETED**
- **Phase 08:** Security Hardening — **COMPLETED**
- **Phase 09:** Release & Automation — **COMPLETED**
- **Phase 10:** POSIX Test Framework — **COMPLETED**

## 7. Docker & Containerization Insights

- **Go Version Alignment:** Always ensure the `golang` base image version in `docker/Dockerfile*` matches or exceeds the `go` version specified in `go.mod`. Failing to do so will break the build during `go mod download`.
- **Debug Image Flexibility:** Use `CMD ["/bin/sh"]` instead of `ENTRYPOINT` in debug images. This allows `docker run -it korego:debug sh` to work as expected, rather than passing `sh` as an argument to the `korego` multicall binary.
- **Scratch Image Purity:** When generating symlinks in a multi-stage Docker build, do **not** `COPY --from=stage /bin/ /bin/`. This pulls in all host OS binaries (like Alpine's BusyBox). Instead, create a dedicated output directory (e.g., `/out/bin`) in the intermediate stage and copy only that to the final `scratch` image.
- **Testing Production:** Use `make smoke-docker` to verify the production image. Use `make docker-run CMD="ls -la"` for ad-hoc testing of specific utilities inside the minimal `scratch` environment.

## 8. BusyBox Test Suite Insights & Agent Learnings

While running and porting the BusyBox test suite to KoreGo, be aware of the following implicit assumptions the suite makes about the utilities it tests:

- **Formatting Rigidity:** Utilities like `wc` must not emit leading padding (e.g., `%7d`), as tests often compare raw string matches against expected output (e.g., `8185` vs `   8185`). 
- **Binary Data Parsing (`NUL` bytes):** The BusyBox test suite actively tests embedded `NUL` bytes (e.g., passing `he\0llo` to `sed` commands). Be careful when parsing text files or command arguments in Go. Do not use standard C-style `0` byte checks as an EOF marker or early-termination signal in parsers (like the `sed` AST builder), because literal `NUL` bytes are valid inputs.
- **Harness Dependencies (`echo -e`):** The `testing.sh` harness often relies on `echo` to generate binary payloads. If `ECHO="korego echo"` is used, ensure `korego echo` fully implements octal (`\0NNN`) and hexadecimal (`\xNN`) escapes. Otherwise, the tests will generate literal backslashes, leading to cascading false-positive failures in downstream tools like `sed` and `grep`.
- **Flag Pre-processing (`find`):** The custom `common.ParseFlags` expects double-dash long flags (`--name`). For tools that use single-dash long flags (like `find -name -exec {} \;`), an argument pre-processing step is required before passing arguments to the flag parser to ensure compatibility without breaking standard POSIX flag logic. `-exec` must be captured as a unit before the flag parser sees it to avoid treating it as bundled short flags (`-e -x -e -c`).

**Always read the active Phase document before writing code!**
