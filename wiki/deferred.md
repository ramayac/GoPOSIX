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

## Known Limitations (Won't Fix)

| Issue | Root Cause |
|-------|------------|
| `date` TZ parsing (3 BusyBox failures) | Go `time` package doesn't parse POSIX TZ strings |
| `fold` NUL handling (1 BusyBox failure) | Echo harness `\0` escape limitation |
| Go `regexp` ≠ POSIX BRE | RE2 engine, no backreferences — documented, by design |
