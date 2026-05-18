# Phase 22 — Hardening III (Daemon-First Pivot)

> **Status:** PLANNING | **Date:** 2026-05-18 | **Trigger:** Benchmark data (Phase 19) + Go SDK discovery
>
> **Key finding:** The Go SDK client with persistent connection achieves **60µs per RPC call — 11× faster than BusyBox fork+exec (680µs)**. The old socat-per-call approach was 3× *slower* than BusyBox. The project has been selling the right architecture with the wrong interface.

---

## 1. What Changed

Phase 19 benchmarking revealed:

| Interface | Per-call latency | vs BusyBox |
|-----------|:---:|:---:|
| `socat` per call (old benchmark) | 2,000µs | **3× slower** |
| BusyBox `fork+exec` | 680µs | baseline |
| **Go SDK persistent conn** | **60µs** | **11× faster** |

The daemon architecture was always correct. But the default Docker image (`FROM scratch`, `ENTRYPOINT ["/bin/goposix"]`) makes every invocation a cold start (7ms Go runtime init), and the only documented way to talk to the daemon was `socat` (worst-case overhead). The **Go SDK already exists** in `pkg/client/` with typed methods for every utility — it's just never been the documented default path.

### Other benchmark wins

| Utility | GoPOSIX | BusyBox | Ratio |
|---------|:------:|:------:|:-----:|
| `grep` on 100MB | 0.16s | 0.86s | **5.4× faster** (RE2 vs POSIX ERE) |

(BusyBox still wins on binary size 11.2×, startup 1.8×, memory 9.5×, and `grep -r` 22× — these are architectural and fine.)

---

## 2. The Pivot

**From:** GoPOSIX is a multicall CLI binary that also has a daemon.
**To:** GoPOSIX is a persistent daemon you talk to via the Go SDK. CLI access is secondary.

### Old Story (wrong)

```
docker run --rm goposix ls -la /     # 7ms cold start, every time
goposix daemon --socket /tmp/s.sock & # manual daemon start
echo '{"jsonrpc":...}' | socat ...    # 2ms per call (socat overhead)
```

### New Story (correct)

```
docker run -d --name goposix goposix:latest        # daemon starts automatically
```

```go
c, _ := client.New("/var/run/goposix.sock")
for i := 0; i < 10000; i++ {
    c.Echo(ctx, "hello")  // 60µs per call
}
```

---

## 3. Milestones

### M1 — Daemon Image Becomes Default ✅ (target: immediate)

| Action | Detail |
|--------|--------|
| Rename current `Dockerfile` → `Dockerfile.cli` | The `FROM scratch` CLI-only image |
| Rename current `Dockerfile.daemon` → `Dockerfile` | Daemon becomes the default build |
| Tag strategy | `goposix:latest` = daemon, `goposix:cli` = CLI |
| `Makefile` targets | `make image` → daemon, `make image-cli` → CLI |
| Update `.goreleaser.yml` | Multi-platform daemon image as primary release artifact |

**Old image tags (deprecated but kept):**
- `goposix:latest` (currently CLI, becomes daemon)
- `goposix:debug` (Alpine + shell, unchanged)
- `goposix:scratch` → renamed to `goposix:cli`

**New image tags:**
- `goposix:latest` = daemon (Alpine base, `ENTRYPOINT ["/bin/goposix", "daemon", ...]`)
- `goposix:cli` = scratch CLI (old `Dockerfile`, `ENTRYPOINT ["/bin/goposix"]`)
- `goposix:debug` = same as before (Alpine + shell, interactive)

### M2 — Go SDK Quickstart in README (target: immediate)

Replace the current `--json` CLI example with an SDK example:

```go
package main

import (
    "context"
    "fmt"
    "github.com/ramayac/goposix/pkg/client"
)

func main() {
    c, _ := client.New("/var/run/goposix.sock")
    defer c.Close()

    // List files as structured data.
    result, _ := c.Ls(context.Background(), "/", nil)
    for _, entry := range result.Entries {
        fmt.Printf("%s %7d %s\n", entry.Mode, entry.Size, entry.Name)
    }
}
```

**Files to update:**
- `README.md` — swap CLI quickstart for SDK quickstart, move CLI to "CLI Usage" section
- `AGENTS.md` — update project identity paragraph
- `CLAUDE.md` — same
- `CONTRIBUTING.md` — mention daemon-first architecture
- `docs/ARCHITECTURE.md` — update ASCII diagram, API-first messaging
- `docs/RPC_QUICKSTART.md` — add Go SDK examples

### M3 — SDK Documentation Page (target: this week)

Create `docs/SDK.md` covering:

- Installation: `go get github.com/ramayac/goposix/pkg/client`
- Connection: `client.New(socketPath, options...)`
- Typed methods for all 77 utilities (reference, not full listing)
- Connection pooling (`WithPoolSize(n)`)
- Error handling, retries, timeouts
- Performance expectations: ~60µs/call for simple commands
- Comparison: CLI invocation (7ms) vs socat (2ms) vs SDK (60µs)

### M4 — Benchmark Integration (target: this week)

- Add SDK benchmark results to `README.md` (key numbers)
- Update `wiki/19_performance_benchmarking.md` with actual measured data (replace the predicted matrix)
- Add Cat F SDK mode to CI smoke test (`make bench-quick`)

### M5 — goposix:cli Entrypoint Smart Routing (target: later)

Make the CLI image detect a running daemon and forward commands through it:

```go
// In cmd/goposix/main.go:
func Main() int {
    // If daemon socket exists, forward command and exit.
    if socketExists("/var/run/goposix.sock") {
        return forwardToDaemon("/var/run/goposix.sock", os.Args)
    }
    // Fall back to direct execution (current behavior).
    return dispatchAndRun(os.Args)
}
```

This gives the CLI image daemon benefits without changing its entrypoint — if you're `docker exec`-ing into a daemon container, commands auto-forward. Deferred because it needs the Go SDK linked into the multicall binary (minor import, but increases binary size slightly).

---

## 4. Documentation Audit

### Files to Update

| File | Current | Change |
|------|---------|--------|
| `README.md` | CLI-first quickstart | SDK-first quickstart, CLI as secondary |
| `README.md` | "Zero Dependencies" | "Near-Zero Dependencies" (already done) |
| `README.md` | No benchmark data | Add key numbers: 60µs/call, 11× faster than BusyBox, grep 5.4× faster |
| `AGENTS.md` §1 | "multicall binary" | "persistent daemon with Go SDK + multicall CLI fallback" |
| `AGENTS.md` §5 | Root protection etc | Add: "Daemon-first: the default image starts the daemon. CLI is secondary." |
| `CLAUDE.md` | Same as AGENTS.md | Sync |
| `CONTRIBUTING.md` | 8-step checklist | Add SDK coverage expectations |
| `docs/ARCHITECTURE.md` | "JSON-RPC 2.0 daemon for programmatic consumers" | Already good after Phase 21, add SDK benchmark numbers |
| `docs/RPC_QUICKSTART.md` | socat examples | Add Go SDK examples alongside socat |
| `wiki/19_performance_benchmarking.md` | Predicted matrix | Replace with actual measured matrix from benchmark runs |
| `wiki/phases.md` | Phase history | Add Phase 22 entry |
| `wiki/index.md` | Phase docs list | Add Phase 22 |

### New Files to Create

| File | Content |
|------|---------|
| `docs/SDK.md` | Go SDK usage guide, typed method reference, performance expectations |
| `wiki/22_hardening_iii.md` | This document |

### Files to Rename

| Old | New | Reason |
|-----|-----|--------|
| `docker/Dockerfile` | `docker/Dockerfile.cli` | CLI is no longer the default |
| `docker/Dockerfile.daemon` | `docker/Dockerfile` | Daemon becomes the default build |

### Files NOT Changed

- `docker/Dockerfile.debug` — unchanged (interactive debugging)
- `docker/Dockerfile.goreleaser` — updated for daemon default (separate task)
- `pkg/client/` — unchanged (already works, just under-documented)
- `internal/daemon/` — unchanged (already works, rate limiter already raised)
- `test/benchmark/` — already updated with SDK benchmark client

---

## 5. Acceptance Criteria

- [ ] M1: `make image` builds daemon image, `make image-cli` builds CLI-only image
- [ ] M1: `docker run -d --name goposix goposix:latest` starts daemon and responds to SDK calls
- [ ] M1: `docker run --rm goposix:cli ls -la /` works as before (backward compatible)
- [ ] M2: README quickstart shows Go SDK example, not CLI `--json` flag
- [ ] M2: AGENTS.md project identity says "persistent daemon with Go SDK" first
- [ ] M3: `docs/SDK.md` exists with connection examples, typed methods, performance expectations
- [ ] M4: README includes benchmark numbers (60µs/call, 11×, grep 5.4×)
- [ ] M4: `wiki/19_performance_benchmarking.md` has actual measured matrix, not predictions
- [ ] `make test` passes (zero regressions)
- [ ] `make testsuite` passes (548/4/10, no new failures)
- [ ] `make bench-quick` passes with SDK mode
- [ ] `make bench-all` completes all categories including 3-mode Cat F

---

## 6. Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Daemon image breaks existing users | Low | High | Keep `goposix:cli` tag; document migration path |
| `FROM scratch` purists object | Medium | Low | CLI image still available; daemon is Alpine (7MB base) |
| SDK hasn't been load-tested beyond 1,000 calls | Medium | Medium | Rate limiter raised; add load test before SCALE=25 benchmarks |
| Binary size increases with SDK linked into multicall | Low | Low | Only for M5 (deferred); daemon image doesn't link SDK into binary |
| ARM64 daemon image build breaks | Low | Medium | Multi-arch build already tested in CI |
| Go SDK breaking changes | Low | High | SDK is already used internally; add integration test for SDK methods |

---

## 7. References

- [Phase 19 — Performance Benchmarking](19_performance_benchmarking.md) — benchmark framework and raw data
- [Phase 20 — Hardening II](20_hardening_ii.md) — `-j` flag audit, coverage, input safety (completed)
- [Phase 21 — Honest Takes](21_honest_takes.md) — de-agentified language audit (completed, page removed)
- [Dockerfile.daemon](../docker/Dockerfile.daemon) — daemon image (already exists)
- [bench_client](../test/benchmark/bench_client/main.go) — Go SDK benchmark client (already exists)
- [Go SDK client](../pkg/client/client.go) — production SDK with typed methods for all 77 utilities
- [Daemon server](../internal/daemon/server.go) — JSON-RPC 2.0 daemon (rate limiter: 100K req/s)
- [Benchmark results](../test/benchmark/results/) — CSV data and generated reports
