# Phase 07a — `awk` (POSIX Text Processing)

> **Status:** ⏳ Not Started | **Depends on:** Phase 07.4 (Tier 5 utilities)
>
> Separated from the main Phase 07 document because `awk` is a substantially
> more complex utility than the rest of Tier 5 — closer to a small programming
> language than a CLI tool. It can be tackled independently without blocking
> other milestones.

---

## Overview

Implement a subset of POSIX `awk` in pure Go. Start with the most commonly
used features and expand incrementally. Full POSIX `awk` compliance is a
stretch goal, not a blocker for release.

---

## 07a.1 — Field Splitting & Print ❌

**Complexity:** Medium — lexer + basic record/field processing.

The minimum viable `awk`: read lines, split into fields, print.

- [ ] Read stdin or files line by line
- [ ] Split records by field separator (`-F` flag, default whitespace)
- [ ] `$0` (whole line), `$1`..`$N` (individual fields)
- [ ] `NF` (number of fields), `NR` (record number)
- [ ] `{ print }`, `{ print $1, $3 }` — basic print actions
- [ ] `--json` output: `[{"nr": 1, "fields": ["a", "b", "c"]}]`
- [ ] Unit tests

---

## 07a.2 — Pattern Matching ❌

**Complexity:** Medium — regex matching with Go's `regexp` package.

Add the ability to filter lines before applying actions.

- [ ] `/regex/ { action }` — match lines against pattern
- [ ] `BEGIN { action }` — run before any input
- [ ] `END { action }` — run after all input
- [ ] Pattern ranges: `/start/,/end/ { action }`
- [ ] Expression patterns: `$3 > 100 { print $1 }`
- [ ] Unit tests

---

## 07a.3 — Variables & Expressions ❌

**Complexity:** Medium-High — expression evaluator with type coercion.

Add variables, arithmetic, and string concatenation.

- [ ] User variables: `{ total += $2 }`
- [ ] Built-in variables: `FS`, `OFS`, `RS`, `ORS`, `FILENAME`
- [ ] Arithmetic: `+`, `-`, `*`, `/`, `%`, `^`
- [ ] String concatenation (implicit, by adjacency)
- [ ] Comparison operators: `<`, `<=`, `==`, `!=`, `>=`, `>`
- [ ] Logical operators: `&&`, `||`, `!`
- [ ] Ternary: `expr ? a : b`
- [ ] Unit tests

---

## 07a.4 — Built-in Functions ❌

**Complexity:** Medium-High — standard library of string/math functions.

Implement the most commonly used POSIX awk built-in functions.

- [ ] String: `length()`, `substr()`, `index()`, `split()`, `sub()`, `gsub()`, `match()`, `sprintf()`
- [ ] String: `tolower()`, `toupper()`
- [ ] Math: `int()`, `sqrt()`, `sin()`, `cos()`, `log()`, `exp()`, `rand()`, `srand()`
- [ ] I/O: `print`, `printf` (with format specifiers)
- [ ] Unit tests

---

## 07a.5 — Control Flow ❌

**Complexity:** High — AST interpreter with control flow.

Add control flow statements to make awk Turing-complete.

- [ ] `if (expr) stmt [else stmt]`
- [ ] `while (expr) stmt`
- [ ] `for (init; cond; incr) stmt`
- [ ] `for (key in array) stmt`
- [ ] `break`, `continue`, `next`, `exit`
- [ ] Statement blocks `{ ... }`
- [ ] Unit tests

---

## 07a.6 — Associative Arrays ❌

**Complexity:** High — hash map implementation with string keys.

- [ ] `array[key] = value` — set
- [ ] `(key in array)` — membership test
- [ ] `delete array[key]` — delete element
- [ ] `for (k in array)` — iteration
- [ ] Multi-dimensional: `array[i, j]` via `SUBSEP`
- [ ] Unit tests

---

## 07a.7 — Multiple Input & Output ❌

**Complexity:** High — file handle management, piping.

- [ ] Multiple file arguments processed in order
- [ ] Output redirection: `print > "file"`, `print >> "file"`
- [ ] Pipe output: `print | "command"` (within shell interpreter context)
- [ ] `getline` — read next line / from file / from command
- [ ] Unit tests

---

## 07a.8 — Compliance & Polish ❌

**Complexity:** High — edge cases, POSIX spec conformance.

- [ ] `-v var=value` — assign variables from CLI
- [ ] `-f progfile` — read program from file
- [ ] Multiple `-e` programs
- [ ] Proper exit codes
- [ ] Comprehensive `--json` structured output mode
- [ ] BusyBox awk test suite integration (`test/busybox_testsuite/awk.tests`)
- [ ] Performance benchmarks on large files

---

## How to Verify

```bash
# Basic field splitting
echo "alice 90\nbob 85" | korego awk '{ print $1 }'

# Sum a column
echo "10\n20\n30" | korego awk '{ sum += $1 } END { print sum }'

# Filter + format
korego awk -F: '$3 >= 1000 { printf "%-20s %s\n", $1, $7 }' /etc/passwd

# JSON mode
echo "a b c" | korego awk --json '{ print $2 }'
```
