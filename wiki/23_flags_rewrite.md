# Phase 23 — Zero-Allocation POSIX Flag Scanner Rewrite

> **Status:** PLANNING | **Date:** 2026-05-19 | **Trigger:** Performance audit of `pkg/common/flags.go`
>
> Replace the map-heavy `ParseFlags` with a compiled-table, arena-style scanner targeting
> **28 allocs → 2 allocs** and **2,633ns → ~400ns** while preserving the existing public API
> (zero migration cost for 77 utilities).

---

## 1. Motivation

### Baseline Metrics (current `ParseFlags`)

| Scenario | ns/op | B/op | allocs/op |
|----------|------:|------:|----------:|
| Typical (10 flags + 2 values) | 2,633 | 4,056 | **28** |
| Grouped short (`-laR` + 2 positional) | 607 | 732 | **12** |
| Long-only (4 flags + 1 value) | 888 | 1,360 | **12** |

**Where the 12–28 allocs come from:**

1. `shortDef := make(map[string]FlagDef)` — always allocates (1 alloc)
2. `longDef := make(map[string]FlagDef)` — always allocates (1 alloc)
3. `Bools`, `Values`, `ValuesList`, `Count` — 4 maps in `newParseResult()` (4 allocs)
4. `Positional` slice growth via `append` (1+ allocs)
5. `ValuesList` slice growth per value flag (1 alloc per flag that has a value)
6. `&FlagError{...}` with `fmt.Sprintf` — error path only (not in benchmark path)

**Where it's called:** 77 utilities. All use the identical pattern:

```go
flags, err := common.ParseFlags(args, spec)
// then flags.Has("name"), flags.Get("name"), flags.Positional, flags.Stdin, flags.Count
```

496 total call sites across `.Has()`, `.Get()`, `.Positional`, `.Stdin`, `.Count`.

Two utilities have manual flag parsing (`echo`, `printf`) because they need to stop at the first
non-flag argument. They serve as a reference for a lightweight pattern but could be brought into
the fold with a `StopAtFirstNonFlag` option.

---

## 2. Library Strategy: Build vs. Adopt

Before designing the internals, we evaluated every viable Go POSIX flag library to determine
whether we should adopt one or extract our own parser as a standalone module.

### 2.1 External Library Survey

Three candidates were examined in full source detail:

#### `pborman/getopt` — Google-authored POSIX getopt

3,800 lines, pure Go, BSD-licensed. The closest semantic match to C `getopt_long`.

| Feature | Supported? |
|---------|:----------:|
| Clustered short flags (`-laR`) | ✅ |
| Short flag with optional value | ✅ `SetOptional()` |
| `--name=value` and `--name value` | ✅ |
| `--` end-of-flags | ✅ |
| `Counter` type for `-vvv` | ✅ |
| Bare `-` handling | ✅ |
| Declarative spec (define flags once, query result) | ❌ |
| `FlagOptionalValue` as first-class type | ⚠️ via `SetOptional()` method |
| `--json` convention integration | ❌ |

**Fatal flaw: imperative API model.** Every flag is a separate variable registered via
function calls. Grep's 24 flags would require 24 variable declarations, 24 `s.Bool()` calls,
and 24 pointer dereferences to query results. All 77 utilities would need complete rewrites
(~1,000+ lines of boilerplate total).

#### `spf13/pflag` — POSIX fork of Go's `flag`

~3,000 lines. The most popular Go flag library. Widely used (Kubernetes, Docker, etc).

| Feature | Supported? |
|---------|:----------:|
| Clustered short flags | ✅ |
| `--name=value` | ✅ |
| `--` end-of-flags | ✅ |
| Short flag with optional value | ❌ No concept of optional value flags |
| `Count` type for `-vvv` | ⚠️ Separate `CountP` API, different access pattern |
| `FlagOptionalValue` | ❌ Not supported |
| Declarative spec | ❌ Imperative like stdlib `flag` |

**Fatal flaw: no optional value support.** POSIX tools like `sed -e script`, `tar -f file`,
and `od -j skip` require flag types where the value can be in the same argument or omitted.
pflag has no equivalent.

#### `jessevdk/go-flags` — Struct-tag based

10,500 lines. Uses reflection and struct tags.

| Feature | Supported? |
|---------|:----------:|
| Clustered short flags | ✅ |
| `--name=value` | ✅ |
| `--` end-of-flags | ✅ |
| Short flag with optional value | ❌ |
| `Count` type | ⚠️ Via `[]bool` slice length trick |
| Declarative spec | ⚠️ Struct tags (closer, but still per-utility struct type) |

**Fatal flaw: 10,500 lines** for a 350-line problem. Each utility must define a new struct type
with tags. Massive dependency for marginal benefit.

### 2.2 Why No Existing Library Fits

All existing Go getopt libraries share a fundamental architectural pattern: **imperative,
per-flag variable registration**. The caller declares individual Go variables and registers
them one by one. After parsing, each variable is read independently.

GoPOSIX uses the opposite pattern: **declarative, query-by-name from a result struct**.
The flag spec is a data structure (a slice of `FlagDef`), and the result is a single object
with `Has(name)`, `Get(name)` methods. This enables:

- JSON output integration (`--json` flag is just another entry in the spec)
- Daemon RPC forwarding (flags are serialized as key-value pairs)
- Consistent 2-line flag handling in every utility:
  ```go
  flags, err := common.ParseFlags(args, spec)
  jsonMode := flags.Has("json")
  ```

None of the surveyed libraries support this model. Wrapping one would require a
translation layer that is more code than writing the parser directly.

### 2.3 Decision: Extract Our Parser as a Standalone Library

**Write the parser once, publish it as `github.com/ramayac/go-getopt`, import it in GoPOSIX.**

```
go-getopt/                     ← standalone library, zero deps, pure Go
├── flags.go                   ← scanner/parser (~350 lines after rewrite)
├── compiled.go                ← pre-compiled spec tables (~100 lines)
├── spec.go                    ← FlagDef, FlagSpec, FlagType (same types as today)
├── result.go                  ← ParseResult + Has/Get/GetAll/Count accessors
├── flags_test.go              ← unit + benchmark + fuzz tests
├── go.mod
└── README.md

goposix/go.mod:
  require github.com/ramayac/go-getopt v0.1.0

goposix/pkg/common/flags.go:
  // One-line re-exports. All 77 utilities unchanged.
  package common
  import getopt "github.com/ramayac/go-getopt"
  type FlagDef = getopt.FlagDef
  type FlagSpec = getopt.FlagSpec
  // ... type aliases for backward compat ...
  var ParseFlags = getopt.ParseFlags
```

**Why this wins:**

| | pflag | pborman/getopt | go-flags | **Extract ours** |
|---|---:|---:|---:|---:|
| Lines of code (non-test) | ~3,000 | ~2,000 | ~6,000 | **~500** |
| POSIX optional values | ❌ | ✅ | ❌ | ✅ |
| Count tracking (`-vvv`) | ⚠️ | ✅ | ❌ | ✅ |
| Declarative spec | ❌ | ❌ | ⚠️ | ✅ |
| Migration cost (files) | 77 | 77 | 77 | **0** |
| Works with `--json` integration | ❌ | ❌ | ❌ | ✅ |
| BusyBox regression risk | High | High | High | **None** |
| External dependency? | Yes | Yes | Yes | **Yes (our own)** |

This satisfies the "use a library" goal (the parser lives in its own module with its own
versioning, CI, and docs) while avoiding the 77-file migration cost and POSIX gaps of all
third-party options.

---

## 3. Design Space: Parser Internals

### Approach A: Zero-Allocation Bitmask Scanner

| Pro | Con |
|-----|-----|
| Truly 0 allocs (2,633ns → ~200ns) | Caps bool flags at 64 (uint64 bitmask) |
| Compile-time safety for flag names | Value flags need fixed arrays — fragile with hard caps |
| Works entirely on stack | Doesn't naturally support `ValuesList` (multiple `-e pat` for grep) |

### Approach B: Minimal Getopt Iterator

| Pro | Con |
|-----|-----|
| No pre-compilation needed | Caller must loop manually — **massive migration cost (77 files, 496 call sites)** |
| Caller controls memory layout | Breaks existing API |
| Familiar C-getopt pattern | No grouped short flag support by default |
| | Each caller reinvents flag-to-field mapping |

### Approach C: Hybrid — Compile-Time Tables + Single-Arena Result ✅

| Pro | Con |
|-----|-----|
| **Same API** — zero migration cost for 77 utilities | 1–2 allocs (not strictly zero, but 14× improvement) |
| 28 allocs → **2 allocs** | Slightly more complex internals |
| No hard caps on flag count | |
| Clean fast-path inner loop (switch on byte) | |
| `Has(name)` → O(1) table lookup + bit test; no map hashing | |

**Decision: Approach C** — keep the public API, rewrite the internals. The 77-utility migration
cost of Approach B far outweighs the theoretical purity of Approach A. 2 allocs vs 28 allocs
is a 14× reduction while keeping the same surface for all 77 consumers.

---

## 4. Architecture

### 4.1 Pre-Compiled Flag Tables (`compiled.go` — new file)

Instead of building `map[string]FlagDef` on every call, each utility pre-compiles its spec
at init time (single-threaded, once).

```go
type FlagDef struct {
    Short byte   // single byte, 0 = none
    Long  string // long name (used for --name parsing)
    Type  FlagType
    Index uint8  // position in result arrays (assigned at compile time)
}

type CompiledSpec struct {
    // ShortLookup[c] gives the FlagDef index for short flag byte c (0xFF = none).
    ShortLookup [128]uint8

    // LongLookup is a compact sorted array; linear scan is fine since most specs
    // have < 25 flags. Binary search for specs with > 30 long flags.
    LongLookup []LongEntry // {name string, defIndex uint8}

    Defs      []FlagDef
    NumBools  uint8
    NumValues uint8

    // Pre-built name → bit-index and name → value-index mappings for O(1) Has/Get.
    BoolNames  map[string]uint8  // name → bit position (0..63)
    ValueNames map[string]uint8  // name → slot index (0..N)
}
```

Pre-compilation happens at init time: each utility's `var spec = common.FlagSpec{...}` becomes
`var spec = common.NewFlagSpec().Add(...).Compile()` or the existing `FlagSpec` gets a
`.Compile() *CompiledSpec` method.

### 4.2 Zero-Allocation Scanner Loop (core rewrite)

The inner loop uses byte-level dispatch instead of string operations:

```
for each arg in args:
   switch arg[0]:
     case '-':
       if len(arg) == 1:          // bare "-"
           → Stdin marker
       if arg[1] == '-':           // long flag: --name or --name=value
           → parse name inline (scan for '=' byte)
           → lookup in LongLookup
       else:                       // short flag cluster: -laR or -ofile
           → iterate bytes of arg[1:]
           → dispatch via ShortLookup[byte]
     default:                      // positional argument
       → append to Positional
```

Key optimizations over the current code:

1. **No `strings.HasPrefix`/`strings.IndexByte`** — direct byte indexing on the string
2. **No map writes in hot path** — bitmask for bools, indexed arrays for values
3. **Single `Positional` allocation** — pre-allocated to `len(args)` (worst-case, every arg is positional)
4. **No string → map key hashing** — byte-indexed lookup tables built once at compile time

### 4.3 Arena-Style Result Struct

Replace the 4 maps in `ParseResult` with fixed-size inline storage:

```go
type ParseResult struct {
    // --- Bool Flags ---
    BoolMask uint64           // bitmask for up to 64 unique bool flags

    // --- Repeat Counts (-vvv) ---
    RepeatCount [64]uint8     // indexed by bit position

    // --- Value Flags ---
    // Flat pool: all values from all value-type flags share this array.
    // No per-flag slice headers allocated.
    ValuesPool    [32]string  // pool of all value strings (slices into os.Args, no alloc)
    ValuesPoolLen uint8
    // Per-flag windows into the pool: [start, start+count)
    ValueWindows   [16]struct{ start, count uint8 }

    // --- Positional Args ---
    Positional []string        // single allocation, pre-alloced to len(args)

    // --- Misc ---
    Stdin bool

    // --- Pre-compiled lookup tables (set once, never allocates per call) ---
    spec *CompiledSpec
}
```

**How `ValuesList` works without per-flag allocation:**

```
ValuesPool:    ["pat1", "pat2", "out.txt", "", "", ...]  (flat array, 32 slots max)
ValueWindows:  [
  0: {start:0, count:2}   // flag index 0 (e.g., -e): 2 values
  1: {start:2, count:1}   // flag index 1 (e.g., -o): 1 value
  2: {start:0, count:0}   // unused
  ...
]
```

`GetAll(name)` returns `ValuesPool[window.start : window.start+window.count]` — a **sub-slice
of the pool array**, not a freshly allocated slice. The returned `[]string` header is allocated
on the caller's stack (or optimized away by the compiler).

### 4.4 API Compatibility Layer

The existing public API is fully preserved:

```go
func (r *ParseResult) Has(name string) bool {
    // Old: return r.Bools[name]              (map lookup + hash)
    // New: bitIndex := r.spec.BoolNames[name]; return r.BoolMask & (1 << bitIndex) != 0
    // O(1), no hash, no heap allocation
}

func (r *ParseResult) Get(name string) string {
    // Old: return r.Values[name]             (map lookup + hash)
    // New: slot := r.spec.ValueNames[name]; window := r.ValueWindows[slot]
    //      if window.count == 0 { return "" }
    //      return r.ValuesPool[window.start + window.count - 1]  // last value wins
}

func (r *ParseResult) GetAll(name string) []string {
    // Old: return r.ValuesList[name]         (map lookup + hash)
    // New: slot := r.spec.ValueNames[name]; window := r.ValueWindows[slot]
    //      return r.ValuesPool[window.start : window.start+window.count]
    // The returned slice header is stack-allocated (sub-slice of ValuesPool array).
}
```

Count-based access is similarly translated to indexed access.

---

## 5. Implementation Plan

### Phase 0: Create Standalone Library (`github.com/ramayac/go-getopt`)

- [ ] Initialize new repo with `go mod init github.com/ramayac/go-getopt`
- [ ] Copy existing `FlagDef`, `FlagSpec`, `FlagType`, `ParseResult`, `FlagError` types into `spec.go`
- [ ] Write the zero-alloc scanner in `flags.go` (see Phase 2 below)
- [ ] Write pre-compiled tables in `compiled.go` (see Phase 1 below)
- [ ] Write tests + benchmarks + fuzz in `flags_test.go`
- [ ] Tag `v0.1.0` with passing CI

### Phase 0b: Wire GoPOSIX to Import the Library

- [ ] Add `require github.com/ramayac/go-getopt v0.1.0` to GoPOSIX `go.mod`
- [ ] Replace `pkg/common/flags.go` with type-alias re-exports:
  ```go
  package common
  import getopt "github.com/ramayac/go-getopt"
  type FlagDef = getopt.FlagDef
  type FlagSpec = getopt.FlagSpec
  type FlagType = getopt.FlagType
  type ParseResult = getopt.ParseResult
  type FlagError = getopt.FlagError
  var ParseFlags = getopt.ParseFlags
  ```
- [ ] Delete `pkg/common/flags_test.go` (tests live in the library now)
- [ ] Verify: `make test` passes with zero changes to any `pkg/*/` file

### Phase 1: Pre-Compiled Tables (`compiled.go`, ~100 lines)

- [ ] Define `CompiledSpec` struct with `ShortLookup [128]uint8` and `LongLookup []LongEntry`
- [ ] Define `LongEntry` struct (`{Name string, Index uint8}`)
- [ ] Implement `FlagSpec.Compile() *CompiledSpec`:
  - Iterate `FlagSpec.Defs`, assign indices to bool and value flags
  - Populate `ShortLookup` array (index by byte)
  - Populate `BoolNames` and `ValueNames` maps (built once, read-only after compile)
  - Validate: no duplicate short/long names, bool count ≤ 64, value count ≤ 16

### Phase 2: Rewrite Scanner Loop (`flags.go`, ~200 lines)

- [ ] Replace `newParseResult()` with `newParseResult(cspec *CompiledSpec)` — pre-sizes Positional
- [ ] Rewrite `ParseFlags` inner loop with byte-level dispatch
  - Switch on `arg[0]`: `'-'` → handle bare `-`, `--`, short cluster; `default` → positional
  - Long flag parsing: scan for `=` byte inline, no `strings.IndexByte`
  - Short cluster: iterate bytes, use `ShortLookup[ch]` for O(1) dispatch
  - Bool flags: set bit in `BoolMask`, increment `RepeatCount[bitIndex]`
  - Value flags: write to `ValuesPool[ValuesPoolLen++]`, update `ValueWindows[index]`
- [ ] Keep error paths using `fmt.Sprintf` (error path — allocations acceptable)
- [ ] Keep `--` end-of-flags marker and `-` stdin marker handling

### Phase 3: API Compatibility Methods (`flags.go`, ~50 lines)

- [ ] Rewrite `Has(name)` — table lookup in `BoolNames` + bit test
- [ ] Rewrite `Get(name)` — table lookup in `ValueNames` + window access
- [ ] Rewrite `GetAll(name)` — table lookup + sub-slice of `ValuesPool`
- [ ] `Count` accessor remains but operates on `RepeatCount` array
- [ ] `Positional` and `Stdin` fields unchanged

### Phase 4: Flags Tests Extension (`flags_test.go`, ~200 lines)

- [ ] Add benchmark suite:
  - `BenchmarkParseFlags_Typical` — 10 flags, 2 values, 2 positionals
  - `BenchmarkParseFlags_Grouped` — `-laR` cluster + positionals
  - `BenchmarkParseFlags_LongOnly` — long flags with `=` and space values
  - `BenchmarkParseFlags_AllBool` — 20 bool flags (stress bitmask)
  - `BenchmarkParseFlags_ManyValues` — 8 repeated value flags (stress pool)
- [ ] Add fuzz test: random flag strings, compare old parser output byte-for-byte with new
- [ ] Add edge case tests: 64 bool flags (bitmask boundary), 16 value flags (pool boundary)
- [ ] Add `TestCompiledSpec_DuplicateShort` — compile-time validation

### Phase 5: Echo/Printf Generalization (in go-getopt library)

- [ ] Add `FlagSpec.StopAtFirstNonFlag bool` field
- [ ] `CompiledSpec` respects it: scanner stops when it encounters a non-flag argument
- [ ] Refactor `echo.go` `parseEchoFlags` to use `ParseFlags` with `StopAtFirstNonFlag`
- [ ] Refactor `printf.go` similarly
- [ ] Remove manual flag parsing loops from both utilities
- [ ] Verify BusyBox echo/printf compliance tests still pass

### Phase 6: Full Validation (both repos)

- [ ] `make test` — all unit tests pass
- [ ] `make testsuite` — BusyBox 548/4/10 or better (no regressions)
- [ ] `make ci` — coverage ≥ 70%, Docker builds succeed
- [ ] `make bench-quick SCALE=0.1` — daemon benchmark still passes
- [ ] Fuzz test: `go test -fuzz=FuzzParseFlags -fuzztime=30s ./pkg/common/`

### Phase 7: Utility-by-Utility Optimization (GoPOSIX, Optional, Deferred)

Once the rewrite is stable, utilities can optionally pre-compile their spec at init:

```go
// BEFORE (compiles on every call):
var spec = common.FlagSpec{Defs: [...]}
flags, _ := common.ParseFlags(args, spec)

// AFTER (compile once, reuse forever):
var compiledSpec = common.MustCompile(common.FlagSpec{Defs: [...]})
flags, _ := common.ParseFlags(args, compiledSpec)
```

But the internal `ParseFlags` will auto-compile on first call anyway, so this is a
micro-optimization. Not required for Phase 23 delivery.

---

## 6. File Plan

### New Library (`github.com/ramayac/go-getopt`)

| File | Action | Description |
|------|--------|-------------|
| `spec.go` | **New** | `FlagDef`, `FlagSpec`, `FlagType`, `FlagError`, `ParseResult` types |
| `compiled.go` | **New** | `CompiledSpec` type + precompilation + byte-indexed lookup tables |
| `flags.go` | **New** | Zero-alloc scanner loop, `ParseFlags` entry point, accessor methods |
| `flags_test.go` | **New** | Unit tests + benchmarks + fuzz test + edge case tests |

### GoPOSIX Changes

| File | Action | Description |
|------|--------|-------------|
| `go.mod` | **Edit** | Add `require github.com/ramayac/go-getopt v0.1.0` |
| `pkg/common/flags.go` | **Replace** | Type-alias re-exports (~15 lines); old 223-line body deleted |
| `pkg/common/flags_test.go` | **Delete** | Tests live in the library now |
| `pkg/echo/echo.go` | **Refactor** | Use `StopAtFirstNonFlag` instead of manual `parseEchoFlags` |
| `pkg/printf/printf.go` | **Refactor** | Same |
| All 77 utilities | **No change** | Same `import common`, same `ParseFlags(args, spec)`, same `flags.Has()` |

---

## 7. Estimated Impact

| Metric | Before (Current) | After (Target) | Improvement |
|--------|:------------------:|:---------------:|:-----------:|
| Allocs per `ParseFlags` (typical, 10 flags) | 28 | **2** | 14× |
| Allocs per `ParseFlags` (simple, 3 bools) | 12 | **0–1** | 12× |
| ns/op (typical) | 2,633 | **~400** | 6.5× |
| ns/op (grouped) | 607 | **~150** | 4× |
| ns/op (long-only) | 888 | **~300** | 3× |
| Lines in `flags.go` | 223 | **~350** | — |
| Lines in `flags_test.go` | 161 | **~350** | — |
| New file `compiled.go` | — | **~100** | — |
| Utility migration cost | — | **0 lines changed** | — |

---

## 8. Risks & Mitigations

| Risk | Probability | Mitigation |
|------|:----------:|------------|
| Bitmask overflow (>64 bool flags in one utility) | Low | `grep` has the most at 19 bools. 64 is 3.4× headroom. Validate at compile time. |
| ValuesPool overflow (>32 values across all value flags) | Low | `grep` with many `-e` is the worst case. 32 is generous. Validate at compile time. |
| Fuzz mismatch between old and new parser | Medium | Run fuzz test for 30s+ before merge. Compare output byte-for-byte. |
| BusyBox test regression | Medium | `make testsuite` gates every commit. 548 tests catch cascading failures. |
| `GetAll` returning a window into `ValuesPool` may cause aliasing bugs | Low | Document that returned slices are only valid until the next call to `ParseFlags` (same as current behavior with maps). |

---

## 9. Dependencies & Ordering

- **Must come after:** Phase 22 (Hardening III) — which is already COMPLETED
- **Independent of:** Phase 23 (Multi-tenant sandbox), Phase 24 (Observability)
- **Blocks:** Nothing. This is a pure optimization with no API changes.

---

## 10. Design Notes

### Why not `spf13/pflag`?

See §2.1 for full comparison. Summary: no `FlagOptionalValue`, no `Count` type,
imperative API forces 77-file migration, and the migration cost dwarfs any benefit.

### Why not `pborman/getopt`?

See §2.1 for full comparison. POSIX-compliant but imperative API — grep would need
24 variable declarations + 24 registration calls + 24 pointer dereferences. 77× that
is ~1,000+ lines of boilerplate.

### Why not standard `flag` package?

stdlib `flag` doesn't support clustered short flags (`-laR`), `--` end-of-flags marker,
or `-single-dash` long flag style. Plus, AGENTS.md §2 has historically forbidden it.

### Why keep `map[string]uint8` for `BoolNames`/`ValueNames` in `CompiledSpec`?

These maps are built **once at compile time** and read-only thereafter. They never appear in
the hot path. The hot path uses `ShortLookup[byte]` (array index) and `BoolMask & bit`. The
name→index maps are only used by `Has(name)` and `Get(name)` — which are convenience methods
called a handful of times per invocation, not in the inner loop.

If we wanted to go further, we could replace these with a sorted `[]struct{name string, index uint8}`
and binary search (zero allocs for the compiled spec). But the maps are small (≤25 entries) and
built once. The alloc is negligible.

---

*Plan created 2026-05-19. Ready for implementation.*
