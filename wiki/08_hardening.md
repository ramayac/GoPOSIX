# Phase 08 — Production Hardening & Security

> **Timeline:** Week 11–12 | **Depends on:** Phase 07

---

## Goal

Harden the daemon and container for production use. Security audit, resource limits, and stability testing.

## Tasks

### 08.1 — Security Audit

- [x] **Path traversal:** All file operations validate paths. No `../../etc/shadow` escapes.
- [x] **Shell sandbox (design + implementation):** `korego.shell.exec` runs with:
  - No network access (no `net.Dial` exposed)
  - Restricted filesystem (configurable allowed paths via `SecurePath`)
  - Execution timeout (30s hardcoded in `interpreter.go`)
  - Memory limit (128MB per stream via `LimitWriter`)
  > **Note:** The sandbox is implemented but lacks tests and formal documentation.
  > Wiring `KOREGO_SHELL_TIMEOUT` from env, writing `internal/shell/interpreter_test.go`,
  > and creating `docs/SECURITY.md` are tracked in [12_road_to_gold.md](12_road_to_gold.md) (12.2).
- [x] **Rate limiting:** Max 100 RPC requests/sec per connection (configurable)
- [x] **Input validation:** All RPC params validated against expected types/ranges
- [x] **Symlink following:** Configurable — refuse to follow symlinks outside allowed paths

### 08.2 — Non-Root Container

- [x] Create user `korego:1000` in builder stage
- [x] Copy `/etc/passwd` and `/etc/group` to scratch image
- [x] `USER korego` in Dockerfile
- [x] Socket created with proper group permissions
- [x] Test: daemon works as non-root

### 08.3 — Resource Management

- [x] Session TTL with automatic cleanup goroutine
- [x] Connection limits (max concurrent connections)
- [x] Request body size limit (default 1MB)
- [x] Response size limit for `--json` (paginate large results)
- [x] Memory profiling with `pprof` endpoint (debug mode only)

### 08.4 — Stability Testing

- [x] 24-hour soak test under moderate load
- [x] Memory leak detection (`go test -race`, `pprof` heap profiles)
- [x] Graceful shutdown under load (SIGTERM during active requests)
- [x] Socket file cleanup on crash (stale socket detection on restart)
- [x] Fuzz testing on JSON-RPC parser (`go test -fuzz`)

### 08.5 — POSIX Coverage Matrix ([posix_coverage.md](posix_coverage.md))

- [x] Matrix covers all 50+ utilities (Completed, see `posix_coverage.md`)
- [x] Each utility lists: implemented flags, missing flags, known deviations
- [x] Overall compliance percentage calculated

*(Note: `awk` has been explicitly deferred from the MVP scope.)*

## Milestone 08

- [x] Security audit complete, no critical findings
- [x] Daemon runs as non-root in container
- [x] 24-hour soak test passes (no memory leaks, no crashes)
- [x] Fuzz testing on JSON-RPC parser finds no panics
- [x] POSIX coverage matrix published, >= 80% overall

## How to Verify

```bash
# Soak test
./korego daemon --socket /tmp/korego.sock &
go test -run TestSoak -timeout 25h ./test/integration/

# Fuzz
go test -fuzz FuzzJSONRPC -fuzztime 10m ./pkg/common/

# Memory profile
curl --unix-socket /tmp/korego.sock http://localhost/debug/pprof/heap > heap.prof
go tool pprof heap.prof

# Non-root test
docker run --rm --user korego korego:prod whoami  # → korego
```
