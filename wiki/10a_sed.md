# Phase 10a — Sed Implementation Plan

> **HISTORICAL — COMPLETED.** This phase is done. Document retained for reference.
>
> **Timeline:** Phase 10 | **Depends on:** Phase 09

---

## Goal

Implement a fully POSIX-compliant (and BusyBox-compatible) `sed` utility for KoreGo. The current implementation in `pkg/sed` only supports a very basic `s///` command. To pass the `busybox_testsuite/sed.tests`, we need a robust parsing and execution engine capable of handling pattern spaces, hold spaces, branching, and complex addressing.

## Architecture

We need to rewrite `pkg/sed` to separate the parsing phase (building an abstract syntax tree of sed commands) from the execution phase (applying the AST to the input stream).

### 1. The AST and Parser
- **Addresses:** Support no address, single address (line number, `$`, `/regex/`), and address ranges (`addr1,addr2`). Support `+N` and `~N` GNU extensions if tested by BusyBox.
- **Commands:** 
  - Subtitution (`s`): Support custom delimiters, flags (`g`, `p`, `w`, `[NUM]`), and backreferences.
  - Deletion (`d`, `D`)
  - Printing (`p`, `P`)
  - Text insertion (`a`, `i`, `c`) with multi-line support (`\` continuations).
  - Next line (`n`, `N`)
  - Hold space operations (`g`, `G`, `h`, `H`, `x`)
  - Branching and Flow Control (`b`, `t`, `T`, `:label`, `q`)
  - Blocks (`{ ... }`)
  - Line numbers (`=`)
  - Write (`w`) and Read (`r`)

### 2. The Execution Engine
- **State Machine:**
  - `PatternSpace`: The current string being modified.
  - `HoldSpace`: The secondary buffer.
  - `LineNumber`: Across all files or per file (depending on context).
  - `Substituted`: A boolean flag for the `t`/`T` branch commands, cleared on read or `t`/`T` execution.
- **Run Loop:**
  - Read line into pattern space.
  - Evaluate commands sequentially.
  - Check address matches to determine if a command should run.
  - Handle early loop terminations (`d`, `n`, `q`).
  - Auto-print pattern space at the end of the script (unless `-n` is specified).

## Task Breakdown

### Step 1: Lexer and Parser Foundation
- [x] Create `parser.go` for `sed`.
- [x] Define the AST structs (`Instruction`, `Address`, `Command`).
- [x] Implement parsing for simple commands without addresses (`p`, `d`, `q`).
- [x] Implement address parsing (line numbers and simple regex).

### Step 2: Substitution and Delimiters
- [x] Expand `s///` parsing to support arbitrary delimiters (e.g., `s@foo@bar@`).
- [x] Implement `g`, `p`, `[NUM]`, and `w` flags for substitution.
- [x] Handle regex backreferences and escape sequences in the replacement string.

### Step 3: Execution Engine & State
- [x] Create `engine.go` with a `SedEngine` struct.
- [x] Implement pattern space and hold space memory.
- [x] Implement the main execution loop over `io.Reader`.
- [x] Implement `n` and `N` commands.

### Step 4: Flow Control and Blocks
- [x] Implement labels (`:`) and branching (`b`).
- [x] Implement test branching (`t`, `T`) and the `Substituted` flag.
- [x] Implement blocks `{ ... }` and nested execution.

### Step 5: Advanced Features & Testing
- [x] Implement text insertion (`a`, `i`, `c`).
- [x] Implement advanced address ranges (e.g., `/regex/,+2`).
- [x] Run the `test/busybox_testsuite/runtest sed` suite continuously.
- [x] Fix edge cases (e.g., empty files, no trailing newline, `-i` behavior across multiple files).
