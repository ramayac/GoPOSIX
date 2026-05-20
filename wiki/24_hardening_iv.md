# Hardening IV: Architecture and Compliance Gaps

## 1. Architecture & Elegance Gaps

### Flag Parsing Friction (Leaky Abstraction)
- **Issue**: The strict custom flag parser (`pkg/common/flags.go`) expects double-dash long flags (`--name`) and standard short flags. POSIX utilities are notoriously inconsistent (e.g., `find -name` using single dashes, `tar xvf` using no dashes, `dd if=file` using key-value pairs). 
- **Impact**: We currently have to "pre-process" arguments before feeding them to the generic flag parser. This is a brittle abstraction that requires constant vigilance as new utilities are ported, increasing the risk of breaking standard POSIX flag logic.
- **Action Item**: Consider extending `pkg/common/flags.go` to natively support utility-specific flag semantics (e.g., single-dash long flags, `dd`-style arguments) without requiring per-utility pre-processing logic.

## 2. Code Quality & Idiomatic Go Gaps

### Daemon Worker "Pool" Allocation
- **Issue**: In `internal/daemon/server.go`, the `WorkerPool` implementation relies on a bounded semaphore channel to cap concurrency (`wp.sem <- struct{}{}`), but still spawns a *new* goroutine (`go func() {...}`) for every single request.
- **Impact**: While goroutines are lightweight, at 60µs per call, the garbage collection (GC) overhead of constantly allocating and destroying goroutines under heavy load will eventually bottleneck JSON-RPC throughput.
- **Action Item**: Refactor the `WorkerPool` into a true thread-pool pattern where a fixed number of worker goroutines sit in a `for` loop, pulling requests from a job channel.

### Bloated Connection Handler
- **Issue**: The `handleConn` function in `server.go` mixes rate-limiting, session management, and JSON decoding directly within the same block.
- **Impact**: Poor separation of concerns makes testing and extending the JSON-RPC daemon difficult.
- **Action Item**: Refactor `handleConn` into a clear middleware chain (e.g., connection -> rate limiter -> session manager -> payload decoder).

## 3. POSIX Compliance & Robustness Gaps

### The NUL Byte / String Assumption
- **Issue**: Go inherently treats strings as UTF-8 sequences. However, POSIX pipelines frequently pass arbitrary binary data or NUL-separated records (e.g., `\0`).
- **Impact**: Utilities like `fold`, `sed`, or `awk` may fail when processing streams that rely on literal NUL bytes or binary data, as they depend on Go's `string` scanning rather than raw `[]byte` buffering. The test `fold with NULs` is already failing due to this.
- **Action Item**: Audit all text-processing utilities to ensure they use `[]byte` where binary data or NUL bytes are possible, avoiding standard C-style `0` byte checks as EOF markers.

### Timezone Parsing Limitations
- **Issue**: Go's standard `time` package does not fully support parsing complex POSIX `TZ` strings (e.g., `TZ=XYZ+1:00`).
- **Impact**: The `date` utility fails several BusyBox tests (`date-@-works`, `date-timezone`) because it relies on standard Go time parsing.
- **Action Item**: Implement or import a custom POSIX TZ parser for `date` to achieve 100% compliance.

### Missing JSON-RPC Stream Tests
- **Issue**: `tee` and `tr` lack explicit JSON-RPC daemon integration tests in `test/posix-json/`. 
- **Impact**: Since these utilities heavily manipulate `stdin` streams, daemonizing them is complex (multiplexing streams over JSON-RPC) and prone to deadlocks. Without explicit daemon integration tests, their success paths over JSON-RPC are unverified.
- **Action Item**: Add dedicated tests for `tee` and `tr` traversing the daemon in JSON-RPC mode.

## 4. Performance & Scalability Gaps

### OOM Ceiling via `LimitWriter`
- **Issue**: `server.go` enforces a hardcoded 50MB `LimitWriter` on the output buffer to prevent Out-Of-Memory (OOM) crashes during JSON-RPC responses.
- **Impact**: Running utilities that produce massive output (e.g., `cat large_file.log` or `grep` over huge files) over the JSON-RPC daemon will silently truncate or fail when hitting the 50MB ceiling.
- **Action Item**: Implement streaming JSON-RPC responses, chunking, or ndjson over the socket for payloads exceeding a certain threshold, rather than caching everything in an in-memory buffer.

## 5. Security & Container Readiness Gaps

### Socket Trust & Multi-Tenancy
- **Issue**: The daemon exposes a Unix socket (`0660`) that acts as a god-object. 
- **Impact**: If an untrusted tenant gains access to this socket, they have full access to `goposix.shell.exec` and other potentially destructive commands. There is currently no Role-Based Access Control (RBAC) governing which utilities can be called over the socket.
- **Action Item**: Implement the deferred multi-tenant sandbox model, including socket authentication and utility-level RBAC.

### Symlink Pollution
- **Issue**: The multicall binary approach and test harnesses rely on creating symlinks (e.g., `/bin/sh -> goposix`).
- **Impact**: Accidental or uncontrolled symlink creation can shadow host OS binaries if volume-mounted incorrectly, breaking downstream host scripts.
- **Action Item**: Ensure `goposix` never registers destructive symlinks (like `sh`) globally by default, and provide isolated environments for test suites to prevent host environment corruption.
