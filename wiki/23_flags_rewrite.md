# Phase 23 — Zero-Allocation POSIX Flag Scanner Rewrite

> **Status:** PLANNING | **Date:** 2026-05-19 | **Branch:** `feat/flags-rewrite`
>
> Move the flag parser out of `pkg/common/flags.go` into an internal library at
> `internal/getopt/`, rewrite internals for zero-allocation byte-level scanning
> (**28 allocs → 2 allocs, 2,633ns → ~400ns**), and re-export via type aliases
> in `pkg/common/` so all 77 utilities compile unchanged.

---

## 1. Current State

| Scenario | ns/op | B/op | allocs/op |
|----------|------:|------:|----------:|
| Typical (10 flags + 2 values) | 2,633 | 4,056 | **28** |
| Grouped short (`-laR` + 2 positional) | 607 | 732 | **12** |
| Long-only (4 flags + 1 value) | 888 | 1,360 | **12** |

Allocs come from 6 heap-allocated `map[string]...` per call (2 for def lookups, 4 in ParseResult).

---

## 2. Location

Parser moves from `pkg/common/flags.go` into `internal/getopt/`:

```
internal/getopt/
├── spec.go        ← FlagDef, FlagSpec, FlagType, FlagError, ParseResult types
├── compiled.go    ← CompiledSpec + byte-indexed lookup tables
├── flags.go       ← scanner loop + ParseFlags + accessor methods
└── flags_test.go  ← unit + benchmark + fuzz tests

pkg/common/flags.go    ← REPLACED: type aliases re-exporting internal/getopt
pkg/common/flags_test.go ← DELETED (tests live in internal/getopt/)
```

`pkg/common/flags.go` becomes a ~15-line re-export shim:

```go
package common

import "github.com/ramayac/goposix/internal/getopt"

type FlagDef       = getopt.FlagDef
type FlagSpec      = getopt.FlagSpec
type FlagType      = getopt.FlagType
type ParseResult   = getopt.ParseResult
type FlagError     = getopt.FlagError

var ParseFlags     = getopt.ParseFlags
```

---

## 3. Architecture

### 3.1 Pre-Compiled Flag Tables (`compiled.go`)

Each `FlagSpec` is compiled once at init time into a `CompiledSpec` with byte-indexed lookups:

```go
type CompiledSpec struct {
    ShortLookup [128]uint8       // short flag byte → def index (0xFF = none)
    LongLookup  []LongEntry      // {name string, defIndex uint8} — linear scan (≤25 entries)
    BoolNames   map[string]uint8 // name → bit position (built once, read-only)
    ValueNames  map[string]uint8 // name → value slot index (built once, read-only)
    Defs        []FlagDef
    NumBools    uint8
    NumValues   uint8
    StopAtFirst bool             // for echo/printf: stop at first non-flag arg
}
```

### 3.2 Zero-Allocation Scanner Loop (`flags.go`)

Byte-level dispatch, no `strings.HasPrefix`, no `strings.IndexByte`, no map writes:

```
for each arg in args:
   switch arg[0]:
     case '-':
       if len(arg) == 1           → Stdin marker
       if arg[1] == '-'            → long flag: scan for '=' byte inline
       else                        → short cluster: iterate bytes, ShortLookup[byte]
     default                       → positional argument
```

Bool flags → set bit in `BoolMask uint64`. Value flags → write into flat `ValuesPool [32]string`.

### 3.3 Arena-Style Result Struct

Replace 4 maps with fixed-size inline storage:

```go
type ParseResult struct {
    BoolMask     uint64
    RepeatCount  [64]uint8
    ValuesPool   [32]string                                 // flat pool, no per-flag allocs
    ValueWindows [16]struct{ start, count uint8 }
    Positional   []string
    Stdin        bool
    spec         *CompiledSpec
}
```

Accessors unchanged — `Has(name)` does `spec.BoolNames[name] → bit test`, `Get(name)` does `spec.ValueNames[name] → window[last]`. No map lookups in the query path either.

---

## 4. Implementation Steps

### Step 1: Create `internal/getopt/`

- [ ] `spec.go` — copy `FlagDef`, `FlagSpec`, `FlagType`, `FlagError`, `ParseResult` from `pkg/common/flags.go`
- [ ] Add `FlagSpec.Compile() *CompiledSpec` method
- [ ] Add `StopAtFirstNonFlag` field to `FlagSpec`

### Step 2: `compiled.go`

- [ ] `CompiledSpec` struct
- [ ] `FlagSpec.Compile()`: populate `ShortLookup[128]uint8`, `LongLookup`, `BoolNames`, `ValueNames`
- [ ] Validate no duplicate names, bool count ≤ 64, value count ≤ 16

### Step 3: `flags.go` — Rewrite scanner

- [ ] `ParseFlags(args []string, spec FlagSpec) (*ParseResult, error)`
- [ ] Inner loop: byte dispatch, no string functions, no maps
- [ ] Error path: `FlagError` with `fmt.Sprintf` (allocs in error path are fine)
- [ ] `Has()`, `Get()`, `GetAll()` accessors on `ParseResult`

### Step 4: `flags_test.go`

- [ ] Port all existing tests from `pkg/common/flags_test.go`
- [ ] Add benchmarks: Typical, Grouped, LongOnly, AllBool, ManyValues
- [ ] Add fuzz test: random args vs old parser output byte-for-byte

### Step 5: Wire GoPOSIX

- [ ] Replace `pkg/common/flags.go` with type-alias re-exports
- [ ] Delete `pkg/common/flags_test.go`
- [ ] Add `StopAtFirstNonFlag: true` to echo/printf specs, remove manual flag loops

### Step 6: Validate

- [ ] `make test` — all unit tests pass
- [ ] `make testsuite` — BusyBox 548+ passed, no regressions
- [ ] `make ci` — coverage ≥ 70%, Docker builds succeed

---

## 5. Target Metrics

| Metric | Before | After |
|--------|:------:|:-----:|
| Allocs (typical, 10 flags) | 28 | **2** |
| Allocs (simple, 3 bools) | 12 | **0–1** |
| ns/op (typical) | 2,633 | **~400** |
| ns/op (grouped) | 607 | **~150** |
| Lines in parser | 223 | **~350** |
| Files touched | — | 5 |
| Utilities changed | — | 0 (echo/printf are refactors, not API changes) |

---

## 6. Risks

| Risk | Mitigation |
|------|-----------|
| Bitmask overflow (>64 bools) | `grep` has 19. Validate at compile time. |
| ValuesPool overflow (>32 values) | `grep -e` with many patterns is worst case. Validate. |
| Fuzz mismatch | Run fuzz 30s+ before merge, compare byte-for-byte against old parser. |
| BusyBox regression | `make testsuite` gates every commit. |
