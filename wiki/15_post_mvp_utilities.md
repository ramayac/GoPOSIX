# Post-MVP Utilities (Phases 15–18)

> **Status:** COMPLETED | **Date:** 2026-05-17

Four phases that added the remaining POSIX utilities and hardened quality gates.

---

## Phase 15 — Tier 1: `dd` + `od` (10 BusyBox tests)

| Utility | BusyBox Tests | Notes |
|---------|:------------:|-------|
| `dd` | 6 | `if`/`of`, `bs`/`ibs`/`obs`, `count`, `skip`, `seek`, `conv`, `status` |
| `od` | 4 | `-b`, `-c`, `-x`, `-f`, `-t`, `-N`, `--traditional`, `--json` |

Both are core data inspection/transformation utilities used in pipelines and forensic
workflows. POSIX-compliant with `--json` structured output.

---

## Phase 16 — Tier 2: 9 Text & Stream Utilities (43 BusyBox tests)

| Utility | BusyBox Tests | Purpose |
|---------|:------------:|---------|
| `unexpand` | 11 | Convert spaces to tabs |
| `comm` | 9 | Compare sorted files |
| `paste` | 5 | Merge lines of files |
| `fold` | 5 | Wrap input lines |
| `sum` | 4 | Checksum + block count |
| `nl` | 4 | Line numbering |
| `expand` | 3 | Convert tabs to spaces |
| `cmp` | 1 | Byte comparison |
| `strings` | 1 | Extract printable strings |

All follow the canonical `Run()` (library) + `run()` (CLI) pattern. ~980 LOC total.

---

## Phase 17 — Tier 3: 12 Utilities (no BusyBox tests)

| Utility | Purpose |
|---------|---------|
| `split` | File splitting by lines/bytes |
| `join` | Relational join on sorted files |
| `tty` | `isatty()` wrapper |
| `link` | `link()` syscall wrapper |
| `unlink` | `unlink()` syscall wrapper |
| `mkfifo` | `mkfifo()` syscall wrapper |
| `nice` | `setpriority()` wrapper |
| `nohup` | SIGHUP immunity + output redirect |
| `logger` | Syslog message submission |
| `logname` | `getlogin()` wrapper |
| `who` | Parse `/var/run/utmp` |
| `cksum` | POSIX CRC-32 algorithm |

Implemented as functional stubs with compliance tests written against GNU coreutils
baseline. ~1,150 LOC total.

---

## Phase 18 — Quality Fixes

| Item | What was fixed |
|------|---------------|
| 18.1 | CI coverage gate: replaced stale 45% check with `make cover-gate` (70%) |
| 18.2 | CI BusyBox baseline: raised from 409 to 477 (now 548) |
| 18.3 | `patch` utility: full unified diff parser, fuzzy matching, 11 BusyBox tests passing |
| 18.4 | `egrep`/`fgrep` dispatch aliases in `pkg/grep` |
| 18.5 | Daemon coverage: 35.9% → 64.6% (+28.7%, 20 new tests) |
| 18.6–9 | Coverage ramp: `diff` +4%, `client` +1.3%, `gzip`/`cut` verified |

**Final post-Phase-18 metrics:** 77 utilities, 548 BusyBox passes (99.3%), 85 test packages,
70.4% overall coverage.
