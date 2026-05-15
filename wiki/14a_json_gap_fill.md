# Phase 14a — JSON/XML Gap Fill (8 Utilities)

> **Status:** Planning | **Date:** 2026-05-15 | **Parent:** [14_xml_output.md](14_xml_output.md)

---

## Goal

Add `--json` and `--xml` structured output to the 8 utilities that currently lack it
(or parse `--json` manually outside the `common.FlagSpec` system). This closes the last
remaining gap before full XML rollout.

---

## Gap Inventory

| # | Utility | `--json` Status | Has Result Type? | Uses FlagSpec? | Root Cause |
|---|---------|----------------|-----------------|----------------|------------|
| 1 | `echo` | Manual parsing | `EchoResult` ✅ | ❌ | Manual `for` loop over `os.Args` checking `-j`/`--json` |
| 2 | `testcmd` | Manual parsing | `TestResult` ✅ | ❌ | Strips `--json`/`-j` before passing args to parser |
| 3 | `sed` | Missing | ❌ | ✅ (no json flag) | Never implemented |
| 4 | `tee` | Missing | ❌ | ✅ (no json flag) | Never implemented |
| 5 | `tr` | Missing | ❌ | ✅ (no json flag) | Never implemented |
| 6 | `sleep` | Missing | ❌ | ✅ (empty spec) | Never implemented |
| 7 | `truefalse` | Missing | ❌ | ❌ (empty spec) | Never implemented |
| 8 | `yes` | Explicitly absent | ❌ | ✅ (empty spec) | Documented as "does not support --json" |

---

## Per-Utility Implementation

### 1. `echo` — Integrate with FlagSpec

**Current:** Echo has a manual arg loop (lines ~97–105) that scans for `-j`/`--json`.
It also manually handles `-n`, `-e`, `-E` outside FlagSpec.

**Fix:** Move all flag parsing to `common.FlagSpec`. Add a proper `spec` variable:

```go
var spec = common.FlagSpec{
    Defs: []common.FlagDef{
        {Short: "n", Type: common.FlagBool},
        {Short: "e", Type: common.FlagBool},
        {Short: "E", Type: common.FlagBool},
        {Short: "j", Long: "json", Type: common.FlagBool},
        {Long: "xml", Type: common.FlagBool},
    },
}
```

In `run()`:
```go
flags, err := common.ParseFlags(args, spec)
if err != nil {
    common.RenderError("echo", 2, "USAGE", err.Error(), false, false, out)
    return 2
}
noNewline := flags.Has("n")
escapeMode := flags.Has("e") && !flags.Has("E")
jsonMode := flags.Has("json")
xmlMode := flags.Has("xml")
words := flags.Positional
```

**Result type** (already exists, add `xml` tags):
```go
type EchoResult struct {
    Text string `json:"text" xml:"text"`
}
```

- [ ] Replace manual flag parsing with `common.ParseFlags` + `spec`
- [ ] Add `{Short: "j", Long: "json", Type: common.FlagBool}` and `{Long: "xml", Type: common.FlagBool}` to spec
- [ ] Add `xml:"text"` tag to `EchoResult.Text`
- [ ] Add `xmlMode` and pass to `common.Render`
- [ ] Update tests

### 2. `testcmd` — Integrate with FlagSpec

**Current:** `testcmd` strips `--json`/`-j` from `args` before calling the expression
parser. This is fragile — it assumes `--json` is always the first arg.

**Fix:** Use `common.ParseFlags` with a proper spec. The tricky part: `test` uses
single-dash long flags like `-n`, `-z`, `-eq` etc. which conflict with POSIX short-flag
bundling. So we must **pre-process** the args to extract `--json`/`--xml` before parsing,
similar to how `find` pre-processes `-exec`.

```go
var spec = common.FlagSpec{
    Defs: []common.FlagDef{
        {Short: "j", Long: "json", Type: common.FlagBool},
        {Long: "xml", Type: common.FlagBool},
    },
}
```

Extract `--json`/`--xml`/`-j` from args, pass the rest through:
```go
jsonMode := false
xmlMode := false
cleanArgs := make([]string, 0, len(args))
for _, a := range args {
    switch a {
    case "--json", "-j":
        jsonMode = true
    case "--xml":
        xmlMode = true
    default:
        cleanArgs = append(cleanArgs, a)
    }
}
```

**Result type** (already exists, add `xml` tags):
```go
type TestResult struct {
    Result bool `json:"result" xml:"result"`
}
```

- [ ] Extract `--json`/`-j`/`--xml` via pre-processing (keep expression args intact)
- [ ] Add `xml:"result"` tag to `TestResult.Result`
- [ ] Add `xmlMode` and pass to `common.Render`/`common.RenderError`
- [ ] Update tests

### 3. `sed` — Add Structured Output

**Current:** `sed` has a full FlagSpec with flags like `-n`, `-e`, `-f`, `-i`, `-r` but
no `--json` flag. The `run()` function processes files, calls the engine, and prints to
stdout — no structured output path.

**Result type:**
```go
type SedResult struct {
    Lines      []string `json:"lines"      xml:"lines"`
    LineCount  int      `json:"lineCount"  xml:"lineCount"`
    Changed    bool     `json:"changed"    xml:"changed"`     // -i in-place
    Scripts    []string `json:"scripts"    xml:"scripts"`     // -e scripts applied
}
```

- [ ] Add `SedResult` type with both `json` and `xml` tags
- [ ] Add `{Short: "j", Long: "json", Type: common.FlagBool}` and `{Long: "xml", Type: common.FlagBool}` to spec
- [ ] Add `jsonMode` / `xmlMode` detection in `run()`
- [ ] Collect output lines into `SedResult.Lines` (already done for text output; reuse buffer)
- [ ] Add `common.Render("sed", result, jsonMode, xmlMode, out, ...)` in json/xml paths
- [ ] Add render tests for both `--json` and `--xml`

### 4. `tee` — Add Structured Output

**Current:** `tee` has a FlagSpec with `-a`/`--append` but no `--json`. Output is
directly written to files + stdout via io.MultiWriter.

**Result type:**
```go
type TeeResult struct {
    BytesWritten int64    `json:"bytesWritten" xml:"bytesWritten"`
    Files        []string `json:"files"        xml:"files"`
}
```

- [ ] Add `TeeResult` type with both `json` and `xml` tags
- [ ] Add `{Short: "j", Long: "json", Type: common.FlagBool}` and `{Long: "xml", Type: common.FlagBool}` to spec
- [ ] Add `jsonMode` / `xmlMode` detection in `run()`
- [ ] Track `BytesWritten` via a counting wrapper around the MultiWriter
- [ ] Add `common.Render("tee", result, jsonMode, xmlMode, out, ...)` in json/xml paths
- [ ] Add render tests for both flags

### 5. `tr` — Add Structured Output

**Current:** `tr` has a FlagSpec with `-d`/`--delete`, `-s`/`--squeeze-repeats`,
`-c`/`--complement` but no `--json`. Output is direct to stdout.

**Result type:**
```go
type TrResult struct {
    Lines       []string `json:"lines"       xml:"lines"`
    LineCount   int      `json:"lineCount"   xml:"lineCount"`
    BytesIn     int64    `json:"bytesIn"     xml:"bytesIn"`
    BytesOut    int64    `json:"bytesOut"    xml:"bytesOut"`
}
```

- [ ] Add `TrResult` type
- [ ] Add `{Short: "j", Long: "json", Type: common.FlagBool}` and `{Long: "xml", Type: common.FlagBool}` to spec
- [ ] Add `jsonMode` / `xmlMode` detection
- [ ] Buffer output lines + track byte counts
- [ ] Add `common.Render` in json/xml paths
- [ ] Add tests

### 6. `sleep` — Add Structured Output

**Current:** `sleep` has an empty `FlagSpec{}`. No flags at all. No output (it just
blocks and exits).

**Result type:**
```go
type SleepResult struct {
    Duration   float64 `json:"duration"   xml:"duration"`   // seconds slept
    Requested  float64 `json:"requested"  xml:"requested"`  // requested duration
    Interrupted bool   `json:"interrupted" xml:"interrupted"` // SIGINT/SIGTERM
}
```

- [ ] Add FlagSpec with `{Short: "j", Long: "json", Type: common.FlagBool}` and `{Long: "xml", Type: common.FlagBool}`
- [ ] Add `SleepResult` type
- [ ] Track requested vs actual sleep duration
- [ ] Detect interrupt (signal caught → exit 0 with `interrupted: true`)
- [ ] Add `common.Render` in json/xml paths
- [ ] Add tests

### 7. `truefalse` — Add Structured Output

**Current:** `truefalse` has no FlagSpec, no flags, no output. It just exits.

**Result type:**
```go
type BoolResult struct {
    ExitCode int  `json:"exitCode" xml:"exitCode"`
    Value    bool `json:"value"    xml:"value"`    // true or false
}
```

- [ ] Add FlagSpec with `{Short: "j", Long: "json", Type: common.FlagBool}` and `{Long: "xml", Type: common.FlagBool}`
- [ ] Add `BoolResult` type
- [ ] In json/xml mode, render result instead of silent exit
- [ ] `true` → `{exitCode: 0, value: true}`, `false` → `{exitCode: 1, value: false}`
- [ ] Add tests

### 8. `yes` — Add Structured Output

**Current:** `yes` uses `common.FlagSpec{}` (empty). The file explicitly documents
"yes does not support --json per spec." Output is an infinite stream.

**Result type:**
```go
type YesResult struct {
    String    string `json:"string"    xml:"string"`     // the repeated string
    Count     int    `json:"count"     xml:"count"`      // total lines output
    Truncated bool   `json:"truncated" xml:"truncated"`  // true in json/xml mode (we stop early)
}
```

**Design note:** `yes` produces infinite output in text mode. In `--json`/`--xml` mode,
the structured result must be finite. Output 1 line for the envelope, or accept a `--count`
flag. For MVP: if `--json` or `--xml` is passed, output a single line and stop.

- [ ] Add FlagSpec with `{Short: "j", Long: "json", Type: common.FlagBool}`, `{Long: "xml", Type: common.FlagBool}`, and `{Short: "n", Long: "count", Type: common.FlagValue}`
- [ ] Add `YesResult` type
- [ ] In json/xml mode: output `n` lines (from `--count`) or 1 line (default), then render result
- [ ] In text mode: existing infinite-loop behavior
- [ ] Add tests

---

## Task Checklist

| # | Utility | Result Type | FlagSpec | run() Logic | Tests |
|---|---------|-------------|----------|-------------|-------|
| 1 | `echo` | `EchoResult` (exists) | Replace manual loop | Add xmlMode | Both flags |
| 2 | `testcmd` | `TestResult` (exists) | Pre-process extraction | Add xmlMode | Both flags |
| 3 | `sed` | `SedResult` | Add both flags | Buffer lines | Both flags |
| 4 | `tee` | `TeeResult` | Add both flags | Track bytes | Both flags |
| 5 | `tr` | `TrResult` | Add both flags | Buffer + count | Both flags |
| 6 | `sleep` | `SleepResult` | Add both flags | Track duration | Both flags |
| 7 | `truefalse` | `BoolResult` | Add both flags | Render result | Both flags |
| 8 | `yes` | `YesResult` | Add both + `--count` | Finite in json/xml | Both flags |

---

## Verification

```bash
# Each utility outputs valid JSON
./korego echo -j hello world | jq .
./korego test -j -n "" | jq .
./korego sed -j 's/foo/bar/' < input.txt | jq .
./korego tee -j out.txt < input.txt | jq .
./korego tr -j 'a-z' 'A-Z' < input.txt | jq .
./korego sleep -j 0.1 | jq .
./korego true -j | jq .
./korego false -j | jq .
./korego yes -j | jq .

# Each utility outputs valid XML
./korego echo --xml hello | xmllint --format -
./korego test --xml -n "" | xmllint --format -
./korego sed --xml 's/foo/bar/' < input.txt | xmllint --format -
./korego tee --xml out.txt < input.txt | xmllint --format -
./korego tr --xml 'a-z' 'A-Z' < input.txt | xmllint --format -
./korego sleep --xml 0.1 | xmllint --format -
./korego true --xml | xmllint --format -
./korego false --xml | xmllint --format -
./korego yes --xml | xmllint --format -

# Unit tests pass
go test ./pkg/echo/... ./pkg/testcmd/... ./pkg/sed/... ./pkg/tee/... \
        ./pkg/tr/... ./pkg/sleep/... ./pkg/truefalse/... ./pkg/yes/... -v
```
