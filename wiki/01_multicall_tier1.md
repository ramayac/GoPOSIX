# Phase 01 — Multicall Dispatcher + Tier 1 Utilities

> **HISTORICAL — COMPLETED.** This phase is done. Document retained for reference.
>
> **Timeline:** Week 1–2 | **Depends on:** Phase 00 | **Blocks:** Phase 02

---

## Goal

Create the single-binary multicall dispatcher and implement the 10 simplest POSIX utilities. These are trivial to implement but prove the entire architecture works end to end.

---

## Tasks

### 01.1 — Command Registry (`internal/dispatch/dispatch.go`)

**Design:**
```go
type Command struct {
    Name    string
    Run     func(args []string) int  // returns exit code
    Usage   string
}

var registry = map[string]Command{}

func Register(cmd Command) {
    registry[cmd.Name] = cmd
}

func Lookup(name string) (Command, bool) {
    cmd, ok := registry[name]
    return cmd, ok
}
```

- [x] Registry is a `map[string]Command`
- [x] Each utility calls `dispatch.Register()` in its `init()` function
- [x] `Lookup()` returns the command or false

---

### 01.2 — Multicall Entry Point (`cmd/korego/main.go`)

**Logic flow:**
```
1. name = filepath.Base(os.Args[0])
2. if name == "korego" AND len(os.Args) > 1:
       name = os.Args[1]
       os.Args = os.Args[1:]  // shift
3. cmd, ok = dispatch.Lookup(name)
4. if !ok → print "unknown command", exit 127
5. exitCode = cmd.Run(os.Args[1:])
6. os.Exit(exitCode)
```

- [x] Symlink dispatch: `/bin/ls` → `/bin/korego` → runs `ls`
- [x] Subcommand dispatch: `korego ls -la` → runs `ls`
- [x] Unknown command → exit code 127 (POSIX standard)
- [x] `korego --help` lists all registered commands
- [x] `korego --version` prints version string

**Test cases:**
```bash
# Symlink test (manual)
ln -s ./korego ./echo
./echo hello       # prints "hello"

# Subcommand test
./korego echo hello  # prints "hello"

# Unknown command
./korego nonexist    # exit 127
```

---

### 01.3 — Tier 1 Utilities

Each utility follows this pattern:
1. **Library** in `pkg/<name>/` — pure function returning a struct
2. **CLI wrapper** — registered in dispatcher, calls library, uses `Render()`

| Utility | Package | Library Return | `--json` Output | Notes |
|---------|---------|---------------|-----------------|-------|
| `echo` | [pkg/echo/](../pkg/echo/) | `EchoResult{Text string}` | `{"data":{"text":"hello"}}` | Support `-n` (no newline), `-e` (escapes) |
| `true` / `false` | [pkg/truefalse/](../pkg/truefalse/) | — | — | Just `os.Exit(0)` / `os.Exit(1)` |
| `yes` | [pkg/yes/](../pkg/yes/) | — | — | Infinite loop printing "y" or arg. No `--json`. |
| `whoami` | [pkg/whoami/](../pkg/whoami/) | `WhoamiResult{User string, UID int}` | `{"data":{"user":"root","uid":0}}` | `os/user.Current()` |
| `hostname` | [pkg/hostname/](../pkg/hostname/) | `HostnameResult{Name string}` | `{"data":{"hostname":"abc"}}` | `os.Hostname()` |
| `uname` | [pkg/uname/](../pkg/uname/) | `UnameResult{Sysname,Node,Release,Version,Machine}` | Full struct | `syscall.Uname()` |
| `pwd` | [pkg/pwd/](../pkg/pwd/) | `PwdResult{Path string}` | `{"data":{"path":"/home"}}` | `os.Getwd()`. Flag: `-P` (physical, resolve symlinks) |
| `printenv` | [pkg/printenv/](../pkg/printenv/) | `PrintenvResult{Vars map[string]string}` | Full env map | Specific var: `printenv HOME` |
| `env` | [pkg/env/](../pkg/env/) | `EnvResult{Vars map[string]string}` | Full env map | Like printenv but also supports `-i` (ignore environment) |

**Per-utility checklist (repeat for each):**
- [x] `pkg/<name>/<name>.go` — library function
- [x] `pkg/<name>/<name>_test.go` — unit tests
- [x] CLI wrapper registered via `init()`
- [x] `--json` flag works via `common.Render()`
- [x] Exit codes match POSIX spec
- [x] `--help` prints usage

---

## Milestone 01

- [x] `korego echo hello` prints `hello`
- [x] `korego echo --json hello` prints `{"command":"echo","data":{"text":"hello"},...}`
- [x] `korego true` exits 0
- [x] `korego false` exits 1
- [x] `korego --help` lists all 10 commands
- [x] Symlink dispatch works (`ln -s korego echo && ./echo hi`)
- [x] Unknown commands exit 127
- [x] All Tier 1 utilities have unit tests passing

## How to Verify

```bash
# Build
CGO_ENABLED=0 go build -o korego ./cmd/korego/

# Unit tests
go test -v -cover ./pkg/echo/ ./pkg/whoami/ ./pkg/uname/ ...

# Manual smoke tests
./korego true ; echo $?                  # 0
./korego false ; echo $?                 # 1
./korego echo hello                      # hello
./korego echo --json hello               # JSON envelope
./korego uname --json                    # structured uname
./korego whoami --json                   # {"user":"...","uid":...}
./korego nonexist ; echo $?             # 127

# Symlink test
ln -sf ./korego ./pwd && ./pwd           # prints cwd
```
