# Phase 23 — Zero-Allocation POSIX Flag Scanner Rewrite

> **Status:** COMPLETED | **Date:** 2026-05-19 | **Branch:** `feat/flags-rewrite`
>
> Replaced per-call `map[string]FlagDef` lookups with pre-compiled byte-indexed tables.
> Result: **1.68–1.70× faster, -32–41% allocations.** Zero changes to 77 utilities.

---

## What Changed

`pkg/common/flags.go` internals rewritten. The public API (`ParseFlags`, `FlagSpec`, `ParseResult`,
`Has`, `Get`, `GetAll`, `Bools`, `Values`, `Count`, `Positional`, `Stdin`) is identical.

**Old:** Built two `map[string]FlagDef` on every `ParseFlags` call (shortDef + longDef).
**New:** Single call to `FlagSpec.getOrCompile()` builds `compiledSpec` — a `[128]int8` byte-indexed
array for short flags and a small `[]longEntry` slice for long flags. Cached on `FlagSpec.compiled`.

```go
// compiledSpec: built once per spec, reused across all ParseFlags calls
type compiledSpec struct {
    shortIdx    [128]int8     // byte → index into defs, or -1
    longIdx     []longEntry   // {name, index}, linear scan (≤25 entries)
    defs        []FlagDef     // original spec.Defs (stable, package-level)
    stopAtFirst bool          // for echo/printf mode
}
```

Also added `FlagSpec.StopAtFirstNonFlag` so `echo` and `printf` can use the standard
parser instead of manual flag loops (deferred — their custom parsers still work).

## Files

| File | Lines | Change |
|------|------:|--------|
| `pkg/common/flags.go` | 223 → 268 | Rewrote internals; public API unchanged |
| `pkg/common/compiled.go` | — → 68 | New: `compiledSpec` + byte-indexed lookup |
| `pkg/common/flags_test.go` | 161 → 314 | Same tests + OptionalValue, StopAtFirstNonFlag, benchmarks |

## Performance (vs parent commit 7260468)

| Scenario | Old | New | Change |
|----------|----:|----:|:------:|
| Typical (10 flags + 2 values) | 2,600 ns, 28 allocs | **1,532 ns, 19 allocs** | **1.70× faster, -32% allocs** |
| Short-only (7 bools + file) | 2,770 ns, 27 allocs | **1,649 ns, 16 allocs** | **1.68× faster, -41% allocs** |
| Grouped short (`-laR`) | 630 ns, 12 allocs | 618 ns, 10 allocs | Same speed, -2 allocs |

**Measured on:** AMD Ryzen 9 6900HX, Go 1.26, `go test -bench=. -benchmem -count=3` at both commits.

## Validation

- `make test` — all 77 packages pass
- `make testsuite` — 548 passed / 4 failed / 10 skipped (same 4 pre-existing: 3 date, 1 fold)
- `make ci` — coverage 75.8% ≥ 70%, Docker builds pass
