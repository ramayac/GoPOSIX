# Task Log: Harden pkg/head/ and add dispatch-call tests

## Summary
Harden `pkg/head/head.go`, fix 3 bugs, add 22 dispatch-call tests.
Coverage: **29.0% → 94.1%** (target ≥55% ✅)

## Bugs Fixed

### 1. Multi-file header misalignment with stdin (functional bug)
**Root cause:** `readers` list was misaligned with `flags.Positional` when `flags.Stdin` inserted stdin at position 0. Header naming used `flags.Positional[i]` where `i` indexed `readers`, causing wrong names or index-out-of-bounds.

**Fix:** Introduced `headInput` struct pairing each reader with its display name. Stdin deduplication: only add stdin from `flags.Stdin` when Positional is empty (since bare `-` also appears in Positional).

### 2. Early return on file open error (functional bug)
**Root cause:** `os.Open(path)` failure called `return 1`, skipping remaining files.

**Fix:** Changed to `continue` after setting `exitCode = 1`, allowing all files to be processed.

### 3. Deferred file close accumulation (resource leak)
**Root cause:** `defer f.Close()` in a loop deferred all closes until function return, potentially exhausting file descriptors for many files.

**Fix:** Close each file immediately after processing via `headInput.closer.Close()`.

### Also fixed:
- Removed dead code: `writer := os.NewFile(os.Stderr.Fd(), "/dev/null")` (unused, fragile)
- Removed `os` import of unused symbol `os.NewFile`

## Tests Added (22 new dispatch-call tests)

### Basic file I/O
- `TestRunCLI_Basic` — default 10 lines from file
- `TestRunCLI_LinesFlag` — `-n 3` short flag with space
- `TestRunCLI_LinesFlagAttached` — `-n5` attached value (POSIX)
- `TestRunCLI_NegativeLines` — `-n -3` skip last 3
- `TestRunCLI_NegativeAttached` — `-n-2` attached negative
- `TestRunCLI_ShortFile` — short file with large -n
- `TestRunCLI_LargeLineCount` — huge -n with small file
- `TestRunCLI_NegativeAll` — negative skip more than total lines

### Byte mode
- `TestRunCLI_BytesFlag` — `-c 5`
- `TestRunCLI_BytesFlagZero` — `-c 0`
- `TestRunCLI_BytesLongFlag` — `--bytes=8`
- `TestRunCLI_LongFlagBytesSpace` — `--bytes 6`

### Multi-file
- `TestRunCLI_MultiFile` — two files with headers
- `TestRunCLI_MultiFileWithDash` — stdin + file with headers
- `TestRunCLI_FileNotFoundContinues` — error on first file, continues to second

### Stdin
- `TestRunCLI_Stdin` — piped input, no file args
- `TestRunCLI_StdinDash` — `head -n 2 -`
- `TestRunCLI_NoArgs` — bare head reads stdin

### JSON
- `TestRunCLI_Json` — `--json` produces valid envelope with correct line data

### Error handling
- `TestRunCLI_FileNotFound` — missing file → exit 1
- `TestRunCLI_InvalidLinesCount` — `-n abc` → exit 2
- `TestRunCLI_InvalidBytesCount` — `-c abc` → exit 2
- `TestRunCLI_UnknownFlag` — `--bogus` → exit 2

### Long flags
- `TestRunCLI_LinesLongFlag` — `--lines=4`
- `TestRunCLI_LongFlagLinesSpace` — `--lines 3`

## Verification
```
CGO_ENABLED=0 go build ./...           ✅
go test -race -cover ./pkg/head/       ✅ (PASS, coverage: 94.1%)
go vet ./pkg/head/                     ✅
```

### Per-function coverage:
- `Run()`:       100.0% (was 96.9%)
- `run()`:       94.2%  (was 0.0%)
- `runNegative()`: 92.9%
- `runBytes()`:   85.7%
- `init()`:      100.0%
- **Total:        94.1%** (target ≥55% ✅)
