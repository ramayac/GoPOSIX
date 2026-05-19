# JSON-RPC Protocol Reference

GoPOSIX uses JSON-RPC 2.0 over a Unix socket as its wire protocol. The typed
[Go SDK](../docs/SDK.md) is the recommended client for Go users (60µs/call, connection
pooling, retry). This document covers the **raw protocol** for non-Go clients,
debugging, and understanding what the SDK does under the hood.

## Overview

The daemon exposes a Unix socket-based JSON-RPC 2.0 API that lets any program:

1. Start a GoPOSIX daemon (lightweight, ~8 MB container image)
2. Create isolated sessions with per-session working directories
3. Execute POSIX utilities and shell scripts
4. Receive structured JSON output (no scraping stdout)
5. Clean up when done

## Quick Start

### Go (recommended)

```go
c, _ := client.New("/tmp/goposix.sock")
defer c.Close()
result, _ := c.Ls(ctx, "/etc", nil)  // typed, 60µs
```

See **[docs/SDK.md](../docs/SDK.md)** for the full Go SDK guide.

### Any Language (raw protocol)

```bash
make example-rpc
```

This runs `examples/rpc_client/main.go` — a self-contained Go program that demonstrates
the full lifecycle using raw JSON-RPC. For non-Go languages, read on.

## Architecture

```
┌─────────────────────┐    Unix socket     ┌─────────────────┐
│  Programmatic       │ ◄──── JSON-RPC ───► │  GoPOSIX Daemon  │
│  Consumer           │                    │  (goposix daemon) │
│  (your code)        │                    └────────┬────────┘
└─────────────────────┘                             │
                                           ┌────────▼────────┐
                                           │  POSIX Utilities │
                                           │  ls, cat, grep,  │
                                           │  find, wc, ...   │
                                           └─────────────────┘
```

## Lifecycle

### 1. Start the Daemon

```bash
goposix daemon -s /tmp/goposix.sock -w 4
```

Flags:
- `-s` — Unix socket path
- `-w` — Worker pool size (default 4)
- `-l` — HTTP observability address (optional, e.g. `:9090`)

### 2. Connect

```go
conn, _ := net.DialTimeout("unix", "/tmp/goposix.sock", 5*time.Second)
```

All communication is newline-delimited JSON over a Unix socket. Each request gets one response.

### 3. Ping (Health Check)

```json
→ {"jsonrpc":"2.0", "method":"goposix.ping", "id":1}
← {"jsonrpc":"2.0", "result":{"pong":true, "uptime":"5s", "version":"0.1.0"}, "id":1}
```

### 4. Create a Session

Sessions isolate state — each has its own CWD and environment variables.

```json
→ {"jsonrpc":"2.0", "method":"goposix.session.create", "id":2}
← {"jsonrpc":"2.0", "result":{"sessionId":"a1b2c3d4", "cwd":"/", "env":{}, "lastActive":"..."}, "id":2}
```

### 5. Set Working Directory

```json
→ {"jsonrpc":"2.0", "method":"goposix.session.setCwd", "params":{"sessionId":"a1b2c3d4", "path":"/etc"}, "id":3}
← {"jsonrpc":"2.0", "result":true, "id":3}
```

Paths are validated to prevent traversal outside the session CWD. The default CWD (`/`) allows all absolute paths.

### 6. Execute Utilities

All JSON-enabled utilities are available as `goposix.<name>` methods:

```json
→ {"jsonrpc":"2.0", "method":"goposix.ls", "params":{"sessionId":"a1b2c3d4"}, "id":4}
← {"jsonrpc":"2.0", "result":{"exitCode":0, "data":{"path":"/etc", "files":[...], "total":16}}, "id":4}
```

```json
→ {"jsonrpc":"2.0", "method":"goposix.wc", "params":{"sessionId":"a1b2c3d4", "path":"hosts"}, "id":5}
← {"jsonrpc":"2.0", "result":{"exitCode":0, "data":{"lines":11, "words":46, "bytes":225, "chars":225}}, "id":5}
```

Standard params: `sessionId` (string), `path` (string), `flags` ([]string), `text` (string, for echo).

### 7. Execute Shell Scripts

```json
→ {"jsonrpc":"2.0", "method":"goposix.shell.exec", "params":{"sessionId":"a1b2c3d4", "script":"echo hello from goposix"}, "id":6}
← {"jsonrpc":"2.0", "result":{"stdout":"hello from goposix\n", "stderr":"", "exitCode":0}, "id":6}
```

The shell interpreter runs with a configurable timeout (default 30s, via `GOPOSIX_SHELL_TIMEOUT`) and 128 MB memory limit per stream.

### 8. Destroy the Session

```json
→ {"jsonrpc":"2.0", "method":"goposix.session.destroy", "params":{"sessionId":"a1b2c3d4"}, "id":7}
← {"jsonrpc":"2.0", "result":true, "id":7}
```

Sessions auto-expire after 30 minutes of inactivity.

### 9. Stop the Daemon

Send SIGTERM to the daemon process. It drains connections and removes the socket file.

## Batch Requests

Send an array of requests to execute multiple calls in one round-trip:

```json
→ [
    {"jsonrpc":"2.0", "method":"goposix.echo", "params":{"text":"a"}, "id":1},
    {"jsonrpc":"2.0", "method":"goposix.echo", "params":{"text":"b"}, "id":2}
  ]
← [
    {"jsonrpc":"2.0", "result":{"exitCode":0, "data":{"text":"a"}}, "id":1},
    {"jsonrpc":"2.0", "result":{"exitCode":0, "data":{"text":"b"}}, "id":2}
  ]
```

## Notifications

Requests without an `id` field are treated as JSON-RPC notifications — no response is sent. Useful for fire-and-forget operations.

## Error Handling

| Code | Meaning |
|------|---------|
| -32700 | Parse error or request too large |
| -32600 | Invalid Request |
| -32601 | Method not found |
| -32602 | Invalid params (includes path traversal) |
| -32000 | Server error / rate limited |

The response envelope includes `exitCode` for utility errors (non-zero = failure) and `data` is `null` on error.

## Example: Multi-Step RPC Task

The full example at `examples/rpc_client/main.go` demonstrates:

1. Start daemon
2. Ping
3. Create session
4. Set CWD to `/etc`
5. List files with `goposix.ls`
6. Count lines with `goposix.wc`
7. Run shell command with `goposix.shell.exec`
8. Read file contents with `goposix.cat`
9. Destroy session
10. Stop daemon

```bash
go run ./examples/rpc_client/main.go
```

## Go Client SDK

For production use, import `pkg/client/` for connection pooling, retry, and typed helper methods. See `docs/SDK.md` for the full guide.

### Quick SDK Example

```go
package main

import (
    "context"
    "fmt"
    "github.com/ramayac/goposix/pkg/client"
)

func main() {
    c, _ := client.New("/tmp/goposix.sock",
        client.WithPoolSize(4),
        client.WithTimeout(30*time.Second),
    )
    defer c.Close()

    // Typed call — returns structured data.
    result, _ := c.Ls(context.Background(), "/etc", nil)
    for _, f := range result.Entries {
        fmt.Println(f.Name)
    }

    // All 77 utilities have typed methods:
    // c.Cat(), c.Grep(), c.Sort(), c.Find(), c.Tar(), ...
}
```

| Feature | raw socket | Go SDK |
|---------|:---:|:---:|
| Typed return values | ❌ | ✅ |
| Connection pooling | ❌ | ✅ |
| Retry with backoff | ❌ | ✅ |
| Context propagation | ❌ | ✅ |
| Per-call latency | ~2ms (socat) | ~60µs |

See also [rpc_api.md](rpc_api.md) for the full RPC API reference.
