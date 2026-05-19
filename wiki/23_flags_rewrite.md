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

## 2. Design Space & Decision

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

## 3. Architecture

### 3.1 Pre-Compiled Flag Tables (`compiled.go` — new file)

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

### 3.2 Zero-Allocation Scanner Loop (core rewrite)

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

### 3.3 Arena-Style Result Struct

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

### 3.4 API Compatibility Layer

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

## 4. Implementation Plan

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

### Phase 5: Echo/Printf Generalization

- [ ] Add `FlagSpec.StopAtFirstNonFlag bool` field
- [ ] `CompiledSpec` respects it: scanner stops when it encounters a non-flag argument
- [ ] Refactor `echo.go` `parseEchoFlags` to use `ParseFlags` with `StopAtFirstNonFlag`
- [ ] Refactor `printf.go` similarly
- [ ] Remove manual flag parsing loops from both utilities
- [ ] Verify BusyBox echo/printf compliance tests still pass

### Phase 6: Full Validation

- [ ] `make test` — all unit tests pass
- [ ] `make testsuite` — BusyBox 548/4/10 or better (no regressions)
- [ ] `make ci` — coverage ≥ 70%, Docker builds succeed
- [ ] `make bench-quick SCALE=0.1` — daemon benchmark still passes
- [ ] Fuzz test: `go test -fuzz=FuzzParseFlags -fuzztime=30s ./pkg/common/`

### Phase 7: Utility-by-Utility Optimization (Optional, Deferred)

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

## 5. File Plan

| File | Action | Description |
|------|--------|-------------|
| `pkg/common/compiled.go` | **New** | `CompiledSpec` type + precompilation + byte-indexed lookup tables |
| `pkg/common/flags.go` | **Rewrite** | Zero-alloc internals, same public API, byte-level scanner loop |
| `pkg/common/flags_test.go` | **Extend** | Benchmarks + fuzz test + edge case tests + bitmask boundary tests |
| `pkg/echo/echo.go` | **Refactor** | Use `StopAtFirstNonFlag` instead of manual `parseEchoFlags` |
| `pkg/printf/printf.go` | **Refactor** | Same |
| All 77 utilities | **No change** | API preserved |

---

## 6. Estimated Impact

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

## 7. Risks & Mitigations

| Risk | Probability | Mitigation |
|------|:----------:|------------|
| Bitmask overflow (>64 bool flags in one utility) | Low | `grep` has the most at 19 bools. 64 is 3.4× headroom. Validate at compile time. |
| ValuesPool overflow (>32 values across all value flags) | Low | `grep` with many `-e` is the worst case. 32 is generous. Validate at compile time. |
| Fuzz mismatch between old and new parser | Medium | Run fuzz test for 30s+ before merge. Compare output byte-for-byte. |
| BusyBox test regression | Medium | `make testsuite` gates every commit. 548 tests catch cascading failures. |
| `GetAll` returning a window into `ValuesPool` may cause aliasing bugs | Low | Document that returned slices are only valid until the next call to `ParseFlags` (same as current behavior with maps). |

---

## 8. Dependencies & Ordering

- **Must come after:** Phase 22 (Hardening III) — which is already COMPLETED
- **Independent of:** Phase 23 (Multi-tenant sandbox), Phase 24 (Observability)
- **Blocks:** Nothing. This is a pure optimization with no API changes.

---

## 9. Design Notes

### Why not `spf13/pflag`?

- Adds an external dependency (violates AGENTS.md "Zero Dependencies" invariant)
- pflag is ~3,000 lines of code — more than the entire current `flags.go`
- pflag doesn't support our `--json` structured output integration
- pflag doesn't support our `FlagOptionalValue` type (`-e[eof-str]`)
- The migration cost (77 files × rewriting flag definitions) dwarfs any benefit

### Why not standard `flag` package?

- AGENTS.md §2 explicitly forbids it: *"Use the custom POSIX-compliant parser in `pkg/common/flags.go`. Do not use the standard library `flag` package."*
- stdlib `flag` doesn't support clustered short flags (`-laR`)
- stdlib `flag` doesn't support `--` end-of-flags marker
- stdlib `flag` uses `-single-dash` for long flags (non-POSIX)

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
