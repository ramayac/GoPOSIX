# Daemon Observability Exports — Options

How to expose GoPOSIX daemon internals (goroutines, memory, sessions, per-method
throughput) so operators can see what's happening — whether through standard OS
tools (`top`, `ps`, `htop`) or through custom observability surfaces.

**Status:** PLANNING — options below; no implementation yet.

---

## What Already Exists

The daemon already has three observability surfaces:

| Surface | Path | Format | What it shows |
|---------|------|--------|---------------|
| Prometheus metrics | `-l :9090` → `GET /metrics` | OpenMetrics text | requests_total, workers_active/max, uptime_s, sessions_active, rate_limited, per-method duration aggregates |
| Health | `GET /healthz` | JSON | `{"status":"ok"}` |
| Readiness | `GET /readyz` | JSON | `{"status":"ready"}` or 503 when draining |
| pprof | `:6060` (only with `GOPOSIX_DEBUG=1`) | pprof protocol | goroutine/heap/CPU profiles, execution traces |

**Gap:** The Prometheus endpoint has no Go runtime stats (goroutine count, heap, GC), no per-session breakdown, and no machine-readable status blob for custom tooling. Also, nothing feeds standard OS tools (`top`, `ps`, `htop`).

---

## Option A: Thread Naming (visible in `top -H` / `htop`)

Linux lets you name OS threads via `prctl(PR_SET_NAME, name)`, visible in
`/proc/<pid>/task/<tid>/comm` and `htop` tree view.

```go
import "golang.org/x/sys/unix"

func nameWorker(id int) {
    runtime.LockOSThread()
    defer runtime.UnlockOSThread()
    unix.Prctl(unix.PR_SET_NAME, fmt.Sprintf("goposix/wrk-%02d", id), 0, 0, 0)
}
```

| What | Value |
|------|-------|
| **What `top` / `htop` sees** | Each worker thread gets a named entry under the daemon process |
| **Effort** | Medium — requires `runtime.LockOSThread()` and careful placement in the worker-pool submit loop |
| **Portability** | Linux only (`PR_SET_NAME`); Go's goroutine→OS-thread mapping is non-deterministic without `LockOSThread` |
| **Limits** | 15-char name max (Linux), goroutine != OS thread, runtime dynamically spawns/retires threads |
| **Value** | Low. Nice-to-have but fragile. `htop` users would see named workers; the mapping leaks over time as OS threads come and go. |

**Verdict:** Marginal. Only worth it if operators already use `htop` as their primary monitoring tool.

---

## Option B: Process Name in `ps` (argv[0] overwrite)

Overwrite the process name shown in `ps aux`. Many daemons do this (PostgreSQL:
`postgres: writer process`, nginx: `nginx: worker process`). On Linux, overwriting
the `argv` memory region updates `/proc/<pid>/cmdline`.

```go
// With github.com/erikdubbelboer/gspt (pure Go, no cgo):
gspt.SetProcTitle("goposix [4w/3s/500K]")  // 4 workers, 3 sessions, 500K calls
```

Or via direct `/proc/self/cmdline` manipulation (stdlib only, ~40 LOC).

| What | Value |
|------|-------|
| **What `ps` sees** | `goposix [3/4 workers, 12 sessions]` |
| **Effort** | Low — 15 min with gspt or a small stdlib helper |
| **Portability** | Linux only (BSD/Mac need different syscalls) |
| **Limits** | 15 chars on `comm` field (standard `ps`); `/proc/self/cmdline` allows more but some tools truncate |
| **Value** | Medium. Makes `ps aux | grep goposix` immediately useful. Zero tooling overhead for operators. |

**Verdict:** Worth doing as a low-effort bonus. No new dependencies needed.

---

## Option C: Go Runtime Stats in Prometheus `/metrics`

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

## Option D: JSON `/status` Endpoint

Add a `GET /status` endpoint returning a rich JSON blob with everything the daemon
knows about itself. Machine-friendly, human-readable, consumable by any tool.

```json
{
  "pid": 42,
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
    "num_gc": 142
  },
  "workers": {
    "active": 2,
    "max": 4,
    "queued": 0
  },
  "sessions": {
    "active": 3,
    "total_created": 12
  },
  "rpc": {
    "total_calls": 500000,
    "rate_limited": 3
  },
  "per_method": [
    { "method": "goposix.echo",  "count": 150000, "avg_ms": 0.05, "sum_ms": 7500.0 },
    { "method": "goposix.ls",    "count": 120000, "avg_ms": 0.22, "sum_ms": 26400.0 },
    { "method": "goposix.grep",  "count": 80000,  "avg_ms": 1.40, "sum_ms": 112000.0 }
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

**Verdict:** Essential foundation. Implement before Option E.

---

## Option E: `gotop` TUI Utility

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

## Option F: CGroups v2 Per-Session Isolation

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
| **Portability** | Linux only, cgroups v2 required. Docker: needs `--privileged` or explicit `--cgroup-parent` + cgroupfs mounts. Scratch container: extra setup (cgroupfs must be mounted by init). |
| **Value** | Elegant in theory, fragile in practice. Kernel-enforced isolation is the gold standard, but most deployments won't grant the privileges. |

**Verdict:** Defer. The right answer for multi-tenant production, but the operational
cost is high. Revisit when per-session resource quotas (Phase 23) are on the table —
cgroups are the enforcement mechanism, not just an observability tool.

---

## Option G: eBPF / bpftrace Probes

Attach eBPF programs to the Go binary using USDT (User Statically-Defined Tracing)
probes, or use bpftrace for dynamic instrumentation.

| What | Value |
|------|-------|
| **Effort** | Very high — requires kernel support, eBPF toolchain, USDT probe insertion in Go |
| **Portability** | Linux 4.x+, not available in scratch containers without additional setup |
| **Value** | Zero for internal daemon observability. eBPF is for kernel-level tracing and performance debugging, not for exposing application metrics that the app already tracks. |

**Verdict:** Out of scope. eBPF is a kernel observability tool, not an application
metrics transport. Use it to trace *why* the daemon is slow, not to monitor *whether*
it's healthy.

---

## Summary Matrix

| Opt | What `top`/`ps` sees? | Effort | New deps? | Portability | Value |
|:---:|------------------------|:------:|:---------:|:-----------:|:-----:|
| A | `htop` shows named worker threads | Medium | ❌ | Linux only | Low |
| B | `ps aux` shows live status | 15 min | ❌ or 1 lib | Linux | Med |
| **C** | Nothing (Prometheus) | **30 min** | ❌ | **All** | **High** |
| **D** | Nothing (JSON) | **50 min** | ❌ | **All** | **High** |
| **E** | Self-contained TUI | 1–3 hr | ❌ or bubbletea | **All** | **High** |
| F | `systemd-cgtop` per-session | High | cgroup mgmt | Linux only | Med |
| G | Nothing (eBPF toolchain) | Very high | eBPF | Linux only | Zero |

---

## Recommended Implementation Phases

If approved, implement in this order (each builds on the previous):

| Phase | What | Time | Dependency |
|:-----:|------|:----:|------------|
| 1 | **C** — Runtime stats in `/metrics` | 30 min | None |
| 2 | **D** — JSON `/status` endpoint | 50 min | None (reads existing counters) |
| 3 | **E** — `gotop` TUI (`pkg/gotop/`) | 1–3 hr | Phase 2 (`/status` data source) |
| 4 | **B** — Process name in `ps` | 15 min | None (nice-to-have bonus) |

Total: **~3–5 hours** for full observability. Options A, F, G are deferred or rejected.

### Phase 1 alone: immediate Grafana value (30 min)
```
curl localhost:9090/metrics | grep goroutines
goposix_goroutines 34
goposix_heap_alloc_bytes 1.29e+07
goposix_gc_pause_ns 412000
```

### Phase 1+2: any tool can monitor (80 min)
```
curl -s localhost:9090/status | jq '.mem.heap_alloc_mb'
12.3
```

### Phase 1+2+3: `goposix gotop` (2–4 hr total)
```
┌─ goposix daemon @ :9090 ──── uptime: 3d 7h ─────────────────┐
│ Goroutines: 34    Workers: [████░░] 2/4    Sessions: 3      │
│ Heap: 12.3 MB     RSS: 45.1 MB            GC pause: 0.4ms   │
│ ...                                                         │
```
