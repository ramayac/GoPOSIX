# RPC Client API

Go client library for the GoPOSIX daemon. Import path: `github.com/ramayac/goposix/pkg/client`.

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "github.com/ramayac/goposix/pkg/client"
)

func main() {
    c, _ := client.New("/var/run/goposix.sock", client.WithPoolSize(4))
    defer c.Close()
    ctx := context.Background()

    ping, _ := c.Ping(ctx)
    fmt.Printf("daemon version: %s\n", ping.Version)

    result, _ := c.Ls(ctx, "/var/log", []string{"-l"})
    for _, f := range result.Files {
        fmt.Printf("%s (%d bytes)\n", f.Name, f.Size)
    }
}
```

## Connection Lifecycle

```go
c, err := client.New("/tmp/goposix.sock",
    client.WithPoolSize(4),
    client.WithTimeout(30*time.Second),
    client.WithMaxRetries(2),
)
defer c.Close()
```

Connections are pooled and reused. The pool grows up to `poolSize` concurrent connections.

## Core Methods

### Call — generic JSON-RPC

```go
var result MyType
err := c.Call(ctx, "goposix.ls", params, &result)
```

### CallRaw — returns raw JSON

```go
raw, err := c.CallRaw(ctx, "goposix.someMethod", params)
```

### Batch — multiple requests in one round-trip

```go
reqs := []client.BatchRequest{
    {Method: "goposix.echo", Params: map[string]string{"text": "a"}},
    {Method: "goposix.echo", Params: map[string]string{"text": "b"}},
}
resps, err := c.Batch(ctx, reqs)
```

### Notify — fire-and-forget (no response)

```go
err := c.Notify(ctx, "goposix.true", nil)
```

## Typed Utility Helpers

### File Inspection

```go
c.Ls(ctx, "/tmp", []string{"-l"})
c.Cat(ctx, "/etc/hosts")
c.Stat(ctx, "/etc/passwd")
c.Find(ctx, "/etc", []string{"-name", "*.conf"})
c.Wc(ctx, "/etc/hosts")
```

### Text Processing

```go
c.Grep(ctx, "pattern", []string{"file.txt"})
c.Head(ctx, "/var/log/syslog", 20)
c.Tail(ctx, "/var/log/syslog", 50)
c.Sort(ctx, []string{"-r"})
c.Cut(ctx, []string{"-d:", "-f1,3"})
c.Uniq(ctx, []string{"-c"})
```

### File Operations

```go
c.Cp(ctx, "/src/file", "/dst/file")
c.Mv(ctx, "/src/file", "/dst/file")
c.Ln(ctx, "/target", "/link", true) // symbolic
c.Rm(ctx, []string{"/tmp/foo"}, false, false)
c.Rmdir(ctx, "/empty/dir")
c.Mkdir(ctx, "/new/dir", true) // mkdir -p
c.Touch(ctx, []string{"/tmp/a", "/tmp/b"})
c.Chmod(ctx, "0644", []string{"/tmp/f"})
c.Chown(ctx, "root", []string{"/tmp/f"})
c.Chgrp(ctx, "staff", []string{"/tmp/f"})
```

### System Info

```go
c.Date(ctx)
c.Uname(ctx)
c.Whoami(ctx)
c.ID(ctx)
c.Hostname(ctx)
c.Pwd(ctx)
c.Df(ctx, "/")
c.Du(ctx, "/tmp")
c.Ps(ctx)
```

### Environment

```go
c.Env(ctx, []string{"-i", "FOO=bar"}, nil)
c.Printenv(ctx, "HOME")
```

### Text Output

```go
c.Echo(ctx, "hello world")
c.Printf(ctx, "hello %s", "world")
c.Basename(ctx, "/etc/hosts")
c.Dirname(ctx, "/etc/hosts")
c.Readlink(ctx, "/proc/self/exe")
```

### Hash & Archive

```go
c.Md5sum(ctx, []string{"file.txt"}, false)
c.Sha256sum(ctx, []string{"file.txt"}, false)
c.Gzip(ctx, []string{"-c", "file.txt"})
c.Tar(ctx, []string{"-tf", "archive.tar"})
```

### Process & Execution

```go
c.Kill(ctx, "SIGTERM", []int{1234})
c.Expr(ctx, []string{"1", "+", "1"})
c.Test(ctx, []string{"-f", "/etc/hosts"})
c.Xargs(ctx, "echo", []string{})
```

### Diff

```go
c.Diff(ctx, "/etc/hosts", "/etc/host.conf")
```

### Session Management

```go
s, _ := c.SessionCreate(ctx)
c.SessionSetCwd(ctx, s.SessionID, "/etc")
c.SessionList(ctx)
c.SessionDestroy(ctx, s.SessionID)
```

### Shell Execution

```go
c.ShellExec(ctx, sessionID, "echo hello && ls -la")
```

The `sessionID` parameter is optional — pass `""` for stateless one-off commands. When provided,
the shell inherits the session's working directory and environment. All executions are subject to
the timeout configured by `GOPOSIX_SHELL_TIMEOUT` (default `30s`, accepts Go duration strings like
`"60s"`, `"5m"`). See [Security Model](security.md) for the full sandbox and resource limit details.

### Ping

```go
c.Ping(ctx)
```

## Context Propagation

All methods accept `context.Context`:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
result, err := c.Ls(ctx, "/tmp", nil)
```

## Retry Behavior

Transient errors (connection refused, broken pipe, timeout, EOF) retry with exponential backoff:

- Attempt 0: immediate
- Attempt 1: 100ms
- Attempt 2: 200ms

Non-retryable errors (RPC errors, invalid params) return immediately.

## Error Handling

RPC errors include standard JSON-RPC 2.0 codes:

| Code | Meaning |
|------|---------|
| -32700 | Parse error |
| -32600 | Invalid Request |
| -32601 | Method not found |
| -32602 | Invalid params (includes path traversal) |
| -32000 | Server error / rate limited |
