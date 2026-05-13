# Phase 11 — Post-MVP Priorities

> **Status:** Planning | **Depends on:** All Phases 00–10 complete

---

## Context

KoreGo MVP is complete: 49 utilities implemented, 97.9% BusyBox pass rate, daemon functional. This phase addresses the gaps that matter most for the stated goals — AI agent tooling and minimal container deployments. Items are ordered by impact on adoption.

---

## 11.1 — Formal JSON Output Schemas

**Why it matters:** The primary differentiator of KoreGo is machine-readable output. Without a formal schema, consumers can't rely on stable JSON shapes — defeating the purpose.

### Tasks

- [x] Define a canonical JSON schema (JSON Schema draft-07) for every utility's `--json` output (`test/schemas/`)
- [x] Add schemas and update `docs/JSON_SCHEMA.md` (now documents all 42 JSON-enabled utilities)
- [x] Add a schema validation step in CI (`make validate-schemas` runs `ajv` against golden fixtures)
- [x] Version the schema — added `"schemaVersion": "1.0"` field to the JSON envelope (`pkg/common/output.go`)

### Acceptance

```bash
# Any utility's JSON output validates against its published schema
./korego ls --json /tmp | ajv validate -s docs/schemas/ls.schema.json
```

---

## 11.2 — End-to-End Agent Integration Example

**Why it matters:** The agent use case is the whole pitch, but there is no demonstration of it. A working example is both the best documentation and the best test.

### Tasks

- [ ] Create `examples/agent/` directory with a self-contained example
- [ ] Implement a minimal Claude tool-use loop that:
  - Starts the KoreGo daemon
  - Creates a session via `korego.session.create`
  - Executes a multi-step task (e.g., find large files, summarize with grep/wc, return structured result)
  - Cleans up the session
- [ ] Provide the same example in Python (using `anthropic` SDK + `socket`) and Go
- [ ] Document the example in `docs/AGENT_INTEGRATION.md` with annotated walkthrough
- [ ] Add an `examples` Makefile target that runs the Go version as a smoke test

### Acceptance

```bash
make example-agent   # runs Go example against a live daemon, exits 0
```

---

## 11.3 — RPC Client Library

**Why it matters:** `pkg/client/client.go` is 61 lines and only used in tests. Anyone building on the daemon has to implement the protocol themselves.

### Tasks

- [ ] Expand `pkg/client/` into a real client library:
  - Connection pooling (configurable pool size)
  - Batch request support (`[]Request` → `[]Response` in one round-trip)
  - Retry with exponential backoff on transient errors
  - Context propagation (`context.Context` on every call)
  - Helper methods per utility (e.g., `client.Ls(ctx, path, flags)`) returning typed structs
- [ ] Publish a thin Python wrapper (`examples/client/korego_client.py`) using `socket` + `json`
- [ ] Document the client in `docs/RPC_API.md` with connection lifecycle and error handling examples
- [ ] Unit tests covering pool exhaustion, timeout, reconnect

### Acceptance

```go
c, _ := client.New("/tmp/korego.sock", client.WithPoolSize(4))
result, _ := c.Ls(ctx, "/var/log", "-l")
fmt.Println(result.Files[0].Name)
```

---

## 11.4 — `awk` Implementation

**Why it matters:** `awk` is mandatory for strict POSIX.2 compliance. The existing docs (`wiki/posix_faq.md`) acknowledge this explicitly. Every serious shell script that processes structured text uses awk. Without it, "POSIX-compliant userland" is a qualified claim.

> Full implementation plan already exists in [07a_awk.md](07a_awk.md). This task is to execute it.

### Tasks

- [ ] Implement `pkg/awk/awk.go` per the plan in `wiki/07a_awk.md`
- [ ] Register in multicall dispatch
- [ ] `--json` output: array of per-record results
- [ ] BusyBox test suite awk tests must pass
- [ ] Unit tests (target: ≥ 20 test cases covering patterns, fields, BEGIN/END, built-in variables)
- [ ] Update `wiki/posix_coverage.md` and README status table

### Acceptance

```bash
echo -e "a 1\nb 2\nc 3" | ./korego awk '{print $2}' 
# → 1\n2\n3

./korego awk --json '{sum += $1} END {print sum}' numbers.txt
# → {"output": "42\n", "exitCode": 0}
```

---

## Milestone 11

- [x] Every utility's `--json` output validates against a published JSON schema
- [ ] A working agent example (Go + Python) runs end-to-end against the daemon
- [ ] `pkg/client` supports connection pooling and typed helper methods
- [ ] `awk` passes BusyBox awk tests and is listed as complete in `posix_coverage.md`

## How to Verify

```bash
# Schema validation
make validate-schemas

# Agent example
make example-agent

# Client library
go test ./pkg/client/... -v

# awk
echo "hello world" | ./korego awk '{print $1}'
go test ./pkg/awk/... -v
make testsuite   # awk tests should now pass
```
