# Phase 16 — MCP Server

> **Status:** Design | **Depends on:** Phase 05 (daemon), Phase 07 (sessions, shell) | **Milestone:** Phase 16

---

## Goal

Expose KoreGo as an MCP (Model Context Protocol) server so external AI agents — Claude Desktop, Claude Code, Cursor, Continue, or any MCP-compatible client — can use KoreGo as their sandboxed Linux environment. The agent brings its own LLM and reasoning; KoreGo provides the "computer."

Two transports share one engine:

```
Local:    korego mcp                                    # stdio transport
Remote:   korego mcp --http :8080                       # HTTP/SSE transport
```

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────┐
│                    KoreGo Binary                         │
│                                                         │
│  ┌──────────┐  ┌──────────┐  ┌──────────────────────┐  │
│  │   CLI    │  │   RPC    │  │    MCP Server         │  │
│  │ (daemon) │  │ (socket) │  │  ┌────────────────┐   │  │
│  └──────────┘  └──────────┘  │  │  Transport     │   │  │
│                              │  │  (stdio/SSE)   │   │  │
│                              │  └───────┬────────┘   │  │
│                              │  ┌───────┴────────┐   │  │
│                              │  │  Tool Registry │   │  │
│                              │  └───────┬────────┘   │  │
│                              │          │             │  │
│                              │  ┌───────┴────────┐   │  │
│                              │  │      Shared     │   │  │
│                              │  │  ┌──────────┐   │   │  │
│                              │  │  │  Shell    │   │   │  │
│                              │  │  │  Exec     │   │   │  │
│                              │  │  ├──────────┤   │   │  │
│                              │  │  │  Session  │   │   │  │
│                              │  │  │  Manager  │   │   │  │
│                              │  │  ├──────────┤   │   │  │
│                              │  │  │  Worker   │   │   │  │
│                              │  │  │  Pool     │   │   │  │
│                              │  │  ├──────────┤   │   │  │
│                              │  │  │  Secure   │   │   │  │
│                              │  │  │  Path     │   │   │  │
│                              │  │  └──────────┘   │   │  │
│                              │  └─────────────────┘   │  │
│                              │                        │  │
│                              │  Existing KoreGo       │  │
│                              │  (ls, cat, sed, grep,  │  │
│                              │   find, diff, tar,     │  │
│                              │   wc, sort, uniq, ...) │  │
│                              └────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
```

---

## MCP Tool Surface

Six curated tools. Shell execution provides maximum flexibility; file tools give structured access to the most impactful operations.

| Tool | Purpose | Key Parameters |
|------|---------|----------------|
| `shell.exec` | Run a shell script in the workspace | `script` (required), `timeout` (default 30s) |
| `file.read` | Read file contents | `path` (required), `offset`, `limit` (default 64KB) |
| `file.write` | Write or overwrite a file | `path` (required), `content` (required) |
| `file.edit` | Targeted edit via KoreGo's `sed` engine | `path` (required), `pattern` (required), `replacement` |
| `file.list` | List directory contents (`ls -la`) | `path` (required) |
| `workspace.set` | Set workspace root and cwd | `path` (required) |

### Tool Schemas

Each tool exposes typed JSON Schema parameter definitions so the LLM receives structured descriptions. Example:

```json
{
  "name": "shell.exec",
  "description": "Run a shell script in the sandboxed workspace. Returns stdout, stderr, and exit code.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "script": {"type": "string", "description": "Shell script to execute"},
      "timeout": {"type": "integer", "description": "Timeout in seconds (default 30, max 300)"}
    },
    "required": ["script"]
  }
}
```

### Return Values

- `shell.exec` → `{stdout, stderr, exitCode}`
- `file.read` → `{path, content, truncated, totalBytes}`
- `file.write` → `{path, bytesWritten}`
- `file.edit` → `{path, modified, diff}`
- `file.list` → `{path, entries: [{name, size, mode, isDir}]}`
- `workspace.set` → `{workspace, cwd}`

All errors return an MCP error object with a descriptive message.

---

## Transport & Protocol

MCP uses JSON-RPC 2.0 — the same protocol as the existing daemon. Two transports:

### Stdio (default)

`korego mcp` reads JSON-RPC messages from stdin, writes to stdout. This is the transport expected by Claude Desktop, Cursor, Continue, and most local MCP clients. The client spawns the process and communicates over pipes.

All logging is suppressed from stdout (either disabled or written to `--log-file`). Stderr is reserved for MCP transport-level logging if enabled.

### HTTP/SSE (`--http :8080`)

For remote agents or server deployments:
- `POST /mcp` — accepts JSON-RPC requests
- `GET /sse` — Server-Sent Events for server→client notifications

Session identified by a header (`X-Session-Id`) or query parameter (`?sessionId=`).

### MCP Lifecycle

```
initialize (client↔server handshake, capabilities exchange)
    │
initialized (server sends tools/list, creates session)
    │
tools/call (loop — agent invokes tools)
    │
shutdown (server destroys session, optional cleanup)
```

---

## Session Model

Each MCP connection gets a session backed by the existing `internal/daemon/session.go` Session type, which provides:

- **Session ID** — unique per connection
- **cwd** — current working directory, confined to workspace root
- **env vars** — inherited from host process, mutable per session
- **Command history** — retained for duration of session
- **Idle TTL** — auto-expire after idle timeout (default 30min, configurable via `--session-ttl`)

### Lifecycle

| Event | Behavior |
|-------|----------|
| `initialize` | Session created, workspace root set to `--workspace` flag |
| `workspace.set` | cwd updated within workspace root (`SecurePath` enforced) |
| Disconnect | Session remains alive until TTL expires |
| Reconnect | Same session ID resumes prior state (cwd, env, history) |
| `shutdown` | Session destroyed, workspace optionally cleaned (`--cleanup`) |
| TTL expiry | Session garbage-collected, workspace cleaned if `--cleanup` |

If the client doesn't track session IDs, each new connection gets a fresh session.

---

## Concurrency

Each MCP connection gets its own goroutine. The worker pool (reused from the daemon) limits concurrent shell executions per session.

Default: **4 concurrent execs per connection**, configurable via `--max-execs`.

File operations (`file.read`, `file.write`, `file.edit`, `file.list`) are serial — they complete synchronously within the request handler. Shell execution is the only async operation since it may be long-running.

---

## Security

| Concern | Mitigation |
|---------|------------|
| Filesystem escape | `common.SecurePath` validates all paths; confined to workspace root |
| Shell abuse | Per-command timeout (default 30s, max 300s); 128MB stream limit |
| Resource exhaustion | Worker pool caps concurrent execs per session (`--max-execs`, default 4) |
| Credential leaks | No tokens in tool params. Git auth (if needed later) uses env vars from host process |
| Host access | Agent operates within workspace directory only |
| Infinite loops | Shell timeout + session TTL provide hard bounds |

---

## Configuration

| Flag | Env Fallback | Default | Description |
|------|-------------|---------|-------------|
| `--http` | `KOREGO_MCP_HTTP` | `""` (stdio mode) | HTTP listen address for SSE transport |
| `--workspace` | `KOREGO_MCP_WORKSPACE` | `/tmp/korego-mcp` | Workspace root directory |
| `--session-ttl` | `KOREGO_MCP_SESSION_TTL` | `30m` | Session idle timeout |
| `--max-execs` | `KOREGO_MCP_MAX_EXECS` | `4` | Max concurrent shell execs per connection |
| `--cleanup` | — | `true` | Remove workspace on session shutdown |
| `--log-file` | — | `""` (disabled) | Write logs to file (required for stdio transport) |

---

## CLI Interface

```
korego mcp                                      # stdio transport (default)
korego mcp --http :8080                         # HTTP/SSE transport
korego mcp --workspace /var/lib/korego/agents   # custom workspace root
korego mcp --session-ttl 1h --max-execs 8       # tune concurrency and TTL
korego mcp --log-file /var/log/korego-mcp.log   # log to file (stdio mode)
korego mcp --no-cleanup                         # keep workspace after shutdown
```

---

## Package Structure

```
internal/mcp/
├── server.go       # MCP protocol: handshake, tool dispatch, JSON-RPC messages
├── transport.go    # Transport layer: stdio and HTTP/SSE
├── tools.go        # Tool definitions (JSON Schema) and handler implementations
└── server_test.go  # Unit tests with mock MCP client

cmd/korego/main.go  # registers "mcp" subcommand via dispatch
```

### Integration Points

| Package | Integrates With |
|---------|----------------|
| `internal/mcp/server.go` | `internal/dispatch/` — subcommand registration |
| `internal/mcp/tools.go` | `internal/shell/interpreter.go` — shell execution |
| `internal/mcp/tools.go` | `internal/daemon/session.go` — session management |
| `internal/mcp/tools.go` | `internal/daemon/ratelimit.go` — worker pool |
| `internal/mcp/tools.go` | `pkg/common/` — SecurePath, file I/O |
| `internal/mcp/transport.go` | `net`, `net/http` — stdlib only |

### Dependencies

No new external dependencies. MCP is JSON-RPC 2.0 — the same protocol the daemon already speaks. All implementation uses stdlib + existing internal packages.

---

## Docker Integration

### Scratch Image Additions

- No new binary dependencies — MCP server is pure Go
- CA certificates already present (for HTTPS in HTTP/SSE mode)
- Workspace directory created at build time

### Compose Example

```yaml
services:
  korego-mcp:
    image: korego:latest
    command: ["mcp", "--http", ":8080", "--workspace", "/var/lib/korego/workspaces"]
    ports:
      - "8080:8080"
    volumes:
      - /data/agent-workspaces:/var/lib/korego/workspaces
    environment:
      - KOREGO_MCP_SESSION_TTL=1h
      - KOREGO_MCP_MAX_EXECS=8
    security_opt:
      - no-new-privileges:true
```

---

## Client Configuration Examples

### Claude Desktop (`claude_desktop_config.json`)

```json
{
  "mcpServers": {
    "korego": {
      "command": "docker",
      "args": ["run", "-i", "--rm", "korego:latest", "mcp"]
    }
  }
}
```

### Claude Code (`.claude/mcp.json`)

```json
{
  "mcpServers": {
    "korego": {
      "type": "stdio",
      "command": "korego",
      "args": ["mcp", "--workspace", "/home/user/projects"]
    }
  }
}
```

### Cursor (`.cursor/mcp.json`)

```json
{
  "mcpServers": {
    "korego": {
      "command": "korego",
      "args": ["mcp"]
    }
  }
}
```

---

## Verification Plan

- **Unit tests:** MCP protocol handshake, each tool handler, session lifecycle, SecurePath enforcement, transport framing (stdio and HTTP).
- **Integration test:** Spawn `korego mcp`, send `initialize` + `tools/list` + `tools/call` sequence via pipes, verify JSON-RPC responses.
- **Agent smoke test:** Configure Claude Code or Cursor to use `korego mcp`, give it a simple task ("list files, read README, write a test file"), verify it completes.
- **Docker smoke test:** `docker run --rm korego:latest mcp` — verify the binary starts and responds to stdin JSON-RPC.
