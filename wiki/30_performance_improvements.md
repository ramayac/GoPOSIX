# Phase 30 — Performance Improvements (30 Actionable Optimizations)

> **Status:** PROPOSED | **Date:** 2026-05-23 | **Author:** Performance Audit  
> **Scope:** 10 tools deeply audited (cat, ls, grep, wc, find, sort, sed, tr, cp, dd) + common infrastructure (flags, output, dispatch, daemon, client SDK)  
> **Goal:** Reduce latency per call, memory allocations, and I/O overhead across the entire GoPOSIX stack

---

## Executive Summary

The GoPOSIX codebase is functionally excellent (96.9% BusyBox pass rate, 80%+ coverage) but has significant performance headroom. The primary performance losses come from:

1. **No buffered writers anywhere** — every `fmt.Fprintf(stdout, ...)` call triggers a syscall per line
2. **No `sync.Pool` usage** — zero buffer reuse across daemon requests (60µs target means every allocation matters)
3. **`fmt.Sprintf` / `fmt.Fprintf` in hot loops** — these use reflection internally and allocate on every call
4. **Double JSON serialization in daemon** — command outputs get serialized to `JSONEnvelope`, then re-parsed, then re-serialized to `RPCResponse`
5. **Hardcoded `os.Stderr`/`os.Stdin` in ~50+ packages** — prevents daemon I/O routing and leaks to host stderr

Estimated combined impact: **2–5× improvement** in daemon RPC latency and **3–10× improvement** in text processing throughput for grep, sort, wc, sed, tr.

---

## Classification

| Severity | Meaning | Impact |
|:--------:|---------|--------|
| 🔴 **Critical** | Directly impacts daemon 60µs target or 5× benchmark gaps | Immediate priority |
| 🟡 **High** | Significant throughput or memory improvement | Next sprint |
| 🟢 **Medium** | Measurable but smaller impact | When touching the code |
| 🔵 **Low** | Marginal or long-term architectural | Backlog |

---

## Improvement 1: Add Buffered Writers to All Output-Heavy Utilities  
**Severity:** 🔴 Critical | **Tools:** ALL (especially grep, sort, cat, ls, wc, find, sed)  
**File:** Every `pkg/<tool>/<tool>.go` that calls `fmt.Fprintln(stdout, ...)`

### Problem
Not a single utility in GoPOSIX wraps the output `io.Writer` in a `bufio.Writer`. Every `fmt.Fprintln(stdout, line)` triggers an individual `write()` syscall. For grep outputting 10,000 matches, that's 10,000 syscalls. BusyBox uses stdio which buffers to 8KB by default.

### Current Code (grep, representative)
```go
// grep.go:526 — called once per match
fmt.Fprintf(stdout, "%s%s\n", prefix, m.Text)
```

### Fix
```go
func grepRun(args []string, stdout, errOut io.Writer, stdinR io.Reader, cwd string) int {
    bw := bufio.NewWriterSize(stdout, 32*1024) // 32KB buffer
    defer bw.Flush()
    // ... use bw everywhere instead of stdout
```

### Impact
For 10K matches: ~10,000 → ~300 write syscalls. Expected **5–30×** throughput improvement for line-heavy output.

---

## Improvement 2: Eliminate Double JSON Serialization in Daemon  
**Severity:** 🔴 Critical | **File:** `internal/daemon/server.go:644–728`

### Problem
The daemon's `processRequest()` captures command output into a `bytes.Buffer`, the command writes a `JSONEnvelope` into it (via `json.NewEncoder`), then the daemon `json.Unmarshal`s the buffer, extracts `.Data`, and re-serializes it into an `RPCResponse`. This is **three JSON operations** (marshal → unmarshal → marshal) per RPC call.

### Current Flow
```
Command → json.Encode(JSONEnvelope) → bytes.Buffer → json.Unmarshal → extract .Data → json.Encode(RPCResponse)
```

### Fix: Direct Data Passing
Add a "daemon mode" where commands return their `data` as a Go `interface{}` directly, bypassing the intermediate JSON encoding:

```go
type Command struct {
    Name  string
    Run   func(args []string, stdin io.Reader, stdout, stderr io.Writer, cwd string) int
    RunDirect func(args []string, ...) (interface{}, int) // NEW: returns structured data
}
```

The daemon calls `RunDirect` when available, skipping the buffer entirely:

```
Command → return data interface{} → json.Encode(RPCResponse{Result: data})
```

### Impact
Eliminates 2 of 3 JSON operations per call. At the current 60µs/call, this could shave **15–25µs** (25–40% of total latency) since JSON marshaling dominates.

---

## Improvement 3: `sync.Pool` for `bytes.Buffer` in Daemon Request Processing  
**Severity:** 🔴 Critical | **File:** `internal/daemon/server.go:644`

### Problem
Every RPC request allocates a new `bytes.Buffer` for command output:
```go
var buf bytes.Buffer
lw := &common.LimitWriter{W: &buf, Limit: 50 * 1024 * 1024}
```
With 16,000 requests/sec (target), this is 16K allocations/sec + GC pressure.

### Fix
```go
var bufPool = sync.Pool{
    New: func() interface{} { 
        return bytes.NewBuffer(make([]byte, 0, 4096)) 
    },
}

// In processRequest:
buf := bufPool.Get().(*bytes.Buffer)
buf.Reset()
defer bufPool.Put(buf)
```

### Impact
Near-zero allocation for typical responses. Reduces GC pauses under sustained load by ~30%.

---

## Improvement 4: `sync.Pool` for `json.Encoder` / `json.Decoder` in Daemon  
**Severity:** 🟡 High | **File:** `internal/daemon/server.go`

### Problem
`json.NewEncoder(conn)` and `json.NewDecoder(conn)` are created per request and per response write. Each allocates internal buffers.

### Current Code
```go
// server.go:377 — called per request
enc := json.NewEncoder(conn)
enc.Encode(res)
```

### Fix
Since JSON encoders/decoders hold a reference to their `io.Writer`/`io.Reader`, pool the buffer and reuse:
```go
var encoderBufPool = sync.Pool{
    New: func() interface{} { return bufio.NewWriterSize(nil, 4096) },
}
```
Or use pre-allocated byte slices with `json.Marshal` + direct `conn.Write`.

---

## Improvement 5: Replace `fmt.Sprintf` with `strconv.AppendInt` in Hot Paths  
**Severity:** 🟡 High | **Tools:** cat, wc, grep, ls, sort

### Problem
`fmt.Sprintf("%6d\t", lineNum)` uses reflection internally. In a tight loop processing millions of lines, this is measurable overhead.

### Current Code (cat.go:87)
```go
prefix = fmt.Sprintf("%6d\t", lineNum)
```

### Fix
```go
var numBuf [20]byte
b := strconv.AppendInt(numBuf[:0], int64(lineNum), 10)
// Pad to 6 chars + tab
prefix = string(pad6(b)) + "\t"
```

Or use a pre-allocated `[]byte` builder pattern:
```go
buf := make([]byte, 0, 8)
buf = append(buf, "     "[:6-len(strconv.Itoa(lineNum))]...)
buf = strconv.AppendInt(buf, int64(lineNum), 10)
buf = append(buf, '\t')
```

### Impact
~3× faster per line number formatting. Matters for `cat -n` on million-line files.

---

## Improvement 6: wc `CountProper()` — Read Bytes, Not Runes  
**Severity:** 🟡 High | **File:** `pkg/wc/wc.go:102-139`

### Problem
`CountProper()` calls `reader.ReadRune()` which does 1–4 byte reads per character. For counting words and lines, a bulk byte-level approach (like the existing `Count()`) is dramatically faster. The `Count()` function at line 38 is actually closer to optimal but is **not used** — `CountProper()` at line 102 is the one called from `run()`.

### Current Code
```go
// line 188 — uses the slow rune-by-rune reader
res, err := CountProper(f)
```

### Fix: Hybrid Approach
Use the bulk `Count()` approach with proper UTF-8 tracking for the `chars` counter only when `-m` is requested:
```go
func CountFast(r io.Reader, needChars bool) (WcResult, error) {
    buf := make([]byte, 32*1024)
    for {
        n, err := r.Read(buf)
        chunk := buf[:n]
        res.Bytes += n
        res.Lines += bytes.Count(chunk, newline)
        // Count words using byte-level state machine (no rune conversion)
        for _, b := range chunk {
            isSpace := b == ' ' || b == '\t' || b == '\n' || b == '\r'
            if isSpace { inWord = false } 
            else if !inWord { inWord = true; res.Words++ }
        }
        if needChars {
            res.Chars += utf8.RuneCount(chunk)
        }
    }
}
```

### Impact
**10–50× faster** for wc on large files. BusyBox currently beats GoPOSIX wc by 1.3×; this would reverse that.

---

## Improvement 7: grep — Use `bytes.Contains` Instead of `strings.Contains` for Fixed-String Mode  
**Severity:** 🟡 High | **File:** `pkg/grep/grep.go:56–113`

### Problem
In `grep -F` (fixed-string) mode, `Run()` calls `scanner.Text()` which allocates a new string per line, then `strings.Contains()`. For bulk text, operating on `[]byte` directly avoids string allocation.

### Fix
```go
for scanner.Scan() {
    lineNum++
    lineBytes := scanner.Bytes() // zero-copy reference
    
    if fixed {
        for _, pat := range fixedPatternsBytes { // pre-converted to []byte
            if bytes.Contains(lineBytes, pat) {
                matchFound = true
                break
            }
        }
    }
    // Only convert to string when actually outputting the line
```

### Impact
Eliminates one string allocation per line. For 100MB grep with 4M lines = 4M fewer allocations.

---

## Improvement 8: grep `-r` — Parallelize Recursive Directory Walk with Goroutines  
**Severity:** 🟡 High | **File:** `pkg/grep/grep.go:274–303`

### Problem
`grep -r` walks directories sequentially with `filepath.Walk()`. The benchmark shows BusyBox is **22× faster** for recursive grep. While BusyBox has its own optimizations, goroutine parallelism can close this gap significantly.

### Fix
```go
if recursive {
    // Walk to collect files
    var files []string
    filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
        if !d.IsDir() { files = append(files, p) }
        return nil
    })
    
    // Process files in parallel
    sem := make(chan struct{}, runtime.NumCPU())
    var mu sync.Mutex
    var wg sync.WaitGroup
    for _, f := range files {
        wg.Add(1)
        sem <- struct{}{}
        go func(path string) {
            defer func() { <-sem; wg.Done() }()
            matches := grepFile(path, re, ...)
            mu.Lock()
            allMatches = append(allMatches, matches...)
            mu.Unlock()
        }(f)
    }
    wg.Wait()
}
```

### Impact
On multi-core systems: **4–8×** improvement for recursive grep. This directly addresses the 22× BusyBox gap identified in benchmarks.

---

## Improvement 9: grep `scanWithContext()` — Stream Instead of Slurping All Lines  
**Severity:** 🟡 High | **File:** `pkg/grep/grep.go:553–632`

### Problem
`scanWithContext()` reads ALL lines into memory (`var allLines []string`) before processing context. For a 100MB file, this allocates ~4M strings in a slice. A ring-buffer/sliding-window approach would use O(beforeCtx) memory instead of O(N).

### Fix: Sliding Window
```go
func scanWithContext(r io.Reader, ...) []ctxLine {
    scanner := bufio.NewScanner(r)
    beforeBuf := make([]string, 0, beforeCtx)
    afterRemaining := 0
    // ... sliding window logic
```

### Impact
Memory: O(N) → O(context_size). For 100MB file: ~400MB → ~1KB.

---

## Improvement 10: sort — Pre-allocate `lineItem` Slices  
**Severity:** 🟡 High | **File:** `pkg/sort/sort.go:295–337`

### Problem
`parseLines()` appends to `items` slice without pre-allocation hint, and each `lineItem` creates 4 separate slices (keys, numVals, validNum, humanVals):
```go
item.keys = make([]string, len(keySpecs))
item.numVals = make([]float64, len(keySpecs))
item.validNum = make([]bool, len(keySpecs))
item.humanVals = make([]humanVal, len(keySpecs))
```
For 1M lines × 1 key = 4M slice headers + 4M backing arrays.

### Fix: Flat Allocation
```go
// Pre-allocate contiguous memory
type lineItemFlat struct {
    original string
    key      string      // single key case (most common)
    numVal   float64
    validNum bool
    humanVal humanVal
}
```
For multi-key, fall back to slices but pre-allocate in bulk:
```go
items := make([]lineItem, 0, estimatedLines)
keysBacking := make([]string, estimatedLines * len(keySpecs))
```

### Impact
~4× fewer allocations for single-key sorts (the common case).

---

## Improvement 11: sort — Use `bufio.Writer` for Output  
**Severity:** 🟡 High | **File:** `pkg/sort/sort.go:578–593`

### Problem
```go
for i, line := range sortedLines {
    fmt.Fprint(w, line)     // One write syscall per line
    fmt.Fprintln(w)         // Another write syscall for newline
}
```
Two syscalls per line for potentially millions of lines.

### Fix
```go
bw := bufio.NewWriterSize(w, 64*1024)
defer bw.Flush()
for _, line := range sortedLines {
    bw.WriteString(line)
    bw.WriteByte('\n')
}
```

---

## Improvement 12: tr — Write Runes via `bufio.Writer`, Not `fmt.Fprint` Per Character  
**Severity:** 🔴 Critical | **File:** `pkg/tr/tr.go:169`

### Problem
This is one of the worst performance patterns in the codebase:
```go
fmt.Fprint(w, string(outRune))  // Called per CHARACTER
```
For a 1MB file (~1M characters), this is 1M `write()` syscalls and 1M string allocations from `string(outRune)`.

### Fix
```go
func Run(r io.Reader, w io.Writer, ...) error {
    bw := bufio.NewWriterSize(w, 32*1024)
    defer bw.Flush()
    reader := bufio.NewReader(r)
    
    for {
        rn, _, err := reader.ReadRune()
        // ...
        bw.WriteRune(outRune)  // Buffered, zero allocation
    }
}
```

### Impact
**100–500×** improvement for tr. From 1M syscalls → ~30 syscalls for 1MB input.

---

## Improvement 13: tr — Cache `expandSet(set2)` Instead of Rebuilding Per Character  
**Severity:** 🟡 High | **File:** `pkg/tr/tr.go:158-159`

### Problem
Inside the per-character loop, when squeeze mode is active:
```go
if len(s2List) > 0 {
    inSqueezeSet = expandSet(set2)[outRune]  // Rebuilds the ENTIRE set every character
}
```
`expandSet()` allocates a map, iterates all character classes, expands ranges — **per character**.

### Fix
```go
// Pre-compute once before the loop:
var squeezeSet map[rune]bool
if squeezeFlag && len(s2List) > 0 {
    squeezeSet = expandSet(set2)
}
// In loop:
if squeezeFlag && squeezeSet[outRune] && outRune == lastWrite {
    continue
}
```

### Impact
From O(N × K) to O(N) where N = input chars, K = character class size. **100×** improvement for squeeze mode on large inputs.

---

## Improvement 14: sed Engine — Use `bufio.Writer` and Reduce `fmt.Fprint` Calls  
**Severity:** 🟡 High | **File:** `pkg/sed/engine.go:43–78`

### Problem
The sed engine prints via `fmt.Fprint(w, s)` and `fmt.Fprint(w, "\n")` — two write syscalls per output line. For stream editing of millions of lines, this adds up.

### Fix
Wrap the output writer in a `bufio.Writer`:
```go
func runEngineInternal(...) int {
    bw := bufio.NewWriterSize(globalOut, 32*1024)
    defer bw.Flush()
    e := &engineState{out: bw, ...}
```
And replace `fmt.Fprint(w, s)` with `bw.WriteString(s)`.

---

## Improvement 15: cat `Run()` — Use `io.Copy` for Unbuffered Pass-Through  
**Severity:** 🟡 High | **File:** `pkg/cat/cat.go:69–98`

### Problem
`cat` with `-n` or `-b` flags uses `bufio.Scanner` + `fmt.Fprintln` per line. The `visLine()` function (cat -v) also uses `strings.Builder` with `visByte()` returning `string` — allocations per byte.

The good news: plain `cat` (no flags) already uses `io.Copy` (line 162). But `cat -n` does not benefit.

### Fix for cat -n
```go
bw := bufio.NewWriterSize(w, 32*1024)
defer bw.Flush()
for scanner.Scan() {
    line := scanner.Bytes()  // zero-copy
    if numberAll {
        lineNum++
        bw.Write(padNumber(lineNum))
        bw.WriteByte('\t')
    }
    bw.Write(line)
    bw.WriteByte('\n')
}
```

### Fix for cat -v  
Replace per-byte `string(b)` allocations with a lookup table:
```go
var visByteTable [256]string
func init() {
    for i := 0; i < 256; i++ {
        visByteTable[i] = computeVisByte(byte(i))
    }
}
func visByte(b byte) string { return visByteTable[b] }
```

---

## Improvement 16: ls `ownerName()` / `groupName()` — Cache UID/GID Lookups  
**Severity:** 🟡 High | **File:** `pkg/ls/ls.go:70–84`

### Problem
`user.LookupId()` and `user.LookupGroupId()` make NSS/passwd file lookups **per directory entry**. For `ls -la` on 10,000 files (most owned by same user), that's 10,000 redundant lookups.

### Fix
```go
var (
    uidCache sync.Map // uint32 → string
    gidCache sync.Map // uint32 → string
)

func ownerName(uid uint32) string {
    if name, ok := uidCache.Load(uid); ok {
        return name.(string)
    }
    u, err := user.LookupId(strconv.Itoa(int(uid)))
    name := strconv.Itoa(int(uid))
    if err == nil { name = u.Username }
    uidCache.Store(uid, name)
    return name
}
```

### Impact
For `ls -la` on 10K files: 10,000 → ~2 NSS lookups. **100×** improvement for ls -la.

---

## Improvement 17: ls — Use `os.ReadDir` Results Directly Instead of Re-statting  
**Severity:** 🟢 Medium | **File:** `pkg/ls/ls.go:164–175`

### Problem
`os.ReadDir()` returns `DirEntry` objects that already have type info. But `buildFileInfo()` calls `os.Lstat(fullPath)` again for every entry, doubling the syscall count.

### Fix
```go
for _, e := range entries {
    info, err := e.Info()  // Uses cached DirEntry info, may avoid extra stat
    if err != nil { continue }
    // Only call os.Lstat if we need symlink target
```

---

## Improvement 18: find — Parallelize Directory Walk with Goroutines  
**Severity:** 🟡 High | **File:** `pkg/find/find.go:103`

### Problem
`filepath.WalkDir()` is single-threaded. For deep directory trees, this is a significant bottleneck. The benchmark notes "Concurrent grep/du [GOROUTINE-TODO]" (bench cat_i).

### Fix
Use `filepath.WalkDir` to discover directories, then fan out with goroutines for subdirectory processing. Or use a custom parallel walker:
```go
func parallelWalkDir(root string, fn func(path string, d fs.DirEntry)) {
    sem := make(chan struct{}, runtime.NumCPU())
    var wg sync.WaitGroup
    // Walk top-level dirs in parallel
}
```

---

## Improvement 19: dd — Use Larger Default Block Sizes  
**Severity:** 🟢 Medium | **File:** `pkg/dd/dd.go:64–68`

### Problem
Default `ibs` and `obs` are 512 bytes (traditional POSIX). For modern SSDs and memory, this results in tiny reads/writes. While the user can override with `bs=`, the default is very conservative.

### Fix
When `bs` is not explicitly set by the user, auto-detect optimal block size:
```go
if ibs <= 0 {
    ibs = 512 // POSIX default
}
// But hint in documentation and consider a "modern" default
// Or: detect if writing to disk vs pipe and adjust
```
Also: avoid allocating `padded := make([]byte, obs)` inside the loop (line 144) when `conv=sync`:
```go
// Pre-allocate once outside the loop
var padded []byte
if opts.convSync {
    padded = make([]byte, obs)
}
// Inside loop:
if opts.convSync && int64(n) < obs {
    copy(padded, buf[:n])
    for i := n; i < int(obs); i++ { padded[i] = 0 }
    outBlock = padded[:obs]
}
```

---

## Improvement 20: Daemon — Avoid Double JSON Unmarshal of Params  
**Severity:** 🟡 High | **File:** `internal/daemon/server.go:413–454, 598–634`

### Problem
`processRequest()` unmarshals `req.Params` into `GoposixParams` at line 601, then again in the deferred logging closure at line 431. That's 2 unmarshals of the same payload.

### Fix
Extract params once and reuse:
```go
var p GoposixParams
var paramsParsed bool
if len(req.Params) > 0 {
    if err := json.Unmarshal(req.Params, &p); err == nil {
        paramsParsed = true
    }
}
// Use p.SessionId in defer without re-parsing
```

---

## Improvement 21: Daemon — Pre-encode Common Error Responses  
**Severity:** 🟢 Medium | **File:** `internal/daemon/server.go:394–401`

### Problem
`writeError()` creates a new `json.Encoder`, a new `Response`, and a new `Error` struct each time. Common errors like "Rate limit exceeded" and "Invalid Request" are identical every time.

### Fix
```go
var preEncodedErrors = map[int][]byte{}

func init() {
    for code, msg := range map[int]string{
        -32700: "Parse error or request too large",
        -32600: "Invalid Request",
        -32601: "Method not found",
        -32000: "Rate limit exceeded",
        -32000: "Server busy",
    } {
        b, _ := json.Marshal(Response{JSONRPC: "2.0", Error: &Error{Code: code, Message: msg}})
        preEncodedErrors[code] = append(b, '\n')
    }
}

func (s *Server) writeError(conn net.Conn, id interface{}, code int, msg string) {
    if id == nil {
        if encoded, ok := preEncodedErrors[code]; ok {
            conn.Write(encoded)
            return
        }
    }
    // fallback to dynamic encoding for requests with IDs
}
```

---

## Improvement 22: Client SDK — Buffer RPC Writes  
**Severity:** 🟢 Medium | **File:** `pkg/client/client.go:374–387`

### Problem
`json.NewEncoder(conn)` writes directly to the Unix socket. For small requests, this is fine, but for batch requests with many items, buffering the entire encoded payload before writing reduces syscall overhead.

### Fix
```go
func (c *Client) doSingle(ctx context.Context, req interface{}, resp interface{}) error {
    // ...
    var buf bytes.Buffer
    enc := json.NewEncoder(&buf)
    enc.Encode(req)
    _, err = conn.Write(buf.Bytes())  // Single write syscall
```

---

## Improvement 23: Flag Parser — Use Map Instead of Linear Scan for Long Flags  
**Severity:** 🟢 Medium | **File:** `pkg/common/compiled.go:61–68`

### Problem
`lookupLong()` does a linear scan over `longIdx` slice. While most commands have < 15 flags (making this O(15)), tools like `grep` have 24 flags. For daemon mode parsing hundreds of requests/sec, a map lookup is constant time.

### Fix
```go
type compiledSpec struct {
    shortIdx    [128]int8
    longMap     map[string]uint8  // O(1) lookup instead of O(n)
    defs        []FlagDef
    stopAtFirst bool
}

func (cs *compiledSpec) lookupLong(name string) *FlagDef {
    if idx, ok := cs.longMap[name]; ok {
        return &cs.defs[idx]
    }
    return nil
}
```

### Impact
24-flag linear scan → O(1) map lookup. Minor per-call but multiplied by every RPC request.

---

## Improvement 24: Hardcoded `os.Stderr`/`os.Stdin` Usage in 50+ Packages  
**Severity:** 🟡 High | **Tools:** wc, basename, chgrp, chmod, chown, cksum, cp, date, df, diff, dirname, du, env, expr, hostname, join, kill, link, ln, logger, logname, md5sum, mkdir, mkfifo, mv, nice, nohup, od, printenv, printf, ps, pwd, readlink, rev, rm, rmdir, sha1sum, sha256sum, sha512sum, sleep, split, stat, tail, tar, touch, truefalse (and more)

### Problem
Many packages directly reference `os.Stderr` or `os.Stdin` instead of using the injected `stderr` and `stdin` parameters from the `Run` function signature. Example from wc.go:
```go
// wc.go:145 — uses os.Stderr instead of injected stderr
fmt.Fprintf(os.Stderr, "wc: %v\n", err)
```

### Impact
1. **Daemon mode:** Error messages go to the daemon's stderr instead of being captured and returned to the RPC client
2. **Testing:** Cannot capture error output in unit tests
3. **Correctness:** Violates the project's own architectural invariant ("pass the `out io.Writer`")

### Fix
Systematic find-and-replace across all affected packages. For wc.go:
```go
// Replace:
fmt.Fprintf(os.Stderr, "wc: %v\n", err)
// With:
fmt.Fprintf(stderr, "wc: %v\n", err)
```

---

## Improvement 25: grep — Avoid Re-Compiling Binary Detection Prefix Buffer  
**Severity:** 🟢 Medium | **File:** `pkg/grep/grep.go:347–366`

### Problem
For each file, grep allocates an 8KB buffer to check for binary content:
```go
prefixBuf := make([]byte, 8192)  // Allocated per file
```
Then it iterates byte-by-byte to check for NUL:
```go
for _, b := range prefix {
    if b == 0 { isBinary = true; break }
}
```
Then uses `io.MultiReader` to reconstruct the reader, which adds overhead.

### Fix
1. Use `sync.Pool` for the prefix buffer
2. Use `bytes.IndexByte(prefix, 0)` instead of byte-by-byte scan
3. For `grep -r` with many files, this saves thousands of allocations

```go
var binaryCheckPool = sync.Pool{
    New: func() interface{} { return make([]byte, 8192) },
}

// In the file processing loop:
prefixBuf := binaryCheckPool.Get().([]byte)
defer binaryCheckPool.Put(prefixBuf)
n, _ := io.ReadFull(r, prefixBuf)
isBinary = bytes.IndexByte(prefixBuf[:n], 0) >= 0
```

---

## Improvement 26: `common.Render()` — Avoid `SetEscapeHTML(false)` Overhead  
**Severity:** 🟢 Medium | **File:** `pkg/common/output.go:31–47`

### Problem
`Render()` creates a new `json.Encoder` per call and calls `SetEscapeHTML(false)`. The encoder then checks this flag for every string it encodes.

### Fix
Pre-encode the envelope header and use `json.Marshal` with custom escaping disabled:
```go
func Render(cmdName string, data interface{}, jsonMode bool, out io.Writer, textFn func()) {
    if jsonMode {
        env := JSONEnvelope{...}
        // Use pre-serialized fixed fields + only marshal data
        b, _ := json.Marshal(env)
        out.Write(b)
        out.Write(newline)
    } else {
        textFn()
    }
}
```

Alternatively, use `json.MarshalIndent` with a `sync.Pool`'d buffer.

---

## Improvement 27: Daemon — Rate Limiter `os.Getenv` Called Per Connection  
**Severity:** 🟢 Medium | **File:** `internal/daemon/server.go:269–282`

### Problem
`handleConn()` calls `os.Getenv("GOPOSIX_RATE_LIMIT")` and `os.Getenv("GOPOSIX_MAX_REQUEST_SIZE")` **per connection**. Environment variables don't change at runtime, so these should be read once at startup.

### Fix
```go
type Server struct {
    // ...
    rateLimit    float64  // Computed once in NewServer
    requestLimit int64    // Computed once in NewServer
}

func NewServer(socketPath string, workers int, httpAddr string) *Server {
    rateLimit := 100.0
    if limitStr := os.Getenv("GOPOSIX_RATE_LIMIT"); limitStr != "" {
        if r, err := strconv.ParseFloat(limitStr, 64); err == nil && r > 0 {
            rateLimit = r
        }
    }
    // ... store in struct
}
```

---

## Improvement 28: Benchmark Suite — Add Go-Native Micro-Benchmarks  
**Severity:** 🟡 High | **File:** `test/benchmark/bench_daemon_test.go`

### Problem
The existing Go benchmark only tests `BenchmarkDaemonEcho`, `BenchmarkDaemonLs`, and `BenchmarkCLIEcho`. Critical hot paths like grep, sort, wc, cat-with-flags, and tr are not benchmarked in Go's testing framework, which means we can't track regressions or validate improvements with `go test -bench`.

### Fix
Add comprehensive micro-benchmarks:
```go
func BenchmarkGrepFixed100MB(b *testing.B) {
    data := generateTestData(100 * 1024 * 1024)
    re := regexp.MustCompile("pattern")
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        grep.Run(bytes.NewReader(data), "", re, nil, false, false, false)
    }
}

func BenchmarkWcCount100MB(b *testing.B) { ... }
func BenchmarkSortLines1M(b *testing.B) { ... }
func BenchmarkTrTranslate1MB(b *testing.B) { ... }
func BenchmarkCatNumberedLines(b *testing.B) { ... }
func BenchmarkDaemonBatchRequest(b *testing.B) { ... }
```

Also add `-benchmem` flag to track allocations:
```makefile
bench:
    go test -bench=. -benchmem -benchtime=5s ./test/benchmark/...
```

---

## Improvement 29: cp `copyRegularFile()` — Use `sendfile()` Syscall on Linux  
**Severity:** 🟢 Medium | **File:** `pkg/cp/cp.go:103–118`

### Problem
`io.Copy(stdout, in)` uses a userspace buffer (default 32KB) to copy data. On Linux, `sendfile()` can do kernel-space zero-copy file transfer.

### Fix
```go
func copyRegularFile(src, dst string, si os.FileInfo, preserve bool) error {
    in, _ := os.Open(src)
    defer in.Close()
    out, _ := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, si.Mode())
    defer out.Close()
    
    // Try sendfile first (Linux zero-copy)
    if _, err := io.Copy(out, in); err != nil {
        // Go's io.Copy already uses sendfile on Linux when both are *os.File
        // But we should ensure we're passing *os.File, not io.Writer
        return err
    }
}
```

Note: Go's `io.Copy` already uses `sendfile` when src is `*os.File` and dst is `*os.File`. The current code already does this correctly! However, for `cp -r` with many small files, the overhead is in `os.Open/os.Create/defer Close` per file. Consider batching or pre-opening files.

---

## Improvement 30: Daemon `processRequest()` — Avoid `strings.NewReader("")` for Empty Stdin  
**Severity:** 🔵 Low | **File:** `internal/daemon/server.go:650–651`

### Problem
```go
if runStdin == nil {
    runStdin = strings.NewReader("")  // Allocates a strings.Reader per request
}
```

### Fix
```go
var emptyReader = strings.NewReader("")

// In processRequest:
if runStdin == nil {
    runStdin = emptyReader
}
```
Note: `strings.Reader` is stateful (has a read position), so reusing a single instance requires resetting or using `io.NopCloser(bytes.NewReader(nil))`:
```go
var emptyReader io.Reader = bytes.NewReader(nil) // immutable, safe for concurrent use
```

---

## Summary Table

| # | Improvement | Severity | Category | Est. Impact |
|:-:|-------------|:--------:|----------|:-----------:|
| 1 | Buffered writers for all utilities | 🔴 | I/O | 5–30× throughput |
| 2 | Eliminate double JSON serialization in daemon | 🔴 | Daemon | -25µs/call |
| 3 | `sync.Pool` for `bytes.Buffer` in daemon | 🔴 | Daemon | -30% GC |
| 4 | `sync.Pool` for JSON encoder/decoder | 🟡 | Daemon | -5µs/call |
| 5 | Replace `fmt.Sprintf` with `strconv.AppendInt` | 🟡 | CPU | 3× per format |
| 6 | wc: byte-level counting instead of rune-level | 🟡 | CPU/I/O | 10–50× |
| 7 | grep: `bytes.Contains` for fixed-string mode | 🟡 | Memory | -4M allocs/100MB |
| 8 | grep -r: parallel directory traversal | 🟡 | Concurrency | 4–8× |
| 9 | grep: streaming context instead of slurp | 🟡 | Memory | O(N) → O(ctx) |
| 10 | sort: pre-allocate `lineItem` slices | 🟡 | Memory | -4× allocs |
| 11 | sort: buffered output writer | 🟡 | I/O | 10× output |
| 12 | tr: buffered writer (per-character `fmt.Fprint`) | 🔴 | I/O | 100–500× |
| 13 | tr: cache `expandSet(set2)` | 🟡 | CPU | 100× squeeze |
| 14 | sed: buffered output writer | 🟡 | I/O | 5–15× |
| 15 | cat -n: buffered output, cat -v: lookup table | 🟡 | I/O/CPU | 5× |
| 16 | ls: cache UID/GID lookups | 🟡 | Syscall | 100× for ls -la |
| 17 | ls: use DirEntry.Info() instead of re-statting | 🟢 | Syscall | 2× fewer stats |
| 18 | find: parallel directory walk | 🟡 | Concurrency | 4–8× |
| 19 | dd: pre-allocate sync padding buffer | 🟢 | Memory | Minor |
| 20 | daemon: avoid double params unmarshal | 🟡 | CPU | -5µs/call |
| 21 | daemon: pre-encode common error responses | 🟢 | CPU | -2µs/error |
| 22 | client SDK: buffer RPC writes | 🟢 | I/O | Minor |
| 23 | flag parser: map for long flags | 🟢 | CPU | Minor |
| 24 | Fix hardcoded os.Stderr in 50+ packages | 🟡 | Correctness | Daemon errors |
| 25 | grep: pool binary detection buffer | 🟢 | Memory | -8KB/file |
| 26 | common.Render: reduce encoder overhead | 🟢 | CPU | Minor |
| 27 | daemon: read env vars once at startup | 🟢 | Syscall | -2 getenv/conn |
| 28 | Add Go-native micro-benchmarks | 🟡 | Testing | Regression tracking |
| 29 | cp: ensure sendfile() path (already works) | 🟢 | I/O | Verified |
| 30 | daemon: avoid allocating empty stdin reader | 🔵 | Memory | Micro |

---

## Recommended Implementation Order

### Sprint 1: Critical Path (Items 1, 2, 3, 12)
These four changes alone could improve daemon RPC throughput by 2× and text tool throughput by 10×:
1. **Item 12** (tr buffered writer) — quickest 100× win
2. **Item 1** (buffered writers everywhere) — systematic, high impact
3. **Item 3** (`sync.Pool` for buffers) — reduces GC pressure
4. **Item 2** (eliminate double JSON) — most architecturally complex but highest daemon impact

### Sprint 2: High-Value (Items 6, 8, 13, 16, 24, 28)
1. **Item 6** (wc fast counting) — reverses BusyBox advantage
2. **Item 8** (parallel grep -r) — addresses 22× BusyBox gap
3. **Item 13** (tr cache expandSet) — easy fix, huge impact
4. **Item 16** (ls UID/GID cache) — easy fix, huge impact
5. **Item 24** (fix os.Stderr) — correctness issue, systematic
6. **Item 28** (micro-benchmarks) — needed to validate all other improvements

### Sprint 3: Medium-Value (Items 5, 7, 9, 10, 11, 14, 15, 17, 18, 20)
Polish and refinement of individual utilities.

### Backlog (Items 4, 19, 21, 22, 23, 25, 26, 27, 29, 30)
Marginal improvements that can be done opportunistically.

---

## Verification Plan

### Automated Benchmarks
```bash
# Before implementing any changes:
go test -bench=. -benchmem -count=5 ./test/benchmark/... > baseline.txt

# After each sprint:
go test -bench=. -benchmem -count=5 ./test/benchmark/... > after.txt
benchstat baseline.txt after.txt

# Full benchmark suite:
make bench-all SCALE=1.0
```

### Manual Verification
- Run `make testsuite` to ensure no BusyBox regressions
- Run `make test` to ensure unit tests pass
- Run `pprof` CPU and memory profiles before and after:
  ```bash
  GOPOSIX_DEBUG=1 ./goposix daemon &
  go tool pprof http://localhost:6060/debug/pprof/heap
  go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30
  ```

---

## See Also

- [Performance Benchmarking Plan](19_performance_benchmarking.md) — existing benchmark infrastructure
- [Performance Quick Reference](performance.md) — how to run benchmarks
- [Architecture](architecture.md) — component layout
- [Lessons Learned](11_lessons_learned.md) — past optimization insights
