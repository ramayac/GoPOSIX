# grep Harden & Test — TASK LOG

## Summary
**Coverage: 16.5% → 85.3%** (target: ≥40%)
**Tests: 18 → 47** (29 new CLI-layer tests)
**Status: ✅ DONE**

## Changes

### 1. Architecture Refactor (`grep.go`)
- Extracted `grepRun(args, out, errOut, stdinR)` from `run()` — the testable core
- `run()` is now a thin wrapper: `return grepRun(args, out, os.Stderr, os.Stdin)`
- All output (stdout + stderr) now goes through injected writers
- Stdin now goes through injected reader
- This makes every code path in the CLI layer testable

### 2. New Flags Added
- `-H` / `--with-filename` — force filename prefix on output
- `-h` / `--no-filename` — suppress filename prefix
- `-R` / `--dereference-recursive` — alias for `-r`

### 3. Context Flags Implemented (`-A`, `-B`, `-C`)
- Added `scanWithContext()` function that reads entire file, marks matches, computes context windows, and outputs with `:`/`-` markers and `--` group separators
- Context mode takes precedence over `-c`, `-l`, `-L`, `-o` (as in GNU grep)

### 4. Bug Fixes
- **`-l` exit code**: now sets exitCode=0 when matches found (was not updating)
- **JSON mode exit code**: now sets exitCode=0 when matches found
- **Empty patterns**: filtered out from `-e` and `-f` sources (prevents empty-regex matching everything)
- **Output routing**: all output now goes through `out`/`errOut` writers instead of directly to `os.Stdout`/`os.Stderr`

### 5. Tests Added (grep_test.go)
29 new CLI-layer tests covering:
- Basic match, count, ignore-case, invert, line-number
- Files-with-matches (`-l`), files-without-match (`-L`)
- Fixed strings (`-F`), word-regexp (`-w`), line-regexp (`-x`)
- Recursive (`-r`, `-R`), file patterns (`-f`), multi-pattern (`-e`)
- Stdin (bare and with `-`), `-f -` (patterns from stdin)
- JSON mode, quiet mode (`-q`)
- Missing pattern, file not found, `-s` suppress errors
- Only-matching (`-o`), extended regex (`-E`)
- Multi-file with filename prefix, `-H`, `-h`
- Context: `-A`, `-B`, `-C`, group separators
- Fixed+ignore-case, fixed+word
- Count multi-file, count stdin
- Empty pattern file, multiple `-e` flags
- Invalid flag, invalid regex, stdin with `-`
- JSON no-match

## Verification
```
go test -race -coverprofile=coverage.out ./pkg/grep/
# 85.3% coverage, all 47 tests pass
CGO_ENABLED=0 go build ./...     # builds clean
go vet ./pkg/grep/               # no warnings
```

## Files Changed
- `pkg/grep/grep.go` — refactored run→grepRun, added flags/context, bug fixes
- `pkg/grep/grep_test.go` — 29 new dispatch-call tests + kept 18 library-layer tests
