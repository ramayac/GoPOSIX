# Post-MVP — Fixes & Utilities

> **Status:** ✅ COMPLETED | **Date:** 2026-05-17
>
> Consolidated from Phases 14–18 (post-MVP fixes, utilities, and quality gates).

---

## 14a — JSON Gap Fill (8 Utilities)

`--json` flag added to utilities that were missing structured output:

- `sleep`, `tee`, `tr`, `yes`, `sed`, `true`, `false`, `chmod`
- All registered in `common.ParseFlags` flag specs
- JSON schemas added in `test/schemas/`

---

## 14b — BusyBox Regression Fix (79 → 3 failures)

**Root cause:** `common.ParseFlags` applied uniformly broke `echo`, `printf`, `expr` — positional args starting with `-` were parsed as flags. Fix: manual flag parsing with `StopAtFirstNonFlag` mode.

**Result:** 79 → 3 failures (96.2% pass rate). Established the rule: infrastructure changes must provide escape hatches.

---

## 14c — Coverage Drive (76.7% → 83.5%)

Targeted test additions for low-coverage packages:

- `date`: POSIX TZ edge cases
- `dd`: status output, block size edge cases
- `cp`: recursive, preserve, update flags
- `grep`: `-F` fixed-string comparisons
- `find`: `-exec`, `-maxdepth`, `-type`

Coverage gate established at ≥80% for CI.

---

## Phase 15 — `dd` + `od` (10 BusyBox tests)

- `dd`: full POSIX implementation with `bs`, `count`, `skip`, `seek`, `if`, `of` operands
- `od`: octal/hex/character dump formats, `-A` address radix, `-j` skip, `-N` count
- 10 BusyBox tests (6 dd + 4 od), all passing

---

## Phase 16 — `patch`, `egrep`, `fgrep`, `expand`, `unexpand`

- `patch`: unified diff application, `-p` strip levels, `-R` reverse
- `egrep` / `fgrep`: implemented as `grep -E` / `grep -F` aliases
- `expand` / `unexpand`: tab expansion (8 tests), `-t` tab stop lists

---

## Phase 17 — `comm`, `paste`, `fold`, `sum`, `nl`, `cmp`, `strings`

- `comm`: 9 BusyBox tests — column-wise file comparison
- `paste`: 5 tests — column merging with delimiters
- `fold`: 4 tests — line wrapping
- `sum`: 4 tests — BSD checksum
- `nl`: 4 tests — line numbering
- `cmp`: 1 test — byte-level comparison
- `strings`: 1 test — printable string extraction

---

## Phase 18.3 — `which`, `realpath`, `pidof`, `seq`, `cal`, `hostid`, `factor`, `uptime`, `wget`, `rev`, `tree`, `tsort`, `sha1sum`, `sha512sum`, `sha3sum`

- 15 utilities added across Post-MVP part III
- All BusyBox tests passing (where applicable)
- JSON-RPC daemon endpoints registered for all
