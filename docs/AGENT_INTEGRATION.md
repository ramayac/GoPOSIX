# Agent Integration Guide

How to use KoreGo as a tool-execution backend for AI agents.

## Overview

KoreGo exposes a Unix socket-based JSON-RPC 2.0 API that lets an AI agent:

1. Start a KoreGo daemon (lightweight, ~8 MB container image)
2. Create isolated sessions with per-session working directories
3. Execute POSIX utilities and shell scripts
4. Receive structured JSON output (no scraping stdout)
5. Clean up when done

## Quick Start

```bash
make example-agent
```

This runs `examples/agent/main.go` — a self-contained Go program that demonstrates the full lifecycle.

## Architecture

```
┌──────────────┐    Unix socket     ┌─────────────────┐
│  AI Agent    │ ◄──── JSON-RPC ───► │  KoreGo Daemon  │
│  (your code) │                    │  (korego daemon) │
└──────────────┘                    └────────┬────────┘
                                             │
                                    ┌────────▼────────┐
                                    │  POSIX Utilities │
                                    │  ls, cat, grep,  │
                                    │  find, wc, ...   │
                                    └─────────────────┘
```

## Lifecycle

### 1. Start the Daemon

```bash
korego daemon -s /tmp/korego.sock -w 4
```

Flags:
- `-s` — Unix socket path
- `-w` — Worker pool size (default 4)
- `-l` — HTTP observability address (optional, e.g. `:9090`)

### 2. Connect

```go
conn, _ := net.DialTimeout("unix", "/tmp/korego.sock", 5*time.Second)
```

All communication is newline-delimited JSON over a Unix socket. Each request gets one response.

### 3. Ping (Health Check)

```json
→ {"jsonrpc":"2.0", "method":"korego.ping", "id":1}
← {"jsonrpc":"2.0", "result":{"pong":true, "uptime":"5s", "version":"0.1.0"}, "id":1}
```

### 4. Create a Session

Sessions isolate state — each has its own CWD and environment variables.

```json
→ {"jsonrpc":"2.0", "method":"korego.session.create", "id":2}
← {"jsonrpc":"2.0", "result":{"sessionId":"a1b2c3d4", "cwd":"/", "env":{}, "lastActive":"..."}, "id":2}
```

### 5. Set Working Directory

```json
→ {"jsonrpc":"2.0", "method":"korego.session.setCwd", "params":{"sessionId":"a1b2c3d4", "path":"/etc"}, "id":3}
← {"jsonrpc":"2.0", "result":true, "id":3}
```

Paths are validated to prevent traversal outside the session CWD. The default CWD (`/`) allows all absolute paths.

### 6. Execute Utilities

All 42 JSON-enabled utilities are available as `korego.<name>` methods:

```json
→ {"jsonrpc":"2.0", "method":"korego.ls", "params":{"sessionId":"a1b2c3d4"}, "id":4}
← {"jsonrpc":"2.0", "result":{"exitCode":0, "data":{"path":"/etc", "files":[...], "total":16}}, "id":4}
```

```json
→ {"jsonrpc":"2.0", "method":"korego.wc", "params":{"sessionId":"a1b2c3d4", "path":"hosts"}, "id":5}
← {"jsonrpc":"2.0", "result":{"exitCode":0, "data":{"lines":11, "words":46, "bytes":225, "chars":225}}, "id":5}
```

Standard params: `sessionId` (string), `path` (string), `flags` ([]string), `text` (string, for echo).

### 7. Execute Shell Scripts

```json
→ {"jsonrpc":"2.0", "method":"korego.shell.exec", "params":{"sessionId":"a1b2c3d4", "script":"echo hello from agent"}, "id":6}
← {"jsonrpc":"2.0", "result":{"stdout":"hello from agent\n", "stderr":"", "exitCode":0}, "id":6}
```

The shell interpreter runs with a 30-second timeout and 128 MB memory limit per stream.

### 8. Destroy the Session

```json
→ {"jsonrpc":"2.0", "method":"korego.session.destroy", "params":{"sessionId":"a1b2c3d4"}, "id":7}
← {"jsonrpc":"2.0", "result":true, "id":7}
```

Sessions auto-expire after 30 minutes of inactivity.

### 9. Stop the Daemon

Send SIGTERM to the daemon process. It drains connections and removes the socket file.

## Batch Requests

Send an array of requests to execute multiple calls in one round-trip:

```json
→ [
    {"jsonrpc":"2.0", "method":"korego.echo", "params":{"text":"a"}, "id":1},
    {"jsonrpc":"2.0", "method":"korego.echo", "params":{"text":"b"}, "id":2}
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

## Example: Multi-Step Agent Task

The full example at `examples/agent/main.go` demonstrates:

1. Start daemon
2. Ping
3. Create session
4. Set CWD to `/etc`
5. List files with `korego.ls`
6. Count lines with `korego.wc`
7. Run shell command with `korego.shell.exec`
8. Read file contents with `korego.cat`
9. Destroy session
10. Stop daemon

```bash
go run ./examples/agent/main.go
```

## Programmatic Client

For production use, import `pkg/client/` for connection pooling, retry, and typed helper methods. See `docs/RPC_API.md`.
