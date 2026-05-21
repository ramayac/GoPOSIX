# Deferred & Future Work

This document serves as the single canonical registry for all active planning phases, deferred architectural enhancements, completed transitions, and documented engine limitations for GoPOSIX.

---

## 📅 Active Planning & Future Phases

### Multi-Agent Observability (Phase 24)
* **Status:** PLANNING
* **Reference Document:** [wiki/24_multi_agent_observability.md](24_multi_agent_observability.md)
* **Details:**
  Adds agent-aware sessions, per-agent audit trails, fine-grained file-level read/write tracking, and performance metrics. Designed for multi-agent collaboration environments sharing a common sandbox workspace directory, allowing developers/agents to audit who performed what filesystem mutations.

---

### Daemon Pipeline Composition (Phase 26)
* **Status:** PLANNING (Deferred to `feat/daemon-pipeline` after Phase 25/Hardening stability)
* **Details:**
  Introduces a new `goposix.pipe` RPC method. It accepts an array of command specifications (commands with flags) and chains them together server-side using standard Go `io.Pipe()`. This allows single-RPC invocation of pipeline operations (e.g., `tar cf - . | gzip > out.tar.gz`) where intermediate payloads are too large or heavy to serialize back and forth through JSON-RPC connection round-trips.

---

## ⏸️ Deferred Architectural Enhancements

### CWD Signature Refactoring (Global State Elimination)
* **Status:** DEFERRED (Workaround active)
* **Reference Issue:** [wiki/24_hardening_iv.md](24_hardening_iv.md) §H6
* **Details:**
  The shell sandbox currently uses `os.Chdir(hc.Dir)` to mutate the process working directory before execution. Since CWD is a global process state, concurrent execution could lead to directory clobbering.
  * **Current Workaround:** Thread-safety is achieved using a global `execMu sync.Mutex` in `internal/shell/interpreter.go` that serializes all executions and performs CWD save/restore.
  * **Target Architecture:** Eliminate `os.Chdir()` and the mutex bottleneck entirely by expanding the `dispatch.Command.Run` signature to accept a CWD parameter (e.g. `(args, stdin io.Reader, stdout io.Writer, stderr io.Writer, cwd string)`). This requires updating all 79 utility implementations to perform path resolutions relative to the passed `cwd` rather than using OS-level relative lookups.

---

### `dd` Structured Output (`--json`)
* **Status:** DEFERRED
* **Details:**
  POSIX standard `dd` relies on non-standard `key=value` operands (e.g., `if=input_file of=output_file bs=1M count=5`) rather than standard single/double dash flags. As a result, `dd` bypasses the unified `common.ParseFlags` entirely. Integrating `--json` support requires designing a custom output schema representing byte copy metrics and retrofitting the custom operand parser to securely support output redirection without breaking BSD/POSIX compatibility.

---

### XML Output Support (`--xml`)
* **Status:** DEFERRED
* **Details:**
  Proposed to mirror the existing structured `--json` output envelopes. However, the `--json` option already covers all programmatically driven consumer cases. Adding `--xml` would require ~2,000 lines of redundant encoding logic across 77 utilities plus full compliance test matrices with no active consumer demand.

---

### Multi-Tenant Sandbox Confinement (Phase 23)
* **Status:** DEFERRED
* **Details:**
  Provides isolated per-session virtual filesystems, command execution allowlists, strict resource quotas (memory, open file descriptors), and subprocess sandboxing. Deferred because GoPOSIX's primary target is single-tenant, multi-agent cooperative development on a shared workspace. Confinement boundaries are currently enforced globally at the session base level, which is optimal for cooperative runtimes.

---

## 🏆 Completed Transitions

### Daemon Stdin Support (Phase 25)
* **Status:** COMPLETED (Implemented & Merged)
* **Details:**
  Expanded the command registry `dispatch.Command.Run` signature to include `stdin io.Reader` and added the `GoposixParams.Stdin` field to JSON-RPC envelopes. Plumbed stdin streams directly through the daemon to 40+ stdin-consuming utilities (grep, sed, tr, wc, sort, cut, etc.). Backed by comprehensive integration and unit tests.

---

## ⚠️ Documented Engine Limitations (Won't Fix)

| Issue | Root Cause | Rationale |
|-------|------------|-----------|
| Go `regexp` ≠ POSIX BRE/ERE | Go standard `regexp` uses the RE2 engine which guarantees $O(N)$ execution time but explicitly lacks support for backreferences and lookaheads. | Essential for container security. Backreferences allow writing regexes that trigger catastrophic backtracking ($O(2^N)$ time complexity), which is a common vector for ReDoS (Regular Expression Denial of Service) attacks. Secure-by-default design choice. |

*(Note: Prior limits regarding `date` TZ parsing and `fold` trailing newlines/NUL preservation have been fully resolved by our custom POSIX parsers and stream-splitting implementations).*
