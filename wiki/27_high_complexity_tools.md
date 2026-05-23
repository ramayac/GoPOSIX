# Phase 27 — High Complexity & Privileged Utilities (Tier 5)

> **Version:** 1.0 | **Date:** 2026-05-23 | **Status:** PLANNING
>
> **Analysis:** 11 High-Complexity / Privileged Utilities (Tier 5)

This document catalogs the final tier of unimplemented BusyBox-tested utilities in **GoPOSIX**. It outlines the requirements, architectural considerations, and precise Go-native implementation strategies needed to implement them with full POSIX and BusyBox parity.

---

## 📋 Tier 5 Utilities Deep-Dive & Implementation Strategies

### 1. Compression & Archival Component

#### 📦 **`ar`** (Archive utility)
* **BusyBox Test Suite**: `ar.tests` (e.g. creating, listing, and extracting archive files).
* **POSIX/GNU Requirements**: Support standard UNIX archive format (`!<arch>\n`), standard fixed-width header records (name, modification time, owner/group, mode, size), and actions `-t` (list), `-x` (extract), `-r` (insert/replace).
* **Implementation Strategy**:
  * Write a pure Go-native LFS (Library Format Specification) byte reader/writer.
  * Avoid external packages. The `.a` archive header is simple 60-byte ASCII padding.
  * Integrate fully with injectable `io.Reader`/`io.Writer` and structured `--json` formatting.

#### 📁 **`cpio`** (Copy archives in/out)
* **BusyBox Test Suite**: `cpio.tests` (e.g., extracting archives, filtering, listing).
* **POSIX/GNU Requirements**: Support standard cpio archive format types: binary, old ASCII (`070707`), new ASCII SVR4/ODC (`070701`, `070702`), and modes `-o` (create), `-i` (extract), `-t` (list), `-d` (make directories).
* **Implementation Strategy**:
  * Parse standard cpio headers (reading magic numbers like `070701`).
  * Process relative path paths, extracting files with matching permissions, resolving hard/symbolic links.
  * Package should achieve `>=80%` coverage by parsing in-memory cpio stream tables.

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

#### 📡 **`rx`** (XMODEM file receiver)
* **BusyBox Test Suite**: `rx.tests` (verifies protocol integrity, handshakes, and errors).
* **POSIX/GNU Requirements**: Non-interactive file receiver using the classic XMODEM protocol (standard 128-byte packets and XMODEM-1K/1024-byte packets) with standard Checksum and CRC-16 integrity verification.
* **Implementation Strategy**:
  * Write XMODEM packet parsing and state machine:
    * Standard Control Characters: `SOH` (0x01), `STX` (0x02), `EOT` (0x04), `ACK` (0x06), `NAK` (0x15), `CAN` (0x18).
    * CRC calculations over byte arrays.
  * Rely entirely on injectable `io.Reader`/`io.Writer` streams to easily mock UART/Serial port behavior in Go unit tests.

---

### 3. Mathematics & Calculators

#### 🧮 **`bc`** & **`dc`** (Arbitrary-precision calculators)
* **BusyBox Test Suite**: `bc.tests` and `dc.tests` (e.g., executing complex scripts, scale calculations, trigonometry, and variables).
* **POSIX/GNU Requirements**:
  * **`dc`**: Stack-based, Reverse-Polish Notation (RPN) calculator.
  * **`bc`**: Interactive, C-like calculator language with variables, arrays, custom functions, control statements (`if`, `for`, `while`), and floating-point scale limits.
* **Implementation Strategy**:
  * Standard math operations will be bridged to Go's standard `math/big` package (`big.Float`, `big.Int`, `big.Rat`) for unlimited precision.
  * **`dc`**: Implement a stack machine that reads character sequences, updates standard registers, and handles math bounds.
  * **`bc`**: Implement a token scanner and an AST parser that evaluates expressions or parses mathematical routines into an execution scope.

---

### 4. Shell & Process Component

#### 🐚 **`ash`** (Standard Command Shell alias)
* **BusyBox Test Suite**: `ash.tests`.
* **POSIX/GNU Requirements**: Standard POSIX sh syntax interpreter.
* **Implementation Strategy**:
  * GoPOSIX already features a 100% Go-native POSIX shell interpreter (`pkg/shell`) powered by `mvdan.cc/sh/v3`.
  * To resolve `ash.tests` integration immediately, register `"ash"` as a dispatch command alias in `cmd/goposix/main.go` that maps directly to the existing `shell` / `sh` execution entry point!

---

### 5. System Administration & Hardware Component

#### 🔌 **`mdev`** (Micro-udev daemon)
* **BusyBox Test Suite**: `mdev.tests` (simulates hotplug events and directory generation).
* **POSIX/GNU Requirements**:
  * Scan `/sys/class` and `/sys/block` on startup to populate `/dev` with hardware devices.
  * Listen to kernel netlink hotplug events, reading event environment variables (like `ACTION`, `DEVPATH`, `SUBSYSTEM`).
  * Parse `/etc/mdev.conf` configurations to dynamically set permissions, ownership, and run target scripts on device updates.
* **Implementation Strategy**:
  * Go-native netlink socket reader utilizing the standard library `syscall` or `golang.org/x/sys/unix`.
  * Use regular expressions (`regexp`) to match `/etc/mdev.conf` lines, and call `unix.Mknod` with dynamic major/minor values.

#### 💽 **`mkfs.minix`** (Minix filesystem generator)
* **BusyBox Test Suite**: `mkfs.minix.tests`.
* **POSIX/GNU Requirements**: Build a V1 or V2 Minix filesystem on a target device or image file, writing appropriate Superblocks, Inode bitmaps, Zone bitmaps, Inodes, and root directory directories.
* **Implementation Strategy**:
  * Define Go struct equivalents for Minix filesystem blocks (`MinixSuperBlock`, `MinixInode`).
  * Pack structures into binary streams (`encoding/binary`) and write sequentially to target block devices or mock files.

#### 💾 **`mount`** (Mount filesystems)
* **BusyBox Test Suite**: `mount.tests` (requires root privileges).
* **POSIX/GNU Requirements**: Mount partition block devices to directory trees supporting filesystem types (`-t`) and custom option lists (`-o` e.g., `ro`, `rw`, `noexec`, `nosuid`).
* **Implementation Strategy**:
  * Bridge flag strings and path directories directly to the Linux standard `unix.Mount` system call.
  * Mock system calls in unit tests to verify proper flag parsing and error boundaries without requiring root permissions during development builds.
