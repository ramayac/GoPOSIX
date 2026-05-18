# Phase 21 — Honest Takes: De-Agentifying GoPOSIX

> **Status:** AUDIT COMPLETE | **Date:** 2026-05-18 | **Branch:** `feat/honest-takes`
>
> This document audits every instance of "agent," "agentic," "AI agent," and
> "agent-ready" language across the entire project and prescribes honest
> replacements. It will be used to execute the changes, then removed.

---

## The Honest Truth

GoPOSIX is a **Go-native multicall POSIX userland with a JSON-RPC daemon and
structured `--json` output on every utility.** That's it. That's what it is.

It is NOT "designed for agentic runtimes." It has zero agent-specific code:
- No LLM tool-calling schema (OpenAI function format, Anthropic tool_use)
- No MCP (Model Context Protocol) server
- No agent loop or planning framework
- No context-window-aware output formatting
- No integration with any agent framework (LangChain, CrewAI, etc.)
- Nothing that knows what an "agent" is, let alone serves one specifically

**What it actually has:** A JSON-RPC daemon that any program — agent, CI runner,
shell script, monitoring system, web service — can connect to and invoke POSIX
utilities with structured JSON responses. This is **programmatic infrastructure,**
no different in kind from a REST API, gRPC service, or D-Bus interface.

The "agentic runtime" framing is marketing, not architecture. We're fixing that.

---

## Audit: Every Instance, Categorized

### Category A: Public-Facing (README, docs/) — HIGHEST PRIORITY

These are what users, evaluators, and the public see first.

| # | File | Line / Section | Current Text | Problem | Recommended Replacement |
|---|------|---------------|-------------|---------|------------------------|
| A1 | `README.md` | Tagline | "featuring structured `--json` output in every utility and a persistent JSON-RPC daemon to eliminate process-spawning overhead" | Accurate. No change needed. | Keep as-is. |
| A2 | `README.md` | Key Features | "Machine-Readable by Default" | Accurate. No change needed. | Keep as-is. |
| A3 | `README.md` | Key Features | "Low-Overhead Execution: A persistent JSON-RPC 2.0 daemon with session management" | Accurate. No change needed. | Keep as-is. |
| A4 | `README.md` | Key Features | "Portable Scripting: Sandboxed shell interpreter" | Accurate. No change needed. | Keep as-is. |
| A5 | `README.md` | Quickstart | `ls --json /` | Accurate. No change needed. | Keep as-is. |
| A6 | `README.md` | Documentation section | "Agent Integration Guide" link | Overstates. This is just an RPC usage guide. | Rename to **"RPC Client Guide (examples/)"** or keep link but rename the doc. |
| A7 | `README.md` | Project Principles | "`--json` Only" | Accurate. | Keep as-is. |
| A8 | `docs/ARCHITECTURE.md` | Line 5 | "JSON-RPC 2.0 daemon for AI agent backends" | Inaccurate framing. | → "JSON-RPC 2.0 daemon for programmatic consumers" |
| A9 | `docs/ARCHITECTURE.md` | ASCII diagram | "AI Agent / User" box | Inaccurate. | → "Programmatic Consumer / CLI User" |
| A10 | `docs/ARCHITECTURE.md` | Line 96 | "Go SDK for agents" | Inaccurate. | → "Go SDK for programmatic RPC clients" |
| A11 | `docs/ARCHITECTURE.md` | Line 138 | "How to use GoPOSIX as an AI agent backend" | Inaccurate. | → "How to use GoPOSIX over JSON-RPC" |
| A12 | `docs/AGENT_INTEGRATION.md` | Entire file title | "Agent Integration Guide" | Entire doc is framed for AI agents but describes generic RPC usage. | Rename file to `docs/RPC_QUICKSTART.md`. Replace all "AI agent" → "RPC client" or "programmatic consumer." |
| A13 | `docs/AGENT_INTEGRATION.md` | Line 1 | "How to use GoPOSIX as a tool-execution backend for AI agents." | Inaccurate. | → "How to interact with GoPOSIX programmatically via JSON-RPC." |
| A14 | `docs/AGENT_INTEGRATION.md` | Line 7 | "lets an AI agent:" | Inaccurate. | → "lets any program:" |
| A15 | `docs/AGENT_INTEGRATION.md` | Line 74 | "echo hello from agent" | Minor but sloppy. | → "echo hello from goposix" |
| A16 | `docs/AGENT_INTEGRATION.md` | Section title | "Example: Multi-Step Agent Task" | Inaccurate. | → "Example: Multi-Step RPC Task" |
| A17 | `docs/AGENT_INTEGRATION.md` | Line 134 | "The full example at `examples/agent/main.go`" | Refers to agent-named file. | → "The full example at `examples/rpc_client/main.go`" |
| A18 | `docs/SECURITY.md` | Line 3 | "The `goposix.shell.exec` RPC method is designed for **trusted input only**." | Accurate. No agent language here. | Keep as-is. No changes needed. |
| A19 | `docs/JSON_SCHEMA.md` | Line 3 | "All GoPOSIX utilities support structured machine-readable output" | Accurate. | Keep as-is. |

### Category B: Code & Examples — HIGH PRIORITY

| # | File | Current | Problem | Replacement |
|---|------|---------|---------|-------------|
| B1 | `examples/agent/main.go` | File and package named "agent" | The example has nothing agent-specific — it's a generic RPC client demo. | Rename directory to `examples/rpc_client/`. Rename binary to `rpc_client`. |
| B2 | `examples/agent/main.go:1` | "Agent example: demonstrates a minimal AI-agent integration" | Inaccurate. | → "RPC client example: demonstrates using GoPOSIX via JSON-RPC" |
| B3 | `examples/agent/main.go:59` | "=== GoPOSIX Agent Example ===" | Inaccurate. | → "=== GoPOSIX RPC Client Example ===" |
| B4 | `examples/agent/main.go:62` | `"goposix-agent-*"` temp dir | Minor. | → `"goposix-rpc-*"` |
| B5 | `examples/agent/main.go:168` | `"echo hello from agent"` | Minor. | → `"echo hello from goposix"` |
| B6 | `examples/agent/main.go:211` | "=== Agent example complete ===" | Minor. | → "=== RPC example complete ===" |
| B7 | `test/integration/agent_test.go` | File named "agent_test.go" | Integration test for daemon+RPC, not agent-specific. | Rename file to `rpc_integration_test.go`. |
| B8 | `test/integration/agent_test.go:10` | "TestAgentWorkflow simulates a real agentic workflow via RPC" | Inaccurate. | → "TestRPCWorkflow simulates a multi-step RPC workflow" |
| B9 | `test/integration/agent_test.go:11` | `func TestAgentWorkflow` | Inaccurate. | → `func TestRPCWorkflow` |
| B10 | `test/integration/agent_test.go:50` | `"goposix-agent-test"` temp dir | Minor. | → `"goposix-rpc-test"` |
| B11 | `Makefile` | Target `example-agent` and description "agent integration example" | Inaccurate. | → Target `example-rpc` with description "RPC integration example" |

### Category C: Wiki / Internal Docs — MEDIUM PRIORITY

These are internal-facing but set the tone for the entire project and influence
future contributors.

| # | File | Section | Current Text | Problem | Replacement |
|---|------|---------|-------------|---------|-------------|
| C1 | `wiki/phases.md` | Phase description | "Agent-Ready Features (sessions, shell, Tier 5)" | Inaccurate — these are RPC features. | → "RPC Features (sessions, shell, advanced utilities)" |
| C2 | `wiki/phases.md` | Phase History table | "Agent Features" | Same. | → "RPC Features" |
| C3 | `wiki/index.md` | Historical Phase Docs row | "Agent Features" | Same. | → "RPC Features" |
| C4 | `wiki/index.md` | Historical Phase Docs row | "Agent Features" (07_agent_features.md ref) | Same. | Rename the wiki file itself? See C5. |
| C5 | `wiki/07_agent_features.md` | File name + title | "Phase 07 — Agent-Ready Features + Tier 5" | Inaccurate. Sessions, shell exec, structured logging are generic RPC/infra features. | Rename file to `wiki/07_rpc_features.md`. Rewrite title to "Phase 07 — RPC Features + Tier 5." Rewrite goal section to say "programmatic consumers" not "agents." |
| C6 | `wiki/07_agent_features.md` | Goal section | "Each agent connection can create a session" | Inaccurate. | → "Each RPC client connection can create a session" |
| C7 | `wiki/posix_faq.md` | ~6 instances | "agentic runtime," "agentic contexts," "GoPOSIX targets agentic runtimes in containers," "provide a useful agentic runtime," "agent capabilities" | Pervasive. This doc frames the entire POSIX compliance discussion around agents, which is the wrong framing. The doc is otherwise excellent on POSIX facts. | Replace ALL instances of "agentic runtime" → "programmatic runtime" or "containerized automation." Replace "agent capabilities" → "structured JSON output." The target audience section should say "programmatic consumers in containers" not "agentic runtimes." |
| C8 | `wiki/posix_coverage.md` | Notes column | "agent-ready userland," "agent directly via JSON structured output" | Inaccurate. | → "programmatic userland," "client directly via JSON structured output" |
| C9 | `wiki/11_post_mvp_priorities.md` | Line 1 context | "AI agent tooling and minimal container deployments" | Inaccurate framing of project goals. | → "programmatic tooling and minimal container deployments" |
| C10 | `wiki/11_post_mvp_priorities.md` | Section title/intro | "agent use case is the whole pitch" | This is the most damning line. It explicitly states the entire project pitch is agentic, which is false. | → "programmatic use case is the primary differentiation" |
| C11 | `wiki/11_post_mvp_priorities.md` | 11.2 section | "End-to-End Agent Integration Example" | Inaccurate. | → "End-to-End RPC Integration Example" |
| C12 | `wiki/11_post_mvp_priorities.md` | 11.2 tasks | "Implement a Go agent that: ... Executes a multi-step task" | Inaccurate. | → "Implement a Go RPC client that: ... Executes a multi-step task" |
| C13 | `wiki/11_lessons_learned.md` | Table row 11.2 | "End-to-end agent integration example" | Inaccurate. | → "End-to-end RPC integration example" |
| C14 | `wiki/11_lessons_learned.md` | Various | "agent example," "the agent example now documents..." | Inaccurate. | → "RPC example," "the RPC example now documents..." |
| C15 | `wiki/11_lessons_learned.md` | Lesson 9 | "The agent example intentionally uses raw socket communication" | Inaccurate. | → "The RPC example intentionally uses raw socket communication" |
| C16 | `wiki/12_road_to_gold.md` | 12.2 section | "tool explicitly targets AI agents with untrusted input" | Inaccurate. | → "tool accepts untrusted programmatic input" |
| C17 | `wiki/12_road_to_gold.md` | 12.2 section | "explicitly positioned for AI agent use" | Inaccurate. | → "designed for programmatic consumption" |
| C18 | `wiki/19_performance_benchmarking.md` | Title/summary | "agentic runtimes — persistent JSON-RPC daemon" | Inaccurate. | → "programmatic runtimes — persistent JSON-RPC daemon" |
| C19 | `wiki/19_performance_benchmarking.md` | Honest Priors section | "agent loop throughput," "adoption evidence for agent backends," "AI agent loop" | Inaccurate. | → "RPC loop throughput," "adoption evidence for RPC backends," "RPC client loop" |
| C20 | `wiki/19_performance_benchmarking.md` | Category J description | "AI agent loop (list files → read file → search → filter)" | This is just a task loop — nothing agent-specific. | → "RPC task loop (list files → read file → search → filter)" |
| C21 | `wiki/19_performance_benchmarking.md` | Conclusion | "For an AI agent making 10,000 ls calls per session" | Inaccurate. | → "For a program making 10,000 RPC calls per session" |
| C22 | `wiki/04_text_processing.md` | Goal | "agentic pipelines" | Inaccurate. | → "programmatic pipelines" |
| C23 | `wiki/09_release_docs.md` | Integration test section | "Simulate a real agentic workflow" | Inaccurate. | → "Simulate a multi-step RPC workflow" |
| C24 | `wiki/09_release_docs.md` | Checklist | "E2E agent test passes," "Stateful sessions for agentic workflows" | Inaccurate. | → "E2E RPC test passes," "Stateful sessions for multi-step workflows" |
| C25 | `wiki/14c_posix_json_gap.md` | Descriptions | "agent pipelines" | Inaccurate. | → "RPC pipelines" |
| C26 | `wiki/18_quality_fixes.md` | 18.3 | "gap in agent-ready toolbelt" | Inaccurate. | → "gap in POSIX coverage" |
| C27 | `wiki/18_quality_fixes.md` | 18.3 purpose | "Critical for agent workflows that apply patches" | Inaccurate. | → "Critical for automated workflows that apply patches" |
| C28 | `wiki/20_hardening_ii.md` | Coverage table | "Go SDK for agents" | Inaccurate. | → "Go SDK for JSON-RPC clients" |
| C29 | `wiki/log.md` | Multiple entries | "agent," "agentic," "agent architecture" references in log timeline | These are historical records. | Add a one-line addendum at the top: "Note: 'agent' references in historical entries below predate the Phase 21 honest-takes audit. The project's positioning has been corrected. See wiki/21_honest_takes.md." |

### Category D: Benchmark Infrastructure — LOW PRIORITY (but visible)

| # | File | Current | Problem | Replacement |
|---|------|---------|---------|-------------|
| D1 | `test/benchmark/cat_j_agent_loop.sh` | File named "agent_loop.sh" | Misleading name. | Rename to `cat_j_rpc_loop.sh`. |
| D2 | `test/benchmark/cat_j_agent_loop.sh` | Comments + log messages: "Simulates a typical AI agent task flow" | Inaccurate. | → "Simulates a typical RPC task loop" |
| D3 | `test/benchmark/cat_j_agent_loop.sh` | Temp dirs: `agent_bench`, `agent_loop` | Minor. | → `rpc_bench`, `rpc_loop` |
| D4 | `test/benchmark/lib/report.sh` | Category description: "End-to-end agent loop" | Inaccurate. | → "End-to-end RPC loop" |
| D5 | `test/benchmark/lib/report.sh` | Conclusion: "For an AI agent making 10,000 calls per session" | Inaccurate. | → "For a program making 10,000 RPC calls per session" |
| D6 | `test/benchmark/lib/report.sh` | "For AI agent backends" / "agent RPC" | Inaccurate. | → "For programmatic backends" / "daemon RPC" |
| D7 | `test/benchmark/runner.sh` | Variable names: `agent_loop` | Minor. | → `rpc_loop` |

### Category E: Other Internal Files

| # | File | Current | Problem | Replacement |
|---|------|---------|---------|-------------|
| E1 | `wiki/13_coverage_and_hardening.md` | Audited; contains mentions of "agent" in coverage context | Minor. | Replace as encountered. |
| E2 | `wiki/16_post_mvp_tier2.md` | "by separate agents if desired" (context: parallel work) | Different meaning of "agent" — means "separate people/processes." | Keep as-is. This is standard English, not the "AI agent" sense. |
| E3 | `wiki/11a_lower_priority.md` | Line 90, 139: `make example-agent` target references | Will be fixed by Makefile rename. | Update when Makefile target changes to `example-rpc`. |
| E4 | `wiki/performance.md` | Lines 67, 84, 134: "1K agent loops," "Agent loop," "cat_j_agent_loop.log" | Same overstatement as benchmark files. | → "RPC loops," "RPC task loop," "cat_j_rpc_loop.log" |
| E5 | `wiki/schema.md` | Line 5: "between the chat agent and the raw repository" | Meta usage meaning "the assistant doing the work." Different from the AI-agent-runtime framing. | Can leave as-is or change to "assistant." Low priority — internal ops doc. |
| E6 | `wiki/operations/query.md` | Line 5: "so the agent does not start from zero" | Same meta usage as E5. | Can leave as-is. Low priority. |
| E7 | `.github/prompts/wiki-*.prompt.md` | `agent: "agent"` and similar in prompt templates | These are internal prompt templates for CI/wiki automation. "agent" here refers to the LLM performing a wiki task — operational, not marketing. | Leave as-is. These are tool configuration, not project framing. |

---

## What Should NOT Change

These are instances where the current wording is accurate and appropriate:

1. **"Machine-readable output"** — Accurate. JSON is machine-readable.
2. **"Structured output"** — Accurate. The JSON envelope has a defined schema.
3. **"`--json` flag"** — Accurate feature description.
4. **"JSON-RPC daemon"** — Accurate. It is a JSON-RPC 2.0 daemon.
5. **"Session management"** — Accurate. Sessions exist and carry state.
6. **"Shell sandbox"** — Accurate security feature.
7. **"Programmatic consumer"** — This is the honest replacement phrase we should use.
8. **AGENTS.md** (line 9) — "Crucially, GoPOSIX is designed for **Agentic Runtimes**" → "Crucially, GoPOSIX is designed for **Programmatic Consumption**" or "**Container-Native Automation.**" This is the instruction file for AIs working on the project, but it sets the identity that every AI sees on every session. It must match the honest framing.
9. **CLAUDE.md** — Same as AGENTS.md — update to match.

---

## The Honest Framing — Master Template

When describing GoPOSIX, use this language:

> **GoPOSIX is a Go-native multicall POSIX userland for `FROM scratch` Docker
> containers. Every utility supports structured `--json` output and can be
> invoked programmatically via a persistent JSON-RPC 2.0 daemon — eliminating
> process-spawning overhead for repeated operations.**

Key phrases:
- "programmatic consumer" not "agent"
- "RPC client" not "agent"
- "multi-step RPC workflow" not "agentic workflow"
- "JSON-RPC daemon" / "RPC backend" not "agent backend"

---

## Rename Plan

| Current | New |
|---------|-----|
| `docs/AGENT_INTEGRATION.md` | `docs/RPC_QUICKSTART.md` |
| `examples/agent/` | `examples/rpc_client/` |
| `test/integration/agent_test.go` | `test/integration/rpc_integration_test.go` |
| `test/benchmark/cat_j_agent_loop.sh` | `test/benchmark/cat_j_rpc_loop.sh` |
| `wiki/07_agent_features.md` | `wiki/07_rpc_features.md` |
| `make example-agent` | `make example-rpc` |

Files that reference the renamed files must be updated (Makefile, wiki/index.md,
wiki/phases.md, wiki/log.md, test/benchmark/runner.sh, test/benchmark/lib/report.sh,
docs/ARCHITECTURE.md, README.md).

---

## Execution Order

1. **First:** Write the honest framing (this document) and get buy-in.
2. **Rename files** (examples, tests, wiki, docs) and update all internal references.
3. **Update docs/** — `ARCHITECTURE.md`, `AGENT_INTEGRATION.md` → `RPC_QUICKSTART.md`.
4. **Update README.md** — Link name, remove agent framing.
5. **Update wiki/** — All files in Category C above.
6. **Update benchmark/** scripts and report templates.
7. **Update AGENTS.md** project description to honest framing.
8. **Verify:** `make test`, `make testsuite`, `make vet` all pass.
9. **Delete this file** (`wiki/21_honest_takes.md`) when all changes are complete.
