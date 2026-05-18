# POSIX Compliance FAQ

> **Purpose:** Demystify what "POSIX-compliant" actually means, what utilities are
> mandatory vs. optional, and where GoPOSIX sits on the compliance spectrum.

---

## What Is POSIX?

**POSIX** stands for **Portable Operating System Interface**. It is a family of
standards published by the IEEE as **IEEE Std 1003.x** and adopted by ISO/IEC as
**ISO/IEC 9945**. The goal is to guarantee source-code-level portability across
Unix-like operating systems.

POSIX is **not a single document**. It is a collection of specifications that
cover different layers of the system:

| Standard              | Covers                                    |
|-----------------------|-------------------------------------------|
| POSIX.1 (1003.1)      | C API, system calls, core library funcs   |
| POSIX.2 (1003.2)      | Shell & Utilities (the command-line layer) |
| POSIX.1b              | Real-time extensions                      |
| POSIX.1c              | Threads (pthreads)                        |

When people say "POSIX-compliant userland" they almost always mean
**POSIX.2 — the Shell & Utilities volume**, which is what GoPOSIX targets.

---

## What Does POSIX.2 (Shell & Utilities) Actually Require?

The **Shell & Utilities** volume defines:

1. **The Shell Command Language** — The `sh` grammar, control flow, variable
   expansion, pipes, redirections, here-documents, etc.
2. **A set of mandatory utilities** — Programs that *must* exist on a conforming
   system (see table below).
3. **Behavioral contracts** — Each utility's required flags, exit codes, output
   format, and error handling.

### The Mandatory Utility List (POSIX.1-2017 / Issue 7)

The standard groups utilities into categories. Here is the **full mandatory list**
(not optional, not XSI-extended — *base* POSIX):

#### File & Directory
`basename`, `cat`, `cd`, `chgrp`, `chmod`, `chown`, `cksum`, `cmp`, `comm`,
`cp`, `cut`, `dirname`, `dd`, `df`, `du`, `file`, `find`, `head`, `id`, `join`,
`link`, `ln`, `ls`, `mkdir`, `mkfifo`, `mv`, `od`, `paste`, `pathchk`, `pwd`,
`rm`, `rmdir`, `tail`, `tee`, `test`, `touch`, `tr`, `tsort`, `umask`,
`uname`, `unlink`, `wc`

#### Text Processing
`awk`, `diff`, `ed`, `grep`, `patch`, `sed`, `sort`, `uniq`

#### Shell & Execution
`alias`, `bg`, `break`, `case`, `colon`, `command`, `continue`, `dot`, `echo`,
`env`, `eval`, `exec`, `exit`, `export`, `false`, `fg`, `getopts`, `hash`,
`jobs`, `kill`, `local`, `logger`, `newgrp`, `nice`, `nohup`, `printf`,
`read`, `readonly`, `return`, `set`, `shift`, `sleep`, `stty`, `time`,
`trap`, `true`, `type`, `ulimit`, `umask`, `unalias`, `unset`, `wait`,
`xargs`

#### Math / Misc
`bc`, `expr`, `fold`, `gencat`, `getconf`, `iconv`, `locale`, `localedef`,
`lp`, `m4`, `mailx`, `man`, `mesg`, `more`, `pax`, `pr`, `ps`, `renice`,
`split`, `strings`, `tabs`, `tput`, `tty`, `write`

> [!IMPORTANT]
> **Yes, `awk` is on the mandatory list.** It is not optional. A system claiming
> full POSIX.2 Shell & Utilities conformance **must** provide `awk`.

---

## Is `awk` Really Required?

**For strict POSIX.2 conformance — yes.**

`awk` is listed in the base utilities, not in an optional extension module. The
POSIX specification defines its full grammar, built-in variables (`NR`, `NF`,
`FS`, `RS`, `OFS`, `ORS`, `FILENAME`, etc.), built-in functions (`length`,
`substr`, `split`, `sub`, `gsub`, `sprintf`, `printf`, `sin`, `cos`, `exp`,
`log`, `sqrt`, `int`, `rand`, `srand`, `system`, `getline`, etc.), and the
`BEGIN`/`END` pattern-action model.

### Why Is `awk` Mandatory?

`awk` fills a critical gap that no other POSIX utility covers:

| Capability                     | Without `awk`               | With `awk`                    |
|--------------------------------|-----------------------------|-------------------------------|
| Field-based text extraction    | `cut` (fixed delimiters)    | Arbitrary patterns & logic    |
| In-line arithmetic on columns  | Requires `expr` + shell     | Native floating-point math    |
| Multi-line record processing   | Nearly impossible in `sh`   | `RS` / `getline` patterns     |
| Report formatting              | `printf` (single values)    | Columnar output with headers  |
| Pattern-action programming     | `grep` + `sed` chains       | Single coherent program       |

Many POSIX shell scripts in the wild — including system startup scripts,
package managers, and build systems — depend on `awk` for tasks that are
impractical with `sed`, `grep`, and `cut` alone. Without it, large classes of
portable scripts simply won't run.

### Can You Be "Mostly POSIX" Without `awk`?

Absolutely. And that's a perfectly valid engineering choice. The real question is
*what you're optimizing for*:

| Goal                                    | Need `awk`? |
|-----------------------------------------|-------------|
| Run arbitrary POSIX shell scripts       | **Yes**     |
| Run Docker entrypoint / init scripts    | Usually no  |
| Run CI/CD pipeline scripts              | Often yes   |
| Pass the Open Group certification       | **Yes**     |
| Provide a useful programmatic runtime   | Depends     |

---

## What Are the Levels of POSIX Conformance?

There is no formal "levels" system, but in practice there's a spectrum:

### 1. Certified POSIX (The Open Group)
A vendor pays The Open Group to run their conformance test suite (VSX) and, upon
passing, earns the right to use the **UNIX®** trademark. Examples:
- macOS (certified UNIX 03)
- IBM AIX
- HP-UX
- Oracle Solaris

> [!NOTE]
> **Linux is NOT certified POSIX.** No Linux distribution has ever gone through
> The Open Group certification. Linux is "mostly POSIX-compatible" by convention
> and extensive testing, but not formally certified.

### 2. De Facto POSIX-Compatible
The system implements the POSIX interfaces faithfully but hasn't been formally
certified. This is where Linux, FreeBSD, and most BSDs sit. This is also the
realistic target for GoPOSIX.

### 3. POSIX-Subset / POSIX-Inspired
The system implements a useful subset of POSIX utilities with compatible behavior
for the flags and features it *does* support, but explicitly omits some utilities
or features. BusyBox falls in this category — it implements most POSIX utilities
but often omits obscure flags.

### 4. POSIX-Adjacent
Tools that share names and rough semantics with POSIX but diverge in meaningful
ways. Think Plan 9 from Bell Labs, or Toybox in some edge cases.

---

## What Does GoPOSIX Need to Claim?

GoPOSIX doesn't need to be certified POSIX. Its target audience is **programmatic
consumers in containers**, not general-purpose Unix workstations. The practical
goals are:

1. **Run standard shell scripts** that automation and CI systems generate.
2. **Provide structured output** (`--json`) that goes beyond what POSIX requires.
3. **Be a self-contained runtime** in a `FROM scratch` Docker image.

For these goals, the right positioning is:

> **GoPOSIX targets POSIX.2 Shell & Utilities compatibility for all implemented
> utilities, while prioritizing the utilities most commonly used in container
> and automation contexts.**

This means:
- Every utility we *do* implement should behave identically to its POSIX
  specification (flags, exit codes, output format).
- We don't need to implement `mailx`, `lp`, `m4`, or `ed` to be useful.
- We **should** implement `awk` because it unlocks a massive class of real-world
  scripts that would otherwise fail in our environment.

---

## The POSIX Utility Tiers for GoPOSIX

Here's how the full POSIX utility set breaks down by practical importance for
a container/programmatic runtime:

### Tier: Critical (Must Have)
These are non-negotiable for running real-world scripts:

`echo`, `printf`, `true`, `false`, `test`/`[`, `env`, `pwd`, `cd`, `ls`,
`cat`, `cp`, `mv`, `rm`, `mkdir`, `rmdir`, `chmod`, `chown`, `chgrp`,
`ln`, `touch`, `head`, `tail`, `wc`, `sort`, `uniq`, `grep`, `sed`,
`cut`, `tr`, `find`, `xargs`, `tee`, `basename`, `dirname`, `sleep`,
`kill`, `ps`, `id`, `uname`, `date`, `df`, `du`, `diff`, `expr`,
`read`, `export`, `set`, `unset`

### Tier: Important (Should Have)
These appear frequently in non-trivial scripts:

`awk`, `comm`, `join`, `paste`, `split`, `fold`, `od`, `cmp`, `dd`,
`mkfifo`, `nice`, `nohup`, `renice`, `tty`, `stty`, `bc`, `file`,
`strings`, `patch`, `more`/`less`

### Tier: Niche (Nice to Have)
Rarely needed in containerized automation contexts:

`ed`, `pax`, `m4`, `mailx`, `lp`, `man`, `write`, `mesg`, `newgrp`,
`tabs`, `tput`, `pr`, `localedef`, `gencat`, `iconv`, `tsort`,
`pathchk`, `logger`, `getconf`

---

## Summary: Do We Need `awk`?

| Question                                            | Answer |
|-----------------------------------------------------|--------|
| Is `awk` required by the POSIX standard?            | **Yes** |
| Can GoPOSIX be useful without it?                    | Yes, for simple scripts |
| Will real-world scripts break without it?            | **Many will, yes** |
| Is it the hardest utility to implement?              | It's among the hardest (it's a full language) |
| Should GoPOSIX implement it?                          | **Yes**, as a capstone utility |
| Does it need to be done first?                       | No — it can come after the simpler utilities |

`awk` is to POSIX what a compiler is to a programming language — technically you
could hand-assemble everything, but nobody wants to. It's the utility that turns
a collection of simple tools into a programmable text-processing system.

---

## References

- [POSIX.1-2017 (Issue 7) — The Open Group](https://pubs.opengroup.org/onlinepubs/9699919799/)
- [IEEE Std 1003.1-2017](https://standards.ieee.org/standard/1003_1-2017.html)
- [The UNIX® Standard](https://unix.org/what_is_unix/single_unix_specification.html)
- [BusyBox utility coverage](https://busybox.net/downloads/BusyBox.html)
