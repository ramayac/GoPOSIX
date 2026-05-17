# Phase 15 — Post-MVP Tier 1: `dd` & `od`

> **Status:** PLANNING | **Date:** 2026-05-16 | **Branch:** `feat/post-mvp`
>
> **Parent:** [todos.md](todos.md) — "Road to 99%" gap list
>
> Two high-value I/O utilities with BusyBox test coverage. Both are core data
> inspection/transformation tools heavily used in pipelines and forensic workflows.

---

## Current State

| Utility | BusyBox Tests | Complexity | Est. LOC |
|---------|---------------|------------|----------|
| `dd`    | 6 old-style   | Low-Medium | ~350 |
| `od`    | 5 new-style   | Low        | ~250 |

Neither exists in `pkg/` or `cmd/goposix/main.go` today.

---

## 15.1 — `dd`

### BusyBox Test Inventory (6 old-style, `test/busybox_testsuite/dd/`)

| # | Test | What it checks |
|---|------|----------------|
| 1 | `dd-accepts-if` | `if=file` input flag |
| 2 | `dd-accepts-of` | `of=file` output flag |
| 3 | `dd-copies-from-standard-input-to-standard-output` | Default stdin→stdout copy |
| 4 | `dd-count-bytes` | `count=N iflag=count_bytes` byte-level truncation |
| 5 | `dd-prints-count-to-standard-error` | Status line to stderr |
| 6 | `dd-reports-write-errors` | Write-failure exit code |

### POSIX Flag Spec

```
dd [if=file] [of=file] [ibs=N] [obs=N] [bs=N] [count=N] [skip=N] [seek=N]
   [conv=notrunc,noerror,sync,fsync] [iflag=flag[,flag...]] [oflag=flag[,flag...]]
   [status=none|noxfer]
```

Core operands to support: `if`, `of`, `bs`, `ibs`, `obs`, `count`, `skip`, `seek`, `conv=notrunc,noerror,sync`, `status=none,noxfer`, `iflag=count_bytes,fullblock`.

### CHECK → TEST → CODE → PASS (sequential, one at a time)

#### Step 1 — CHECK: Read all 6 BusyBox tests
```bash
cat test/busybox_testsuite/dd/*
```
Understand exact expected behavior for each test.

#### Step 2 — TEST: Write Go unit tests in `pkg/dd/dd_test.go`
- `TestDd_StdinToStdout` — byte-for-byte pipe copy
- `TestDd_IfOf` — named file in/out
- `TestDd_CountBytes` — truncation at byte N
- `TestDd_CountBlocks` — `count=N` with default block size
- `TestDd_SkipSeek` — offset-based I/O
- `TestDd_ConvNotrunc` — `oflag=notrunc`
- `TestDd_StatusNone` — suppress stderr status line

#### Step 3 — CODE: Implement `pkg/dd/dd.go`

**Signature (library layer):**
```go
func Run(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error
```

**Operand parser:** Custom key=value parser (`if=`, `of=`, `bs=`, etc.) because these
are not standard flags. POSIX specifies `dd` uses `operand=value` syntax.

**Core loop:**
```
while count > 0 and input not exhausted:
    read ibs bytes from input
    apply conv transformations
    write obs bytes to output
    count--
```
Default block size: 512 bytes.

#### Step 4 — PASS: Verify against BusyBox
```bash
make testsuite  # confirm all 6 dd tests pass
```

### Registration Checklist
- [ ] `pkg/dd/dd.go` with `init()` → `dispatch.Register`
- [ ] `pkg/dd/dd_test.go`
- [ ] Add `_ "github.com/ramayac/goposix/pkg/dd"` to `cmd/goposix/main.go`
- [ ] Add `./pkg/dd/...` to `PKG_DIRS` in `Makefile`
- [ ] Run `make vet test build` → clean
- [ ] Run `make testsuite` → 0 new failures
- [ ] Update this doc status

---

## 15.2 — `od`

### BusyBox Test Inventory (5 new-style, `test/busybox_testsuite/od.tests`)

| # | Test | What it checks |
|---|------|----------------|
| 1 | `od` | Default octal dump |
| 2 | `od -b` | Octal byte dump (`-b` flag) |
| 3 | `od -c` | Character dump (`-c` flag) |
| 4 | `od -x` | Hex dump (`-x` flag) |
| 5 | `od -N` | Limit bytes (`-N count` flag) |

### POSIX Flag Spec

```
od [-A address_base] [-j skip] [-N count] [-t type_string] [file...]
```

Type specifiers: `a` (named char), `c` (char), `d` (signed decimal), `f` (float),
`o` (octal), `u` (unsigned decimal), `x` (hex). Size suffixes: `C` (char), `S` (short),
`I` (int), `L` (long).

Core flags needed for BusyBox: `-b`, `-c`, `-x`, `-N N`.

### CHECK → TEST → CODE → PASS

#### Step 1 — CHECK: Read `test/busybox_testsuite/od.tests`
```bash
cat test/busybox_testsuite/od.tests
```

#### Step 2 — TEST: Write `pkg/od/od_test.go`
- `TestOd_Default` — default format (octal 2-byte shorts)
- `TestOd_OctalBytes` — `-b` flag
- `TestOd_Char` — `-c` flag
- `TestOd_Hex` — `-x` flag
- `TestOd_Count` — `-N N` truncation
- `TestOd_FromStdin` — pipe input
- `TestOd_Json` — `--json` structured output

#### Step 3 — CODE: Implement `pkg/od/od.go`

**Signature (library layer):**
```go
type OdResult struct {
    Records []string `json:"records"`
}

func Run(args []string, in io.Reader) (*OdResult, error)
```

Core logic:
- Read input in fixed-size blocks (default 16 bytes)
- Format each block according to type specifier
- Offset display in octal
- `-A` controls address radix (default octal)

#### Step 4 — PASS
```bash
make testsuite  # confirm all 5 od tests pass
```

### Registration Checklist
- [ ] `pkg/od/od.go` with `init()` → `dispatch.Register`
- [ ] `pkg/od/od_test.go`
- [ ] Add import to `cmd/goposix/main.go`
- [ ] Add to `PKG_DIRS` in `Makefile`
- [ ] `make vet test build` clean
- [ ] `make testsuite` → 0 new failures
- [ ] Update this doc

---

## Milestone 15

```
[ ] 15.1 — dd implemented, 6/6 BusyBox tests pass
[ ] 15.2 — od implemented, 5/5 BusyBox tests pass
```

**Combined BusyBox pass count increase: +11 (477 → 488)**

---

## How to Verify

```bash
# dd
echo "hello world" | ./goposix dd bs=5 count=2
./goposix dd if=/dev/zero of=/tmp/test.dd bs=1024 count=10
make testsuite  # 6 dd tests pass

# od
echo -n "hello" | ./goposix od -x
echo -n "hello" | ./goposix od --json
make testsuite  # 5 od tests pass
```
