# Phase 23 — Multi-Tenant Sandbox (Linux Experiment Security)

> **Status:** DEFERRED | **Date:** 2026-05-18 | **Deferred:** 2026-05-18
>
> **Goal:** Transform GoPOSIX from a single-tenant daemon into a secure multi-tenant
> environment where each RPC session has isolated filesystem, resource limits, command
> restrictions, and audit trail — enabling browser-based Linux playgrounds, CTF platforms,
> coding interview tools, and container-native sandboxes.
>
> **Deferral rationale:** The current primary use case is single-tenant, multi-agent
> collaboration on a shared filesystem (one user, many trusted agents operating on the
> same repo). Workspace isolation would break the shared-repo model. The observability
> and audit trail components of this phase have been extracted into
> [Phase 24 — Multi-Agent Observability](24_multi_agent_observability.md). The
> multi-tenant isolation features (workspace isolation, command allowlist, subprocess
> sandboxing) are deferred until a concrete untrusted-user use case emerges.

---

## 1. Why This Matters

GoPOSIX's daemon architecture is uniquely suited for multi-tenant command execution:

| Property | GoPOSIX | Traditional (Docker-per-user) |
|----------|---------|-------------------------------|
| Per-user overhead | One goroutine (~4KB stack) | One container (~10MB+) |
| Startup latency | Session create = 8-byte random ID | Container create = 500ms+ |
| Concurrent users | Thousands on one daemon | Limited by kernel process table |
| Output format | Structured JSON, auditable | Raw text, must parse |
| Command control | Allowlist at daemon level | Must configure per-container |

But the current implementation has **no multi-tenant isolation** — all sessions share one filesystem, one set of resources, and one unrestricted command set. This phase closes those gaps.

---

## 2. Current Security Posture (Already Exists)

These primitives are solid and need no changes:

| Feature | Location | Detail |
|---------|----------|--------|
| Session manager | `internal/daemon/session.go` | Random session IDs, CWD tracking, env vars, TTL cleanup |
| Path traversal prevention | `pkg/common/security.go:SecurePath()` | Blocks `../` escape from base directory |
| Output limiting | `common.LimitWriter` (50MB daemon, 128MB shell, 50MB response) | Prevents OOM from infinite output |
| Input limiting | `io.LimitReader` (1MB RPC, 256MB text utils) | Prevents memory exhaustion from large inputs |
| Rate limiting | Token bucket, 100K req/s per connection | Prevents DoS |
| Non-root default | UID 1000 | No privilege escalation |
| Root protection | `rm --no-preserve-root` guard | Can't recursively remove `/` |
| Structured output | `--json` on all 77 utilities | Parseable, auditable, injection-safe |
| Single process | Daemon handles all requests via goroutines | No fork bombs, no process-table exhaustion |

---

## 3. Gap Analysis

### 🔴 23.1 — Filesystem Isolation Per Session (CRITICAL)

**Problem:** All sessions share `/`. User A runs `echo secret > /tmp/pass` and User B runs `cat /tmp/pass`.

**Current state:** `SecurePath` exists but always resolves against `"/"` as base because `session.CWD` defaults to `"/"`. No per-session namespace.

**Design:** Add a `Workspace` field to `Session`. On creation, allocate a private directory under `/var/lib/goposix/sessions/<session-id>/`. All path operations resolve against this workspace. `ls /` shows the workspace root, not the host root.

```go
type Session struct {
    ID         string
    CWD        string            // relative to Workspace
    Workspace  string            // e.g., /var/lib/goposix/sessions/abc123/
    Env        map[string]string
    LastActive time.Time
}

func (sm *SessionManager) Create() *Session {
    id := randomHex(8)
    ws := filepath.Join("/var/lib/goposix/sessions", id)
    os.MkdirAll(ws, 0700)
    // Seed with standard directory structure.
    os.MkdirAll(filepath.Join(ws, "tmp"), 01777)
    os.MkdirAll(filepath.Join(ws, "home"), 0755)
    return &Session{ID: id, Workspace: ws, CWD: "/home", ...}
}
```

**Files changed:** `internal/daemon/session.go`, `internal/daemon/server.go` (resolve paths against workspace)

**Risk:** Utilities that access absolute system paths (`/etc/hostname`, `/proc`, `/sys`) need a **bind-mount** of read-only system directories into each workspace, or a **fallback** that allows read-only access to specific host paths.

### 🟠 23.2 — Command Allowlist (HIGH)

**Problem:** Any RPC client can call any utility, including `rm -rf`, `kill`, `chmod`, `sh`, `daemon stop`.

**Design:** `--allowed-commands` flag on daemon startup.

```go
// In server.go:
type Server struct {
    // ...
    allowedCommands map[string]bool  // nil = allow all (current behavior)
}

func (s *Server) checkCommand(method string) bool {
    if s.allowedCommands == nil {
        return true
    }
    cmdName := strings.TrimPrefix(method, "goposix.")
    return s.allowedCommands[cmdName]
}
```

**Default allowlist for "Linux experiment" use case:**
```
ls, cat, echo, pwd, whoami, hostname, uname, date, env, printenv,
wc, head, tail, grep, sort, uniq, cut, tr, nl, wc, tee, fold,
mkdir, rmdir, touch, cp, stat, readlink, basename, dirname,
find, du, df, ps, sleep, true, false, yes, test, expr, od, printf
```

**Blocklist (default blocked for experiments):**
```
rm, mv, kill, chmod, chown, chgrp, ln, link, unlink,
tar, gzip, sha256sum, md5sum, cksum, sum,
sh, shell, daemon, nice, nohup, xargs, logger
```

**Files changed:** `internal/daemon/server.go`, `pkg/daemon/daemon.go` (new flag)

### 🟠 23.3 — Per-Session Resource Quotas (HIGH)

**Problem:** One session running `sort` on a large file can consume all available memory. No fairness between sessions.

**Design:** Soft limits enforced by the daemon, hard limits via cgroups (Linux-only).

**Soft limits (daemon level — portable):**
```go
type Session struct {
    // ...
    MaxFiles  int       // max files created (default 1000)
    FileCount int       // current count
    MaxOutput int64     // max total output bytes (default 10MB)
    OutputUsed int64    // running total
}
```

**Hard limits (cgroups — Linux only, optional):**
```
# Per-session cgroup on session create:
mkdir /sys/fs/cgroup/goposix/session-<id>/
echo 268435456 > memory.max        # 256MB
echo "100 100000" > cpu.max        # 1 CPU core
echo $$ > cgroup.procs             # assign daemon goroutine (best-effort)
```

**Files changed:** `internal/daemon/session.go` (quota fields), `internal/daemon/server.go` (enforce on file creation, output accumulation)

### 🟡 23.4 — Structured Audit Trail (MEDIUM)

**Problem:** No record of who ran what. Can't answer "what did session X do?"

**Design:** JSON-lines audit log with configurable destination.

```go
type AuditEvent struct {
    Time      time.Time `json:"time"`
    SessionID string    `json:"sessionId"`
    Method    string    `json:"method"`
    Args      []string  `json:"args"`
    ExitCode  int       `json:"exitCode"`
    Duration  int64     `json:"durationMs"`
    Error     string    `json:"error,omitempty"`
}
```

**Output options (flag: `--audit-log`):**
- `file:///var/log/goposix/audit.jsonl` — rotated daily
- `socket:///var/run/goposix-audit.sock` — stream to external collector
- `stdout` — for Docker log driver capture
- `none` — disable (default for backward compatibility)

**Files changed:** `internal/daemon/audit.go` (new), `internal/daemon/server.go` (emit events), `pkg/daemon/daemon.go` (new flag)

### 🟡 23.5 — Subprocess Sandboxing (MEDIUM)

**Problem:** `find -exec`, `xargs`, and shell `exec` spawn child processes that inherit daemon permissions.

**Current state:** The shell sandbox already uses `SecurePath` for `openHandler`. But `execHandler` falls through to `interp.DefaultExecHandler` which runs the actual binary.

**Design:** Wrap child processes with:
1. Inherit session workspace as root directory
2. Drop all capabilities via `prctl(PR_CAPBSET_DROP, ...)`
3. Set `RLIMIT_NPROC` to prevent fork bombs from within the sandbox
4. Kill child processes on session destroy

**Linux-only:** Use `unshare(CLONE_NEWNS)` + `pivot_root` to give the child its own mount namespace rooted at the session workspace. This requires `CAP_SYS_ADMIN` or user namespaces.

**Fallback (no root):** `chroot`-like path rewriting in the shell interpreter's `execHandler` — all paths are resolved against the session workspace before being passed to `exec.Cmd`.

**Files changed:** `internal/shell/interpreter.go` (execHandler), `internal/daemon/server.go` (child process tracking)

### ⚪ 23.6 — Network Isolation (LOW)

**Problem:** The shell sandbox and `find -exec` could invoke network tools if they exist in the container.

**Fix:** This is a deployment concern, not a code concern. The daemon Dockerfile should drop all capabilities:

```dockerfile
# In Dockerfile (daemon image):
# Already non-root. Add:
RUN apk add --no-cache libseccomp  # optional: seccomp profiles
```

Recommended `docker run` flags for experiments:
```
docker run -d --name goposix-experiment \
  --cap-drop=ALL \
  --security-opt=no-new-privileges \
  --read-only \
  --tmpfs /var/lib/goposix/sessions:rw,noexec,nosuid,size=1G \
  -v goposix-audit:/var/log/goposix \
  goposix:latest --allowed-commands=/etc/goposix/experiment-allowlist.conf
```

**Files changed:** `docker/Dockerfile` (add security annotations as comments), `docs/operations/experiment-deployment.md` (new)

---

## 4. Implementation Plan

### Step 1 — Filesystem Isolation (4h, unblocks multi-tenant)
- [ ] Add `Workspace` field to `Session` struct
- [ ] `SessionManager.Create()` allocates workspace directory
- [ ] `SessionManager.Destroy()` cleans up workspace
- [ ] `server.go`: resolve all paths against `session.Workspace + session.CWD`
- [ ] Seed workspace with `/tmp`, `/home`
- [ ] Handle `/etc/hostname`, `/proc/*`, `/sys/*` as read-only host fallback
- [ ] Add `--session-dir` flag to daemon (default `/var/lib/goposix/sessions`)
- [ ] Tests: session A creates file, session B cannot see it

### Step 2 — Command Allowlist (1h)
- [ ] Add `allowedCommands map[string]bool` to `Server`
- [ ] Parse `--allowed-commands` flag (file path or comma-separated list)
- [ ] Check in `handleRequest()` before dispatch
- [ ] Ship default config files for experiment use case
- [ ] Tests: allowed command succeeds, blocked command returns -32601

### Step 3 — Audit Trail (2h)
- [ ] Define `AuditEvent` struct
- [ ] Create `AuditLogger` with file/socket/stdout backends
- [ ] Emit event after every RPC call (success or failure)
- [ ] Add `--audit-log` flag to daemon
- [ ] Tests: audit event contains session ID, method, exit code, duration

### Step 4 — Resource Quotas (3h)
- [ ] Add `MaxFiles`, `FileCount`, `MaxOutput`, `OutputUsed` to `Session`
- [ ] Increment `FileCount` on `touch`, `mkdir`, `cp` (writes)
- [ ] Increment `OutputUsed` on every RPC response
- [ ] Reject operations exceeding limits with -32000 error
- [ ] Add `--max-files-per-session`, `--max-output-per-session` flags
- [ ] Tests: quota exhaustion returns error

### Step 5 — Subprocess Sandbox (6h)
- [ ] Track child PIDs per session
- [ ] Kill all child PIDs on session destroy
- [ ] Set `RLIMIT_NPROC` on child processes
- [ ] `pivot_root` or path-rewriting exec handler
- [ ] Tests: `find -exec cat /etc/shadow {} \;` in session workspace fails

### Step 6 — Deployment Documentation (1h)
- [ ] `docs/operations/experiment-deployment.md` — full docker run command with security flags
- [ ] Example systemd unit for production experiment hosting
- [ ] Example nginx reverse-proxy config for WebSocket JSON-RPC

**Total estimated effort:** ~17h

---

## 5. New Daemon Flags (Cumulative)

```
goposix daemon \
  --socket /var/run/goposix.sock \
  --workers 4 \
  --listen-addr :8080 \
  --session-dir /var/lib/goposix/sessions \
  --allowed-commands /etc/goposix/experiment-allowlist.conf \
  --audit-log file:///var/log/goposix/audit.jsonl \
  --max-files-per-session 1000 \
  --max-output-per-session 10485760
```

---

## 6. Acceptance Criteria

- [ ] Two concurrent sessions cannot see each other's files (`echo a > /tmp/x` in session A, `cat /tmp/x` fails in session B)
- [ ] Blocked commands return JSON-RPC error code -32601 ("Command not available")
- [ ] Audit log contains structured entries for every RPC call
- [ ] Session exceeding file quota gets -32000 error
- [ ] Session exceeding output quota gets truncated response with warning
- [ ] Child processes from `find -exec` / shell are killed on session destroy
- [ ] `SecurePath` still prevents `../` traversal within workspace
- [ ] `make test` passes (zero regressions)
- [ ] `make testsuite` passes (548/4/10, no new failures)
- [ ] Existing single-tenant behavior unchanged when flags are not set (backward compatible)

---

## 7. Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Workspace isolation breaks utilities that need host paths | High | Medium | Read-only bind-mount for `/etc`, `/proc`, `/sys`; extensive test matrix |
| `pivot_root` requires `CAP_SYS_ADMIN` | High | High | Fall back to path rewriting in exec handler; document that full sandbox needs privileged Docker |
| Session cleanup races with in-flight requests | Medium | Medium | Reference-count sessions; defer workspace deletion until all goroutines drain |
| Audit log I/O blocks RPC handler | Medium | Low | Async audit writer with ring buffer; drop events under backpressure, don't block |
| Command allowlist breaks BusyBox test suite | Medium | Medium | Run testsuite with allowlist disabled (backward-compatible default) |
| Per-session cgroups require root | High | Low | Cgroups are optional; soft limits work everywhere |
| Filesystem seed (`/tmp`, `/home`) collides with host | Low | Medium | Workspace is always under `--session-dir`, never on host root |

---

## 8. References

- [Phase 22 — Hardening III](22_hardening_iii.md) — Daemon-first pivot (prerequisite: daemon is the default)
- [Phase 19 — Performance Benchmarking](19_performance_benchmarking.md) — Benchmark framework and results
- [Phase 20 — Hardening II](20_hardening_ii.md) — Input limits, coverage, flag audit
- [Session manager](../internal/daemon/session.go) — Current session implementation
- [SecurePath](../pkg/common/security.go) — Path traversal prevention
- [Shell sandbox](../internal/shell/interpreter.go) — Current shell confinement
- [Daemon server](../internal/daemon/server.go) — RPC dispatch and path resolution
- [Go SDK client](../pkg/client/client.go) — Typed RPC client for all utilities
