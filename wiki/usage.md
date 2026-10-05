---
status: current
description: "Usage guide: CLI, daemon, Docker, JSON-RPC."
references: [source:docker/Dockerfile, source:cmd/goposix/main.go, external:https://github.com/ramayac/goposix]
---

# Usage Guide

GoPOSIX is a single, statically-linked binary that works as both a CLI multicall tool
and a persistent JSON-RPC 2.0 daemon. This guide covers both modes.

---

## 1. CLI Mode

### Local build & run

```bash
# Build
make build
# or: go build -ldflags="-s -w" -o goposix ./cmd/goposix/

# Version
./goposix --version
# → goposix version (latest)

# Self-upgrade
./goposix --upgrade
# → upgrading goposix from <old> to <new>...
# → goposix upgraded to <new>

# Run a command
./goposix ls -la /
./goposix echo hello
./goposix cat /etc/hostname
```

### Docker (CLI image — FROM scratch, ~10 MB)

```bash
docker build -t goposix:cli -f docker/Dockerfile.cli .

# One-shot invocation
docker run --rm goposix:cli ls -la /
docker run --rm goposix:cli echo "hello from scratch"
docker run --rm goposix:cli stat /bin/goposix
```

### Symlink mode

When the binary is named after a utility, it auto-dispatches:

```bash
ln -s goposix /usr/local/bin/ls
/usr/local/bin/ls -la /home   # dispatches to goposix ls
```

---

## 2. Daemon Mode

### Docker (daemon image — 9.7 MB, FROM scratch)

The default image starts the JSON-RPC daemon:

```bash
# Build
docker build -t goposix:latest -f docker/Dockerfile .

# Start daemon
docker run -d --name goposix -p 8080:8080 goposix:latest

# Health check
curl http://localhost:8080/healthz
# → {"status":"ok"}

# Prometheus metrics
curl http://localhost:8080/metrics
```

### Docker Compose

```yaml
# docker-compose.yml
services:
  goposix:
    build:
      context: .
      dockerfile: docker/Dockerfile
    ports:
      - "8080:8080"
    volumes:
      - goposix-data:/home/goposix
    healthcheck:
      test: ["CMD", "/bin/goposix", "echo", "ok"]
      interval: 10s
      timeout: 3s
      retries: 3

volumes:
  goposix-data:
```

```bash
docker compose up -d
curl http://localhost:8080/healthz
```

### Local daemon

```bash
# Start the daemon on a Unix socket
./goposix daemon --socket /tmp/goposix.sock --workers 4

# Or with HTTP metrics
./goposix daemon --socket /tmp/goposix.sock --workers 4 --listen-addr :8080
```

Short flags also work:

```bash
./goposix daemon -s /tmp/goposix.sock -w 4 -l :8080
```

---

## 3. Talking to the Daemon

### Go over the Unix socket

```go
package main

import (
    "encoding/json"
    "fmt"
    "log"
    "net"
    "time"
)

func main() {
    conn, err := net.DialTimeout("unix", "/tmp/goposix.sock", 5*time.Second)
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()

    enc := json.NewEncoder(conn)
    enc.Encode(map[string]interface{}{
        "jsonrpc": "2.0",
        "method":  "goposix.echo",
        "params":  map[string]string{"text": "hello"},
        "id":      1,
    })

    var resp map[string]interface{}
    json.NewDecoder(conn).Decode(&resp)
    fmt.Println(resp)
}
```

See [rpc_quickstart.md](rpc_quickstart.md) for the full protocol reference.

### Raw JSON-RPC over Unix socket

```bash
# Ping
echo '{"jsonrpc":"2.0","method":"goposix.ping","id":1}' | \
  socat - UNIX-CONNECT:/tmp/goposix.sock

# Echo
echo '{"jsonrpc":"2.0","method":"goposix.echo","params":{"text":"hello"},"id":1}' | \
  socat - UNIX-CONNECT:/tmp/goposix.sock

# List files
echo '{"jsonrpc":"2.0","method":"goposix.ls","params":{"path":"/etc"},"id":1}' | \
  socat - UNIX-CONNECT:/tmp/goposix.sock
```

### Raw JSON-RPC over HTTP

When the daemon has `--listen-addr :8080`:

```bash
curl -s -X POST http://localhost:8080/rpc \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","method":"goposix.echo","params":{"text":"hello"},"id":1}'
```

> **Note:** The HTTP endpoint only serves the `/healthz`, `/readyz`, and `/metrics`
> observability routes. JSON-RPC is over the Unix socket. For full HTTP JSON-RPC,
> place a reverse proxy (nginx, caddy) in front that forwards to the Unix socket.

---

## 4. Smart Forwarding

When a daemon is running locally, the CLI automatically forwards commands to it:

```bash
# Start daemon in background
./goposix daemon -s /tmp/goposix.sock &

# These will auto-forward to the daemon (no cold start)
./goposix echo hello     # 60µs via daemon, not 7ms cold start
./goposix ls -la /       # same
```

The forwarding is transparent — exit codes, stdout, and stderr are preserved.
If the daemon socket is unavailable, the CLI falls back to cold-start execution.

---

## 5. JSON Output

Every utility supports `--json` for structured, machine-readable output:

```bash
./goposix ls --json /etc
```

```json
{
  "command": "ls",
  "version": "<latest>",
  "schemaVersion": "1.0",
  "exitCode": 0,
  "data": {
    "files": [
      {"name": "hosts", "size": 258, "isDir": false},
      {"name": "hostname", "size": 13, "isDir": false}
    ],
    "total": 2
  }
}
```

---

## 6. Docker Images

| Tag | Base | Size | Purpose |
|-----|------|:----:|---------|
| `goposix:latest` | `FROM scratch` | ~10 MB | Daemon (JSON-RPC + HTTP metrics) |
| `goposix:cli` | `FROM scratch` | ~10 MB | One-shot CLI invocations |
| `goposix:debug` | `alpine:3.20` | ~28 MB | Shell + strace for interactive debugging |

```bash
# Production daemon
docker run -d --name goposix -p 8080:8080 goposix:latest

# One-shot CLI
docker run --rm goposix:cli ls -la /

# Interactive debug
docker run -it --rm goposix:debug sh
```

---

## 7. Quick Recipes

### Count lines of code in a project

```bash
./goposix find . -name '*.go' | xargs ./goposix wc -l
```

### Monitor a log file

```bash
./goposix tail -f /var/log/app.log | ./goposix grep ERROR
```

### Check disk usage

```bash
./goposix du -sh /home/user/project
./goposix df -h /
```

### Process JSON

```bash
./goposix cat data.json | ./goposix grep "status" | ./goposix sort | ./goposix uniq -c
```

### Self-upgrade in CI

```bash
curl -sL https://github.com/ramayac/GoPOSIX/releases/latest/download/goposix_linux_amd64.tar.gz | \
  tar xz -C /usr/local/bin/ goposix
```

---

## See Also

- [Architecture](architecture.md) — Component flow, packages, Docker images
- [RPC Quickstart](rpc_quickstart.md) — JSON-RPC client API
- [Self-Upgrade](self_upgrade.md) — `--version`, `--upgrade` details
- [Security](security.md) — Security model, deployment posture
