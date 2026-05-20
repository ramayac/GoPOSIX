# Go SDK Guide

> Moved from `docs/SDK.md`.

The `pkg/client` package is the **primary interface** to GoPOSIX. It provides a typed,
connection-pooled Go client for the JSON-RPC 2.0 daemon with ~60µs per-call latency —
11× faster than BusyBox fork+exec.

## Installation

```bash
go get github.com/ramayac/goposix/pkg/client
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "github.com/ramayac/goposix/pkg/client"
)

func main() {
    // Connect to the daemon.
    c, _ := client.New("/var/run/goposix.sock")
    defer c.Close()

    // Typed APIs — no JSON marshaling, no string scraping.
    result, _ := c.Ls(context.Background(), "/", nil)
    for _, entry := range result.Entries {
        fmt.Printf("%s %7d %s\n", entry.Mode, entry.Size, entry.Name)
    }
}
```

## Connection

```go
c, err := client.New(socketPath, options...)
```

| Option | Default | Description |
|--------|---------|-------------|
| `client.WithPoolSize(n)` | 4 | Max concurrent connections to the daemon |
| `client.WithTimeout(d)` | 30s | Per-call deadline |
| `client.WithMaxRetries(n)` | 2 | Retry count on transient errors (connection refused, broken pipe) |

### Connection Lifecycle

- `client.New()` returns immediately — it does not dial until the first call
- Each call acquires a connection from the pool (blocking if pool is full)
- Connections are reused across calls; idle connections are not pruned
- `c.Close()` closes all idle connections

## Core API

### Single Calls

```go
// Call executes a JSON-RPC method. result is unmarshaled from the response.
func (c *Client) Call(ctx context.Context, method string, params interface{}, result interface{}) error
```

```go
// CallRaw returns the raw JSON result for custom unmarshaling.
func (c *Client) CallRaw(ctx context.Context, method string, params interface{}) (json.RawMessage, error)
```

### Notifications

```go
// Notify sends a fire-and-forget request (no response).
func (c *Client) Notify(ctx context.Context, method string, params interface{}) error
```

### Batch Requests

```go
// Batch sends multiple requests in one round-trip. No retry on batch.
func (c *Client) Batch(ctx context.Context, reqs []BatchRequest) ([]BatchResponse, error)

type BatchRequest struct {
    Method string
    Params interface{}
}

type BatchResponse struct {
    Result json.RawMessage
    Error  *rpcError
}
```

## Typed Utility Methods

All 77 POSIX utilities have typed helper methods on `*client.Client`. Each method
handles parameter marshaling and returns the utility's specific result type.

### Filesystem

| Method | Signature |
|--------|-----------|
| `Ls` | `(ctx, path string, flags []string) (*LsResult, error)` |
| `Cat` | `(ctx, path string) (*CatResult, error)` |
| `Stat` | `(ctx, path string) (*StatResult, error)` |
| `Readlink` | `(ctx, path string) (*ReadlinkResult, error)` |
| `Find` | `(ctx, basePath string, flags []string) ([]FindEntry, error)` |
| `Du` | `(ctx, path string) ([]DirInfo, error)` |
| `Df` | `(ctx, path string) ([]FSInfo, error)` |

### File Operations

| Method | Signature |
|--------|-----------|
| `Mkdir` | `(ctx, path string, parents bool) (*MkdirResult, error)` |
| `Rmdir` | `(ctx, path string) (*RmdirResult, error)` |
| `Rm` | `(ctx, paths []string, recursive, force bool) (*RmResult, error)` |
| `Cp` | `(ctx, from, to string) (*CpResult, error)` |
| `Mv` | `(ctx, from, to string) (*MvResult, error)` |
| `Ln` | `(ctx, target, link string, symbolic bool) (*LnResult, error)` |
| `Touch` | `(ctx, paths []string) (*TouchResult, error)` |
| `Chmod` | `(ctx, mode string, paths []string) (*ChmodResult, error)` |
| `Chown` | `(ctx, owner string, paths []string) (*ChownResult, error)` |
| `Chgrp` | `(ctx, group string, paths []string) (*ChownResult, error)` |
| `Basename` | `(ctx, path string) (*BasenameResult, error)` |
| `Dirname` | `(ctx, path string) (*DirnameResult, error)` |

### Text Processing

| Method | Signature |
|--------|-----------|
| `Echo` | `(ctx, text string) (*EchoResult, error)` |
| `Printf` | `(ctx, format string, args ...string) (*PrintfResult, error)` |
| `Head` | `(ctx, path string, n int) (*HeadResult, error)` |
| `Tail` | `(ctx, path string, n int) (*TailResult, error)` |
| `Wc` | `(ctx, path string) (*WcResult, error)` |
| `Sort` | `(ctx, flags []string) (*SortResult, error)` |
| `Uniq` | `(ctx, flags []string) ([]UniqItem, error)` |
| `Cut` | `(ctx, flags []string) (*CutResult, error)` |
| `Grep` | `(ctx, pattern string, flags []string) ([]GrepMatch, error)` |
| `Diff` | `(ctx, file1, file2 string) (*DiffResult, error)` |

### Archival

| Method | Signature |
|--------|-----------|
| `Tar` | `(ctx, flags []string) ([]TarFileStat, error)` |
| `Gzip` | `(ctx, flags []string) ([]GzipStat, error)` |
| `Md5sum` | `(ctx, paths []string, check bool) (json.RawMessage, error)` |
| `Sha256sum` | `(ctx, paths []string, check bool) (json.RawMessage, error)` |

### System

| Method | Signature |
|--------|-----------|
| `Ps` | `(ctx) ([]ProcessInfo, error)` |
| `Kill` | `(ctx, signal string, pids []int) (*KillResult, error)` |
| `Date` | `(ctx) (*DateInfo, error)` |
| `ID` | `(ctx) (*IDInfo, error)` |
| `Whoami` | `(ctx) (*WhoamiResult, error)` |
| `Hostname` | `(ctx) (*HostnameResult, error)` |
| `Uname` | `(ctx) (*UnameResult, error)` |
| `Pwd` | `(ctx) (*PwdResult, error)` |
| `Env` | `(ctx, flags []string, vars map[string]string) (*EnvVarsResult, error)` |
| `Printenv` | `(ctx, name string) (*EnvVarsResult, error)` |
| `Test` | `(ctx, flags []string) (*TestResult, error)` |
| `Expr` | `(ctx, expression []string) (*ExprResult, error)` |
| `Xargs` | `(ctx, command string, flags []string) ([]ExecEntry, error)` |

### Session & Shell

| Method | Signature |
|--------|-----------|
| `SessionCreate` | `(ctx) (*SessionInfo, error)` |
| `SessionSetCwd` | `(ctx, sessionID, path string) error` |
| `SessionList` | `(ctx) ([]SessionInfo, error)` |
| `SessionDestroy` | `(ctx, sessionID string) error` |
| `ShellExec` | `(ctx, sessionID, script string) (*ExecResult, error)` |

### Health

| Method | Signature |
|--------|-----------|
| `Ping` | `(ctx) (*PingResult, error)` |

## Error Handling

```go
result, err := c.Ls(ctx, "/nonexistent", nil)
if err != nil {
    var rpcErr *rpcError
    if errors.As(err, &rpcErr) {
        fmt.Printf("RPC error %d: %s\n", rpcErr.Code, rpcErr.Message)
    }
    return
}
```

| Code | Meaning |
|------|---------|
| -32700 | Parse error or request too large |
| -32600 | Invalid Request |
| -32601 | Method not found |
| -32602 | Invalid params (includes path traversal) |
| -32000 | Server error / rate limited |

Transient errors (connection refused, broken pipe, EOF, daemon shutdown) are
automatically retried with exponential backoff up to the configured `maxRetries`.

## Performance

| Pattern | Latency | Notes |
|---------|:------:|-------|
| `c.Echo(ctx, "hi")` | **60µs** | Persistent connection, connection pool |
| `socat` pipe to socket | ~2,000µs | Per-call process overhead |
| CLI `goposix echo hi` | ~7,000µs | Go runtime cold start |
| BusyBox `echo hi` | ~680µs | C fork+exec (baseline) |

For bulk operations, batch multiple calls:

```go
reqs := []client.BatchRequest{
    {Method: "goposix.echo", Params: map[string]string{"text": "a"}},
    {Method: "goposix.echo", Params: map[string]string{"text": "b"}},
}
results, _ := c.Batch(ctx, reqs)
```

## Context Propagation

All methods accept `context.Context` for cancellation and deadlines:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

result, err := c.Ls(ctx, "/large_directory", nil)
```

## Advanced: Raw Calls

For utilities without typed wrappers, or when you need full control:

```go
raw, err := c.CallRaw(ctx, "goposix.tee", map[string]interface{}{
    "sessionId": sid,
    "flags":     []string{"--json"},
})
```

## Comparison: CLI vs SDK

| | CLI (`goposix ls`) | Go SDK (`c.Ls()`) |
|---|---|---|
| **Latency** | 7ms (cold start) | 60µs (persistent) |
| **Output** | Text (parse with `--json`) | Typed Go structs |
| **Connection** | One-shot | Pooled, persistent |
| **Error handling** | Exit codes | Go error values + RPC codes |
| **Concurrency** | None | Goroutine-safe connection pool |
| **Batch** | Shell loops | Native batch RPC |
| **Use case** | One-off admin commands | High-throughput programmatic loops |
