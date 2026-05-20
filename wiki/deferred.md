# Deferred Items

> **Date:** 2026-05-19 | **Summary of all deferred/planned future work for GoPOSIX**

---

## XML Output (`--xml`)

Structured XML envelope mirroring the existing `--json` format. The JSON envelope
already covers all machine-readable use cases. XML would require ~2,000 LOC across
77 utilities + an XML test suite. No consumer has requested it.

**Status:** DEFERRED (design complete, not implemented)

---

## Multi-Tenant Sandbox (Phase 23)

Per-session filesystem isolation, command allowlists, resource quotas, and subprocess
sandboxing. Deferred because the primary use case is single-tenant, multi-agent
collaboration on a shared filesystem. Workspace isolation would break the shared-repo
model. Will revisit when an untrusted-user use case emerges.

**Status:** DEFERRED

---

## Multi-Agent Observability (Phase 24)

**Doc:** [24_multi_agent_observability.md](24_multi_agent_observability.md) — concrete design, ~376 lines

Agent-aware sessions, per-agent audit trail, file-level read/write tracking, and
per-agent metrics. Single-tenant but multi-agent — all agents share the same filesystem
but need attribution for debugging. This is the most likely next active phase.

**Status:** PLANNING

---

## Daemon Stdin Support (Phase 25 — ACTIVE)

**Doc:** [25_daemon_stdin.md](25_daemon_stdin.md)

Add a `stdin` field to the JSON-RPC `GoposixParams` struct so the Go SDK can pass
input to stdin-consuming utilities (grep, sed, sort, wc, tr, head, tail, cut, tee,
uniq, fold, expand, nl, paste, join, comm, diff, patch, cksum, md5sum, sha256sum,
sum, od, strings, and others — 40+ utilities total).

The utilities already accept injectable `io.Reader` for stdin via their `*Run()`
variants (e.g., `catRun(args, out, errOut, stdin)`). The gap is purely in the
dispatch interface (`Command.Run` has no stdin parameter) and the daemon protocol
(`GoposixParams` has no `stdin` field).

**Performance note:** Not a performance play. JSON-RPC round-trip overhead cancels
any savings from avoiding temp files for small inputs. The benefit is capability:
40+ utilities are unreachable via the Go SDK today because they require stdin.

**Status:** PLANNING (implementation in `feat/daemon-stdin`)

---

## Daemon Pipeline Composition (Phase 26 — PLANNED)

New `goposix.pipe` RPC method that accepts a pipeline specification (array of
commands with flags) and wires them together server-side via `io.Pipe()`. Enables
single-RPC execution of chains like `tar cf - . | gzip > /tmp/out.tar.gz` where
intermediate data is too large to serialize through JSON-RPC.

**Performance note:** In-process chaining in the Go SDK (e.g., `c.Ls()` → filter
in Go → `len(results)`) is faster than server-side pipes for most cases. This
feature exists for (a) pipelines with prohibitively large intermediate data and
(b) drop-in compatibility when porting shell scripts to the SDK.

**Status:** PLANNING (deferred to `feat/daemon-pipeline` after stdin land)

---

## Known Limitations (Won't Fix)

| Issue | Root Cause |
|-------|------------|
| `date` TZ parsing (3 BusyBox failures) | Go `time` package doesn't parse POSIX TZ strings |
| `fold` NUL handling (1 BusyBox failure) | Echo harness `\0` escape limitation |
| Go `regexp` ≠ POSIX BRE | RE2 engine, no backreferences — documented, by design |
