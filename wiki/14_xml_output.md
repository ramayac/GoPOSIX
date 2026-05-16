# Phase 14 — XML Output Support

> **Status:** DEFERRED (design complete, not implemented) | **Date:** 2026-05-15 | **Depends on:** Phases 00–10 complete, 14a complete

---

## Goal

Add `--xml` flag support to every KoreGo utility, producing a structured XML
envelope that mirrors the existing `--json` / `-j` format. The two flags are mutually
exclusive — `--xml` takes precedence when both are passed.

> **Note:** `--xml` is the **only** flag for XML output. There is **no short form**
> (`-x` is reserved for future POSIX flags).

**Invariant:** XML output follows the same schema contract as JSON. A consumer that
can parse the JSON envelope must be able to parse the XML envelope with an equivalent
schema. Field names, types, presence/absence rules, and error codes are identical.

---

## Architecture

### XML Envelope

```
JSON:  {"command":"ls","version":"0.1.0","schemaVersion":"1.0","exitCode":0,"data":{...},"error":null}

XML:   <korego command="ls" version="0.1.0" schemaVersion="1.0" exitCode="0">
         <data>
           <!-- utility-specific payload, same structure as JSON -->
         </data>
       </korego>
```

Error envelope:

```
JSON:  {"command":"ls","version":"...","schemaVersion":"1.0","exitCode":2,
        "data":null,
        "error":{"code":"ENOENT","message":"No such file or directory"}}

XML:   <korego command="ls" version="..." schemaVersion="1.0" exitCode="2">
         <error>
           <code>ENOENT</code>
           <message>No such file or directory</message>
         </error>
       </korego>
```

### Implementation Strategy

Go's `encoding/xml` uses `xml:"..."` struct tags (not `json:"..."` tags). Every result
struct must carry both tag sets. The envelope uses a `DataWrapper` with `,innerxml` to
embed pre-serialized payload XML.

```go
// pkg/common/output.go

type XMLElement struct {
    XMLName       xml.Name     `xml:"korego"`
    Command       string       `xml:"command,attr"`
    Version       string       `xml:"version,attr"`
    SchemaVersion string       `xml:"schemaVersion,attr"`
    ExitCode      int          `xml:"exitCode,attr"`
    Data          *DataWrapper `xml:"data,omitempty"`
    Error         *XMLError    `xml:"error,omitempty"`
}

type DataWrapper struct {
    Inner []byte `xml:",innerxml"`
}

type XMLError struct {
    Code    string `xml:"code"`
    Message string `xml:"message"`
}
```

### Render/Error Signature Extension

Current signatures accept `jsonMode bool`. They are extended to accept `xmlMode bool` as
well. When both `jsonMode` and `xmlMode` are true, XML wins.

```go
// Before
func Render(cmdName string, data interface{}, jsonMode bool, out io.Writer, textFn func())
func RenderError(cmdName string, exitCode int, errCode, message string, jsonMode bool, out io.Writer)

// After
func Render(cmdName string, data interface{}, jsonMode, xmlMode bool, out io.Writer, textFn func())
func RenderError(cmdName string, exitCode int, errCode, message string, jsonMode, xmlMode bool, out io.Writer)
```

---

## Scope

### Packages Modified

| Layer | Count | Files |
|-------|-------|-------|
| **Foundation** | 2 | `pkg/common/output.go`, `pkg/common/output_test.go` |
| **Utilities (already have `--json`)** | **44** | Every utility `*.go` file + `*_test.go` |
| **Gap-fill utilities (missing `--json`)** | **8** | echo, sed, sleep, tee, testcmd, tr, truefalse, yes — **COMPLETED** in [14a_json_gap_fill.md](14a_json_gap_fill.md) |
| **POSIX-XML test suite** | 2 | `test/posix-xml/` (mirrors `test/posix-json/`) |
| **Total packages** | **56** | ~112 files |

### What Changes Per Utility

For each utility that already has `--json`:

1. **Struct tags** — Add `xml:"fieldName"` next to every `json:"fieldName"`
2. **FlagSpec** — Add `{Long: "xml", Type: common.FlagBool}` (no short form; `-x` is reserved)
3. **run() / Run()** — Add `xmlMode := flags.Has("xml")` after `jsonMode`
4. **Render calls** — Add `xmlMode` as the new parameter
5. **RenderError calls** — Same, add `xmlMode`
6. **Tests** — Add a `--xml` output test alongside each existing `--json` output test

Example diff for a typical utility:

```diff
 // pkg/ls/ls.go

 type FileInfo struct {
-    Name    string    `json:"name"`
-    Size    int64     `json:"size"`
+    Name    string    `json:"name"    xml:"name"`
+    Size    int64     `json:"size"    xml:"size"`
     ...
 }

 var spec = common.FlagSpec{
     Defs: []common.FlagDef{
         ...
         {Short: "j", Long: "json", Type: common.FlagBool},
+        {Long: "xml", Type: common.FlagBool},
     },
 }

 // In run():
 jsonMode := flags.Has("j")
+xmlMode  := flags.Has("x")

 // In all Render calls:
-common.Render("ls", results, jsonMode, out, func() {
+common.Render("ls", results, jsonMode, xmlMode, out, func() {
```

### Gap-Fill Utilities (8 packages) ✅ JSON Complete

**JSON gap fill is COMPLETED** (Phase 14a). These packages now have proper `--json` support.
Adding `--xml` is the next step:

| Utility | Current JSON Status | Work Needed |
|---------|-------------------|-------------|
| `echo` | Manual arg parsing | Move to FlagSpec, add result struct, add both flags |
| `testcmd` | Manual arg parsing (strips `--json` before flag parse) | Move to FlagSpec, add both flags |
| `sed` | None | Add `SedResult` type, FlagSpec, both flags, Render calls |
| `sleep` | None | Add `SleepResult` type, FlagSpec, both flags, Render calls |
| `tee` | None | Add `TeeResult` type, FlagSpec, both flags, Render calls |
| `tr` | None | Add `TrResult` type, FlagSpec, both flags, Render calls |
| `truefalse` | None | Add `BoolResult` type, FlagSpec, both flags, Render calls |
| `yes` | Explicitly documented "does not support --json" | Add `YesResult` type, FlagSpec, both flags, Render calls |

### Packages NOT Modified

| Package | Reason |
|---------|--------|
| `pkg/common/` | Only output.go changes; flags.go unchanged (`--xml` is just another FlagBool) |
| `pkg/client/` | JSON-RPC protocol layer; XML is not a wire protocol here |
| `pkg/daemon/` | Daemon CLI launcher; no output format flag needed |
| `internal/daemon/` | JSON-RPC 2.0 server; protocol is fixed |
| `internal/shell/` | Shell interpreter; structured output not applicable |
| `cmd/korego/` | Multicall dispatcher; no output format flag needed |

---

## Implementation Phases

### Phase 14.1 — Foundation (pkg/common/output.go)

- [ ] Add `XMLElement`, `DataWrapper`, `XMLError` types to `output.go`
- [ ] Add `xml Import` to output.go
- [ ] Extend `Render()` signature: `jsonMode, xmlMode bool`
- [ ] Extend `RenderError()` signature: `jsonMode, xmlMode bool`
- [ ] Implement XML marshaling path in both functions
- [ ] Add `encoding/xml` to imports
- [ ] **Mutual exclusion:** When both are true, XML wins. Add a note in the docstring.
- [ ] Write `output_test.go` tests:
  - `TestRenderXML` — basic envelope with data
  - `TestRenderErrorXML` — error envelope
  - `TestRenderXMLOverJSON` — XML wins when both true
  - `TestRenderJSONOnly` — JSON still works
  - `TestRenderTextOnly` — text mode unaffected
  - `TestRenderXMLSpecialChars` — `<`, `>`, `&` in data
  - `TestRenderXMLNestedStruct` — nested objects serialize correctly

### Phase 14.2 — Batch A: Core Utilities (18 packages)

High-ROI, most-used utilities first:

| # | Utility | Files to change |
|---|---------|----------------|
| 1 | `ls` | ls.go, ls_test.go |
| 2 | `cat` | cat.go, cat_test.go |
| 3 | `grep` | grep.go, grep_test.go |
| 4 | `find` | find.go, find_test.go |
| 5 | `sed` | sed.go, sed_test.go **(gap fill)** |
| 6 | `sort` | sort.go, sort_test.go |
| 7 | `wc` | wc.go, wc_test.go |
| 8 | `head` | head.go, head_test.go |
| 9 | `tail` | tail.go, tail_test.go |
| 10 | `tar` | tar.go, tar_test.go |
| 11 | `gzip` | gzip.go, gzip_test.go |
| 12 | `diff` | diff.go, diff_test.go |
| 13 | `cp` | cp.go, cp_test.go |
| 14 | `mv` | mv.go, mv_test.go |
| 15 | `rm` | rm.go, rm_test.go |
| 16 | `stat` | stat.go, stat_test.go |
| 17 | `echo` | echo.go, echo_test.go **(gap fill)** |
| 18 | `printf` | printf.go, printf_test.go |

**Per-utility work:**
- [ ] Add `xml:"..."` tags to all fields in the result struct(s)
- [ ] Add `{Long: "xml", Type: common.FlagBool}` to FlagSpec (no short form)
- [ ] Add `xmlMode := flags.Has("xml")` (and `jsonMode` if missing)
- [ ] Update all `common.Render()` and `common.RenderError()` calls with xmlMode
- [ ] Add `--xml` output test: run with `-x`, parse XML, verify envelope + data
- [ ] Add `-x` + `-j` test: XML wins

### Phase 14.3 — Batch B: Remaining Utilities (26 packages)

| # | Utilities |
|---|-----------|
| 1–6 | `basename`, `dirname`, `mkdir`, `rmdir`, `touch`, `readlink` |
| 7–12 | `chmod`, `chown`, `chgrp`, `ln`, `pwd`, `date` |
| 13–18 | `cut`, `tr` **(gap fill)**, `tee` **(gap fill)**, `uniq`, `du`, `df` |
| 19–26 | `env`, `printenv`, `whoami`, `hostname`, `uname`, `id`, `ps`, `kill` |

Same per-utility checklist as Phase 14.2.

### Phase 14.4 — Batch C: Gap-Fill Utilities (8 packages)

> **Full plan:** [14a_json_gap_fill.md](14a_json_gap_fill.md) — result types, FlagSpec design,
> and per-utility implementation checklist for all 8 gap-fill utilities.

These packages currently lack `--json` (or parse it manually outside `FlagSpec`).
Adding `--xml` is the right moment to add both flags properly.

| # | Utility | Current Status |
|---|---------|---------------|
| 1 | `echo` | Has `EchoResult` type, parses `--json` manually — needs FlagSpec integration |
| 2 | `testcmd` | Has `TestResult` type, strips `--json` before parsing — needs FlagSpec integration |
| 3 | `sed` | No structured output — needs `SedResult` type + both flags |
| 4 | `tee` | No structured output — needs `TeeResult` type + both flags |
| 5 | `tr` | No structured output — needs `TrResult` type + both flags |
| 6 | `sleep` | No structured output — needs `SleepResult` type + both flags |
| 7 | `truefalse` | No structured output — needs `BoolResult` type + both flags |
| 8 | `yes` | Documented as "does not support --json" — needs `YesResult` type + both flags |

Plus 4 already-JSON utilities: `expr`, `xargs`, `sha256sum`, `md5sum` (add `--xml` only).

### Phase 14.5 — Integration & Validation

- [ ] **`test/posix-xml/` directory** — Create mirror of `test/posix-json/`:
  - `test/posix-xml/xml_test.go` — daemon-based XML output validation
  - Validates XML well-formedness (`xml.NewDecoder`)
  - Validates envelope structure (required attrs: command, version, schemaVersion, exitCode)
  - Validates mutual exclusion (XML wins when both `--xml` and `--json`)
  - Tests structured output semantics (exit codes, data payloads, error envelopes)
- [ ] **Makefile:** Add `./test/posix-xml/...` to `PKG_DIRS`
- [ ] **CI gate:** Add `make test-xml` target that runs all tests with `-run TestXML`
- [ ] **Smoke test:** `make smoke-docker` extended with `--xml` variants
- [ ] **Coverage:** No regression below 50% overall
- [ ] **Compliance:** `make testsuite` must stay at 409+ passed

---

## Struct Tag Convention

Every field carrying a `json:"..."` tag receives an equivalent `xml:"..."` tag:

```go
// Before
type CatResult struct {
    Lines     []string `json:"lines"`
    LineCount int      `json:"lineCount"`
}

// After
type CatResult struct {
    Lines     []string `json:"lines"     xml:"lines"`
    LineCount int      `json:"lineCount" xml:"lineCount"`
}
```

Rules:
1. Tag name matches the JSON name exactly (camelCase)
2. `omitempty` is NOT used on XML tags (XML doesn't support it the same way)
3. Fields of type `interface{}` use `xml:",any"` or are wrapped
4. Time fields use `xml:"fieldName"` — Go's `time.Time` marshals to RFC3339 in XML

---

## XML Specifics & Edge Cases

### Special Characters

Go's `encoding/xml` handles `<`, `>`, `&`, `"`, `'` automatically via entity escaping.
Tests must verify that binary data, shell output with `<>&`, and JSON-like strings
survive round-tripping without breakage.

### Arrays / Slices

XML has no native array type. Go's `encoding/xml` serializes slices as repeated child
elements:

```xml
<data>
  <files>
    <name>README.md</name>
    <size>1024</size>
  </files>
  <files>
    <name>main.go</name>
    <size>512</size>
  </files>
</data>
```

This differs from JSON arrays but is semantically equivalent. Tests must verify correct
child-element count for multi-item results.

### Empty vs Nil

| JSON | XML |
|------|-----|
| `"data":null` | `<data></data>` or element omitted with `omitempty` |
| `"data":{}` | `<data></data>` |
| `"error":null` | element omitted |

The envelope omits `<data>` when DataWrapper is nil (via `omitempty`) and omits
`<error>` when XMLError is nil. This matches JSON behavior.

### Unicode

XML declares `encoding="UTF-8"` in the prolog. Go's `encoding/xml` emits the
standard header `<?xml version="1.0" encoding="UTF-8"?>`. All KoreGo strings
are Go-native UTF-8 — no transcoding needed.

---

## Task Summary

| Phase | Packages | Files | New Tests | Est. LOC |
|-------|----------|-------|-----------|----------|
| 14.1 Foundation | 1 | 2 | 7 | ~120 |
| 14.2 Core (Batch A) | 18 | 36 | ~90 | ~600 |
| 14.3 Remaining (Batch B) | 26 | 52 | ~78 | ~520 |
| 14.4 Gap-Fill (Batch C) | 12 | 24 | ~36 | ~300 |
| 14.5 Integration | 3 | 4 | ~12 | ~180 |
| **Total** | **60** | **~118** | **~223** | **~1,720** |

Per-utility estimate: ~10 lines of struct tag additions + ~5 lines of flag/logic = ~15 LOC
per utility × 52 utilities = ~780 LOC for the core changes. Plus tests at ~10 LOC/test.

---

## Verification

```bash
# Foundation tests
cd pkg/common && go test -run XML -v

# Per-utility XML round-trip (example)
./korego ls --xml /tmp | xmllint --format -
./korego cat --xml /etc/hostname | xmllint --format -
./korego stat --xml /etc/passwd | xmllint --format -

# Mutual exclusion
./korego ls -j --xml /tmp | head -1   # must output XML, not JSON

# POSIX-XML test suite
go test ./test/posix-xml/... -v

# All XML tests pass
go test ./... -run XML

# No regression
make ci               # coverage gate passes
make testsuite        # 409+ passed
make smoke-docker     # scratch image works
```

---

## Risks

| Risk | Mitigation |
|------|------------|
| `encoding/xml` marshals `interface{}` poorly | Use `DataWrapper` with `,innerxml` — pre-serialize data to XML bytes |
| XML tags nearly double struct tag lines | Mechanical, low-risk. Go vet catches tag typos. |
| Gap-fill utilities have no result types | Design minimal result types (1–3 fields each). Reuse patterns from similar utilities. |
| `testcmd` flag ordering is fragile | `testcmd` currently strips `--json`/`-j` before parsing. Rewrite to use proper FlagSpec with both flags. |
| `echo` has manual flag parsing | Echo doesn't use `common.ParseFlags` at all. Re-architect to use FlagSpec. |
| XML output size larger than JSON | Acceptable. XML is ~20–30% larger. Not a correctness concern. |
