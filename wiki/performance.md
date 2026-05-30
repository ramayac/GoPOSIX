# Performance Benchmarking — Quick Reference

> **Branch:** `feat/performance` | **Status:** IMPLEMENTING | **Plan:** [30_performance_improvements.md](30_performance_improvements.md)

---

## TL;DR

```bash
make bench-all SCALE=1.0    # 8 min — standard comparison
make bench-pub               # 40 min — publication quality (SCALE=5.0)
make bench-quick SCALE=0.1  # 30 sec — CI smoke test
```

---

## Commands

### Run Benchmarks

| Command | What It Does | Approx Time |
|---------|-------------|:-----------:|
| `make bench-quick SCALE=0.1` | Cat A (startup) + Cat H (sizes) | ~30 s |
| `make bench-all SCALE=1.0` | All 10 categories | ~8 min |
| `make bench-smoke` | Alias for `bench-all SCALE=0.1` | ~30 s |
| `make bench-pub` | Alias for `bench-all SCALE=5.0` | ~40 min |
| `make bench-stress` | Alias for `bench-all SCALE=25.0` | ~3 h |
| `make bench-cat CAT=a SCALE=1.0` | Single category | varies |
| `make bench-shell` | Interactive shell in bench container | — |

### View Results

| Command | What It Does |
|---------|-------------|
| `make bench-fetch` | Copy results from Docker volume to `test/benchmark/results/` |
| `make bench-report` | Generate `summary.md` + `narrative.md` from latest results |

### Build

```bash
make bench-image    # Build the benchmark Docker image (goposix:bench)
```

### Clean

```bash
docker volume rm goposix-bench-data    # Nuke all results
docker rmi goposix:bench               # Remove the benchmark image
```

---

## Scale Factor (`SCALE`)

All workload sizes multiply by `SCALE`. Default is `1.0`.

| Value | Tier | Files | Text | Daemon Reqs | Use Case |
|:-----:|------|------:|-----:|:-----------:|----------|
| 0     | null | 0 | 0 | 0 | Static-only (Cat H only) |
| 0.1   | smoke | 100 | 10 MB | 10 | CI pre-merge |
| 0.5   | dev | 500 | 50 MB | 50 | Local iteration |
| 1.0   | standard | 1,000 | 100 MB | 100 | Daily baseline |
| 5.0   | publication | 5,000 | 500 MB | 500 | Blog / conf |
| 25.0  | stress | 25,000 | 2.5 GB | 2,500 | Find cliffs |
| 100.0 | extreme | 100,000 | 10 GB | 10,000 | Prove ceiling |

Hard caps: 500K files, 10 GB text, 100K daemon requests, 1K RPC task loops.

---

## Categories

| Key | Full Name | Friendly | What It Measures |
|:---:|-----------|----------|------------------|
| `a` | `cat_a_startup` | `startup` | Cold-start latency: `true`, `echo`, `pwd`, `whoami` |
| `b` | `cat_b_bulk_create` | `bulk_create` | Bulk `touch` on N files |
| `c` | `cat_c_bulk_ls` | `bulk_ls` | `ls -1` and `ls -la` on N files |
| `d` | `cat_d_bulk_move` | `bulk_move` | `mv` N files, then `rm` N files |
| `e` | `cat_e_text_throughput` | `text` | `cat`, `wc`, `grep`, `sort`, `grep -r` on scaled text |
| `f` | `cat_f_daemon_vs_process` | `daemon` | JSON-RPC daemon vs fork+exec — **killer feature** |
| `g` | `cat_g_memory` | `memory` | RSS: single, idle daemon, loaded daemon, BusyBox |
| `h` | `cat_h_sizes` | `sizes` | Binary size, symlink count (no runtime) |
| `i` | `cat_i_concurrent` | `concurrent` | Concurrent grep/du [GOROUTINE-TODO] |
| `j` | `cat_j_rpc_loop` | `rpc` | RPC task loop: ls→cat→grep→wc→find, N iterations |

```bash
# Any of these work:
make bench-cat CAT=f
make bench-cat CAT=daemon
make bench-cat CAT=cat_f_daemon_vs_process
```

---

## Expected Results (Priors)

### BusyBox Wins Here (no contest)

| Metric | GoPOSIX | BusyBox | Ratio |
|--------|--------:|--------:|:-----:|
| Binary size | ~8.6 MB | ~790 KB | **11:1** |
| Startup (`true`) RSS | ~29 MB | ~3.4 KB | **9:1** |
| Single `echo` wall time | ~10 ms | ~0.5 ms | **20:1** |

### GoPOSIX Wins Here

| Metric | Expected Margin | Why |
|--------|:--------------:|-----|
| Daemon 1,000 sequential calls | **5–100×** | No fork+exec per call |
| RPC task loop 50 iterations | **10–50×** | Connection reuse, no shell parsing |
| Concurrent file traversal | **2–8×** (aspirational) | Go goroutines |

### Break-even

For **10–50 sequential operations**, GoPOSIX daemon overcomes its startup cost.
Below that, BusyBox wins. Above that, GoPOSIX dominates.

---

## Output Files

After a run, results land in the Docker volume `goposix-bench-data`:

```
/data/results/2026-05-18T120000_scale1.0/
├── summary.md           ← Human-readable with formatted tables
├── narrative.md         ← Prose for README / blog posts
├── raw.csv              ← Machine-readable: category,test,sample,wall,user,sys,rss
├── run_config.txt       ← Scale, host, CPU count
├── cat_h_sizes_data.csv
├── cat_a_startup_data.csv
├── cat_a_startup.log    ← Category log with medians & findings
├── ...
└── cat_j_rpc_loop.log
```

**To get them locally:**

```bash
make bench-fetch
# Results appear at: test/benchmark/results/2026-05-18T120000_scale1.0/
make bench-report
# Opens the latest result
```

---

## Architecture

```
┌─────────────────────────────────────────────┐
│                make bench-all                │
│                   SCALE=1.0                  │
└─────────────────┬───────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────┐
│          Docker: goposix:bench               │
│  ┌─────────────────────────────────────┐    │
│  │         runner.sh (orchestrator)     │    │
│  │    sources lib/harness.sh (timing)   │    │
│  └──────────┬──────────────────────────┘    │
│             │ for each category:             │
│             ▼                                │
│  ┌──────────────────────────────────────┐   │
│  │  cat_a.sh  cat_b.sh  ...  cat_j.sh   │   │
│  │  ┌─────────────┐  ┌───────────────┐  │   │
│  │  │ bench_run()  │  │  scaled()     │  │   │
│  │  │ time -f ...  │  │  N = base × S │  │   │
│  │  └──────┬───────┘  └───────────────┘  │   │
│  │         │                              │   │
│  │         ▼                              │   │
│  │    /bin/goposix ls       CSV rows →    │   │
│  │    /bin/busybox ls       stdout        │   │
│  └──────────────────────────────────────┘   │
│                                             │
│  ┌──────────────────────────────────────┐   │
│  │         lib/report.sh                 │   │
│  │    summary.md + narrative.md          │   │
│  └──────────────────────────────────────┘   │
└─────────────────────────────────────────────┘
```

---

## Adding a New Category

1. Create `test/benchmark/cat_k_newthing.sh`:

    ```bash
    #!/bin/sh
    set -u
    . "$(dirname "$0")/lib/harness.sh"
    N=$(scaled 1000)
    echo "category,test,sample,wall_sec,user_sec,sys_sec,rss_kb"
    bench_run "newthing_${N}_goposix" 5 "/bin/goposix <args>"
    bench_run "newthing_${N}_busybox" 5 "/bin/busybox <args>"
    ```

2. `chmod +x test/benchmark/cat_k_newthing.sh`
3. Add to `ALL_CATEGORIES` in `runner.sh`
4. Optionally add to a `QUICK_CATEGORIES` group
5. Rebuild: `make bench-image`

---

## Troubleshooting

| Symptom | Fix |
|---------|-----|
| `Permission denied` in `/data/` | Rebuild image (`make bench-image`) — the volume may have stale permissions |
| `nc: command not found` | Shouldn't happen — BusyBox provides `nc` in Alpine |
| `daemon failed to start` | Check `dmesg` for Unix socket limits; try `ulimit -n 4096` |
| Wall times = `0.00` | Normal for sub-10ms ops. BusyBox `time` precision is centiseconds. Use higher SCALE or heavier commands. |
| `File exists` warnings during build | Expected — Alpine BusyBox symlinks collide with GoPOSIX symlinks. GoPOSIX wins the race. Benign. |

---

## See Also

- [Architecture](architecture.md) — GoPOSIX component layout
- [Architecture](architecture.md) — GoPOSIX component layout
- Speed Targets: `<1ms daemon latency, <15MB binary, <100ms CLI startup` (see [phases.md](phases.md))


---

## Performance Optimizations (12/30 completed)

> **Status:** PARTIALLY IMPLEMENTED | **Date:** 2026-05-23

### Completed (Sprint 1 & 2)

1. **wc Ultra-Fast ASCII Scanning** — 64KB buffer Peek-based scanning. 10KB of text in 25µs, 2 allocations.
2. **tr Translation & Squeezing (100–500× speedup)** — Cached squeeze sets, 32KB `bufio.Writer`.
3. **ls UID/GID Cache & Buffered Printing** — `sync.Map` caches with 30s TTL, 32KB buffered writer.
4. **Daemon latency 2× reduction** — `BenchmarkDaemonLs` from 1.18ms to 0.61ms/op.
5. Zero functional regressions across unit and BusyBox tests.

### Remaining Optimizations (18)

| # | Improvement | Severity | Tool |
|---|-------------|----------|------|
| 1 | Eliminate double JSON serialization in daemon | 🔴 Critical | daemon |
| 2 | `sync.Pool` for JSON encoder/decoder | 🟡 High | daemon |
| 3 | Replace `fmt.Sprintf` with `strconv.AppendInt` | 🟡 High | cat, wc, grep, ls, sort |
| 4 | grep: `bytes.Contains` for fixed-string mode | 🟡 High | grep |
| 5 | grep `-r`: parallelize directory walk | 🟡 High | grep |
| 6 | grep `scanWithContext()`: sliding window | 🟡 High | grep |
| 7 | sort: pre-allocate `lineItem` slices | 🟡 High | sort |
| 8 | sort: `bufio.Writer` for output | 🟡 High | sort |
| 9 | sed: `bufio.Writer`, reduce `fmt.Fprint` | 🟡 High | sed |
| 10 | cat: `io.Copy` unbuffered, pre-computed lookup table | 🟡 High | cat |
| 11 | ls: `DirEntry.Info()` instead of re-statting | 🟢 Medium | ls |
| 12 | find: parallelize directory walk | 🟡 High | find |
| 13 | dd: larger default block sizes | 🟢 Medium | dd |
| 14 | Client SDK: buffer RPC writes | 🟢 Medium | client |
| 15 | wc: memory-mapped I/O for large files | 🟢 Medium | wc |
| 16 | tr: use `bytes.Map` for single-byte translations | 🟢 Medium | tr |
| 17 | cp: `io.CopyBuffer` with larger buffer | 🟢 Medium | cp |
| 18 | daemon: connection write pooling | 🟢 Medium | daemon |
