# Phase 03 — Tier 2: Filesystem Utilities

> **Timeline:** Week 3–4 | **Depends on:** Phase 00, 01 | **Blocks:** Phase 04, 05

---

## Goal

Implement the 13 core filesystem utilities with full POSIX flag compatibility and `--json` output.

---

## Utilities

### 03.1 — `ls` (`pkg/ls/`)

**Most complex utility in the project.** Get it right and everything else follows.

**Library returns:** `[]FileInfo{Name, Path, Size, Mode, ModTime, IsDir, Owner, Group, Inode, Links, Target}`

**Flags:** `-a`, `-A`, `-l`, `-R`, `-h`, `-t`, `-r`, `-S`, `-1`, `-d`, `-i`, `-s`

**`--json` output:**
```json
{"command":"ls","data":{"path":"/tmp","files":[{"name":"foo.txt","size":1024,"mode":"-rw-r--r--","isDir":false}],"total":3},"exitCode":0}
```

**Tests:**
- [x] `-a` includes `.` and `..`; `-A` includes dotfiles but NOT `.`/`..`
- [x] `-laR` grouped flags work (tests POSIX flag parser)
- [x] Symlinks show target
- [x] Exit code 2 on "no such file"
- [x] Compare output against `/bin/ls` for same directory

### 03.2 — `cat` (`pkg/cat/`)

Streaming reader. Flags: `-n` (number lines), `-b` (number non-blank), `-s` (squeeze blank).

Must handle stdin (`cat -` or `cat` with no args) and pipes.

`--json` for small files: `{"lines":[...], "lineCount": N}`. For large files: NDJSON stream.

- [x] `echo hello | cat` reads stdin
- [x] `cat file1 file2` concatenates
- [x] `cat nonexistent` exits 1 with error to stderr

### 03.3 — `mkdir`

- [x] `-p` creates parents
- [x] `-m 0755` sets permissions
- [x] `--json` → `{"created": ["/path/a", "/path/b"]}`
- [x] Exit 1 if exists (without `-p`)

### 03.4 — `rmdir`

- [x] Removes empty directories only
- [x] `-p` removes parents
- [x] Exit 1 if not empty

### 03.5 — `rm`

- [x] `-r` recursive, `-f` force, `-v` verbose
- [x] **Safety: `rm -rf /` REFUSED** without `--no-preserve-root`
- [x] `--json` → `{"removed": [...], "errors": [...]}`

### 03.6 — `cp`

- [x] `-r` recursive, `-p` preserve permissions/timestamps
- [x] `-i` interactive (prompt before overwrite)
- [x] `--json` → `{"copied": [{"from":"...", "to":"..."}]}`

### 03.7 — `mv`

- [x] Rename or move, `-f` force, `-i` interactive
- [x] Cross-device move: copy + delete fallback
- [x] `--json` → `{"moved": [{"from":"...", "to":"..."}]}`

### 03.8 — `touch`

- [x] Creates file or updates mtime
- [x] `-t` specific timestamp, `-r ref` reference file's time

### 03.9 — `ln`

- [x] Hard link by default, `-s` for symbolic
- [x] `-f` force (remove existing)

### 03.10 — `stat`

- [x] Full file info display
- [x] `--json` returns comprehensive struct (mode, size, uid, gid, atime, mtime, ctime, inode, links, blocks)

### 03.11 — `readlink`

- [x] Prints symlink target
- [x] `-f` canonicalize (resolve all symlinks)

### 03.12 — `basename` / `dirname`

- [x] `basename /path/to/file.txt` → `file.txt`
- [x] `basename file.txt .txt` → `file` (suffix removal)
- [x] `dirname /path/to/file` → `/path/to`

---

## Compliance Testing

Per-utility behavior is verified through the BusyBox test suite (`make testsuite`, 479+ tests) and per-package unit tests. The previous `test/compliance/` bash scripts comparing against the host system's GNU Coreutils have been removed — the BusyBox suite provides broader, standardized coverage without depending on the CI runner's installed tool versions.

---

## Milestone 03

- [x] `korego ls --json /` returns valid JSON array of FileInfo
- [x] `korego ls -laR /tmp` output matches `/bin/ls -laR /tmp` format
- [x] `korego cat file | korego grep pattern` — pipes work
- [x] `korego rm -rf /` is refused
- [x] `korego stat --json /etc/passwd` returns full stat struct
- [x] All 13 utilities have unit tests with > 85% coverage
- [x] BusyBox test suite verifies utility behavior (479+ tests)

## How to Verify

```bash
make test
make testsuite
docker run --rm korego:dev ls --json /bin
docker run --rm korego:dev stat --json /bin/korego
```
