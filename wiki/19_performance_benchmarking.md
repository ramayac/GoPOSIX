# Phase 19 — Performance Benchmarking (GoPOSIX vs BusyBox)

> **Status:** IMPLEMENTING | **Date:** 2026-05-18 | **Author:** Deepseek v4 pro | **Branch:** `feat/performance`
>
> This document defines a rigorous, reproducible, and **honest** performance benchmarking
> framework comparing GoPOSIX against BusyBox. The goal is not to cherry-pick wins but to
> produce defensible evidence of where each tool excels, informing adoption decisions for
> AI agent backends and minimal container deployments.

---

## 1. Why Benchmark?

GoPOSIX's primary adoption pitch is:

> "A Go-native POSIX userland designed for agentic runtimes — persistent JSON-RPC daemon,
>  structured output, zero-fork execution loops."

The performance argument is implicit: **a persistent daemon avoids process-spawning overhead
for repeated operations**. But this claim has never been measured. Neither have the tradeoffs
(Go runtime startup cost, binary size, memory footprint).

A rigorous benchmark suite serves three purposes:

| Purpose | Audience |
|---------|----------|
| **Adoption evidence** | Engineers evaluating GoPOSIX vs BusyBox for agent backends |
| **Marketing** | README badges, blog posts, conference talks |
| **Hot-path identification** | Developers optimizing the GoPOSIX codebase |

---

## 2. Honest Priors

Before writing a single line of benchmark code, we state our expectations honestly.
These priors come from understanding Go's runtime characteristics vs C's
and from the architecture of both tools.

### Where BusyBox Will Win (no contest)

| Metric | Why | Expected margin |
|--------|-----|-----------------|
| **Binary size** | C, stripped, no runtime | ~800 KB vs ~10 MB (12:1) |
| **Container image size** | BusyBox + musl vs Go static + tzdata + ca-certs | ~7 MB vs ~12 MB |
| **Single-invocation cold start** | C enters `main()` in µs; Go pays runtime init, GC bootstrap | 10–50× |
| **Per-invocation RSS** | C `brk()`-only heap vs Go's pre-allocated arena | 5–20× |

### Where GoPOSIX Can Win

| Metric | Why | Expected margin |
|--------|-----|-----------------|
| **Daemon amortized latency** | One process, N requests; BusyBox pays fork+exec per call | 5–100× for bulk calls |
| **Concurrent operations** | Go goroutines for parallel file traversal, regex matching | 2–8× on multi-core |
| **Structured output latency** | JSON emitted directly; BusyBox needs `| jq` or parsing | 1.5–3× |
| **Agent loop throughput** | No shell invocation, no parsing, typed client SDK | 10–50× end-to-end |

### Where it's a Fair Fight

| Metric | Why | Expected |
|--------|-----|----------|
| **Text I/O throughput** | Both saturate pipe/disk bandwidth for simple filters (cat, wc) | ±10% |
| **Bulk filesystem ops** | Both bottleneck on kernel VFS; overhead difference matters less | ±20% |
| **Regex matching** | Go `regexp` is RE2 (linear, no backrefs) vs BusyBox POSIX ERE | Depends on pattern |

---

## 3. Test Harness Design

### 3.1 Sandboxing

All benchmarks run inside **Docker containers** to ensure:
- Identical kernel (host kernel shared)
- Identical filesystem (tmpfs for bulk ops to eliminate disk I/O variance)
- Isolated CPU/memory accounting via cgroups
- No host-noise contamination

We use two images:

| Image | Binary | Size | Role |
|-------|--------|------|------|
| `goposix:bench` | GoPOSIX compiled with `-ldflags="-s -w"` | ~12 MB | GoPOSIX under test |
| `alpine:3.20` | BusyBox v1.36.1 (808 KB) | ~7 MB | Baseline under test |

> **Note:** We compile a dedicated `goposix:bench` image that includes `time`, `strace`,
> and benchmark scripts. This is distinct from the production `scratch` image and the
> `goposix:debug` image. It builds on the debug Dockerfile pattern but adds benchmarking
> tooling and a shell.

### 3.2 Metrics Collected

For every benchmark run, collect:

| Metric | Tool | Unit |
|--------|------|------|
| Wall-clock elapsed | `/usr/bin/time -v` or `time` builtin | seconds (3 d.p.) |
| User CPU time | `/usr/bin/time -v` | seconds (2 d.p.) |
| System CPU time | `/usr/bin/time -v` | seconds (2 d.p.) |
| Max RSS | `/usr/bin/time -v` | KB |
| Exit code | `$?` | integer |
| Binary size | `ls -la` | bytes |
| Image size | `docker images` | MB |

### 3.3 Statistical Rigor

- **Warm-up:** 3 discarded warm-up runs before measurement (cache priming)
- **Samples:** 10 measured runs per scenario
- **Reporting:** median, p95, min, max for each metric
- **Outlier detection:** any run deviating >3σ is flagged and re-run
- **Cooldown:** 1s sleep between samples to avoid thermal throttling artifacts

### 3.4 Scale Configuration — One Knob for All Workload Sizes

Every benchmark script reads a single environment variable: `BENCH_SCALE` (float, default `1.0`).
All workload sizes — file counts, iteration counts, text file sizes, request counts — are
computed as `base × BENCH_SCALE`, clamped to integer ≥1. This single knob lets you dial
between a fast smoke test and a multi-hour stress run without editing any script.

#### Scale Tiers

| Tier | `BENCH_SCALE` | Time (approx) | Use Case |
|------|:------------:|---------------|----------|
| **smoke** | 0.1 | ~30 s | CI pre-merge gate ("does it still run?") |
| **dev** | 0.5 | ~3 min | Local iteration during development |
| **standard** | 1.0 | ~8 min | Default — cross-commit comparison |
| **publication** | 5.0 | ~40 min | Blog post / conference talk numbers |
| **stress** | 25.0 | ~3 h | Find asymptotic cliffs, GC pressure, kernel limits |
| **extreme** | 100.0 | ~12 h | Prove scaling ceiling; only on dedicated hardware |

> `BENCH_SCALE=0` is valid — it skips all variable-workload benchmarks and runs only
> fixed-cost categories (Cat A startup, Cat H sizes, Cat G single-invocation RSS).

#### Scale-to-Workload Mapping

The table below shows how `BENCH_SCALE` translates to concrete parameter values.
When a category has multiple N levels, each level scales independently.

| Category | Parameter | Base (`SCALE=1`) | `SCALE=0.1` | `SCALE=5` | `SCALE=25` |
|----------|-----------|:----------------:|:-----------:|:---------:|:----------:|
| B — touch | file count | 1,000 | 100 | 5,000 | 25,000 |
| C — ls | file count | 1,000 / 10,000 | 100 / 1,000 | 5,000 / 50,000 | 25,000 / 250,000 |
| D — mv/rm | file count | 1,000 | 100 | 5,000 | 25,000 |
| E — text | big.txt size | 100 MB | 10 MB | 500 MB | 2.5 GB |
| E — grep -r | small file count | 1,000 | 100 | 5,000 | 25,000 |
| F — daemon | request count | 10 / 50 / 100 / 1,000 | 1 / 5 / 10 / 100 | 50 / 250 / 500 / 5,000 | 250 / 1,250 / 2,500 / 25,000 |
| G — mem | concurrent reqs | 100 | 10 | 500 | 2,500 |
| J — agent | loop iterations | 10 | 1 | 50 | 250 |

> **Hard cap:** Individual workload parameters are clamped to prevent runaway runs.
> Max file count: 500,000. Max text file: 10 GB. Max daemon requests: 100,000.
> Max loop iterations: 1,000. Crossing a cap emits a warning in the results but
> does not abort the run.

#### How Scripts Consume It

```bash
# Every benchmark script starts with:
BENCH_SCALE="${BENCH_SCALE:-1.0}"

# Workload parameters are computed:
FILE_COUNT=$(awk "BEGIN { printf \"%d\", int(1000 * $BENCH_SCALE + 0.5) }")
MAX_FILE_COUNT=500000
if [ "$FILE_COUNT" -gt "$MAX_FILE_COUNT" ]; then
  echo "WARNING: file count $FILE_COUNT capped at $MAX_FILE_COUNT" >&2
  FILE_COUNT=$MAX_FILE_COUNT
fi

# Safety floor: never go below 1
if [ "$FILE_COUNT" -lt 1 ]; then FILE_COUNT=1; fi
```

#### Runner Integration

The master `runner.sh` passes `BENCH_SCALE` into the Docker container and stamps it
into the results directory name so you can compare runs at different scales:

```
test/benchmark/results/
├── 2026-05-18T120000_scale1.0/    # standard
├── 2026-05-18T120500_scale0.1/    # smoke (CI)
├── 2026-05-18T121000_scale5.0/    # publication
└── latest -> 2026-05-18T121000_scale5.0/
```

The `summary.md` header includes the scale factor so reports are self-documenting:

```markdown
# GoPOSIX vs BusyBox — Performance Benchmark
> Run: 2026-05-18T12:10:00Z | Scale: 5.0× | Duration: 38m 12s | Host: AMD EPYC 7763, 64C
```

#### Makefile Invocation

```bash
# Dev iteration (fast)
make bench-all SCALE=0.5

# Publication-quality numbers
make bench-all SCALE=5.0

# CI gate (smoke)
make bench-quick SCALE=0.1

# Stress test overnight
make bench-all SCALE=25.0
```

---

## 4. Benchmark Categories

### Cat A — Single-Invocation Latency (Startup Overhead)

**Hypothesis:** BusyBox wins by 10–50× due to Go runtime initialization.

**Method:** Time a single invocation of the simplest possible command.

| Test | Command | Description |
|------|---------|-------------|
| A1 | `true` | Zero-work command, pure startup overhead |
| A2 | `echo hello` | Minimal work (write+exit) |
| A3 | `pwd` | One syscall (getcwd) |
| A4 | `whoami` | One syscall (getuid → /etc/passwd) |

**For each:** 10 samples, warm cache, in-container timing.

**GoPOSIX invocation:** `goposix true`
**BusyBox invocation:** `busybox true`

---

### Cat B — Bulk File Creation (touch, mkdir)

**Hypothesis:** Roughly even. Both bottleneck on VFS; GoPOSIX might have slight edge
due to goroutine-parallel touch.

**Method:** Create N empty files in parallel on a tmpfs, time the total operation.

| Test | N (at `SCALE=1`) | Command |
|------|:-----------------:|---------|
| B1 | 100 | `for i in $(seq $N); do touch /tmp/bench/$i; done` |
| B2 | 1,000 | same |
| B3 | 10,000 | same |
| B4 | 10,000 (parallel) | `xargs -P4 touch` pattern (if supported) |

> **Scaling:** All N values scale linearly with `BENCH_SCALE`. N is computed as
> `int(base × BENCH_SCALE)`. At `SCALE=0.1`, B3 runs with N=1,000.
> At `SCALE=5`, B3 runs with N=50,000.

**Pre-condition:** `mkdir -p /tmp/bench && rm -rf /tmp/bench/*`

---

### Cat C — Bulk Directory Listing (ls)

**Hypothesis:** BusyBox `ls` uses `getdents64` directly; GoPOSIX uses `os.ReadDir` +
sort. Sorting overhead may be measurable at scale.

**Method:** Create N files, time a single `ls -1` (no sort, list only) and `ls -la`
(full stat).

| Test | N (at `SCALE=1`) | Command |
|------|:-----------------:|---------|
| C1 | 1,000 | `ls -1 /tmp/bench` |
| C2 | 10,000 | `ls -1 /tmp/bench` |
| C3 | 10,000 | `ls -la /tmp/bench` |

> **Scaling:** N = `int(base × BENCH_SCALE)`. At `SCALE=25`, C2 creates and lists
> 250,000 files. Ensure tmpfs has sufficient inodes (`-o nr_inodes=500000`).

---

### Cat D — Bulk File Move / Remove (mv, rm)

**Hypothesis:** Both bottleneck on VFS `rename` syscall; overhead difference negligible.

| Test | N (at `SCALE=1`) | Command |
|------|:-----------------:|---------|
| D1 | 1,000 | `mv /tmp/bench/* /tmp/bench2/` |
| D2 | 1,000 | `rm -rf /tmp/bench2` |

> **Scaling:** N = `int(1000 × BENCH_SCALE)`. D1+D2 form a coupled pair that must
> complete together — files moved in D1 become the files removed in D2. They share
> the same scaled N.

---

### Cat E — Text Processing Throughput (cat, grep, wc, sort)

**Hypothesis:** Both saturate memory bandwidth for simple filters. GoPOSIX may have
advantage on `grep -r` (concurrent file traversal).

**Method:** Generate a large text file (100 MB × `BENCH_SCALE` of line-based text), time operations.

| Test | Command | Workload (at `SCALE=1`) |
|------|---------|:------------------------:|
| E1 | `cat /tmp/big.txt > /dev/null` | 100 MB |
| E2 | `wc -l /tmp/big.txt` | 100 MB |
| E3 | `grep -c "pattern" /tmp/big.txt` | 100 MB |
| E4 | `sort /tmp/big.txt > /dev/null` | 100 MB (CPU-bound) |
| E5 | `grep -r "pattern" /tmp/manyfiles/` | 1,000 small files |

> **Scaling:** Text file size = `int(100 MB × BENCH_SCALE)`, capped at 10 GB.
> Small file count for E5 = `int(1000 × BENCH_SCALE)`. At `SCALE=5`, E4 sorts
> 500 MB — ensure the container has sufficient memory or swap.

---

### Cat F — Daemon vs Process-per-Call (The GoPOSIX Killer Feature)

**Hypothesis:** For N sequential calls, GoPOSIX daemon amortizes startup cost.
BusyBox pays fork+exec per call. Break-even is expected around N=5–15 calls.

**Method:** Make N sequential calls to the same utility, timed end-to-end.

| Test | N (at `SCALE=1`) | Calls | GoPOSIX Mode | BusyBox Mode |
|------|:-----------------:|-------|-------------|-------------|
| F1 | 100 | `echo hello` | daemon JSON-RPC | process spawn |
| F2 | 1,000 | `echo hello` | daemon JSON-RPC | process spawn |
| F3 | 100 | `ls /tmp` | daemon JSON-RPC | process spawn |
| F4 | 100 | `stat /etc/hostname` | daemon JSON-RPC | process spawn |

> **Scaling:** Request counts = `int(base × BENCH_SCALE)`. At `SCALE=25`, F2 runs
> 25,000 sequential daemon calls. Cap: 100,000. This category is the strongest
> scaling story — larger N widens the GoPOSIX advantage.

**GoPOSIX daemon setup:**
```sh
goposix daemon --socket /tmp/goposix-bench.sock &
# then N JSON-RPC calls via netcat or Go client
```

**BusyBox setup:**
```sh
for i in $(seq N); do busybox echo hello > /dev/null; done
```

**Key output:** Latency amortization curve (ms per call vs N). The asymptote is the
per-request dispatch time in the GoPOSIX daemon vs the per-call `fork+exec` time in BusyBox.

---

### Cat G — Memory Footprint

**Hypothesis:** BusyBox uses 5–20× less RSS per invocation. GoPOSIX daemon has a
fixed RSS cost but amortizes it.

**Method:** Measure RSS during steady-state operation.

| Test | Description | Workload (at `SCALE=1`) |
|------|-------------|:------------------------:|
| G1 | Single `echo` invocation RSS | 1 invocation |
| G2 | GoPOSIX daemon idle RSS (0 requests) | 0 requests |
| G3 | GoPOSIX daemon under load | 100 concurrent requests |
| G4 | BusyBox sequential echo calls (peak observed RSS) | 100 sequential calls |

**Tool:** `ps -o rss,vsz,cmd` or `/usr/bin/time -v`

> **Scaling:** G3 and G4 concurrent/sequential counts = `int(100 × BENCH_SCALE)`.
> At `SCALE=25`, G3 fires 2,500 concurrent goroutine-scheduled requests.
> G1 is fixed (1 invocation), G2 is fixed (idle baseline).

---

### Cat H — Binary & Image Size (Static Analysis)

**Hypothesis:** BusyBox wins 12:1 on binary size, ~2:1 on image size.

**Method:** Static measurement of built artifacts.

| Metric | GoPOSIX | BusyBox (alpine:3.20) |
|--------|---------|------------------------|
| Binary size (stripped) | `ls -la goposix` | `ls -la /bin/busybox` |
| Docker image size | `docker images goposix:latest` | `docker images alpine:3.20` |
| Function count (symlinks) | `--list-commands | wc -l` | `busybox --list | wc -l` |

---

### Cat I — Concurrent Operations (Go's Goroutine Advantage)

**Hypothesis:** GoPOSIX can parallelize filesystem traversal and text processing
using goroutines. BusyBox is strictly sequential (single-threaded C).

**Method:** Operations that can benefit from concurrency.

| Test | Command | Description |
|------|---------|-------------|
| I1 | `find /tmp -name "*.txt" -exec wc -l {} \;` | Sequential by nature |
| I2 | `grep -r "pattern" /tmp/manyfiles/` | May parallelize file reads |
| I3 | `du -sh /tmp/manyfiles/` | Recursive disk usage |

> **Note:** GoPOSIX's `find`, `grep`, and `du` utilities would need to be
> instrumented or rewritten to use goroutine-parallel traversal for this
> benchmark to show an advantage. This category is aspirational — it measures
> the *potential* of Go, not the current implementation. Mark tests as
> `[GOROUTINE-TODO]` until parallel implementations exist.

---

### Cat J — End-to-End Agent Loop Simulation

**Hypothesis:** This is the benchmark that matters most for adoption. A realistic
AI agent loop (list files → read file → search → filter) run through the GoPOSIX
daemon vs BusyBox via shell spawns.

**Method:** Simulate a typical agent task flow:

```
1. ls -la /workspace
2. cat /workspace/README.md
3. grep "TODO" /workspace/README.md
4. wc -l /workspace/README.md
5. find /workspace -name "*.go" | head -20
```

Run this sequence `ITERATIONS` times, timed end-to-end.

| Mode | How |
|------|-----|
| GoPOSIX daemon | Go client SDK (connection reused) or bash loop hitting Unix socket |
| BusyBox | Shell script `for` loop with process-per-command |

**Metric:** Total wall-clock time for `ITERATIONS` iterations (default 10, scales
with `BENCH_SCALE`). GoPOSIX daemon should win decisively here.

> **Scaling:** `ITERATIONS = int(10 × BENCH_SCALE)`, capped at 1,000.
> At `SCALE=5`, the agent loop runs 50 iterations — this is where the gap
> between persistent daemon and process-per-command becomes a crater.

---

## 5. Implementation Plan

### 5.1 File Layout

```
test/benchmark/
├── bench_daemon_test.go       # Existing Go daemon micro-benchmarks
├── Dockerfile.bench            # Benchmark Docker image (GoPOSIX + time + strace)
├── runner.sh                   # Master benchmark runner (runs all categories)
├── lib/
│   ├── harness.sh             # Shared timing functions, stats collection
│   └── report.sh              # Markdown/CSV/JSON report generator
├── cat_a_startup.sh
├── cat_b_bulk_create.sh
├── cat_c_bulk_ls.sh
├── cat_d_bulk_move.sh
├── cat_e_text_throughput.sh
├── cat_f_daemon_vs_process.sh
├── cat_g_memory.sh
├── cat_h_sizes.sh
├── cat_i_concurrent.sh
├── cat_j_agent_loop.sh
└── results/                    # Timestamped benchmark results
    └── 2026-05-18T000000/
        ├── summary.md
        ├── raw.csv
        └── cat_*.log
```

### 5.2 Makefile Targets

```makefile
SCALE ?= 1.0

# Build the benchmark Docker image
.PHONY: bench-image
bench-image:
	docker build -t goposix:bench -f test/benchmark/Dockerfile.bench .

# Run all benchmarks inside Docker
.PHONY: bench-all
bench-all: bench-image
	docker run --rm --privileged \
	  -e BENCH_SCALE=$(SCALE) \
	  -v /tmp/goposix-bench:/data \
	  goposix:bench /bench/runner.sh --all

# Run a single category (e.g., make bench-cat CAT=startup SCALE=5)
.PHONY: bench-cat
bench-cat: bench-image
	docker run --rm --privileged \
	  -e BENCH_SCALE=$(SCALE) \
	  -v /tmp/goposix-bench:/data \
	  goposix:bench /bench/runner.sh --cat $(CAT)

# Quick sanity check (Cat A + Cat H only, ~30s at SCALE=0.1)
.PHONY: bench-quick
bench-quick: bench-image
	docker run --rm --privileged \
	  -e BENCH_SCALE=$(SCALE) \
	  -v /tmp/goposix-bench:/data \
	  goposix:bench /bench/runner.sh --quick

# Generate report from latest results
.PHONY: bench-report
bench-report:
	@test/benchmark/lib/report.sh test/benchmark/results/$$(ls -t test/benchmark/results/ | head -1)

# Convenience: common scale presets
.PHONY: bench-smoke bench-pub bench-stress
bench-smoke: SCALE=0.1
bench-smoke: bench-all
bench-pub: SCALE=5.0
bench-pub: bench-all
bench-stress: SCALE=25.0
bench-stress: bench-all
```

### 5.3 Dockerfile.bench

```dockerfile
# Stage 1: Build GoPOSIX for benchmarking
FROM golang:1.26-alpine AS builder
RUN apk add --no-cache git
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /goposix ./cmd/goposix/

# Stage 2: Benchmark image (Alpine + GoPOSIX + BusyBox + tooling)
FROM alpine:3.20
RUN apk add --no-cache time strace bc coreutils procps
COPY --from=builder /goposix /bin/goposix
RUN /bin/goposix --list-commands > /commands.txt && \
    while IFS= read -r cmd; do ln -s /bin/goposix "/bin/${cmd}"; done < /commands.txt
COPY test/benchmark/ /bench/
RUN chmod +x /bench/*.sh /bench/lib/*.sh
WORKDIR /data
ENTRYPOINT ["/bench/runner.sh"]
```

### 5.4 Harness Library (`lib/harness.sh`)

Core functions:

```bash
# Scale factor: single knob controlling all workload sizes.
# Default 1.0 (standard). Set via environment: BENCH_SCALE=5.0 make bench-all
BENCH_SCALE="${BENCH_SCALE:-1.0}"

# Clamp a scaled integer parameter.
# Usage: FILE_COUNT=$(scaled 1000)   # → 1000 at SCALE=1, 5000 at SCALE=5
# Second arg is the cap (default: no cap beyond MAX_FILE_COUNT).
scaled() {
  local base="$1"
  local cap="${2:-500000}"
  local val
  val=$(awk "BEGIN { v = int($base * $BENCH_SCALE + 0.5); if (v < 1) v = 1; printf \"%d\", v }")
  if [ "$val" -gt "$cap" ]; then
    echo "WARNING: scaled value $val (base=$base × scale=$BENCH_SCALE) capped at $cap" >&2
    val=$cap
  fi
  echo "$val"
}

# Run a command N times, collect timing metrics
# Usage: bench_run "label" 10 "command string"
bench_run() {
  local label="$1" samples="$2" cmd="$3"
  # Warmup: 3 discarded runs
  for i in 1 2 3; do eval "$cmd" >/dev/null 2>&1; done
  # Measured runs
  for i in $(seq "$samples"); do
    /usr/bin/time -f "%e %U %S %M" -o /tmp/bench_tmp \
      sh -c "$cmd" >/dev/null 2>&1
    read -r wall user sys rss < /tmp/bench_tmp
    echo "$label,$i,$wall,$user,$sys,$rss"
    sleep 1
  done
}

# Compute median from CSV column
bench_median() {
  sort -n | awk '{arr[NR]=$1} END {if (NR%2==1) print arr[(NR+1)/2]; else print (arr[NR/2]+arr[NR/2+1])/2}'
}
```

---

## 6. Expected Results Matrix (Predictions)

This matrix predicts outcomes before implementation. It serves as a design
check: if we see wildly different results, either our expectations or our
methodology is wrong.

| Category | GoPOSIX Wins? | BusyBox Wins? | Predicted Margin | Confidence |
|----------|---------------|---------------|------------------|------------|
| A — Startup | ❌ | ✅ | 10–50× BusyBox | High |
| B — Bulk Create | ≈ | ≈ | ±10% | High |
| C — Bulk LS | ❌ (single) | ✅ (single) | 1.2–2× BusyBox | Medium |
| D — Bulk Move/RM | ≈ | ≈ | ±5% | High |
| E — Text I/O | ≈ | ≈ | ±10% | Medium |
| F — Daemon vs Fork | ✅ | ❌ | 5–100× GoPOSIX at N=100+ | High |
| G — Memory | ❌ | ✅ | 5–20× BusyBox per invocation | High |
| H — Size | ❌ | ✅ | 12:1 binary, 2:1 image | Certain |
| I — Concurrent | ✅ (potential) | ❌ | 2–8× GoPOSIX | Low (speculative) |
| J — Agent Loop | ✅ | ❌ | 10–50× GoPOSIX | Medium-High |

### The Narrative

> **BusyBox wins on resource economy (size, memory, single-shot latency).**
> **GoPOSIX wins on sustained throughput for repeated operations (daemon mode, agent loops).**
>
> For a one-off `ls`, use BusyBox. For an AI agent making 10,000 `ls` calls per session,
> use GoPOSIX. The break-even on total cost is around **10–50 sequential operations**
> for latency and **30–100 operations** for total CPU, depending on the utility.

---

## 7. CI Integration (Optional — Phase 2)

Once the benchmark harness is stable, we can add a lightweight CI gate:

```yaml
# .github/workflows/bench.yml
perf-bench:
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v4
    - run: make bench-quick
    - uses: benchmark-action/github-action-benchmark@v1
      with:
        tool: 'custom'
        output-file-path: test/benchmark/results/latest/raw.csv
        github-token: ${{ secrets.GITHUB_TOKEN }}
        auto-push: true
        alert-threshold: '200%'
```

This tracks performance over time and alerts on regressions.

> **Deferred:** CI benchmarking is sensitive to GitHub Actions runner noise.
> We recommend running benchmarks locally on a dedicated, quiesced machine for
> publication-quality results. CI is for regression detection only.

---

## 8. Implementation Order

| Step | What | Effort | Priority |
|------|------|--------|----------|
| 1 | `Dockerfile.bench` + build infrastructure | 2h | P0 |
| 2 | `lib/harness.sh` + `lib/report.sh` | 3h | P0 |
| 3 | Cat H — Size (trivial, no runtime) | 0.5h | P0 |
| 4 | Cat A — Startup latency | 1h | P0 |
| 5 | Cat F — Daemon vs Process (killer feature) | 2h | P0 |
| 6 | Cat B, C, D — Bulk filesystem ops | 2h | P1 |
| 7 | Cat E — Text throughput | 1.5h | P1 |
| 8 | Cat G — Memory footprint | 1h | P1 |
| 9 | Cat J — Agent loop simulation | 2h | P1 |
| 10 | Cat I — Concurrent (aspirational) | 2h | P2 |
| 11 | CI integration | 1h | P2 |
| 12 | Write narrative report from results | 2h | P1 |

**Total:** ~20h for complete implementation.

---

## 9. How to Read the Results

After running `make bench-all`, the output is:

```
test/benchmark/results/2026-05-18T120000/
├── summary.md      # Human-readable summary with formatted tables
├── raw.csv         # Machine-readable: label,sample,wall,user,sys,rss
├── cat_a.log       # Raw timing output
├── cat_b.log
├── ...
└── narrative.md    # Polished prose for README / blog post
```

### Example `summary.md` snippet:

```markdown
## Cat A — Single-Invocation Latency (ms, median of 10)

| Test | GoPOSIX | BusyBox | Ratio | Winner |
|------|---------|---------|-------|--------|
| true | 12.3 | 0.4 | 30.8× | BusyBox |
| echo hello | 13.1 | 0.5 | 26.2× | BusyBox |
| pwd | 13.8 | 0.6 | 23.0× | BusyBox |

## Cat F — Daemon vs Process (ms for N sequential calls)

| N | GoPOSIX Daemon | BusyBox Fork | Ratio | Winner |
|---|----------------|--------------|-------|--------|
| 10 | 8.2 | 5.1 | 1.6× | BusyBox |
| 50 | 12.4 | 25.0 | 0.5× | GoPOSIX |
| 100 | 17.1 | 50.3 | 0.34× | GoPOSIX |
| 1000 | 89.2 | 510.1 | 0.17× | GoPOSIX |
```

---

## 10. Acceptance Criteria

- [ ] `make bench-image` builds the benchmark Docker image successfully
- [ ] `make bench-quick` runs Cat A + Cat H and produces a `summary.md`
- [ ] `make bench-all` runs all categories and produces a complete results directory
- [ ] All benchmark scripts exit 0 with valid CSV output
- [ ] Results are reproducible within ±15% on the same hardware
- [ ] A `narrative.md` is generated that tells the honest GoPOSIX vs BusyBox story
- [ ] The narrative includes at least one chart (ASCII or generated) showing the
      daemon amortization curve (Cat F)

---

## 11. Known Limitations & Caveats

1. **Docker overhead:** Both binaries run in identical containers, but Docker's
   `--privileged` flag is needed for accurate CPU timing. Without it, cgroup
   accounting adds noise.

2. **Host kernel:** Both binaries share the host kernel. Filesystem cache
   effects from one container can affect the other. Warmup runs mitigate this.

3. **Go GC pauses:** GoPOSIX's GC may introduce latency spikes. Our p95 measurement
   captures this; BusyBox's malloc/free has no comparable stop-the-world phase.

4. **Goroutine parallelization not yet implemented:** Cat I (concurrent ops) is
   aspirational. Current GoPOSIX utilities are mostly sequential, matching BusyBox's
   architecture. This category measures *potential*, not current reality.

5. **One BusyBox version:** We compare against BusyBox v1.36.1 (Alpine 3.20).
   BusyBox v1.37+ may have different performance characteristics.

---

## 12. References

- [GoPOSIX Architecture](../docs/ARCHITECTURE.md) — component layout, binary size targets
- [Phase 13 — Coverage & Hardening](13_coverage_and_hardening.md) — speed targets
  ("<1ms daemon latency, <15MB binary, <100ms CLI startup")
- [Phase 12 — Road to Gold](12_road_to_gold.md) — current project state
- [test/benchmark/bench_daemon_test.go](../test/benchmark/bench_daemon_test.go) — existing Go-level daemon benchmarks
- [docker/Dockerfile.debug](../docker/Dockerfile.debug) — debug image pattern to extend
- [Makefile](../Makefile) — `make bench` target (Go micro-benchmarks only)
