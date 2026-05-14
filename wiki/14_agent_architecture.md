# Phase 14 — Autonomous Coding Agent (Rejected)

> **Status:** Rejected | **Superseded by:** [Phase 16 — MCP Server](16_mcp_server.md)

This design was rejected in favor of Phase 16. The architectural insight: KoreGo's natural role is as a *tool backend* for external agents (via MCP), not as an agent itself. Building an LLM agent inside KoreGo duplicates what external agents (Claude Code, Cursor, etc.) already do well, while adding maintenance burden for LLM provider integrations, ReAct loop logic, and prompt engineering. Instead, Phase 16 exposes KoreGo as an MCP server so any MCP-compatible agent can use KoreGo as its sandboxed Linux environment.

---

## Goal

Add an autonomous background agent to KoreGo that receives a natural-language task, fetches a git repository, plans and executes changes using a ReAct (Reasoning + Acting) loop, verifies the result, and opens a pull request — all from within the KoreGo scratch container.

Two invocation modes share one engine:

```
CLI:    korego agent run --prompt "Fix null check in auth.go" --repo https://github.com/user/repo.git
RPC:    {"method":"korego.agent.run", "params":{"prompt":"...","repo":"..."}}
```

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────┐
│                    KoreGo Binary                         │
│                                                         │
│  ┌──────────┐  ┌──────────┐  ┌──────────────────────┐  │
│  │   CLI    │  │   RPC    │  │    Agent Engine       │  │
│  │ (agent   │  │ (daemon  │  │  ┌──────┐ ┌────────┐  │  │
│  │  run)    │  │  method) │  │  │ReAct │ │State   │  │  │
│  └────┬─────┘  └────┬─────┘  │  │Loop  │ │Machine │  │  │
│       │             │        │  └──┬───┘ └───┬────┘  │  │
│       └──────┬──────┘        │     │         │       │  │
│              │               │  ┌──┴─────────┴──┐    │  │
│              ▼               │  │  Tool Executor │    │  │
│     ┌────────────────┐       │  └──┬──────┬─────┘    │  │
│     │  Agent Engine  │       │     │      │          │  │
│     └───────┬────────┘       │  ┌──┴──┐ ┌─┴──────┐   │  │
│             │                │  │Shell│ │Git Ops │   │  │
│    ┌────────┼────────┐       │  │Exec │ │(go-git)│   │  │
│    │        │        │       │  └─────┘ └────────┘   │  │
│    ▼        ▼        ▼       │                       │  │
│ ┌─────┐ ┌─────┐ ┌────────┐  │  ┌──────────────────┐ │  │
│ │LLM  │ │Git  │ │  PR    │  │  │Existing KoreGo   │ │  │
│ │API  │ │Ops  │ │Creator │  │  │(ls, cat, sed,    │ │  │
│ │HTTP │ │     │ │(GitHub)│  │  │ grep, find, sh,  │ │  │
│ └─────┘ └─────┘ └────────┘  │  │ diff, tar, ...)  │ │  │
│                              │  └──────────────────┘ │  │
└─────────────────────────────────────────────────────────┘
```

---

## Agent Loop (ReAct)

The agent follows a ReAct loop — the LLM drives execution step by step until it decides the task is complete:

```
RECEIVE TASK
    │
    ▼
SETUP ─── Clone repo, create work branch, init session
    │
    ▼
OBSERVE ── Read repo structure, relevant files, git log
    │
    ▼
THINK ──── LLM reasons about what to do next
    │
    ├── "I'm done" ──▶ VERIFY ──▶ COMMIT & PR ──▶ REPORT
    │
    └── "I need to act" ──▶ ACT (shell / file edit / git)
                                  │
                                  ▼
                              OBSERVE result ──▶ THINK (loop)
```

### Loop rules

- **Max iterations:** Configurable limit per task (default 50). Prevents infinite loops.
- **Timeout:** Per-task wall-clock timeout (default 600s). Includes LLM call time.
- **Stop condition:** LLM emits a final message indicating completion, or timeout/iteration limit hit.
- **No human-in-the-loop:** The agent does not pause for approval. It works autonomously.

---

## Tool Set

The agent has access to these tools, exposed to the LLM as function/tool definitions:

### Shell Execution
- **What:** Run arbitrary shell commands in the workspace
- **Implementation:** Reuses `internal/shell/interpreter.go` (KoreGo shell with builtins, pipes via Go channels)
- **Safety:** Timeout per command (default 30s). Commands run as korego user within workspace directory.
- **Full access:** No command allowlist — the agent controls the workspace and can run anything.

### File Operations
- **What:** Read, write, edit files
- **Implementation:** Direct Go file I/O for reads/writes; existing KoreGo utilities (`cat`, `sed`, `grep`) for structured operations
- **Diff generation:** Before/after snapshots for each file edit, so the agent can review its changes

### Git Operations
- **What:** Clone, branch, status, diff, stage, commit, push
- **Implementation:** `go-git` (pure Go, no CGO). Wraps common workflows.
- **Auth:** HTTPS + token or SSH key from environment/configuration
- **Branch naming:** Auto-generated from prompt slug + timestamp (e.g., `korego/fix-null-check-20260513`)

### PR Creation
- **What:** Open a pull request on GitHub or GitLab
- **Implementation:** `net/http` + provider REST API (no SDK needed — single endpoint each)
- **PR body:** Generated by LLM summarizing what changed and why

### Network Access (optional future tool)
- **What:** Web search or fetch documentation
- **Gated:** Requires explicit configuration to enable outbound HTTP beyond LLM API and git remote

---

## LLM Provider Interface

Provider-agnostic interface so the agent works with OpenAI, Anthropic, or local models:

```go
type LLMProvider interface {
    Chat(ctx context.Context, messages []Message, tools []Tool) (*Response, error)
}
```

### Providers

| Provider | Endpoint | Notes |
|----------|----------|-------|
| `openai` | `https://api.openai.com/v1` | GPT-4, GPT-5; function calling |
| `anthropic` | `https://api.anthropic.com/v1` | Claude; tool use |
| `local` | User-configured (Ollama, vLLM) | OpenAI-compatible endpoint |

### Configuration

- `--llm` / `agent.llm.provider` — provider name
- `--model` / `agent.llm.model` — model name
- `--llm-endpoint` / `agent.llm.endpoint` — override API base URL
- `--api-key-env` / `agent.llm.api_key_env` — env var holding the API key

---

## State Machine

```
                    ┌─────────────┐
                    │   PENDING   │
                    └──────┬──────┘
                           │ task received
                           ▼
                    ┌─────────────┐
                    │  SETTING_UP │──── timeout / clone failure ──▶ FAILED
                    └──────┬──────┘
                           │ repo cloned, branch created
                           ▼
                    ┌─────────────┐
              ┌────▶│  EXECUTING  │
              │     └──────┬──────┘
              │            │ LLM returns "done"
              │            ▼
              │     ┌─────────────┐
              │     │  VERIFYING  │──── verification fails, loop back ──▶ EXECUTING
              │     └──────┬──────┘
              │            │ tests pass / no regressions
              │            ▼
              │     ┌─────────────┐
              │     │ COMMITTING  │──── commit/push/PR failure ──▶ FAILED
              │     └──────┬──────┘
              │            │ PR created
              │            ▼
              │     ┌─────────────┐
              │     │  COMPLETED  │
              │     └─────────────┘
              │
              └─── LLM emits tool call ──▶ EXECUTING (loop)
```

Any state can transition to `CANCELLED` if `korego agent cancel` is called.

---

## CLI Interface

```
korego agent run \
  --prompt "Fix the null pointer check in pkg/auth/login.go" \
  --repo https://github.com/user/repo.git \
  --base main \
  --llm anthropic \
  --model claude-sonnet-4-6 \
  --api-key-env ANTHROPIC_API_KEY \
  --timeout 600s \
  --workspace /var/lib/korego/workspaces \
  --max-iterations 50

korego agent status [--task-id <id>]     # Show current/last task
korego agent cancel --task-id <id>       # Cancel running task
korego agent log --task-id <id>          # Show execution log
korego agent list [--limit 10]           # List recent tasks
```

### Flag Reference

| Flag | Env Fallback | Default | Description |
|------|-------------|---------|-------------|
| `--prompt` | — | *required* | Natural language task description |
| `--repo` | `KOREGO_AGENT_REPO` | *required* | Git repository URL |
| `--base` | — | `main` | Base branch for the PR |
| `--llm` | `KOREGO_LLM_PROVIDER` | `openai` | LLM provider |
| `--model` | `KOREGO_LLM_MODEL` | `gpt-4o` | Model name |
| `--llm-endpoint` | `KOREGO_LLM_ENDPOINT` | provider default | Override API URL |
| `--api-key-env` | — | `LLM_API_KEY` | Env var for API key |
| `--timeout` | `KOREGO_AGENT_TIMEOUT` | `600s` | Task timeout |
| `--max-iterations` | — | `50` | Max ReAct loop iterations |
| `--workspace` | `KOREGO_AGENT_WORKSPACE` | `/tmp/korego-agent` | Workspace root |
| `--git-token-env` | — | `GITHUB_TOKEN` | Env var for git token |
| `--cleanup` | — | `true` | Remove workspace after completion |

---

## RPC Interface

### `korego.agent.run`

```json
// Request
{
  "jsonrpc": "2.0",
  "method": "korego.agent.run",
  "params": {
    "prompt": "Fix null check in auth.go",
    "repo": "https://github.com/user/repo.git",
    "base": "main",
    "llm": "anthropic",
    "model": "claude-sonnet-4-6",
    "timeout": "600s",
    "sessionId": "abc-123"
  },
  "id": 1
}

// Response (immediate — task runs asynchronously)
{
  "jsonrpc": "2.0",
  "result": {
    "taskId": "task-abc123",
    "status": "executing"
  },
  "id": 1
}
```

### `korego.agent.status`

```json
// Request
{"jsonrpc":"2.0","method":"korego.agent.status","params":{"taskId":"task-abc123"},"id":2}

// Response
{
  "jsonrpc": "2.0",
  "result": {
    "taskId": "task-abc123",
    "status": "completed",
    "prUrl": "https://github.com/user/repo/pull/42",
    "branch": "korego/fix-null-check-20260513",
    "iterations": 8,
    "durationMs": 45200
  },
  "id": 2
}
```

### `korego.agent.cancel`

```json
{"jsonrpc":"2.0","method":"korego.agent.cancel","params":{"taskId":"task-abc123"},"id":3}
→ {"jsonrpc":"2.0","result":{"status":"cancelled"},"id":3}
```

### `korego.agent.list`

```json
{"jsonrpc":"2.0","method":"korego.agent.list","params":{"limit":10},"id":4}
→ {"jsonrpc":"2.0","result":{"tasks":[{"taskId":"...","status":"...","prompt":"..."}]},"id":4}
```

### `korego.agent.log`

```json
{"jsonrpc":"2.0","method":"korego.agent.log","params":{"taskId":"task-abc123"},"id":5}
→ {"jsonrpc":"2.0","result":{"entries":[{"timestamp":"...","level":"info","message":"..."}]},"id":5}
```

---

## Workspace Layout

```
/var/lib/korego/workspaces/
└── <task-id>/
    ├── repo/                # Cloned repository
    ├── agent.log            # Structured JSON log
    ├── plan.json            # LLM's plan (if plan-first)
    └── diff/                # Before/after snapshots per file edit
```

Workspace is cleaned up after task completion by default (`--cleanup true`). Set `--cleanup false` to retain for inspection.

---

## Security Model

| Concern | Mitigation |
|---------|------------|
| Host filesystem access | Agent runs in workspace directory. No access outside without explicit volume mounts. |
| Network egress | Only LLM API endpoint, git remote, and PR provider API are reachable. |
| Credential exposure | Tokens read from env vars or mounted secrets files. Never logged. |
| Privilege escalation | Agent runs as existing `korego` user (uid 1000, no sudo). |
| Infinite loops | Hard iteration limit + wall-clock timeout per task. |
| Malicious prompts | Agent only operates on the specified repo. No access to KoreGo internals or daemon config. |
| Shell injection | Shell executor uses the existing KoreGo interpreter which does not fork/exec to host. |

---

## Docker Integration

### Scratch Image Additions

- `go-git` compiles in statically (pure Go, no system deps)
- CA certificates already present (for HTTPS to LLM + git remotes)
- Workspace directory created at build time with `korego` user ownership

### Compose Example

```yaml
services:
  korego-agent:
    image: korego:latest
    command: ["daemon", "--socket", "/var/run/korego.sock"]
    volumes:
      - /data/agent-workspaces:/var/lib/korego/workspaces
      - /run/secrets/gh_token:/run/secrets/gh_token:ro
    environment:
      - KOREGO_LLM_PROVIDER=anthropic
      - KOREGO_LLM_MODEL=claude-sonnet-4-6
      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}
      - GITHUB_TOKEN_FILE=/run/secrets/gh_token
    security_opt:
      - no-new-privileges:true
```

---

## Package Structure

```
pkg/agent/
├── agent.go          # State machine, task lifecycle, ReAct loop entry point
├── planner.go        # LLM interaction: system prompt, message history, tool definitions
├── executor.go       # Tool dispatch: routes LLM tool calls to shell/git/file operations
├── gitops.go         # go-git wrapper: clone, branch, stage, commit, push
├── pr.go             # PR creation via GitHub/GitLab REST API
├── workspace.go      # Workspace creation, cleanup, path management
├── types.go          # Shared types: Task, Step, ToolCall, AgentState
└── llm/
    ├── provider.go   # LLMProvider interface + factory
    ├── openai.go     # OpenAI-compatible provider
    └── anthropic.go  # Anthropic provider
```

### Integration Points

| Package | Integrates With |
|---------|----------------|
| `pkg/agent/agent.go` | `internal/dispatch/` — registers `agent` subcommand and RPC methods |
| `pkg/agent/executor.go` | `internal/shell/interpreter.go` — executes shell commands |
| `pkg/agent/gitops.go` | `github.com/go-git/go-git/v5` — git operations |
| `pkg/agent/pr.go` | `net/http` — GitHub/GitLab REST API |
| `pkg/agent/llm/` | `net/http` — LLM provider REST APIs |

---

## Dependencies to Add

| Module | Purpose | CGO? |
|--------|---------|------|
| `github.com/go-git/go-git/v5` | Git clone, branch, commit, push | No |
| `github.com/go-git/go-git/v5/plumbing/transport/http` | HTTPS auth for git | No |
| None required for LLM | `net/http` + `encoding/json` handle all provider APIs | No |

No LLM SDK is required — the provider APIs (OpenAI chat completions, Anthropic messages) are simple enough for stdlib HTTP. If provider-specific features (streaming, caching) become needed, `github.com/sashabaranov/go-openai` is a lightweight option.

---

## Open Design Decisions

These are deferred to implementation planning:

1. **System prompt design** — The exact system prompt that defines the agent's behavior, tool use conventions, and stopping criteria.
2. **Streaming vs batch** — Whether to stream LLM responses (better UX for CLI mode) or batch (simpler for RPC mode).
3. **Task persistence** — Whether tasks survive daemon restart. Likely: in-memory only for MVP, disk-backed queue later.
4. **Multi-repo tasks** — Should a single task be able to span multiple repos? Likely: no for MVP.
5. **Tool result truncation** — How to handle large tool outputs (e.g., `cat` on a 10MB file). Likely: truncate to N tokens with a note.
6. **Retry on LLM error** — How many retries on API failure before marking task as FAILED.

---

## Verification Plan

- **Unit tests:** Each component (state machine, gitops, PR creator, LLM provider) tested independently with mocks.
- **Integration test:** Agent given a trivial task on a local test repo ("add a comment to README") and full loop exercised.
- **Compliance test:** Agent run against a real repo with a real LLM; PR inspected for correctness.
- **Docker smoke test:** `make smoke-docker` extended to run `korego agent run --prompt "..." --repo "..."` inside the scratch container.
