# Phase 11 — Post-MVP Priorities

> **Status:** COMPLETED (11.1–11.3 done via Phase 12; 11.4 awk → [07a_awk.md](07a_awk.md)) | **Date:** 2026-05-13
> 
> Lessons learned, insights, and gotchas are documented in [11_lessons_learned.md](11_lessons_learned.md).

---

## Context

GoPOSIX MVP is complete: 49 utilities implemented, 97.9% BusyBox pass rate, daemon functional. This phase addresses the gaps that matter most for the stated goals — programmatic tooling and minimal container deployments. Items are ordered by impact on adoption.

---

## 11.1 — Formal JSON Output Schemas

**Why it matters:** The primary differentiator of GoPOSIX is machine-readable output. Without a formal schema, consumers can't rely on stable JSON shapes — defeating the purpose.

### Tasks

- [x] Define a canonical JSON schema (JSON Schema draft-07) for every utility's `--json` output (`test/schemas/`)
- [x] Add schemas and update `docs/JSON_SCHEMA.md` (now documents all 42 JSON-enabled utilities)
- [x] Add a schema validation step in CI (`make validate-schemas` runs `ajv` against golden fixtures)
- [x] Version the schema — added `"schemaVersion": "1.0"` field to the JSON envelope (`pkg/common/output.go`)

### Acceptance

```bash
# Any utility's JSON output validates against its published schema
./goposix ls --json /tmp | ajv validate -s docs/schemas/ls.schema.json
```

---

## 11.2 — End-to-End Agent Integration Example

**Why it matters:** The agent use case is the whole pitch, but there is no demonstration of it. A working example is both the best documentation and the best test.

### Tasks

- [x] Create `examples/agent/` directory with a self-contained example (`examples/agent/main.go`)
- [x] Implement a Go RPC client that:
  - Starts the GoPOSIX daemon
  - Creates a session via `goposix.session.create`
  - Executes a multi-step task (ls, wc, shell.exec, cat)
  - Cleans up the session
- [x] Provide the example in Go (`examples/agent/main.go`)
- [x] Document the example in `docs/AGENT_INTEGRATION.md` with annotated walkthrough
- [x] Add `make example-rpc` target that runs the Go version as a smoke test

### Acceptance

```bash
make example-rpc   # runs Go example against a live daemon, exits 0
```

---

## 11.3 — RPC Client Library

**Why it matters:** `pkg/client/client.go` is 61 lines and only used in tests. Anyone building on the daemon has to implement the protocol themselves.

### Tasks

- [x] Expand `pkg/client/` into a real client library:
  - Connection pooling (configurable pool size)
  - Batch request support (`[]Request` → `[]Response` in one round-trip)
  - Retry with exponential backoff on transient errors
  - Context propagation (`context.Context` on every call)
  - Helper methods per utility (e.g., `client.Ls(ctx, path, flags)`) returning typed structs (42 utilities covered)
- [ ] Publish a thin Python wrapper — deferred (Go-only scope per user direction)
- [x] Document the client in `docs/RPC_API.md` with connection lifecycle and error handling examples
- [x] Unit tests covering pool exhaustion, timeout, reconnect (20 tests in `pkg/client/client_test.go`)

### Acceptance

```go
c, _ := client.New("/tmp/goposix.sock", client.WithPoolSize(4))
result, _ := c.Ls(ctx, "/var/log", "-l")
fmt.Println(result.Files[0].Name)
```

---

## 11.4 — `awk` Implementation

> **Full plan:** [07a_awk.md](07a_awk.md) — the canonical awk document.
>
> `awk` is mandatory for strict POSIX.2 compliance and is the last unimplemented
> POSIX utility. It is the **Platinum gate** for this project. All task details,
> acceptance criteria, and sub-phase breakdowns live in 07a_awk.md.

---

## Milestone 11

- [x] Every utility's `--json` output validates against a published JSON schema
- [x] A working RPC example (Go) runs end-to-end against the daemon (`make example-rpc`)
- [x] `pkg/client` supports connection pooling and typed helper methods (42 utilities)
- [ ] `awk` implemented (see [07a_awk.md](07a_awk.md)) — passes BusyBox awk tests, listed as complete in `posix_coverage.md`

## Remaining Work

| # | Task | Status | Where Tracked |
|---|------|--------|---------------|
| 11.4 | `awk` implementation (Platinum gate) | ⏳ Deferred | [07a_awk.md](07a_awk.md), [12_road_to_gold.md](12_road_to_gold.md) (12.5) |
| 12.3 | Coverage gate → 70% (70.5% actual, enforced via `Makefile`) | ✅ Complete | [13_coverage_and_hardening.md](13_coverage_and_hardening.md) |

## How to Verify

```bash
# Schema validation
make validate-schemas

# Agent example
make example-rpc

# Client library
go test ./pkg/client/... -v

# Coverage
make cover-pct   # ≥70% enforced via Makefile; see [coverage policy](13_coverage_and_hardening.md)

# awk (see 07a_awk.md for full acceptance criteria)
echo "hello world" | ./goposix awk '{print $1}'
```
