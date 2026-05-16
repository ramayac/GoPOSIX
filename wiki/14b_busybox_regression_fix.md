# BusyBox Test Suite Regression Fix — 2026-05-15

## Summary

After running `make testsuite`, 79 BusyBox tests were failing (a major regression from the previously reported 1 failure / 10 skipped). This investigation fixed 33 of those failures across 3 critical utilities: **diff** (12), **printf** (21), and **test** (9 — all 9 test failures resolved).

## Root Causes & Fixes

### 1. Echo → Diff (12 failures)

**Bug:** `echo` used `common.ParseFlags` to parse all arguments, which treated any argument starting with `-` as a potential flag. When diff tests created test data via `echo -ne "--- -\n+++ input\n..."`, the `---` prefix was parsed as a flag group `-`, `-`, `-`, causing "unknown flag: --" errors.

**Fix:** Rewrote `echo.go` to use manual flag parsing (`parseEchoFlags`). Only `-n`, `-e`, `-E`, and `--json` are recognized; everything else (including `-neEZ` with an unknown char `Z`) is treated as literal text. Removed the `-j` short flag for `--json` to avoid collisions.

**Also fixed:**
- **Non-opts flag group** (`echo -neEZ`): Previously had partial side-effects (flags `-neE` set but `Z` invalid → inconsistent state). Now accumulates flags atomically — if any char in the group is unknown, the entire arg is treated as literal text.
- **Octal escapes** (`echo -ne '\41z'`): Previously only handled `\0NNN` octal. Now handles any `\NNN` (backslash followed by 1–3 octal digits). This also fixed `\041`, `\0041`, etc.

### 2. Printf (21 failures)

**Bug 1:** `printf` used `common.ParseFlags` which treated `-5` as a flag.

**Fix 1:** Manual flag parsing — only `--json` is accepted; all other args are positional.

**Bug 2:** `\c` escape was handled in processEscapes (returning early), but Format then re-processed the truncated format string in an infinite loop.

**Fix 2:** Handle `\c` at the Format level. processEscapes now passes `\c` through as literal chars; `processOnePass` detects the `\c` sequence and stops format reuse.

**Bug 3:** Format string was not reused for remaining arguments. POSIX requires re-processing the format when there are more args than specifiers.

**Fix 3:** Outer loop in `Format`: after one pass through the format, if args remain, reset format position and repeat until all args consumed.

**Bug 4:** Width/precision `*` not parsed. `%*.*f` should pull width/precision from args.

**Fix 4:** Added `*` handling in the conversion parser — consumes next arg for width/precision value.

**Bug 5:** Length modifiers (`z`, `l`, `L`, `h`, `hh`, `ll`, `j`, `t`) not stripped. Go's `fmt` package doesn't support `%zd`, `%ld`, etc.

**Fix 5:** Skip/consume length modifiers when parsing the conversion specification.

**Bug 6:** `%i` sent directly to Go's `fmt.Sprintf` which doesn't support `%i`.

**Fix 6:** Map `%i` → `%d` before passing to Go's formatter.

**Bug 7:** `%0*d` treated `0` as width digit, not as zero-padding flag.

**Fix 7:** Flags (`-`, `+`, ` `, `0`, `#`) are now parsed before width digits.

**Bug 8:** Negative width not handled (left-justify).

**Fix 8:** When width < 0, add `-` flag and negate width.

**Bug 9:** Negative precision should treat precision as unspecified.

**Fix 9:** When precision < 0, clear hasPrecision flag.

**Bug 10:** Invalid numeric args (`-`, `bad`, `123bad`) not producing errors.

**Fix 10:** `parseInt`/`parseUint`/`parseFloat` now detect unparseable input and return the remaining suffix. `formatError` interleaves error messages with output (matching POSIX behavior with `2>&1`).

**Bug 11:** `%b` conversion (process escapes in arg) not implemented.

**Fix 11:** Added `doBConv` and `processEscapesForB` — processes `\n`, `\t`, `\\`, `\0NNN`, `\xNN` in the argument.

**Bug 12:** Character constants like `'"x` not handled.

**Fix 12:** Added character constant parsing in `doIntConv` — `"x` → 120.

**Bug 13:** Bare `%` and unknown `%r` should abort with error.

**Fix 13:** `processOnePass` returns false for invalid format chars, stopping further processing and setting exit code 1.

**Bug 14:** Format string reuse with 0 args → infinite loop on empty format.

**Fix 14:** Added `len(state.format) == 0` check in the reuse loop.

### 3. Test/`[` Utility (9 failures)

**Bug 1:** `test !` (single NOT) errored with "unexpected end of expression".

**Fix 1:** In `parseNot`, when `!` is consumed and the remaining expression is empty, return `true` (NOT of empty/false).

**Bug 2:** `test -f` (unary op without argument) errored with "missing argument for -f".

**Fix 2:** When a unary operator has no argument, treat it as a non-empty string → `true`.

**Bug 3:** `test '!' = '!'` — `!` was consumed as NOT operator instead of string LHS of `=`.

**Fix 3:** Added lookahead in `parseNot`: if `!` is immediately followed by a binary operator (`=`, `==`, `!=`, `-eq`, etc.), don't treat it as NOT — let `parsePrimary` handle it as a string.

**Bug 4:** `test '(' = '('` — `(` was consumed as paren-start instead of string LHS.

**Fix 4:** Added lookahead in `parsePrimary`: if `(` is immediately followed by a binary operator, treat it as a string.

**Bug 5:** `test a -a !` — the trailing `!` (NOT of empty) failed.

**Fix 5:** Same as Fix 1 — `parseNot` handles trailing `!` by returning true.

## Flag Design Decision: No `-j` Short Form

The `-j` short flag for `--json` was removed from **echo** and **printf**. Rationale:

- `echo -j` would conflict with literal text `-j` being passed to echo
- `printf -j` would conflict with negative numbers (`printf '%d\n' -5`)
- These utilities have no standard flags beyond their own; everything else is data
- `--json` (long form only) avoids all collision risk

For future consideration: other utilities like `tar` have `-j` as bzip2 in POSIX, creating a real conflict with `-j` → `--json`. All utilities should ideally use long-form only for `--json`.

## Hardening Tests Added

### echo_test.go
- `TestBusyBox_Echo_ArgumentStartingWithDashes` — `---` prefix
- `TestBusyBox_Echo_ArgumentIsDash` — `--- hello` literal
- `TestBusyBox_Echo_NonOptsFlagGroup` — `-neEZ` atomically treated as text
- `TestBusyBox_Echo_OctalEscapeNNN` — `\41z` → `!z`
- `TestBusyBox_Echo_OctalEscapeZeroNNN` — `\041` → `!`

### printf_test.go
- `TestBusyBox_Printf_StopOnC` — `\c` stops output
- `TestBusyBox_Printf_StopOnC_InFormat` — `%s\c` stops after first arg
- `TestBusyBox_Printf_ReuseFormatForRemainingArgs` — format reuse with multiple args
- `TestBusyBox_Printf_BConversion` — `%b` escape processing
- `TestBusyBox_Printf_CharConstants` — `"x`, `'y` → char values
- `TestBusyBox_Printf_StarWidthPrecision` — `%*.*f` from args
- `TestBusyBox_Printf_NegativeWidth` — `%*f` with -23
- `TestBusyBox_Printf_NegativePrecision` — `%.*f` with -12
- `TestBusyBox_Printf_NegativeBoth` — both negative
- `TestBusyBox_Printf_LengthModifiers` — `%zd`, `%ld`, `%Ld`
- `TestBusyBox_Printf_InvalidNumber` — bad input with interleaved errors
- `TestBusyBox_Printf_BarePercent` — `%` error
- `TestBusyBox_Printf_UnknownConversion` — `%r` error
- `TestBusyBox_Printf_ZeroFlag` — `%0*d`
- `TestBusyBox_Printf_ArgumentStartingWithDash` — `-5` as string arg

### testcmd_test.go
- `TestBusyBox_Test_BangOnly` — `test !`
- `TestBusyBox_Test_UnaryWithoutArg` — `test -f`
- `TestBusyBox_Test_BangUnary` — `test ! -f`
- `TestBusyBox_Test_AndBang` — `test a -a !`
- `TestBusyBox_Test_UnaryEqualsOr` — `test -f = a -o b`
- `TestBusyBox_Test_BangEqualsString` — `test '!' = '!'`
- `TestBusyBox_Test_ParenEqualsString` — `test '(' = '('`
- `TestBusyBox_Test_BangBangEquals` — `test '!' '!' = '!'`
- `TestBusyBox_Test_BangParenEquals` — `test '!' '(' = '('`

## Remaining Failures (46)

Non-targeted utilities still have failures: basename, cat, cp, date, du, expr, find, grep, gunzip, hostname, ls, mv, tail, tar, touch, wc, xargs. These are tracked separately.

## Key Insight

The `common.ParseFlags` approach, while elegant for structured utilities like `ls`, `grep`, etc., is fundamentally wrong for utilities where arguments can be arbitrary text starting with `-` (echo, printf). The fix is to use manual flag parsing that stops at the first non-flag argument, which is the correct POSIX behavior for these utilities.
