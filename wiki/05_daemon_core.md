# Phase 05 — JSON-RPC Daemon Core

> **HISTORICAL — COMPLETED.** This phase is done. Document retained for reference.
>
> **Timeline:** Week 6–7 | **Depends on:** Phase 03, 04

---

## Goal

Build the persistent JSON-RPC 2.0 daemon that listens on a Unix socket, routes methods to `pkg/` libraries, and handles concurrency. Source: [pkg/daemon/](../pkg/daemon/), [pkg/client/](../pkg/client/)

## Tasks

### 05.1 — Daemon Entry Point

`korego daemon --socket /var/run/korego.sock --workers 4 --log-level info`

- [x] PID file at `/var/run/korego.pid`
- [x] Signal handling: `SIGTERM`/`SIGINT` → graceful shutdown (drain in-flight, close socket, remove PID)
- [x] `SIGHUP` → reload config (future)

### 05.2 — Unix Socket Listener (`internal/daemon/server.go`)

- [x] `net.Listen("unix", socketPath)`
- [x] Remove stale socket file on start (`os.Remove` if exists)
- [x] Set socket permissions to `0660`
- [x] Accept loop with error handling

### 05.3 — JSON-RPC Router

Method naming: `korego.<utility>` (e.g., `korego.ls`, `korego.echo`)

```json
// Request
{"jsonrpc":"2.0","method":"korego.ls","params":{"path":"/tmp","flags":["-la"]},"id":1}

// Success Response
{"jsonrpc":"2.0","result":{"files":[...]},"id":1}

// Error Response
{"jsonrpc":"2.0","error":{"code":1002,"message":"not found","data":{"path":"/nope"}},"id":1}
```

- [x] Router maps method string → handler function
- [x] Auto-register all utilities from dispatcher registry
- [x] Unknown method → error code `-32601`
- [x] Invalid params → error code `-32602`

### 05.4 — Worker Pool

```go
type WorkerPool struct {
    sem chan struct{} // buffered channel as semaphore
}

func NewWorkerPool(size int) *WorkerPool {
    return &WorkerPool{sem: make(chan struct{}, size)}
}

func (wp *WorkerPool) Submit(ctx context.Context, fn func()) error {
    select {
    case wp.sem <- struct{}{}:
        go func() {
            defer func() { <-wp.sem }()
            fn()
        }()
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}
```

- [x] Bounded goroutine pool (default: `runtime.NumCPU()`)
- [x] Context-based timeout for queued requests
- [x] Metrics: active workers, queued requests, completed, errors

### 05.5 — Batch Processing

- [x] Detect JSON array `[{...},{...}]` → batch mode
- [x] Process each request concurrently via worker pool
- [x] Return results array in same order as requests
- [x] Single request failure doesn't abort batch

### 05.6 — Health Check

`korego.ping` → `{"pong":true, "uptime":"2h30m", "version":"0.3.0", "workers":{"active":2,"total":4}}`

### 05.7 — Client Library (`pkg/client/`)

```go
client, _ := koregoclient.Dial("/var/run/korego.sock")
result, _ := client.Call("korego.ls", map[string]interface{}{"path": "/tmp"})
```

- [x] Go client for tests and future integrations
- [x] Connection pooling
- [x] Timeout support

## Milestone 05

- [x] `korego daemon` starts and listens on Unix socket
- [x] `korego.ls` over socket returns file list without binary restart
- [x] Batch of 10 commands returns 10 results
- [x] 100 concurrent requests complete without errors
- [x] `korego.ping` returns health status
- [x] Graceful shutdown completes within 5s

## How to Verify

```bash
# Start daemon
./korego daemon --socket /tmp/korego.sock &

# RPC via socat
echo '{"jsonrpc":"2.0","method":"korego.ls","params":{"path":"/"},"id":1}' \
  | socat - UNIX-CONNECT:/tmp/korego.sock

# Batch
echo '[{"jsonrpc":"2.0","method":"korego.echo","params":{"text":"a"},"id":1},
       {"jsonrpc":"2.0","method":"korego.echo","params":{"text":"b"},"id":2}]' \
  | socat - UNIX-CONNECT:/tmp/korego.sock

# Health
echo '{"jsonrpc":"2.0","method":"korego.ping","id":99}' \
  | socat - UNIX-CONNECT:/tmp/korego.sock

# Load test
go test -run TestDaemonConcurrent -timeout 60s ./test/integration/
```
