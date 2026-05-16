# Phase 00 — Foundation Libraries

> **HISTORICAL — COMPLETED.** This phase is done. Document retained for reference.
>
> **Timeline:** Week 1 | **Depends on:** Nothing | **Blocks:** Everything

---

## Goal

Build the three shared libraries that every utility depends on. Nothing else can start until these are solid.

---

## Tasks

### 00.1 — POSIX Flag Parser ([pkg/common/](../pkg/common/))

**Why first:** Every POSIX utility needs `-laR` style grouped short flags. Go's `flag` package doesn't support this. This must exist before any utility can be implemented.

**Requirements:**
- [x] Parse grouped short flags: `-laR` → `-l`, `-a`, `-R`
- [x] Parse long flags: `--all`, `--recursive`
- [x] Parse `--key=value` and `--key value`
- [x] Handle `--` (end of flags, everything after is positional)
- [x] Handle `-` (stdin convention)
- [x] Unknown flags return error + exit code 2 (POSIX standard)
- [x] Flags can appear in any order, mixed with positional args
- [x] Support flag repetition counting (`-vvv` → verbosity=3)

**Test cases:**
```go
// pkg/common/flags_test.go
func TestGroupedShortFlags(t *testing.T) {
    args := []string{"-laR", "/tmp"}
    result, err := ParseFlags(args, FlagSpec{...})
    assert(result.Has("l") && result.Has("a") && result.Has("R"))
    assert(result.Positional[0] == "/tmp")
}

func TestEndOfFlags(t *testing.T) {
    args := []string{"--", "-not-a-flag"}
    result, _ := ParseFlags(args, FlagSpec{})
    assert(len(result.Positional) == 1)
    assert(result.Positional[0] == "-not-a-flag")
}

func TestUnknownFlag(t *testing.T) {
    args := []string{"-z"}
    _, err := ParseFlags(args, FlagSpec{})
    assert(err != nil)
    assert(err.ExitCode == 2)
}
```

**Acceptance:** 100% test coverage. All edge cases pass.

---

### 00.2 — JSON Output Envelope (`pkg/common/output.go`)

**Why:** Every utility with `--json` must produce the same envelope format. Define it once.

**Envelope schema:**
```json
{
  "command": "ls",
  "version": "0.1.0",
  "exitCode": 0,
  "data": { ... },
  "error": null
}
```

On error:
```json
{
  "command": "ls",
  "version": "0.1.0",
  "exitCode": 2,
  "data": null,
  "error": {
    "code": "ENOENT",
    "message": "no such file or directory: /nope"
  }
}
```

**Implementation:**
```go
// pkg/common/output.go
type JSONEnvelope struct {
    Command  string      `json:"command"`
    Version  string      `json:"version"`
    ExitCode int         `json:"exitCode"`
    Data     interface{} `json:"data"`
    Error    *ErrorInfo  `json:"error"`
}

type ErrorInfo struct {
    Code    string `json:"code"`
    Message string `json:"message"`
}

// Render checks if --json is set. If yes, marshal envelope. If no, call textFn.
func Render(cmdName string, data interface{}, jsonMode bool, textFn func()) {
    if jsonMode {
        env := JSONEnvelope{Command: cmdName, Data: data, ExitCode: 0}
        json.NewEncoder(os.Stdout).Encode(env)
    } else {
        textFn()
    }
}
```

**Test cases:**
- [x] `--json` produces valid JSON that passes `json.Unmarshal`
- [x] Envelope always has all 5 keys (command, version, exitCode, data, error)
- [x] Error case sets `exitCode != 0`, `data: null`, `error: {...}`
- [x] Non-JSON mode calls the text formatter, not JSON

**Acceptance:** Schema documented. All utilities will import and use this.

---

### 00.3 — JSON-RPC 2.0 Types (`pkg/common/jsonrpc.go`)

**Why:** The daemon (Phase 05) and client library need these types. Defining them now means utilities can be designed to return RPC-compatible results from day one.

**Types:**
```go
// Per JSON-RPC 2.0 spec (https://www.jsonrpc.org/specification)
type RPCRequest struct {
    JSONRPC string          `json:"jsonrpc"` // must be "2.0"
    Method  string          `json:"method"`
    Params  json.RawMessage `json:"params,omitempty"`
    ID      interface{}     `json:"id,omitempty"` // string | int | null
}

type RPCResponse struct {
    JSONRPC string      `json:"jsonrpc"`
    Result  interface{} `json:"result,omitempty"`
    Error   *RPCError   `json:"error,omitempty"`
    ID      interface{} `json:"id"`
}

type RPCError struct {
    Code    int         `json:"code"`
    Message string      `json:"message"`
    Data    interface{} `json:"data,omitempty"`
}

// Standard error codes
const (
    ErrParse          = -32700
    ErrInvalidRequest = -32600
    ErrMethodNotFound = -32601
    ErrInvalidParams  = -32602
    ErrInternal       = -32603
    // Custom
    ErrPermission     = 1001
    ErrNotFound       = 1002
    ErrTimeout        = 1003
)
```

**Test cases:**
- [x] `RPCRequest` serializes/deserializes correctly
- [x] `RPCResponse` with `result` has no `error` field
- [x] `RPCResponse` with `error` has no `result` field
- [x] Batch: `[]RPCRequest` parses correctly
- [x] ID can be string, int, or null

**Acceptance:** Full round-trip serialization tests pass.

---

## Milestone 00 — Foundation Complete

- [x] `pkg/common/flags.go` — POSIX flag parser with 100% coverage
- [x] `pkg/common/output.go` — JSON envelope with schema documented
- [x] `pkg/common/jsonrpc.go` — JSON-RPC 2.0 types with round-trip tests
- [x] All three packages have zero external dependencies
- [x] `go vet ./pkg/common/...` passes clean

## How to Verify

```bash
CGO_ENABLED=0 go test -v -cover ./pkg/common/...
# Expected: PASS, coverage > 95%
```
