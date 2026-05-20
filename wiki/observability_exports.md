# GoPOSIX Observability

How to expose GoPOSIX daemon internals (goroutines, memory, sessions, per-method
throughput, per-agent attribution) so operators can see what's happening — whether
through standard OS tools (`top`, `ps`, `htop`), Prometheus/Grafana, or a custom
`gotop` TUI.

**Status:** Options A, B, D implemented on branch `feat/observability`. Options C, E
planned. Multi-agent observability (Phase 24) is PLANNING — no implementation yet.

---

## What Already Exists

| Surface | Path | Format | What it shows |
|---------|------|--------|---------------|
| Prometheus metrics | `-l :9090` → `GET /metrics` | OpenMetrics text | requests_total, workers_active/max, uptime_s, sessions_active, rate_limited, per-method duration aggregates, Go runtime stats |
| JSON status | `GET /status` | JSON | Full daemon snapshot: pid, uptime, goroutines, heap, GC, workers, sessions, per-method, per-session |
| Health | `GET /healthz` | JSON | `{"status":"ok"}` |
| Readiness | `GET /readyz` | JSON | `{"status":"ready"}` or 503 when draining |
| pprof | `:6060` (only with `GOPOSIX_DEBUG=1`) | pprof protocol | goroutine/heap/CPU profiles, execution traces |
| Process title | `/proc/<pid>/cmdline` | Text | `goposix daemon [W:3/4 S:12 C:500K]` — live in `ps aux` |
| Thread names | `/proc/<pid>/task/<tid>/comm` | Text | `goposix/wrk-00` through `goposix/wrk-NN` — visible in `htop` |

---

## Part 1 — Infrastructure Exports (Tools A–G)

These are the OS-level and HTTP-level options for exporting daemon health to
external tooling. Each option below has a letter code for cross-referencing.

### Option A: Thread Naming (visible in `top -H` / `htop`) ✅ IMPLEMENTED

Linux lets you name OS threads via `prctl(PR_SET_NAME, name)`, visible in
`/proc/<pid>/task/<tid>/comm` and `htop` tree view. Each worker goroutine is
pinned to an OS thread and named `goposix/wrk-NN`.

**Implementation:** `internal/daemon/server.go` — `WorkerPool.Submit()` calls
`setThreadName()` which uses `runtime.LockOSThread()` + `unix.Prctl(PR_SET_NAME)`.

| What | Value |
|------|-------|
| **What `top` / `htop` sees** | Each worker thread named `goposix/wrk-00` through `goposix/wrk-NN` under the daemon process |
| **Effort** | Medium — requires `runtime.LockOSThread()` and careful placement in the worker-pool submit loop |
| **Portability** | Linux only (`PR_SET_NAME`); `_darwin.go` provides a no-op stub |
| **Limits** | 15-char name max on Linux; goroutine != OS thread; runtime dynamically spawns/retires threads |
| **Value** | Low. Nice-to-have. `htop` users see named workers. |

**Verdict:** Marginal but implemented. Only worth it if operators use `htop` as their primary monitoring tool.

---

### Option B: Process Name in `ps` (argv[0] overwrite) ✅ IMPLEMENTED

Overwrite the process name shown in `ps aux`. Many daemons do this (PostgreSQL:
`postgres: writer process`, nginx: `nginx: worker process`). The title is updated
every 5 seconds with live daemon state.

**Example output in `ps aux`:**
```
ramayac  12345  0.1  0.5  45678  12345 ?  Ssl  10:00  0:02 goposix daemon [W:3/4 S:12 C:500K]
```

**Implementation:** `internal/daemon/proctitle_linux.go` — finds the original argv+env
memory area at init time via `unsafe.StringData`, then overwrites it with the new title.

| What | Value |
|------|-------|
| **What `ps` sees** | `goposix daemon [W:3/4 S:12 C:500K]` — workers active/max, session count, total RPC calls |
| **Effort** | Low — stdlib only, `unsafe` for argv area discovery |
| **Portability** | Linux only; macOS stub |
| **Limits** | Title cannot exceed original argv+env combined length |
| **Value** | Medium. `ps aux | grep goposix` immediately useful. Zero tooling overhead. |

**Verdict:** Implemented as a low-effort bonus.

---

### Option C: Go Runtime Stats in Prometheus `/metrics` (PLANNING)

Add Go runtime telemetry to the existing `/metrics` endpoint — **pure stdlib,
no imports, ~40 lines in `observability.go`**.

```go
import "runtime"
var mem runtime.MemStats
runtime.ReadMemStats(&mem)

// New Prometheus metrics:
// goposix_goroutines              gauge     runtime.NumGoroutine()
// goposix_heap_alloc_bytes        gauge     mem.HeapAlloc
// goposix_heap_sys_bytes          gauge     mem.HeapSys
// goposix_heap_idle_bytes         gauge     mem.HeapIdle
// goposix_stack_inuse_bytes       gauge     mem.StackInuse
// goposix_gc_pause_ns             gauge     mem.PauseNsRecent (last GC)
// goposix_num_cpu                 gauge     runtime.NumCPU()
// goposix_gomaxprocs              gauge     runtime.GOMAXPROCS(0)
// goposix_mallocs_total           counter   mem.Mallocs
// goposix_frees_total             counter   mem.Frees
// goposix_num_gc_cycles           counter   mem.NumGC
// goposix_total_alloc_bytes       counter   mem.TotalAlloc
```

| What | Value |
|------|-------|
| **What `top` / `ps` sees** | Nothing. Prometheus/Grafana only. |
| **Effort** | ~30 min — read `runtime.MemStats`, format OpenMetrics lines |
| **Portability** | All platforms (std `runtime` package) |
| **Value** | High. Enables Grafana dashboards with goroutine graphs, heap pressure, GC frequency — the standard Go service monitoring picture. |

**Verdict:** Should be done regardless. 30 minutes, zero risk, big monitoring payoff.

---

### Option D: JSON `/status` Endpoint ✅ IMPLEMENTED

Returns a rich JSON blob with everything the daemon knows about itself.
Machine-friendly, human-readable, consumable by any tool. This is the foundation
for Option E (`gotop` TUI) and external monitoring.

**Implementation:** `internal/daemon/observability.go` — `handleStatus()` handler,
`StatusSnapshot` struct, registered at `/status` in the HTTP mux.

**Example:**
```bash
curl -s localhost:9090/status | jq .
```

```json
{
  "pid": 12345,
  "uptime_s": 3600,
  "version": "0.1.0",
  "goroutines": 34,
  "gomaxprocs": 4,
  "num_cpu": 8,
  "mem": {
    "heap_alloc_mb": 12.3,
    "heap_sys_mb": 28.1,
    "stack_inuse_mb": 0.8,
    "gc_pause_ms": 0.4,
    "num_gc": 142,
    "total_alloc_mb": 450.2,
    "mallocs": 5000000,
    "frees": 4800000
  },
  "workers": {
    "active": 2,
    "max": 4
  },
  "sessions": {
    "active": 3,
    "total_created": 12
  },
  "rpc": {
    "total_calls": 500000,
    "rate_limited": 3
  },
  "connection_pool": {
    "active_connections": 2,
    "max_connections": 100,
    "total_connections": 500
  },
  "per_method": [
    { "method": "goposix.echo",  "count": 150000, "avg_ms": 0.05 },
    { "method": "goposix.ls",    "count": 120000, "avg_ms": 0.22 }
  ],
  "per_session": [
    { "id": "a1b2c3", "age_s": 120, "calls": 142, "cwd": "/tmp" }
  ]
}
```

| What | Value |
|------|-------|
| **Effort** | ~50 min — one new handler in `observability.go`, one `StatusSnapshot` struct, tie into existing counters |
| **Portability** | All platforms |
| **Value** | High. This is the data source for Option E (`gotop` TUI) and for any external monitoring tool. Single `curl` gives the operator everything. |

**Verdict:** ✅ Implemented.

---

### Option E: `gotop` TUI Utility (PLANNING)

A `pkg/gotop/` utility (or standalone `cmd/gotop/` binary) that hits the daemon's
`/status` endpoint at 1–2 Hz and renders a live `htop`-like TUI. Pure Go, zero
external dependencies (bubbletea or termdash for TUI, or raw ANSI).

```
┌─ goposix daemon @ :9090 ────────────────── uptime: 1h 23m ─┐
│ Goroutines: 34    Workers: [████░░] 2/4    Sessions: 3      │
│ Heap: 12.3 MB     RSS: 45.1 MB            GC pause: 0.4ms   │
│ RPC calls: 500K   Rate limited: 3                          │
├────────────────────────────────────────────────────────────┤
│ Method             │  Count   │ Total(ms) │ Avg(ms) │  %   │
│ goposix.echo       │  150,000 │    7,500  │   0.05  │ 30.0 │
│ goposix.ls         │  120,000 │   26,400  │   0.22  │ 24.0 │
│ goposix.grep       │   80,000 │  112,000  │   1.40  │ 16.0 │
│ goposix.shell.exec │   50,000 │ 2,250,000 │  45.00  │ 10.0 │
├────────────────────────────────────────────────────────────┤
│ Sessions                                   [q quit] [r ref]│
│ a1b2c3 · 2min · 142 calls · /tmp                           │
│ d4e5f6 · 5min · 450 calls · /var/data                      │
│ g7h8i9 · 30s  ·   3 calls · /                              │
└────────────────────────────────────────────────────────────┘
```

| What | Value |
|------|-------|
| **Effort** | 1–3 hours depending on TUI library choice (bubbletea ~200 LOC, raw ANSI ~150 LOC) |
| **Portability** | All platforms (pure Go TUI) |
| **Dependencies** | Optional — `bubbletea` for polished TUI, or raw ANSI escape codes for zero-dependency |
| **Value** | High. The best UX for operators. One command to see everything. |

**Verdict:** The "crown jewel" of observability. Implement after C and D.

**Modes:**
- **Daemon-connected:** `goposix gotop --daemon :9090` — queries `/status` every 1s, shows daemon telemetry
- **Standalone:** `goposix gotop` (no daemon) — reads local `/proc` like standard `top`, shows only process-level metrics

---

### Option F: CGroups v2 Per-Session Isolation (DEFERRED)

Put each daemon session in its own Linux cgroup. The kernel then tracks memory,
CPU, and I/O per session — visible in `systemd-cgtop` or `/sys/fs/cgroup/`.

```
$ systemd-cgtop
Control Group                    Tasks   %CPU   Memory
/sys/fs/cgroup/goposix.session.a1b2    1   0.2    4.3M
/sys/fs/cgroup/goposix.session.c3d4    3   1.1   12.7M
```

| What | Value |
|------|-------|
| **Effort** | High — cgroup management (mkdir, echo pid, cleanup on session destroy), requires root/CAP_SYS_ADMIN or pre-delegated subtree |
| **Portability** | Linux only, cgroups v2 required. Docker: needs `--privileged` or explicit `--cgroup-parent` + cgroupfs mounts. Scratch container: extra setup. |
| **Value** | Elegant in theory, fragile in practice. Kernel-enforced isolation is the gold standard, but most deployments won't grant the privileges. |

**Verdict:** Defer. Revisit when per-session resource quotas (Phase 23) are on the table — cgroups are the enforcement mechanism, not just an observability tool.

---

### Option G: eBPF / bpftrace Probes (REJECTED)

eBPF programs attached to the Go binary using USDT probes.

| What | Value |
|------|-------|
| **Effort** | Very high — requires kernel support, eBPF toolchain, USDT probe insertion in Go |
| **Portability** | Linux 4.x+, not available in scratch containers without additional setup |
| **Value** | Zero for internal daemon observability. eBPF is for kernel-level tracing, not application metrics. |

**Verdict:** Rejected. Out of scope.

---

### Infrastructure Summary Matrix

| Opt | What `top`/`ps` sees? | Effort | New deps? | Portability | Value | Status |
|:---:|------------------------|:------:|:---------:|:-----------:|:-----:|:------:|
| A | `htop` shows named worker threads | Medium | ❌ | Linux only | Low | ✅ |
| B | `ps aux` shows live status | 15 min | ❌ | Linux | Med | ✅ |
| **D** | JSON `/status` endpoint | 50 min | ❌ | **All** | **High** | ✅ |
| C | Nothing (Prometheus) | 30 min | ❌ | All | High | PLANNING |
| E | Self-contained TUI | 1–3 hr | ❌ or bubbletea | All | High | PLANNING |
| F | `systemd-cgtop` per-session | High | cgroup mgmt | Linux only | Med | DEFERRED |
| G | Nothing (eBPF toolchain) | Very high | eBPF | Linux only | Zero | REJECTED |

---

## Part 2 — Multi-Agent Observability (Phase 24)

> **Status:** PLANNING — design below; no implementation yet.
>
> **Goal:** Give a single-tenant GoPOSIX daemon the ability to attribute every RPC
> operation to a specific agent, trace which files were read or written, and expose
> per-agent metrics — so that multiple trusted agents operating on the same shared
> filesystem can be debugged, audited, and monitored without workspace isolation.

### Why Multi-Agent, Not Multi-Tenancy

Phase 23 assumed untrusted users who must not see each other's files. The current
use case is different: **multiple trusted agents, one user, one shared repo.**
Agents are concurrent processes operating on shared mutable state — they *must*
see each other's files. The problem isn't security; it's **correctness under
concurrency** and **debuggability.**

When Agent A and Agent B both write to `/home/user/repo/go.mod` in the same
microsecond, the audit log should tell you who wrote what, when, and with what
result.

### What's Missing for Multi-Agent

| Gap | Current state | Needed |
|-----|--------------|--------|
| Agent identity | Sessions are anonymous (random hex IDs) | `AgentID` + optional labels on `Session` |
| Files touched | No tracking of which files a command read/wrote | Per-operation file list in structured log |
| Per-agent metrics | Prometheus broken down by method only | Per-agent request counts, error rates, latencies |
| Queryable audit history | `slog` to stderr — ephemeral, unstructured | Retained structured log, queryable endpoint |
| Session log export | No `POST /sessions/:id/log` equivalent | In-memory ring buffer per session, exportable |
| Error attribution | Exit codes logged but not attributable post-hoc | "Agent X has a 23% error rate on `sed`" |

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

**Backward compatible:** Omitting `agentId` and `labels` is valid.

### 24.2 — Per-Operation File Tracking

Extend the RPC response to include which files were read or written. This is the
hardest part because POSIX utilities use `os.Open`, `os.Create`, `os.Remove`, etc.
and there's no central filesystem intercept layer.

**Approach: opt-in file tracking per session.** Not every session needs file
tracking (performance overhead). Add a `TrackFiles bool` field to `Session`.
When enabled, the daemon injects a `FileTracker` into the RPC context.

**Risk:** Every filesystem utility must route through `FileTracker` instead of
`os.Open` directly — a cross-cutting concern affecting 40+ packages. Mitigation:
start with a small subset (echo redirection, cat, cp, rm) and expand incrementally.

### 24.3 — Structured Audit Log

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
- `socket:///var/run/goposix-audit.sock` — stream to external collector
- `stdout` — for Docker log driver capture
- `none` — disable (default)

### 24.4 — Per-Agent Prometheus Metrics

```
# HELP goposix_rpc_total_agent Count of RPC calls per agent.
# TYPE goposix_rpc_total_agent counter
goposix_rpc_total_agent{agent="build-agent"} 1542
goposix_rpc_total_agent{agent="lint-agent"} 891
```

### 24.5 — Session Log Export & Query

RPC endpoint: `goposix.session.log` — returns events filtered by time range and limit.

### Implementation Plan (~12h)

| Step | What | Time |
|:----:|------|:----:|
| 1 | Agent-aware sessions | 1h |
| 2 | Audit trail | 3h |
| 3 | Per-agent metrics | 1.5h |
| 4 | Session log query API | 1.5h |
| 5 | File tracking (incremental, 40+ packages) | 4h |
| 6 | Integration test (multi-agent scenario) | 1h |

### Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| File tracking requires touching 40+ utility packages | High | High | Incremental rollout: start with 5 utilities, expand per demand |
| FileTracker adds syscall overhead per file operation | Medium | Medium | Gate behind `TrackFiles bool` on session; disabled by default |
| Per-agent metrics cardinality explosion | Low | Medium | Agent IDs are bounded (fixed set, not user-generated UUIDs) |
| Audit log I/O blocks RPC handler | Medium | Low | Async audit writer with ring buffer; drop events under backpressure |
| Ring buffer memory grows with session count | Low | Medium | Configurable `--audit-ring-size` (default 1000) |

---

## References

- [Architecture](architecture.md) — Component flow, key packages
- [Security model](security.md) — Current security posture, RPC-level protections
- [Session manager](session.go) — Current session implementation
- [Observability server](observability.go) — Prometheus metrics, health, /status
- [Daemon server](server.go) — RPC dispatch, structured logging, metrics recording
- [Phase 22 — Hardening III](22_hardening_iii.md) — Daemon-first pivot (prerequisite)
- [deferred.md](deferred.md) — Phase 23 (Multi-Tenant Sandbox) for audit trail and quota design
