# Phase 24 — Multi-Agent Observability

> **Status:** PLANNING | **Date:** 2026-05-18 | **Trigger:** Multi-agent collaboration use case; extracted from deferred Phase 23
>
> **Goal:** Give a single-tenant GoPOSIX daemon the ability to attribute every RPC
> operation to a specific agent, trace which files were read or written, and expose
> per-agent metrics — so that multiple trusted agents operating on the same shared
> filesystem can be debugged, audited, and monitored without workspace isolation.

---

## 1. Why This, Not Multi-Tenancy

Phase 23 assumed untrusted users who must not see each other's files. The current use
case is different: **multiple trusted agents, one user, one shared repo.** Agents are
concurrent processes operating on shared mutable state — they *must* see each other's
files. The problem isn't security; it's **correctness under concurrency** and
**debuggability.**

When Agent A and Agent B both write to `/home/user/repo/go.mod` in the same
microsecond, the audit log should tell you:

- Who wrote what
- When they wrote it
- Which file was affected
- What exit code they got
- How long it took

Without this, multi-agent debugging is hell. With it, you can answer "did Agent A
silently overwrite Agent B's change?" without instrumenting every agent individually.

### Single-tenant, multi-agent vs multi-tenant

| Property | Multi-tenant (Phase 23) | Multi-agent (Phase 24) |
|----------|------------------------|------------------------|
| Filesystem | Isolated per user (workspaces) | Shared (same repo, same `/`) |
| Identity | User is untrusted stranger | Agent is trusted collaborator |
| Goal | Prevent reads/writes across users | Trace what each agent did |
| Allowlist | Block dangerous commands | Attribute commands to agents |
| Audit | Compliance (who ran `rm -rf /`?) | Debugging (which agent corrupted `go.mod`?) |
| Quotas | Fairness between users | Safety (no agent OOMs the daemon) |

---

## 2. Current Observability Posture (Already Exists)

These primitives are solid:

| Feature | Location | Detail |
|---------|----------|--------|
| Prometheus metrics | `internal/daemon/observability.go` | `/metrics` endpoint: request counts, active workers, uptime, sessions, per-method durations, rate-limited count |
| Structured logging | `internal/daemon/server.go` (defer block) | `slog.Info("rpc handled", method, sessionId, durationMs, cmd, exitCode, error)` |
| Session manager | `internal/daemon/session.go` | Random session IDs, CWD tracking, env vars, TTL cleanup |
| Health check | `observability.go` | `/healthz`, `/readyz` endpoints |

### What's missing for multi-agent

| Gap | Current state | Needed |
|-----|--------------|--------|
| Agent identity | Sessions are anonymous (just random hex IDs) | `AgentID` + optional labels on `Session` |
| Files touched | No tracking of which files a command read/wrote | Per-operation file list in structured log |
| Per-agent metrics | Prometheus broken down by method only | Per-agent request counts, error rates, latencies |
| Queryable audit history | `slog` to stderr — ephemeral, unstructured | Retained structured log, queryable endpoint |
| Session log export | No `POST /sessions/:id/log` equivalent | In-memory ring buffer per session, exportable |
| Error attribution | Exit codes logged but not attributable post-hoc | "Agent X has a 23% error rate on `sed`" |

---

## 3. Design

### 24.1 — Agent-Aware Sessions

Add agent identity and metadata to the `Session` struct.

```go
type Session struct {
    ID         string            `json:"sessionId"`
    AgentID    string            `json:"agentId,omitempty"`    // "build-agent", "lint-agent"
    Labels     map[string]string `json:"labels,omitempty"`     // arbitrary key-value
    CWD        string            `json:"cwd"`
    Env        map[string]string `json:"env"`
    LastActive time.Time         `json:"lastActive"`
}
```

**Session create RPC gains optional fields:**

```json
{
    "jsonrpc": "2.0",
    "method": "goposix.session.create",
    "params": {
        "agentId": "build-agent",
        "labels": {"pipeline": "ci", "commit": "abc123"}
    },
    "id": 1
}
```

**Backward compatible:** Omitting `agentId` and `labels` is valid. Existing callers are unaffected.

**Files changed:** `internal/daemon/session.go`, `internal/daemon/server.go` (session.create handler)

### 24.2 — Per-Operation File Tracking

Extend the RPC response to include which files were read or written. This is the
hardest part because POSIX utilities use `os.Open`, `os.Create`, `os.Remove`, etc.
and there's no central filesystem intercept layer.

**Approach: filesystem wrapper.** Introduce a `FileTracker` that wraps `os.File`
operations and records paths. Utilities already use `os.Open` directly — this needs
a refactor to route through a session-aware VFS layer.

```go
// pkg/common/vfs.go (new file)
type FileTracker struct {
    mu       sync.Mutex
    reads    []string
    writes   []string
    deletes  []string
}

func (ft *FileTracker) Open(name string) (*os.File, error) {
    ft.mu.Lock()
    ft.reads = append(ft.reads, name)
    ft.mu.Unlock()
    return os.Open(name)
}

func (ft *FileTracker) Create(name string) (*os.File, error) {
    ft.mu.Lock()
    ft.writes = append(ft.writes, name)
    ft.mu.Unlock()
    return os.Create(name)
}

func (ft *FileTracker) Remove(name string) error {
    ft.mu.Lock()
    ft.deletes = append(ft.deletes, name)
    ft.mu.Unlock()
    return os.Remove(name)
}

func (ft *FileTracker) Snapshot() FileAccessReport {
    ft.mu.Lock()
    defer ft.mu.Unlock()
    return FileAccessReport{
        Reads:   append([]string{}, ft.reads...),
        Writes:  append([]string{}, ft.writes...),
        Deletes: append([]string{}, ft.deletes...),
    }
}
```

**Alternative: opt-in file tracking per session.** Not every session needs file
tracking (performance overhead). Add a `TrackFiles bool` field to `Session`.
When enabled, the daemon injects a `FileTracker` into the RPC context.

**Risk:** Every filesystem utility (`ls`, `cat`, `cp`, `rm`, `grep`, `sed`, `sort`,
`find`, `touch`, `mkdir`, ...) must route through `FileTracker` instead of `os.Open`
directly. This is a cross-cutting concern affecting 40+ packages. Mitigation:
start with a small subset (echo redirection, cat, cp, rm) and expand incrementally.

**Files changed:** `pkg/common/vfs.go` (new), `internal/daemon/server.go` (inject tracker), 40+ utility packages (incremental)

### 24.3 — Structured Audit Log (Lifted from Phase 23, Reframed)

Phase 23's audit trail design is correct — it just needs a different purpose
(debugging, not compliance).

```go
type AuditEvent struct {
    Time        time.Time         `json:"time"`
    SessionID   string            `json:"sessionId"`
    AgentID     string            `json:"agentId,omitempty"`
    Method      string            `json:"method"`
    Args        []string          `json:"args"`
    ExitCode    int               `json:"exitCode"`
    DurationMs  float64           `json:"durationMs"`
    Error       string            `json:"error,omitempty"`
    FilesRead   []string          `json:"filesRead,omitempty"`
    FilesWrote  []string          `json:"filesWrote,omitempty"`
    FilesDelete []string          `json:"filesDelete,omitempty"`
}
```

**Output options (flag: `--audit-log`):**
- `file:///var/log/goposix/audit.jsonl` — rotated daily
- `socket:///var/run/goposix-audit.sock` — stream to external collector (e.g., Vector, Fluentd)
- `stdout` — for Docker log driver capture (e.g., Loki, CloudWatch)
- `none` — disable (default)

**In-memory ring buffer:** For querying recent history, keep a per-session ring buffer
of the last N events (configurable, default 1000). Expose via:

```
POST /sessions/{sessionId}/log
GET  /sessions/{sessionId}/log?limit=100&method=goposix.echo
```

**Files changed:** `internal/daemon/audit.go` (new), `internal/daemon/server.go` (emit events), `internal/daemon/observability.go` (new HTTP endpoints)

### 24.4 — Per-Agent Prometheus Metrics

Current metrics aggregate by method only. Extend to include agent identity.

```
# HELP goposix_rpc_total_agent Count of RPC calls per agent.
# TYPE goposix_rpc_total_agent counter
goposix_rpc_total_agent{agent="build-agent"} 1542
goposix_rpc_total_agent{agent="lint-agent"} 891

# HELP goposix_rpc_errors_agent Count of RPC errors per agent.
# TYPE goposix_rpc_errors_agent counter
goposix_rpc_errors_agent{agent="build-agent"} 3
goposix_rpc_errors_agent{agent="lint-agent"} 0

# HELP goposix_rpc_duration_ms_agent Sum of RPC call durations per agent in milliseconds.
# TYPE goposix_rpc_duration_ms_agent counter
goposix_rpc_duration_ms_agent{agent="build-agent"} 45021.30
goposix_rpc_duration_ms_agent{agent="lint-agent"} 8923.10
```

**Alertable patterns:**
- `rate(goposix_rpc_errors_agent[5m]) / rate(goposix_rpc_total_agent[5m]) > 0.1` → Agent error rate spike
- `goposix_rpc_duration_ms_agent` 99th percentile exceeding threshold → Agent slowdown
- Agent with zero requests for N minutes → Agent may have crashed

**Files changed:** `internal/daemon/observability.go` (new metric series)

### 24.5 — Session Log Export & Query

For forensics: "what did agent X do between 14:00 and 14:05?"

**RPC endpoint:**

```json
// Request
{
    "jsonrpc": "2.0",
    "method": "goposix.session.log",
    "params": {
        "sessionId": "abc123",
        "limit": 100,
        "since": "2026-05-18T14:00:00Z",
        "until": "2026-05-18T14:05:00Z"
    },
    "id": 1
}

// Response
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": {
        "sessionId": "abc123",
        "agentId": "build-agent",
        "events": [
            {
                "time": "2026-05-18T14:02:03.456Z",
                "method": "goposix.echo",
                "args": ["module A v2"],
                "exitCode": 0,
                "durationMs": 0.06,
                "filesWrote": ["/home/user/repo/go.mod"]
            }
        ]
    }
}
```

**Files changed:** `internal/daemon/server.go` (new session.log handler), `internal/daemon/session.go` (ring buffer)

---

## 4. Implementation Plan

### Step 1 — Agent-Aware Sessions (1h)
- [ ] Add `AgentID`, `Labels` fields to `Session` struct
- [ ] Accept `agentId`, `labels` in `goposix.session.create` params
- [ ] Add `--agent-id`, `--label` flags to `goposix session create` CLI
- [ ] Backward-compatible: omitting them produces sessions with empty AgentID
- [ ] Tests: session with agentId, session without (no regression)

### Step 2 — Audit Trail (3h)
- [ ] Define `AuditEvent` struct with AgentID, FilesRead/Wrote/Delete
- [ ] Create `AuditLogger` with file/socket/stdout backends
- [ ] Emit event after every RPC call (success or failure)
- [ ] Add `--audit-log` flag to daemon
- [ ] In-memory ring buffer per session (configurable size)
- [ ] Tests: audit event contains session ID, agent ID, method, exit code, duration

### Step 3 — Per-Agent Metrics (1.5h)
- [ ] Add `goposix_rpc_total_agent`, `goposix_rpc_errors_agent`, `goposix_rpc_duration_ms_agent`
- [ ] Emit in `/metrics` endpoint alongside existing per-method series
- [ ] Tests: agent A gets 10 requests, agent B gets 5 — counters reflect this

### Step 4 — Session Log Query API (1.5h)
- [ ] Add `goposix.session.log` RPC method
- [ ] Support `limit`, `since`, `until` params
- [ ] Return structured event list from ring buffer
- [ ] Tests: query returns events in time range, respects limit

### Step 5 — File Tracking (4h) ⚠️ Highest effort, incremental
- [ ] Create `pkg/common/vfs.go` with `FileTracker`
- [ ] Add `TrackFiles bool` to `Session`
- [ ] Wire FileTracker into RPC context when enabled
- [ ] Pilot: `cat`, `echo` (with redirection), `cp`, `rm`, `touch`
- [ ] Populate `FilesRead`/`FilesWrote`/`FilesDelete` in AuditEvent
- [ ] Tests: `echo foo > /tmp/x` → audit event includes `/tmp/x` in FilesWrote

### Step 6 — Integration Test (1h)
- [ ] Multi-agent scenario: agent A and agent B concurrently write to same file
- [ ] Audit log captures both writes with distinct agent IDs
- [ ] Per-agent metrics show independent request counts
- [ ] Session log query returns events from both agents

**Total estimated effort:** ~12h

---

## 5. New Daemon Flags

```
goposix daemon \
  --socket /var/run/goposix.sock \
  --listen-addr :8080 \
  --audit-log file:///var/log/goposix/audit.jsonl \
  --audit-ring-size 1000
```

Session creation (CLI):
```
goposix session create --agent-id build-agent --label pipeline=ci --label commit=abc123
```

---

## 6. Acceptance Criteria

- [ ] Session creation with `agentId` and `labels` works; omission is backward-compatible
- [ ] Every RPC call emits an AuditEvent with sessionId, agentId, method, exitCode, durationMs
- [ ] `--audit-log=stdout` produces JSON-lines consumable by Docker log drivers
- [ ] `--audit-log=file://...` produces JSON-lines with daily rotation
- [ ] `/metrics` includes `goposix_rpc_total_agent`, `goposix_rpc_errors_agent`, `goposix_rpc_duration_ms_agent`
- [ ] `goposix.session.log` returns events filtered by time range and limit
- [ ] File tracking (when enabled) populates `FilesRead`/`FilesWrote`/`FilesDelete` in audit events
- [ ] Two agents writing to the same file produce distinct audit events with correct agent IDs
- [ ] `make test` passes (zero regressions)
- [ ] `make testsuite` passes (548/4/10, no new failures)
- [ ] Existing single-agent behavior unchanged when `agentId` is not set

---

## 7. Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| File tracking requires touching 40+ utility packages | High | High | Incremental rollout: start with 5 utilities, expand per demand. Untracked calls emit audit events without file lists — no data loss, just less detail. |
| FileTracker adds syscall overhead per file operation | Medium | Medium | Gate behind `TrackFiles bool` on session. Disabled by default. Only pay the cost when you need it. |
| Per-agent metrics cardinality explosion | Low | Medium | Agent IDs are bounded (fixed set, not user-generated UUIDs). Prometheus handles thousands of label values. |
| Audit log I/O blocks RPC handler | Medium | Low | Async audit writer with ring buffer; drop events under backpressure, don't block. |
| Ring buffer memory grows with session count | Low | Medium | Configurable `--audit-ring-size` (default 1000). 1000 sessions × 1000 events × ~200 bytes = 200MB worst case. Acceptable. |
| session.log query does full ring-buffer scan | Medium | Low | Ring buffer is small (1000 events). Linear scan is µs-scale. Add time-indexed skip list only if profiling shows it's needed. |

---

## 8. References

- [Phase 23 — Multi-Tenant Sandbox](deferred.md) — Audit trail and quota design (extracted and reframed here)
- [Phase 22 — Hardening III](22_hardening_iii.md) — Daemon-first pivot (prerequisite)
- [Security model](security.md) — Current security posture, RPC-level protections
- [Architecture](architecture.md) — Component flow, key packages
- [Session manager](../internal/daemon/session.go) — Current session implementation
- [Observability server](../internal/daemon/observability.go) — Prometheus metrics and health endpoints
- [Daemon server](../internal/daemon/server.go) — RPC dispatch, structured logging, metrics recording
