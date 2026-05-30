# Shell Integration — CLI-to-Daemon Forwarding

> Moved from `docs/SHELL_INTEGRATION.md`.

How to make every CLI command (`ls`, `cat`, `grep`, ...) talk to a GoPOSIX daemon
at daemon speed (~100µs) without launching a new Go process. This is a portable
shell-layer pattern — no second binary, no CGO, no additional build targets.

Use this in any project that embeds GoPOSIX or exposes a JSON-RPC daemon over a
Unix socket.

---

## The Pattern

```
┌──────────────┐     shell alias/function      ┌──────────────────┐
│  $ ls -la /  │ ──── JSON-RPC over socket ──► │  goposix daemon  │
│  $ cat file  │      (socat / nc / /dev/tcp)  │  (already running)│
└──────────────┘                                └──────────────────┘
```

No cold start. No Go process. The shell sends JSON, reads JSON, prints text.
Latency: ~100µs (dominated by `socat`/`nc` process startup — still 70× better
than 7ms Go cold start).

---

## Option 1: socat (most portable)

Works on any system with `socat` installed (Alpine, Debian, BusyBox).

```bash
# ~/.bashrc or /etc/profile.d/goposix.sh

GOPOSIX_SOCK="${GOPOSIX_SOCKET:-/var/run/goposix.sock}"

_goposix_rpc() {
    local cmd="$1"; shift
    local method="goposix.$cmd"
    local args_json="["
    local sep=""
    for arg in "$@"; do
        args_json+="${sep}\"${arg}\""
        sep=","
    done
    args_json+="]"

    local request="{\"jsonrpc\":\"2.0\",\"method\":\"${method}\",\"params\":{\"rawOutput\":true,\"flags\":${args_json}},\"id\":1}"

    local response
    response=$(echo "$request" | socat -T2 - UNIX-CONNECT:"$GOPOSIX_SOCK" 2>/dev/null)

    # Extract stdout and exit code from the response.
    local stdout
    stdout=$(echo "$response" | sed -n 's/.*"stdout":"\([^"]*\)".*/\1/p' | sed 's/\\n/\n/g')
    local exit_code
    exit_code=$(echo "$response" | sed -n 's/.*"exitCode":\([0-9]*\).*/\1/p')

    printf '%s' "$stdout"
    return "${exit_code:-0}"
}

# Generate aliases for all commands.
_goposix_commands() {
    goposix --list-commands 2>/dev/null || echo ""
}

if [ -S "$GOPOSIX_SOCK" ]; then
    for _cmd in $(_goposix_commands); do
        eval "${_cmd}() { _goposix_rpc ${_cmd} \"\$@\"; }"
    done
    unset _cmd
fi
```

**Caveats with this approach:**
- `sed` parsing of JSON is fragile — escape sequences in output may break it
- Multi-line output needs the `\n` → newline conversion
- `socat` adds ~500µs per invocation (still 14× faster than Go cold start)

---

## Option 2: Python one-liner (robust JSON parsing)

If Python is available, use it for proper JSON handling:

```bash
GOPOSIX_SOCK="${GOPOSIX_SOCKET:-/var/run/goposix.sock}"

_goposix_rpc() {
    local cmd="$1"; shift
    python3 -c "
import json, socket, sys
req = json.dumps({
    'jsonrpc': '2.0',
    'method': f'goposix.${cmd}',
    'params': {'rawOutput': True, 'flags': sys.argv[1:]},
    'id': 1
})
s = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
s.connect('${GOPOSIX_SOCK}')
s.sendall((req + '\n').encode())
resp = json.loads(s.recv(4096))
s.close()
r = resp.get('result', {})
sys.stdout.write(r.get('stdout', ''))
sys.exit(r.get('exitCode', 0))
" "$@"
}

# Generate aliases...
for _cmd in $(goposix --list-commands 2>/dev/null); do
    eval "${_cmd}() { _goposix_rpc ${_cmd} \"\$@\"; }"
done
```

**Tradeoffs:**
- Robust JSON parsing (no `sed` fragility)
- Python cold start: ~20ms (worse than Go!)
- Only use if Python is already resident (warm interpreter)

---

## Option 3: Go helper binary (fastest, simplest to maintain)

A ~50-line Go program that does nothing but forward JSON-RPC. No dispatcher,
no utilities imported — just `net`, `encoding/json`, `os`. Binary size: ~2MB.

```go
// cmd/goposix-fwd/main.go
package main

import (
    "encoding/json"
    "fmt"
    "net"
    "os"
    "strings"
)

func main() {
    if len(os.Args) < 2 {
        fmt.Fprintln(os.Stderr, "usage: goposix-fwd <command> [args...]")
        os.Exit(2)
    }
    socket := os.Getenv("GOPOSIX_SOCKET")
    if socket == "" {
        socket = "/var/run/goposix.sock"
    }

    cmd := os.Args[1]
    req, _ := json.Marshal(map[string]interface{}{
        "jsonrpc": "2.0",
        "method":  "goposix." + cmd,
        "params": map[string]interface{}{
            "rawOutput": true,
            "flags":     os.Args[2:],
        },
        "id": 1,
    })

    conn, err := net.Dial("unix", socket)
    if err != nil {
        fmt.Fprintf(os.Stderr, "%s: daemon unreachable: %v\n", cmd, err)
        os.Exit(126)
    }
    defer conn.Close()

    conn.Write(append(req, '\n'))

    var resp struct {
        Result struct {
            ExitCode int    `json:"exitCode"`
            Stdout   string `json:"stdout"`
        } `json:"result"`
        Error struct {
            Message string `json:"message"`
        } `json:"error"`
    }
    json.NewDecoder(conn).Decode(&resp)

    if resp.Error.Message != "" {
        fmt.Fprintf(os.Stderr, "%s: %s\n", cmd, resp.Error.Message)
        os.Exit(126)
    }

    os.Stdout.WriteString(resp.Result.Stdout)
    os.Exit(resp.Result.ExitCode)
}
```

Then symlink or alias:
```bash
# Build once
CGO_ENABLED=0 go build -ldflags="-s -w" -o /usr/local/bin/goposix-fwd ./cmd/goposix-fwd/

# Alias all commands through the forwarder
for cmd in $(goposix --list-commands); do
    ln -sf /usr/local/bin/goposix-fwd "/usr/local/bin/${cmd}"
done
```

This is the same idea as M5 but in a separate 2MB binary instead of the full
10MB multicall binary. Cold start: ~3ms (fewer init() functions). Still not
60µs, but 2× better than the full binary.

---

## Option 4: Pure bash with /dev/tcp (zero dependencies)

Bash's builtin `/dev/tcp` pseudo-device can talk to sockets. No `socat`, no
`nc`, no Python. Works on any bash.

```bash
_goposix_rpc() {
    local cmd="$1"; shift
    local args_json="["
    local sep=""
    for arg in "$@"; do
        args_json+="${sep}\"${arg//\"/\\\"}\""
        sep=","
    done
    args_json+="]"

    local request="{\"jsonrpc\":\"2.0\",\"method\":\"goposix.${cmd}\",\"params\":{\"rawOutput\":true,\"flags\":${args_json}},\"id\":1}"

    exec 3<>"/dev/tcp/127.0.0.1/8080" 2>/dev/null || {
        # Fallback: try Unix socket via socat if available
        if command -v socat >/dev/null 2>&1; then
            echo "$request" | socat -T2 - UNIX-CONNECT:"$GOPOSIX_SOCK" 2>/dev/null
        fi
        return 126
    }
    echo "$request" >&3
    local response
    read -r response <&3
    exec 3>&-

    # ... extract stdout/exitCode as in Option 1
}
```

**Limitations:**
- `/dev/tcp` only works with TCP, not Unix sockets — needs the daemon's HTTP port
- Disabled in some hardened bash builds
- Same `sed` JSON fragility as Option 1

---

## Recommendation Matrix

| Scenario | Best Option | Why |
|----------|-------------|-----|
| **Docker container** (Alpine) | Option 1 (socat) | `socat` is already in Alpine; 500µs vs 7ms cold start |
| **Any system with Go** | Option 3 (thin binary) | Proper JSON, proper exit codes, 3ms start |
| **Embedded/minimal** | Option 4 (bash /dev/tcp) | Zero dependencies, works everywhere bash works |
| **Desktop/workstation** | Option 1 or 3 | socat is common; thin binary is clean |
| **Production server** | Just use the Go SDK | Don't forward CLI at all — call `pkg/client` directly |

---

## When NOT to Use Shell Forwarding

1. **You're writing a Go program.** Use `pkg/client` directly at 60µs/call.
2. **You're doing bulk operations.** Shell forwarding still spawns a new process
   (socat/nc/python) per invocation — the Go SDK's persistent connection is the
   right tool for loops.
3. **You need stdin.** Shell forwarding can't stream stdin through JSON-RPC.
4. **You need stderr capture.** The daemon's stderr goes to its own log, not
   back through the RPC response.

Shell forwarding is for interactive use and shell scripts where 500µs is
"fast enough" and the Go SDK isn't available.

---

## Integration Script (drop-in)

Save as `/etc/profile.d/goposix-daemon.sh`:

```bash
# If the daemon is running, alias all GoPOSIX commands through it.
GOPOSIX_SOCK="${GOPOSIX_SOCKET:-/var/run/goposix.sock}"

if [ -S "$GOPOSIX_SOCK" ] && command -v socat >/dev/null 2>&1; then
    _goposix_rpc() {
        local cmd="$1"; shift
        local args=""
        for a in "$@"; do args="${args}\"${a}\","; done
        args="[${args%,}]"
        local req="{\"jsonrpc\":\"2.0\",\"method\":\"goposix.${cmd}\",\"params\":{\"rawOutput\":true,\"flags\":${args}},\"id\":1}"
        local resp
        resp=$(echo "$req" | socat -T2 - UNIX-CONNECT:"$GOPOSIX_SOCK" 2>/dev/null)
        local out
        out=$(echo "$resp" | sed 's/.*"stdout":"//;s/","exitCode".*//' | sed 's/\\n/\n/g')
        local ec
        ec=$(echo "$resp" | grep -o '"exitCode":[0-9]*' | grep -o '[0-9]*')
        printf '%s' "$out"
        return "${ec:-0}"
    }

    for _cmd in $(goposix --list-commands 2>/dev/null); do
        eval "${_cmd}() { _goposix_rpc ${_cmd} \"\$@\"; }"
    done
    unset _cmd _goposix_rpc
fi
```

---

## Portability Notes

- The JSON-RPC protocol is the same for any GoPOSIX deployment — socket path
  is the only variable (configurable via `GOPOSIX_SOCKET`).
- The `rawOutput` parameter (added in M5) is what makes this work — without it,
  all output would be JSON envelopes, not human-readable text.
- For projects that embed GoPOSIX's daemon but use a different socket path,
  set `GOPOSIX_SOCKET` before sourcing this script.
