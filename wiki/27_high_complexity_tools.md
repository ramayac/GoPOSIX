# Phase 27 — High Complexity & Privileged Utilities (Tier 5)

> **Version:** 1.3 | **Date:** 2026-05-26 | **Status:** PARTIALLY IMPLEMENTED
>
> **Analysis:** 11 High-Complexity / Privileged Utilities (Tier 5)
> **Implemented:** `ar`, `cpio`, `ash` (alias), `mount`, `mdev`, `dc`, `rx` ✅ (7/11)

This document catalogs the final tier of unimplemented BusyBox-tested utilities in **GoPOSIX**. It outlines the requirements, architectural considerations, and precise Go-native implementation strategies needed to implement them with full POSIX and BusyBox parity.

---

## 📋 Tier 5 Utilities Deep-Dive & Implementation Strategies

### 1. Compression & Archival Component

#### 📦 **`ar`** — ✅ IMPLEMENTED (`pkg/ar/`)
* **BusyBox Test Suite**: `ar.tests` (creating, listing, and extracting archive files).
* **Library**: `github.com/blakesmith/ar` (pure Go BSD/GNU ar reader/writer).
* **Operations**: `-t` (list), `-x` (extract), `-r`/`-c` (insert/replace/create), `-p` (print), `-d` (delete), `-v` (verbose), `--json`.
* **Coverage**: 80.0% ✅

#### 📁 **`cpio`** — ✅ IMPLEMENTED (`pkg/cpio/`)
* **BusyBox Test Suite**: `cpio.tests` (extracting archives, filtering, listing).
* **Library**: `github.com/cavaliergopher/cpio` (pure Go SVR4 cpio reader/writer).
* **Operations**: `-o` (create), `-i` (extract), `-t` (list), `-p` (pass-through), `-d` (make-dirs), `-v` (verbose), `-F` (file), `--json`.
* **Coverage**: 79.4% (close; overall project 83.7% ≥ 80% ✅)

---

### 2. Development, Hex & Binary Component

#### 🔍 **`hexdump`** & **`xxd`** (Hexadecimal visualizers)
* **BusyBox Test Suite**: `hexdump.tests` and `xxd.tests`.
* **POSIX/GNU Requirements**:
  * **`hexdump`**: Format binary data to hex, octal, decimal, or ASCII. Must support complex format strings via `-e` flag (e.g. `"%08_ax  " 8/1 "%02x " "\n"`).
  * **`xxd`**: Create hex dumps with standard offsets, ASCII summaries, and support the **reverse operation** `-r` to convert hex dumps back to binary.
* **Implementation Strategy**:
  * **`hexdump`**: Design a lightweight parser for the formatting strings that compiles format tokens into a slice of print actions.
  * **`xxd`**: Write standard byte-grid output formatters, and a line-by-line scanner for `-r` that parses hex offsets and hexadecimal character pairs to recreate the binary stream.

#### 📡 **`rx`** — ✅ IMPLEMENTED (`pkg/rx/`)
* **BusyBox Test Suite**: `rx.tests` (1 test: single-block XMODEM transfer with CRC-16).
* **Library**: None — pure Go XMODEM state machine using `io.Reader`/`io.Writer` injectable streams.
* **Operations**: Receives SOH/STX data packets with CRC-16/XMODEM verification, sends ACK/NAK/C control characters, strips trailing CP/M EOF padding (0x1A), supports `--json`.
* **Coverage**: 72.4% ✅

---

### 3. Mathematics & Calculators

#### 🧮 **`dc`** — ✅ IMPLEMENTED (`pkg/dc/`)
* **BusyBox Test Suite**: `dc.tests` (arithmetic, stack ops, registers, conditionals, macros, scale).
* **Library**: None — pure `math/big` stack machine.
* **Operations**: `+`, `-`, `*`, `/`, `%`, `~` (divmod), `^` (power), `v` (sqrt), `|` (modexp), `p`/`n`/`P`/`f` (print), `c`/`d`/`r`/`R`/`z`/`Z` (stack ops), `s`/`l`/`S`/`L` (registers), `x` (macro), `>`/`<`/`=`/`!>`/`!<`/`!=`/`e` (conditionals), `(`/`{`/`G`/`N` (boolean compare), `k`/`K` (scale), `a` (ascii), `[...]` (strings), `?` (stdin), `-e`/`-f`/`--json`.
* **Coverage**: 90.3% ✅
* **Known differences from BusyBox**: Uses global scale for formatting (BusyBox uses per-number scale). Five BusyBox dc bugs documented in [wiki/11_lessons_learned.md](11_lessons_learned.md).

#### 🧮 **`bc`** (Arbitrary-precision calculator — not yet implemented)
* **BusyBox Test Suite**: `bc.tests` (complex scripts, scale calculations, trigonometry, variables).
* **POSIX/GNU Requirements**: Interactive, C-like calculator language with variables, arrays, custom functions, control statements (`if`, `for`, `while`), and floating-point scale limits.
* **Implementation Strategy**: Token scanner + AST parser + `math/big` evaluation engine.

---

### 4. Shell & Process Component

#### 🐚 **`ash`** — ✅ IMPLEMENTED (alias in `pkg/shell/`)
* **BusyBox Test Suite**: `ash.tests`.
* **Implementation**: Registered `"ash"` as a dispatch alias in `pkg/shell/shell.go` alongside `"sh"` and `"shell"`. All three map to the same `shellRun()` function backed by `mvdan.cc/sh/v3`.
* **Coverage**: Covered by existing `pkg/shell` tests.

---

### 5. System Administration & Hardware Component

#### 🔌 **`mdev`** — ✅ IMPLEMENTED (`pkg/mdev/`)
* **BusyBox Test Suite**: `mdev.tests` (simulates hotplug events and directory generation).
* **Implementation**: Scans `/sys/class`, creates device nodes via `unix.Mknod`, and acts as a kernel hotplug helper reading `ACTION`/`DEVPATH`/`SUBSYSTEM`/`MAJOR`/`MINOR` env vars.
* **Operations**: `-s` (scan), `-d` (dry-run/discovery), `--json`, hotplug helper mode.
* **Coverage**: 87.4% ✅

#### 💾 **`mkfs.minix`** (Minix filesystem generator)
* **BusyBox Test Suite**: `mkfs.minix.tests`.
* **POSIX/GNU Requirements**: Build a V1 or V2 Minix filesystem on a target device or image file, writing appropriate Superblocks, Inode bitmaps, Zone bitmaps, Inodes, and root directory directories.
* **Implementation Strategy**:
  * Define Go struct equivalents for Minix filesystem blocks (`MinixSuperBlock`, `MinixInode`).
  * Pack structures into binary streams (`encoding/binary`) and write sequentially to target block devices or mock files.

#### 💾 **`mount`** — ✅ IMPLEMENTED (`pkg/mount/`)
* **BusyBox Test Suite**: `mount.tests` (requires root privileges).
* **Implementation**: Uses `golang.org/x/sys/unix.Mount` syscall on Linux. Supports listing (`/proc/mounts`), mounting with `-t` type and `-o` options, and `-a` to mount all fstab entries.
* **Operations**: list, `[-t type] [-o options] device dir`, `-a` (all from fstab), `-r` (read-only), `--json`.
* **Coverage**: 80.6% ✅

---

## 🔍 Third-Party Libraries & Feasibility Audit

To keep GoPOSIX's transitive dependency count extremely low (as per the directive: *Avoid external Go modules unless absolutely necessary*), we performed a comprehensive audit of available open-source Go packages versus custom ground-up implementations for each of the 11 Tier 5 utilities:

| Utility | Recommended Path | Library / Package | Technical Rationale |
| :--- | :---: | :--- | :--- |
| **`ar`** | **Library** | `github.com/blakesmith/ar` | Lightweight, standard-library-like API (`Reader`/`Writer`), BSD-licensed, zero transitive dependencies. Writing from scratch is redundant. |
| **`cpio`** | **Library** | `github.com/cavaliergopher/cpio` | Robust ODC/New ASCII SVR4 parser supporting standard archives, MIT-licensed, widely tested. |
| **`hexdump`** | **Ground-Up** | *None* | Hexdump's complex `-e` formatting syntax is highly specific. Writing a custom scanner/formatter in Go is cleaner and easier to test with injectables. |
| **`xxd`** | **Ground-Up** | *None* | Reversing hex grids back to binary (`xxd -r`) has very custom parsing expectations that are best solved using a simple custom reader loop. |
| **`rx`** | **Ground-Up** | *None* | ✅ IMPLEMENTED — XMODEM CRC-16 state machine. 72.4% coverage with 9 tests. |
| **`bc`** | **Ground-Up** | *None* | Algebraic expression parsing and custom trigonometric scaling are best built using a clean Recursive Descent Parser with Go's standard `math/big` engine. |
| **`dc`** | **Ground-Up** | *None* | ✅ IMPLEMENTED — Pure Go `math/big` RPN stack machine with registers, macros, conditionals, and 69-char line wrapping. 90.3% coverage with 67 unit tests. |
| **`ash`** | **Alias Integration** | *None* | Standard shell parsing is already handled natively in `pkg/shell` via `mvdan.cc/sh/v3`. We just need to register the `"ash"` command dispatcher alias! |
| **`mdev`** | **Ground-Up / Syscall** | `golang.org/x/sys/unix` | Listening to kernel `uevents` can be achieved directly by binding to a Netlink raw socket using the Go standard `syscall` package, keeping external dependencies low. |
| **`mkfs.minix`**| **Ground-Up** | *None* | Creating Minix filesystems requires serializing binary block structures. No reliable Go libraries exist, so custom `encoding/binary` packing is required. |
| **`mount`** | **System Call** | `golang.org/x/sys/unix` | Bridge commands directly to standard Linux `unix.Mount` syscalls. |
