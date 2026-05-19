# Phase 19 — Performance Benchmarking (GoPOSIX vs BusyBox)

> **Status:** DONE | **Date:** 2026-05-18 | **Key finding:** Go SDK achieves 60µs per RPC call — 11× faster than BusyBox fork+exec (680µs)

Benchmarked on 16 vCPU host, Docker with `--privileged`, tmpfs for bulk ops. All numbers are medians. Full raw data in `test/benchmark/results/`.

---

## 1. Key Results

### Three-Mode Comparison

| Interface | Per-call latency | vs BusyBox |
|-----------|:---:|:---:|
| `socat` per call (old approach) | 2,000µs | **3× slower** |
| BusyBox `fork+exec` | 680µs | baseline |
| **Go SDK persistent conn** | **60µs** | **11× faster** |

### Category Results (SCALE=1.0)

| Category | GoPOSIX | BusyBox | Winner | Key Metric |
|----------|:---:|:---:|:---:|------|
| A — Startup (`true`) | 6.8ms | 3.7ms | BusyBox 1.8× | Go runtime init ~4ms |
| B — Bulk Create (10K touch) | 5.1s | 2.7s | BusyBox 1.9× | Fork overhead amortized |
| C — Bulk LS (10K files) | 0.22s | 0.07s | BusyBox 3.1× | Both VFS-bound |
| D — Bulk Move/RM (1K) | ±5% | ±5% | Tie | Kernel-bottlenecked |
| **E — grep (100MB)** | **0.16s** | 0.86s | **GoPOSIX 5.4×** | RE2 vs POSIX ERE |
| E — wc/sort/cat | varies | varies | BusyBox ~1.3× | I/O-bound ops |
| E — grep -r (1K files) | slower | faster | BusyBox 22× | BusyBox recursive is highly optimized |
| F — Daemon via socat | 2.0ms | 0.68ms | BusyBox 2.8× | socat per-call overhead |
| **F — Daemon via Go SDK** | **0.06ms** | 0.68ms | **GoPOSIX 11×** | Persistent connection pooling |
| G — Memory (RSS) | 29MB | 3KB | BusyBox 9.5× | Go runtime arena vs C `brk()` |
| H — Binary Size | 8.7MB | 790KB | BusyBox 11.2× | Static Go binary vs stripped C |
| I — Concurrent | — | — | Tie | Goroutine parallelization not yet implemented |
| J — RPC Task Loop (SDK) | faster | — | GoPOSIX | 5 typed calls/iter, 2.1× BusyBox |

### Break-Even

**~3 RPC calls.** SDK connection setup cost is amortized after 3 calls. After that,
every additional SDK call is 60µs vs BusyBox 680µs — 11× savings per call.

### Honest Narrative

> BusyBox wins on resource economy (size 11×, memory 9×, single-shot 1.8×).
> GoPOSIX wins on sustained programmatic throughput via the Go SDK: 60µs per RPC.
> GoPOSIX grep is 5.4× faster on large files (RE2 engine).
> The daemon architecture was always correct — socat was the bottleneck.

---

## 2. Benchmark Harness

All benchmarks run in Docker containers for identical kernel, isolated cgroups, and no
host-noise contamination. A single `BENCH_SCALE` environment variable controls all workload
sizes (0.1 = smoke, 1.0 = standard, 5.0 = publication).

| Tier | SCALE | Time | Use Case |
|------|:-----:|------|----------|
| smoke | 0.1 | ~30s | CI pre-merge |
| dev | 0.5 | ~3min | Local iteration |
| standard | 1.0 | ~8min | Cross-commit comparison |
| publication | 5.0 | ~40min | Blog/conference numbers |
| stress | 25.0 | ~3h | Find asymptotic cliffs |

Commands: `make bench-quick SCALE=0.1` (smoke), `make bench-all SCALE=1.0` (full).

---

## 3. Implementation (all scripts in `test/benchmark/`)

| Script | Purpose |
|--------|---------|
| `lib/harness.sh` | Timing, stats, `scaled()` helper |
| `lib/report.sh` | `summary.md` + `narrative.md` generator |
| `runner.sh` | Master orchestrator (`--all`, `--quick`, `--cat`) |
| `cat_a_startup.sh` through `cat_j_rpc_loop.sh` | 10 category scripts |
| `bench_client/` | Go SDK benchmark client (60µs measurement) |
| `Dockerfile.bench` | Alpine + GoPOSIX + BusyBox + tooling |

Results output: `test/benchmark/results/<timestamp>/summary.md`, `raw.csv`, `narrative.md`

---

## 4. Methodology

- 3 warm-up runs discarded, 10 measured samples, median/p95/min/max reported
- `BENCH_SCALE` controls workload sizes via `parameter = base × SCALE`
- Hard caps: 500K files, 10GB text, 100K daemon requests, 1K loop iterations
- See [performance.md](performance.md) for the quick-reference guide
