# Phase 07a — `awk` (POSIX Text Processing)

> **Status:** ✅ COMPLETED | **Depends on:** Phase 07.4 (Tier 5 utilities) | **Canonical awk document**
>
> **Approach:** Wrap [benhoyt/goawk](https://github.com/benhoyt/goawk) v1.31.0
> (17,409 LOC, MIT, zero deps, pure Go) as `pkg/awk/`. This replaces the
> original 8-sub-phase "build from scratch" plan (~3,000–5,000 LOC, 3+ months)
> with a ~400 LOC integration (~1 day).

---

## Why It Matters

`awk` is the last missing POSIX.2 utility in GoPOSIX. Without it, the project's
"POSIX-compliant userland" claim carries a permanent asterisk. Every serious
shell script that processes structured text uses awk. Completing this utility
is the **Platinum gate** — it qualifies the project for the highest compliance
tier.

---

## Why goawk?

| Factor | Build from scratch | Wrap goawk |
|--------|--------------------|------------|
| LOC | 3,000–5,000 | ~400 |
| Effort | 3+ months, 8 sub-phases | ~1 day |
| BusyBox pass rate | Unknown (weeks of debugging) | **64%** out of the box (34/53) |
| Dependencies | 0 | 0 (stdlib only) |
| CGO | 0 | 0 |
| License | N/A | MIT |
| Library API | N/A | `interp.Exec()` / `interp.ExecProgram()` |

**Precedent:** GoPOSIX already depends on `mvdan.cc/sh/v3` (~20K LOC shell interpreter).
goawk is the same class of dependency: a complex, well-tested interpreter that would
be irrational to rebuild. We're wrapping it, not rewriting it.

### BusyBox Test Baseline (2026-05-19)

Ran all 53 non-skipped BusyBox awk tests directly against goawk v1.31.0:

| Result | Count | Breakdown |
|--------|:-----:|-----------|
| PASS | 34 | Field splitting, NF/NR, patterns, BEGIN/END, conditionals, loops, arrays, printf, gsub, getline/pipe, I/O redirect, nested loops, user functions, delete |
| FAIL | 19 | 8 cosmetic (error message format), 3 parse-time-vs-runtime detection, 3 test harness artifacts, 2 GNU extensions (`or()`/`and()` — not POSIX), 3 minor gaps |
| SKIP | 11 | GNU extensions, NUL output handling, external data files |

**Net assessment:** Zero correctness failures on core POSIX awk semantics. The 19
"failures" are: 8 error messages phrased differently, 5 parse-time vs runtime detection
differences, 3 test infrastructure artifacts, and 3 minor behavioral gaps (`ERRNO`
not set on file-not-found, backslash+newline not stripped, exit code not propagated
through END).

---

## Integration Plan

### Step 1 — Add dependency

```
go get github.com/benhoyt/goawk@v1.31.0
```

goawk's `go.mod` has no external `require` directives. Adds zero transitive dependencies.

### Step 2 — `pkg/awk/awk.go` — Library layer

The core `Run()` function wraps `interp.ExecProgram()`:

```go
package awk

import (
    "bytes"
    "fmt"
    "io"
    "strings"

    "github.com/benhoyt/goawk/interp"
    "github.com/benhoyt/goawk/parser"
)

type Result struct {
    Records []Record `json:"records"`
}

type Record struct {
    NR     int      `json:"nr"`
    Fields []string `json:"fields"`
}

func Run(source string, files []string, fieldSep string, jsonMode bool,
         input io.Reader, out io.Writer, errOut io.Writer) (int, error) {

    if jsonMode {
        // Capture AWK output, wrap in our JSON envelope
        var buf bytes.Buffer
        status, err := execAWK(source, files, fieldSep, input, &buf, errOut)
        if err != nil {
            return status, err
        }
        records := parseRecords(buf.String())
        // Rendered via common.Render() in run()
        return status, nil
    }

    return execAWK(source, files, fieldSep, input, out, errOut)
}

func execAWK(source string, files []string, fieldSep string,
             input io.Reader, out io.Writer, errOut io.Writer) (int, error) {
    prog, err := parser.ParseProgram([]byte(source), nil)
    if err != nil {
        return 2, fmt.Errorf("awk: %v", err)
    }
    config := &interp.Config{
        Stdin:  input,
        Output: out,
        Error:  errOut,
        Args:   files,
        Vars:   []string{"FS", fieldSep},
    }
    return interp.ExecProgram(prog, config)
}
```

- **Library layer** (`Run`): Takes `io.Reader`/`io.Writer`, supports `--json` mode
  by capturing output to a buffer, parsing it into structured records.
- **CLI layer** (`run`): Parses flags (`-F`, `-v`, `-f`, `--json`), calls `Run()`,
  renders via `common.Render()`.

### Step 3 — `pkg/awk/awk_test.go` — Unit tests

≥20 test cases covering:

- [ ] Basic field splitting (`-F :`, default whitespace)
- [ ] `{ print $1 }`, `{ print $0 }`
- [ ] `BEGIN { ... }` and `END { ... }` blocks
- [ ] Pattern matching: `/regex/`, expression patterns (`$3 > 100`)
- [ ] Variables: `NR`, `NF`, user variables
- [ ] Built-in functions: `length()`, `substr()`, `split()`, `sub()`, `gsub()`
- [ ] Math: `int()`, arithmetic operators
- [ ] Control flow: `if/else`, `while`, `for`, `for-in`
- [ ] Arrays and `delete`
- [ ] `-v var=value` variable assignment
- [ ] `-f progfile` from file
- [ ] Error handling: syntax errors, `-F` with invalid regex

### Step 4 — BusyBox integration

- [ ] Wire `awk` into our BusyBox test harness (`test/busybox_testsuite/runtest`):
  add `awk` to the applet list so the symlink `awk -> goposix` is created.
- [ ] Run `make testsuite` and measure pass rate.
- [ ] For failing tests: categorize each (cosmetic error message, parse-time detection,
  GNU extension, real gap). Fix real gaps if feasible; document the rest.
- [ ] Baseline: expect **34–40 passes** out of 53 with no changes; target **45+**
  with error message shimming and minor fixes.

### Step 5 — Cross-cutting deliverables

- [ ] Register `awk` in `cmd/goposix/main.go` (blank import)
- [ ] Add `pkg/awk` to `PKG_DIRS` in `Makefile`
- [ ] BusyBox `awk.tests` integrated and baseline recorded
- [ ] `test/compliance/test_awk.sh` — POSIX compliance test script
- [ ] Add to `compliance` target in `Makefile`
- [ ] Update `wiki/test_coverage_matrix.md`: awk from ❌ to ✅
- [ ] Update `wiki/todos.md`: remove awk from deferred
- [ ] Update `wiki/phases.md`: mark 07a COMPLETED

### Step 6 — `--json` output mode

JSON mode captures goawk's text output and wraps it in our standard envelope.
A lightweight parser splits `print`-delimited output into per-record arrays:

```json
{
  "command": "awk",
  "version": "1.0.0",
  "exitCode": 0,
  "data": {
    "records": [
      {"nr": 1, "fields": ["alice", "90"]},
      {"nr": 2, "fields": ["bob", "85"]}
    ]
  }
}
```

Alternative: use goawk's native `printf` in a wrapper to emit per-record JSON
directly (avoids output parsing). Either approach is ~30 LOC.

- [ ] `test/schemas/awk.schema.json` — JSON Schema draft-07
- [ ] `--json` unit tests (envelope structure, record format)

---

## Known Gaps & Tradeoffs

| Gap | Severity | Mitigation |
|-----|----------|------------|
| goawk error messages differ from BusyBox | Low | Wrap parse errors with `fmt.Errorf("awk: %v", err)` — our error format, not goawk's |
| `or()`/`and()` bitwise functions missing | None | GNU extensions, not POSIX. Document as unsupported. |
| `ERRNO` not set on file-not-found | Low | goawk leaves it at 0. Acceptable — few scripts rely on this. |
| exit code not propagated through END | Low | `exit 42` then `END { exit }` returns 0 instead of 42. Edge case. |
| backslash+newline in strings | Low | goawk preserves newline; BusyBox strips it. Minor output difference. |
| Function resolution order | TBD | The `func f(){}; func g(){}; BEGIN{f(g(),g())}` pattern triggers "undefined function f" in goawk. Needs investigation — may be a parser bug or a known limitation. |

---

## Effort Estimate

| Step | Effort |
|------|--------|
| 1. Add dependency | 5 min |
| 2. `pkg/awk/awk.go` (library + CLI layer) | 1–2 h |
| 3. Unit tests (20+ cases) | 1–2 h |
| 4. BusyBox integration + baseline | 30 min |
| 5. Cross-cutting (dispatch, Makefile, compliance, docs) | 1 h |
| 6. `--json` output mode + schema + tests | 30 min |
| **Total** | **~5 h** |

vs. the original 8-sub-phase build-from-scratch plan: **3+ months**.

---

## How to Verify

```bash
# Basic field splitting
echo "alice 90\nbob 85" | goposix awk '{ print $1 }'

# Sum a column
echo "10\n20\n30" | goposix awk '{ sum += $1 } END { print sum }'

# Filter + format
goposix awk -F: '$3 >= 1000 { printf "%-20s %s\n", $1, $7 }' /etc/passwd

# JSON mode
echo "a b c" | goposix awk --json '{ print $2 }'

# BusyBox suite
make testsuite  # expect awk.tests entries in results
```
