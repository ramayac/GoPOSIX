# Docker Compose

Run KoreGo as a Unix socket sidecar alongside an application container.

## Architecture

```
┌──────────────────┐     ┌──────────────────┐
│   app container  │────▶│ korego container │
│  (Python / Node) │     │  (daemon mode)   │
└──────────────────┘     └────────┬─────────┘
                                  │
                    /var/run/korego/korego.sock
                    (shared emptyDir volume)
```

## Setup

`examples/docker-compose.yml` provides a working example with an Alpine-based test client.

```bash
docker compose -f examples/docker-compose.yml up --build
```

## Volumes

| Path | Purpose |
|------|---------|
| `/var/run/korego/` | Shared Unix socket directory (must be writable by both containers) |

## Configuration

| Env var | Default | Purpose |
|---------|---------|---------|
| `KOREGO_SOCKET` | `/var/run/korego/korego.sock` | Socket path to connect to |
| `KOREGO_WORKERS` | `4` | Worker pool size for the daemon |
| `KOREGO_SHELL_TIMEOUT` | `30` | Shell execution timeout in seconds |

## Healthcheck

```yaml
healthcheck:
  test:
    [
      "CMD",
      "sh",
      "-c",
      "echo '{\"jsonrpc\":\"2.0\",\"method\":\"korego.ping\",\"id\":1}' | nc -U /var/run/korego/korego.sock",
    ]
  interval: 10s
  timeout: 3s
  retries: 3
```

## Security

- The daemon container runs as `nobody` (uid 65534) with read-only root filesystem.
- The socket is mode `0660` — both containers must share a group or run as the same user.
- For multi-tenant setups, run one daemon per application instance.

## Troubleshooting

**"Connection refused"**: The socket volume is not shared between containers. Check `volumes:` config.

**"Permission denied"**: Socket permissions (`0660`) don't match the client container's user. Either:
- Run both containers as the same uid, or
- Add both users to a shared group
