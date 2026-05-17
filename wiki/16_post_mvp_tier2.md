# Phase 16 — Post-MVP Tier 2: Text & Stream Utilities

> **Status:** PLANNING | **Date:** 2026-05-16 | **Branch:** `feat/post-mvp`
>
> **Parent:** [todos.md](todos.md) — BusyBox gap list
>
> Nine utilities covering text formatting, column manipulation, and stream tools.
> All have BusyBox test coverage. Ordered by test count (most→least).

---

## Current State

| # | Utility | BusyBox Tests | Complexity | Est. LOC |
|---|---------|---------------|------------|----------|
| 16.1 | `unexpand` | 11 new-style | Trivial | ~120 |
| 16.2 | `comm` | 9 new-style | Trivial | ~150 |
| 16.3 | `paste` | 5 old-style | Trivial | ~130 |
| 16.4 | `fold` | 5 new-style | Trivial | ~100 |
| 16.5 | `sum` | 4 new-style | Trivial | ~100 |
| 16.6 | `nl` | 4 new-style | Trivial | ~120 |
| 16.7 | `expand` | 3 new-style | Trivial | ~90 |
| 16.8 | `cmp` | 1 old-style | Trivial | ~90 |
| 16.9 | `strings` | 1 old-style | Trivial | ~80 |

**Total estimated LOC:** ~980 | **BusyBox pass gain:** +43

None exist in `pkg/` today.

---

## Common Pattern

All Tier 2 utilities share the same simple CLI pattern:
- Read from files or stdin
- Transform text line-by-line or byte-by-byte
- Write to stdout
- Support `--json` structured output

Each follows the canonical `Run()` library-layer + `run()` CLI-glue pattern
established in `pkg/cat/cat.go`. Test files follow the `catRun()` injectable pattern.

---

## 16.1 — `unexpand`

**Purpose:** Convert leading spaces to tabs in each line.

**BusyBox tests:** `test/busybox_testsuite/unexpand.tests` (11 cases)

**Flags:** `-a` (convert all spaces, not just leading), `-t N` (tab width, default 8)

### CHECK → TEST → CODE → PASS

1. **CHECK:** Read `unexpand.tests` — understand `-a`, `-t`, and `--first-only` behaviors
2. **TEST:** `pkg/unexpand/unexpand_test.go`
   - Default conversion (leading spaces → tabs)
   - `-a` all-spaces mode
   - `-t 4` custom tab stop
   - Mixed tabs+spaces preservation
3. **CODE:** `pkg/unexpand/unexpand.go`
   - Core: walk each line, count leading spaces, replace groups of N with `\t`
   - `--json`: return `{"lines": [...]}`
4. **PASS:** `make testsuite` — 11 new passes

---

## 16.2 — `comm`

**Purpose:** Compare two sorted files line by line. Output 3 columns: lines only
in file1, lines only in file2, lines in both.

**BusyBox tests:** `test/busybox_testsuite/comm.tests` (9 cases)

**Flags:** `-1`, `-2`, `-3` (suppress columns), `--total` (summary count)

### CHECK → TEST → CODE → PASS

1. **CHECK:** Read `comm.tests` — column suppression and total flag
2. **TEST:** `pkg/comm/comm_test.go`
   - Two identical files → all lines in col 3
   - Disjoint files → cols 1 and 2
   - `-1` suppression, `-23` suppression, `--total`
3. **CODE:** `pkg/comm/comm.go`
   - Core: merge-scan two sorted `[]string` slices line by line
   - `--json`: `{"unique_to_file1": [...], "unique_to_file2": [...], "common": [...]}`
4. **PASS:** `make testsuite` — 9 new passes

---

## 16.3 — `paste`

**Purpose:** Merge lines of files horizontally, separated by tabs.

**BusyBox tests:** 5 old-style in `test/busybox_testsuite/paste/`

**Flags:** `-d DELIM` (delimiter, default tab), `-s` (serialize — transpose rows to columns)

### CHECK → TEST → CODE → PASS

1. **CHECK:** Read old-style paste test scripts
2. **TEST:** `pkg/paste/paste_test.go`
   - Two files, default tab join
   - `-d :` custom delimiter
   - `-s` serialize mode
   - Unequal-length file handling
3. **CODE:** `pkg/paste/paste.go`
   - Core: read all files, iterate max(lines) indices, join with delimiter
   - `--json`: `{"records": [["a","1"], ["b","2"]]}`
4. **PASS:** `make testsuite` — 5 new passes

---

## 16.4 — `fold`

**Purpose:** Wrap input lines to specified width.

**BusyBox tests:** `test/busybox_testsuite/fold.tests` (5 cases)

**Flags:** `-w N` (width, default 80), `-b` (count bytes not characters), `-s` (break at spaces)

### CHECK → TEST → CODE → PASS

1. **CHECK:** Read `fold.tests`
2. **TEST:** `pkg/fold/fold_test.go`
   - Default 80-char wrap
   - `-w 20` custom width
   - `-s` space-aware breaks
   - `-b` byte counting
3. **CODE:** `pkg/fold/fold.go`
   - Core: scan line, emit chunks of width N
   - `--json`: `{"lines": [...]}`
4. **PASS:** `make testsuite` — 5 new passes

---

## 16.5 — `sum`

**Purpose:** Compute checksum and block count of files.

**BusyBox tests:** `test/busybox_testsuite/sum.tests` (4 cases)

**Flags:** `-r` (BSD algorithm, default), `-s` (System V algorithm)

### CHECK → TEST → CODE → PASS

1. **CHECK:** Read `sum.tests` — BSD and SysV checksums
2. **TEST:** `pkg/sum/sum_test.go`
   - BSD checksum (`-r`) single file
   - SysV checksum (`-s`) single file
   - Multi-file output
   - Stdin input
3. **CODE:** `pkg/sum/sum.go`
   - BSD: 16-bit sum, block count in 1K units, filename
   - SysV: 32-bit sum with modulo, 512-byte blocks
   - `--json`: `{"files": [{"name": "...", "checksum": N, "blocks": N}]}`
4. **PASS:** `make testsuite` — 4 new passes

---

## 16.6 — `nl`

**Purpose:** Number lines of files.

**BusyBox tests:** `test/busybox_testsuite/nl.tests` (4 cases)

**Flags:** `-b TYPE` (body numbering style), `-v N` (starting number), `-w N` (width)

### CHECK → TEST → CODE → PASS

1. **CHECK:** Read `nl.tests`
2. **TEST:** `pkg/nl/nl_test.go`
   - Default numbering (non-empty lines only)
   - `-b a` (all lines)
   - `-v 10` starting value
   - `-w 3` width padding
3. **CODE:** `pkg/nl/nl.go`
   - Core: iterate lines, prepend number if line qualifies
   - `--json`: `{"lines": [{"number": 1, "text": "..."}]}`
4. **PASS:** `make testsuite` — 4 new passes

---

## 16.7 — `expand`

**Purpose:** Convert tabs to spaces in each line.

**BusyBox tests:** `test/busybox_testsuite/expand.tests` (3 cases)

**Flags:** `-t N` (tab width, default 8), `-i` (initial tabs only — leading)

### CHECK → TEST → CODE → PASS

1. **CHECK:** Read `expand.tests`
2. **TEST:** `pkg/expand/expand_test.go`
   - Default tab→spaces conversion
   - `-t 4` custom tab stop
   - `-i` leading-only mode
3. **CODE:** `pkg/expand/expand.go`
   - Core: walk each character, expand `\t` to N spaces based on column position
   - `--json`: `{"lines": [...]}`
4. **PASS:** `make testsuite` — 3 new passes

---

## 16.8 — `cmp`

**Purpose:** Compare two files byte by byte. Reports first differing byte.

**BusyBox tests:** 1 old-style in `test/busybox_testsuite/cmp/`

**Flags:** `-s` (silent — only exit code), `-l` (list all differences), `-n N` (limit bytes)

### CHECK → TEST → CODE → PASS

1. **CHECK:** Read cmp test script
2. **TEST:** `pkg/cmp/cmp_test.go`
   - Identical files → exit 0, no output
   - One-byte difference → "file1 file2 differ: byte N, line M"
   - `-s` silent mode
   - `-l` verbose listing
   - `-n 10` byte limit
3. **CODE:** `pkg/cmp/cmp.go`
   - Core: `io.ReadFull` both files in parallel blocks, compare byte-by-byte
   - `--json`: `{"equal": bool, "first_diff": {"byte": N, "line": M, "val1": X, "val2": Y}}`
4. **PASS:** `make testsuite` — 1 new pass

---

## 16.9 — `strings`

**Purpose:** Extract printable character sequences from binary files.

**BusyBox tests:** 1 old-style in `test/busybox_testsuite/strings/`

**Flags:** `-n N` (minimum length, default 4), `-t FORMAT` (radix: `d`, `o`, `x`)

### CHECK → TEST → CODE → PASS

1. **CHECK:** Read strings test script
2. **TEST:** `pkg/strings/strings_test.go`
   - Binary file with embedded text
   - `-n 2` shorter sequences
   - `-t x` hex offset prefix
   - Empty input, all-binary input
3. **CODE:** `pkg/strings/strings.go`
   - Core: scan bytes, accumulate runs of printable chars, emit runs ≥ `-n` length
   - Printable set: ASCII 0x20–0x7E (plus tab)
   - `--json`: `{"strings": [{"offset": N, "value": "..."}]}`
4. **PASS:** `make testsuite` — 1 new pass

---

## Registration Checklist (per utility)

For each 16.1–16.9:

- [ ] `pkg/<name>/<name>.go` — library layer `Run()` + CLI glue `run()` with `init()` → `dispatch.Register`
- [ ] `pkg/<name>/<name>_test.go` — unit tests targeting ≥70% coverage
- [ ] Add `_ ".../goposix/pkg/<name>"` to `cmd/goposix/main.go`
- [ ] Add `./pkg/<name>/...` to `PKG_DIRS` in `Makefile`
- [ ] Run `make vet test build` → clean
- [ ] Run `make testsuite` → corresponding tests pass

---

## Execution Order

Implement in this order (descending BusyBox test count, highest yield first):

```
unexpand (11) → comm (9) → paste (5) → fold (5) → sum (4) →
nl (4) → expand (3) → cmp (1) → strings (1)
```

Each utility is self-contained with no cross-dependencies. They can be
implemented in parallel by separate agents if desired.

---

## Milestone 16

```
[ ] 16.1 — unexpand: 11/11 BusyBox tests pass
[ ] 16.2 — comm: 9/9 BusyBox tests pass
[ ] 16.3 — paste: 5/5 BusyBox tests pass
[ ] 16.4 — fold: 5/5 BusyBox tests pass
[ ] 16.5 — sum: 4/4 BusyBox tests pass
[ ] 16.6 — nl: 4/4 BusyBox tests pass
[ ] 16.7 — expand: 3/3 BusyBox tests pass
[ ] 16.8 — cmp: 1/1 BusyBox test passes
[ ] 16.9 — strings: 1/1 BusyBox test passes
```

**Combined BusyBox pass gain: +43 (477 → 520+)**

---

## How to Verify

```bash
# Per-utility
echo -e "        hello\tworld" | ./goposix unexpand
./goposix comm <(echo a; echo b) <(echo b; echo c)
./goposix paste file1 file2 -d :
echo "long line here..." | ./goposix fold -w 10
./goposix sum *.go
./goposix nl README.md
echo -e "a\tb\tc" | ./goposix expand -t 4
./goposix cmp file1 file2
./goposix strings /bin/goposix

# Full suite
make testsuite  # +43 passes expected
```
