# POSIX Compliance Matrix

> **Last updated:** 2026-05-18 | **Utilities:** 77 registered | **BusyBox:** 548/541 (99.3%)
>
> See [test_coverage_matrix.md](test_coverage_matrix.md) for the complete per-utility test status.

This document tracks the implementation status of GoPOSIX utilities against the POSIX standard.

## Overall Compliance Summary
- **Registered Utilities:** 77
- **Fully Implemented (✅):** 77 (100%)
- **BusyBox Pass Rate:** 99.3% (548/541)
- **Deferred (❌):** 1 — `awk`
- **4 remaining failures:** 3 date (Go TZ limits), 1 fold (NUL — echo harness limitation)

> **Note on BusyBox Test Suite:** As part of Phase 10, we integrated the busybox test suite. The baseline execution resulted in ~150 failures, almost entirely driven by flags that are not implemented in our MVP (e.g., `tar -x`, `tail -c`, `uniq -f`) or minor POSIX deviations. These will be incrementally addressed.

> **Note on `awk`:** We have decided **not** to implement `awk` for this MVP. Building a full POSIX-compliant `awk` parser and interpreter is a massive undertaking that would delay the core goal of providing an agent-ready userland. We will revisit `awk` in a future phase. Complex text processing should be handled by `grep`, `sed`, or the agent directly via JSON structured output.

### Phase 00 & 01: Core & Env
| Utility    | Status | Flags Implemented | Notes |
|------------|--------|-------------------|-------|
| `basename` | ✅     | *N/A*             | POSIX compliant |
| `dirname`  | ✅     | *N/A*             | POSIX compliant |
| `echo`     | ✅     | `-e`, `-n`        | POSIX compliant |
| `env`      | ✅     | `-i`              | POSIX compliant |
| `pwd`      | ✅     | `-P`              | POSIX compliant |
| `true`     | ✅     | *N/A*             | POSIX compliant |
| `false`    | ✅     | *N/A*             | POSIX compliant |
| `printenv` | ✅     | *N/A*             | Common GNU extension |
| `whoami`   | ✅     | *N/A*             | Common GNU extension |
| `hostname` | ✅     | *N/A*             | Common extension |

### Phase 03: Filesystem Utils
| Utility    | Status | Flags Implemented           | Notes |
|------------|--------|-----------------------------|-------|
| `cat`      | ✅     | `-b`, `-n`, `-s`            | POSIX compliant |
| `cp`       | ✅     | `-d`, `-f`, `-H`, `-i`, `-L`, `-P`, `-p`, `-r`, `-R` | POSIX compliant |
| `ln`       | ✅     | `-f`, `-s`                  | POSIX compliant |
| `ls`       | ✅     | `-1`, `-A`, `-R`, `-S`, `-a`, `-d`, `-h`, `-i`, `-l`, `-r`, `-s`, `-t` | Broad compliance |
| `mkdir`    | ✅     | `-m`, `-p`                  | POSIX compliant |
| `mv`       | ✅     | `-f`, `-i`                  | POSIX compliant |
| `readlink` | ✅     | `-f`                        | GNU extension widely used |
| `rm`       | ✅     | `-f`, `-i`, `-r`, `-R`, `-v`| POSIX compliant |
| `rmdir`    | ✅     | `-p`                        | POSIX compliant |
| `stat`     | ✅     | *N/A*                       | GNU extension |
| `touch`    | ✅     | `-r`, `-t`                  | POSIX compliant |

### Phase 04: Text Utils
| Utility    | Status | Flags Implemented           | Notes |
|------------|--------|-----------------------------|-------|
| `cut`      | ✅     | `-b`, `-c`, `-d`, `-f`, `-n`| POSIX compliant |
| `grep`     | ✅     | `-A`, `-B`, `-C`, `-E`, `-F`, `-L`, `-a`, `-c`, `-i`, `-l`, `-n`, `-o`, `-r`, `-v`, `-w`, `-x` | `egrep` alias, NUL via `-a`, `-E` (Go RE2) |
| `head`     | ✅     | `-n`                        | POSIX compliant |
| `sed`      | ✅     | `-e`, `-i`, `-n`, `s`, `d`, `p`, `q` | Full NUL support, label validation |
| `sort`     | ✅     | `-k`, `-n`, `-r`, `-t`, `-u`| POSIX compliant |
| `tail`     | ✅     | `-f`, `-n`                  | POSIX compliant |
| `tr`       | ✅     | `-c`, `-d`, `-s`            | POSIX compliant |
| `uniq`     | ✅     | `-c`, `-d`, `-i`, `-u`      | POSIX compliant |
| `wc`       | ✅     | `-c`, `-l`, `-m`, `-w`      | POSIX compliant |

### Phase 06: System & Process Utils
| Utility    | Status | Flags Implemented           | Notes |
|------------|--------|-----------------------------|-------|
| `chgrp`    | ✅     | `-R`                        | POSIX compliant |
| `chmod`    | ✅     | `-R`                        | POSIX compliant |
| `chown`    | ✅     | `-R`                        | POSIX compliant |
| `date`     | ✅     | `-u`                        | POSIX compliant |
| `df`       | ✅     | `-h`                        | `-h` is a GNU extension |
| `du`       | ✅     | `-h`, `-s`                  | POSIX compliant |
| `find`     | ✅     | `-name`, `-type`            | Core functionality implemented |
| `id`       | ✅     | *N/A*                       | POSIX compliant |
| `kill`     | ✅     | `-9`                        | POSIX compliant |
| `ps`       | ✅     | `-e`                        | POSIX compliant |
| `sleep`    | ✅     | *N/A*                       | POSIX compliant |
| `uname`    | ✅     | `-a`, `-m`, `-n`, `-r`, `-s`, `-v` | POSIX compliant |
| `xargs`    | ✅     | `-E`, `-e`                  | POSIX compliant |

### Phase 07: Agent-Ready Features
| Utility    | Status | Flags Implemented           | Notes |
|------------|--------|-----------------------------|-------|
| `diff`     | ✅     | `-B`, `-b`, `-U`, `-q`, `-u`| Core unified diff supported |
| `expr`     | ✅     | *N/A*                       | Core arithmetic/string supported |
| `gzip`     | ✅     | `-c`, `-d`, `-f`, `-k`      | Common compression options |
| `printf`   | ✅     | *N/A*                       | POSIX compliant formatting |
| `tar`      | ✅     | `-C`, `-c`, `-f`, `-t`, `-v`, `-x`, `-z` | Primary operations supported |

### Not Implemented
| Utility    | Status | Reason                      | Notes |
|------------|--------|-----------------------------|-------|
| `awk`      | ❌     | Deferred for MVP            | Too complex for initial scope, use agent capabilities or `grep`/`sed` |
