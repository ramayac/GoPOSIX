# RPC Client API — Type Signature Reference

> **This is a type-signature catalog.** For usage, connection lifecycle, error handling,
> context propagation, retry behavior, and performance, see [sdk.md](sdk.md).

Import path: `github.com/ramayac/goposix/pkg/client`.

---

## Core Low-Level Methods

```go
func (c *Client) Call(ctx context.Context, method string, params interface{}, result interface{}) error
func (c *Client) CallRaw(ctx context.Context, method string, params interface{}) (json.RawMessage, error)
func (c *Client) Batch(ctx context.Context, reqs []BatchRequest) ([]BatchResponse, error)
func (c *Client) Notify(ctx context.Context, method string, params interface{}) error
```

---

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

The `sessionID` parameter is optional — pass `""` for stateless one-off commands.
See [security.md](security.md) for sandbox and resource limit details.

### Ping

```go
c.Ping(ctx)
```

---

## See Also

- [sdk.md](sdk.md) — Go SDK guide: connection, error handling, context propagation, retry, performance.
- [rpc_quickstart.md](rpc_quickstart.md) — Raw JSON-RPC protocol for non-Go clients.
