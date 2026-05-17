# Phase 17 — Post-MVP Tier 3: No-BusyBox-Test Utilities

> **Status:** COMPLETED | **Date:** 2026-05-16 | **Branch:** `feat/post-mvp`
>
> **Parent:** [todos.md](todos.md) — known deviations / future work
>
> Twelve utilities with NO BusyBox test suite coverage.
> Each requires writing our own compliance baseline before implementation.
> Ordered by utility value and implementation complexity.

---

## Current State

| # | Utility | Complexity | Est. LOC | Notes |
|---|---------|------------|----------|-------|
| 17.1 | `split` | Medium | ~180 | File splitting by lines/bytes |
| 17.2 | `join` | Medium | ~200 | Relational join on sorted files |
| 17.3 | `tty` | Trivial | ~50 | `isatty()` wrapper |
| 17.4 | `link` | Trivial | ~50 | `link()` syscall wrapper |
| 17.5 | `unlink` | Trivial | ~50 | `unlink()` syscall wrapper |
| 17.6 | `mkfifo` | Trivial | ~60 | `mkfifo()` syscall wrapper |
| 17.7 | `nice` | Low | ~80 | `setpriority()` wrapper |
| 17.8 | `nohup` | Low | ~100 | SIGHUP immunity + output redirect |
| 17.9 | `logger` | Low | ~100 | Syslog message submission |
| 17.10 | `logname` | Trivial | ~40 | `getlogin()` wrapper |
| 17.11 | `who` | Low | ~120 | Parse `/var/run/utmp` |
| 17.12 | `cksum` | Low | ~120 | POSIX CRC-32 algorithm |

**Total estimated LOC:** ~1,150
**No BusyBox pass gain** (these utilities have no BusyBox tests)

None exist in `pkg/` today.

---

## Key Difference from Tier 1 & 2

Tier 3 utilities lack BusyBox test coverage, so the workflow changes:

```
RESEARCH → BENCHMARK → TEST → CODE → PASS (local)
```

1. **RESEARCH:** Read POSIX spec + GNU coreutils/BusyBox source for expected behavior
2. **BENCHMARK:** Run the GNU utility (on host system) to capture expected output
3. **TEST:** Write Go tests that encode the benchmarked behavior
4. **CODE:** Implement against those tests
5. **PASS:** Verify against both Go tests and host GNU utility

---

## 17.1 — `split`

**Purpose:** Split a file into fixed-size pieces.

**POSIX flags:** `-l N` (lines per file, default 1000), `-b N` (bytes per file),
`-a N` (suffix length, default 2), `-d` (numeric suffixes), `--filter=CMD` (pipe to command)

**Output files:** `xaa`, `xab`, `xac`, ... (or `x00`, `x01` with `-d`)

### RESEARCH → TEST → CODE

1. **BENCHMARK:** `split -l 10 /etc/passwd` on host GNU
2. **TEST:** `pkg/split/split_test.go`
   - `-l 5` line-based split
   - `-b 100` byte-based split
   - `-a 3` suffix length
   - `-d` numeric suffixes
   - Pipe-to-stdin input
3. **CODE:** `pkg/split/split.go`
   - Core: read input, write chunks to `xaa`, `xab`, ...
   - `--json`: `{"files": ["xaa", "xab", ...], "chunks": N}`
4. **PASS:** Compare output files against GNU split

---

## 17.2 — `join`

**Purpose:** Relational database-style join of two sorted files on a key field.

**POSIX flags:** `-1 FIELD` (join field in file1), `-2 FIELD` (join field in file2),
`-t CHAR` (field separator), `-a FILENUM` (unpairable lines from file N),
`-v FILENUM` (only unpairable lines), `-o FORMAT` (output format)

### RESEARCH → TEST → CODE

1. **BENCHMARK:** `join -t: -1 1 -2 3 file1 file2` on host GNU
2. **TEST:** `pkg/join/join_test.go`
   - Default join on first field
   - `-1 2 -2 1` custom fields
   - `-t :` custom delimiter
   - `-a 1` unpaired lines
   - `-v 2` only unpaired from file2
   - Unsorted input error
3. **CODE:** `pkg/join/join.go`
   - Core: merge-scan with equality predicate on key
   - `--json`: `{"records": [{"fields": [...]}]}`
4. **PASS:** Compare against GNU join

---

## 17.3 — `tty`

**Purpose:** Print the file name of the terminal connected to standard input.

**POSIX flags:** `-s` (silent — only exit code)

### RESEARCH → TEST → CODE

1. **BENCHMARK:** `tty` and `tty -s` on host
2. **TEST:** `pkg/tty/tty_test.go`
   - Connected to terminal → prints path
   - Piped input → "not a tty", exit 1
   - `-s` silent mode
3. **CODE:** `pkg/tty/tty.go`
   - Core: `term.IsTerminal(fd)` check, then `ttyname(fd)`
   - `--json`: `{"is_tty": bool, "path": "..."}` (path empty if not tty)
4. **PASS:** Verify against GNU tty

---

## 17.4 — `link`

**Purpose:** Create a hard link. Basically `ln` without the `-s` flag, but as a
standalone utility. Trivial syscall wrapper.

**POSIX:** `link FILE1 FILE2` (no flags)

### RESEARCH → TEST → CODE

1. **BENCHMARK:** `link /etc/hosts /tmp/testlink`
2. **TEST:** `pkg/link/link_test.go`
   - Create hard link, verify same inode
   - Non-existent source → error
   - Existing target → error (no `-f`)
3. **CODE:** `pkg/link/link.go`
   - Core: `os.Link(src, dst)` with error wrapping
   - `--json`: `{"source": "...", "target": "..."}`
4. **PASS:** Verify against GNU link

---

## 17.5 — `unlink`

**Purpose:** Remove a single file or symlink. Basically `rm` for one file, but
as a standalone utility. Trivial syscall wrapper.

**POSIX:** `unlink FILE` (no flags)

### RESEARCH → TEST → CODE

1. **BENCHMARK:** `unlink /tmp/testfile`
2. **TEST:** `pkg/unlink/unlink_test.go`
   - Remove file → gone
   - Remove symlink → link gone, target stays
   - Non-existent path → error
   - Directory → EISDIR error
3. **CODE:** `pkg/unlink/unlink.go`
   - Core: `os.Remove(path)` (POSIX unlink works for files and symlinks)
   - `--json`: `{"removed": "..."}`
4. **PASS:** Verify against GNU unlink

---

## 17.6 — `mkfifo`

**Purpose:** Create a FIFO (named pipe).

**POSIX flags:** `-m MODE` (permission mode, like `chmod`)

### RESEARCH → TEST → CODE

1. **BENCHMARK:** `mkfifo /tmp/testpipe && ls -l /tmp/testpipe`
2. **TEST:** `pkg/mkfifo/mkfifo_test.go`
   - Create FIFO, verify with `os.ModeNamedPipe`
   - `-m 0644` custom mode
   - Existing file → error (no `-f`)
3. **CODE:** `pkg/mkfifo/mkfifo.go`
   - Core: `syscall.Mkfifo(path, mode)` (already have `golang.org/x/sys` dep)
   - `--json`: `{"path": "...", "mode": "..."}`
4. **PASS:** Verify against GNU mkfifo

---

## 17.7 — `nice`

**Purpose:** Run a command with modified scheduling priority.

**POSIX flags:** `-n ADJUSTMENT` (niceness increment, default 10)

### RESEARCH → TEST → CODE

1. **BENCHMARK:** `nice -n 5 sleep 1` (check `echo $?`)
2. **TEST:** `pkg/nice/nice_test.go`
   - Default `nice` (increment 10)
   - `-n 5` custom adjustment
   - `-n -10` negative adjustment (may need root/cap_sys_nice)
   - Command exit code propagation
3. **CODE:** `pkg/nice/nice.go`
   - Core: `syscall.Setpriority(syscall.PRIO_PROCESS, 0, prio)` then `syscall.Exec`
   - `--json`: `{"adjustment": N, "command": [...], "exit_code": N}`
4. **PASS:** Compare behavior against GNU nice

---

## 17.8 — `nohup`

**Purpose:** Run a command immune to SIGHUP, with output redirected to `nohup.out`.

**POSIX:** `nohup COMMAND [ARG...]` (no flags)

### RESEARCH → TEST → CODE

1. **BENCHMARK:** `nohup echo hello && cat nohup.out`
2. **TEST:** `pkg/nohup/nohup_test.go`
   - Output redirects to `nohup.out` when stdout is terminal
   - Piped stdout → no redirect, pipe passes through
   - SIGHUP ignored by child
   - `nohup.out` append mode
3. **CODE:** `pkg/nohup/nohup.go`
   - Core: set `signal(SIGHUP, SIG_IGN)`, check if stdout is tty, redirect if so, `exec`
   - `--json`: `{"command": [...], "output_file": "nohup.out"}` (or null)
4. **PASS:** Compare against GNU nohup

---

## 17.9 — `logger`

**Purpose:** Submit messages to the system logger (syslog).

**POSIX flags:** `-p PRI` (priority, default `user.notice`), `-t TAG` (tag),
`-s` (also log to stderr)

### RESEARCH → TEST → CODE

1. **BENCHMARK:** `logger -t test "hello world"` and check syslog/journal
2. **TEST:** `pkg/logger/logger_test.go`
   - Basic message via Unix domain socket `/dev/log`
   - `-t mytag` custom tag
   - `-p local0.info` facility/priority
   - Pipe stdin as message
3. **CODE:** `pkg/logger/logger.go`
   - Core: connect to `/dev/log` (or fallback UDP 514), format RFC 5424 message
   - `--json`: `{"priority": "...", "tag": "...", "message": "..."}`
4. **PASS:** Verify against GNU logger

---

## 17.10 — `logname`

**Purpose:** Print the user's login name.

**POSIX:** `logname` (no flags, no operands)

### RESEARCH → TEST → CODE

1. **BENCHMARK:** `logname` on host
2. **TEST:** `pkg/logname/logname_test.go`
   - Returns login name (not effective uid name)
   - No arguments → exit 0
3. **CODE:** `pkg/logname/logname.go`
   - Core: `os.Getenv("LOGNAME")` or fallback to `user.Current()`
   - `--json`: `{"logname": "..."}`
4. **PASS:** Verify against GNU logname

---

## 17.11 — `who`

**Purpose:** Display who is logged on (parse `/var/run/utmp` or `/run/utmp`).

**POSIX flags:** `-q` (quick — names + count only), `-s` (default), `-H` (header)

### RESEARCH → TEST → CODE

1. **BENCHMARK:** `who` on host
2. **TEST:** `pkg/who/who_test.go`
   - Default format: name, terminal, time, host
   - `-q` quick mode
   - `-H` header line
   - Empty utmp handling
3. **CODE:** `pkg/who/who.go`
   - Core: read `/var/run/utmp` (`/run/utmp` on modern Linux), parse utmp structs
   - Use `golang.org/x/sys/unix` for `Utmp` type
   - `--json`: `{"users": [{"name": "...", "terminal": "...", "time": "...", "host": "..."}]}`
4. **PASS:** Compare against GNU who

---

## 17.12 — `cksum`

**Purpose:** POSIX-standard CRC-32 checksum and byte/word count.

**POSIX flags:** No flags, operates on file list or stdin.

The POSIX CRC algorithm is well-defined in IEEE Std 1003.1-2017.

### RESEARCH → TEST → CODE

1. **BENCHMARK:** `cksum /etc/hosts` on host
2. **TEST:** `pkg/cksum/cksum_test.go`
   - Known-answer CRC values for fixed inputs
   - Multi-file output format
   - Stdin input
   - Empty file → CRC 0, but correct polynomial handling
3. **CODE:** `pkg/cksum/cksum.go`
   - Core: POSIX CRC-32 polynomial (0x04C11DB7), reflected, XOR 0xFFFFFFFF
   - Output: `CRC BYTE_COUNT FILENAME`
   - `--json`: `{"files": [{"name": "...", "checksum": N, "bytes": N}]}`
4. **PASS:** Compare CRCs against GNU cksum for known files

---

## Registration Checklist (per utility)

For each 17.1–17.12:

- [x] `pkg/<name>/<name>.go` — library layer + CLI glue + `init()` → `dispatch.Register`
- [x] `pkg/<name>/<name>_test.go` — unit tests targeting ≥70% coverage
- [x] Add `_ ".../goposix/pkg/<name>"` to `cmd/goposix/main.go`
- [x] Add `./pkg/<name>/...` to `PKG_DIRS` in `Makefile`
- [x] Run `make vet test build` → clean
- [x] Manual comparison against host GNU utility for behavioral parity
- [x] Verify `--json` output against JSON schemas

---

## Execution Order

Prioritize by utility value and simplicity:

```
link → unlink → logname → tty → mkfifo → split →
nice → nohup → join → cksum → logger → who
```

Trivial syscall wrappers first (quick wins), then medium-complexity tools.

---

## Milestone 17

```
[x] 17.1 — split
[x] 17.2 — join
[x] 17.3 — tty
[x] 17.4 — link
[x] 17.5 — unlink
[x] 17.6 — mkfifo
[x] 17.7 — nice
[x] 17.8 — nohup
[x] 17.9 — logger
[x] 17.10 — logname
[x] 17.11 — who
[x] 17.12 — cksum
```

No BusyBox pass count change — these utilities aren't in the BusyBox suite.

---

## How to Verify

```bash
# Trivial wrappers
./goposix link /etc/hosts /tmp/checklink && stat /tmp/checklink
./goposix unlink /tmp/checklink
./goposix logname
./goposix tty

# Medium
echo "hello" | ./goposix split -l 1 -d
./goposix mkfifo /tmp/testpipe
./goposix nice -n 5 sleep 1
echo "test from goposix" | ./goposix logger -t goposix
./goposix cksum /etc/hosts
./goposix who

# Compare against GNU
diff <(./goposix join -t: file1 file2) <(join -t: file1 file2)
```
