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
* **Reference Issue:** [wiki/hardening.md](hardening.md) §H6
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

### Smart CLI Forwarding (Milestone M5)
* **Status:** DEFERRED
* **Reference Document:** [wiki/hardening.md](hardening.md)
* **Details:**
  Adds dynamic entry point routing to the multicall binary via `forwarder.go`. When executing a command, the CLI binary checks for an active daemon socket (e.g., `/var/run/goposix.sock`). If the socket is active, the CLI bypasses Go runtime cold-start/dispatch latency by automatically and transparently forwarding the command, environment, and standard I/O streams directly to the running daemon.
  * **Current Status:** The core detection and JSON-RPC forwarding logic is fully implemented in `forwarder.go` at the repository root. However, wiring this routing mechanism into the main entry point `cmd/goposix/main.go` remains deferred to avoid routing complications in custom environments and wait-loop synchronization issues during initial startup.

---

### StopAtFirstNonFlag Integration for `echo`/`printf`
* **Status:** DEFERRED
* **Reference Document:** [wiki/23_flags_rewrite.md](23_flags_rewrite.md)
* **Details:**
  Refactoring the custom `echo` and `printf` commands to utilize the standard POSIX-compliant flag parser (`common.ParseFlags`) instead of using specialized manual argument scanning loops.
  * **Target Architecture:** Leveraging the newly added `FlagSpec.StopAtFirstNonFlag` capability of the unified zero-allocation flag scanner. This will ensure standard flag parsing conformance and eliminate bespoke argument parsing code from utility codebases. Currently deferred because the existing custom manual loops are robust and fully pass the BusyBox compliance suite.

---

### CGroups v2 Per-Session Isolation
* **Status:** DEFERRED
* **Reference Document:** [wiki/observability_exports.md](observability_exports.md)
* **Details:**
  Leveraging Linux cgroups v2 control group structures to isolate and restrict resource usage (CPU, memory, tasks, and I/O) on a per-session basis for daemon operations.
  * **Target Architecture:** Move each daemon execution session into a dedicated cgroup (e.g., `/sys/fs/cgroup/goposix.session.<id>`). This allows system administrators to monitor resource usage via tools like `systemd-cgtop` or enforce strict resource quotas.
  * **Challenges / Blockers:** Highly complex cgroup management lifecycle (directories creation, PID migration, cleanup), requires system root privileges or pre-delegated user subtrees (not viable in most container configurations), and lacks portability (Linux cgroups v2 only). Deferred until Phase 23 (Multi-Tenant Sandbox Confinement) is revisited.

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
