# The Loop — Product & Technical Specification [AI generated]

## Vision

The Loop is a self-hosted Go application with a bundled React web UI for creating and running AI agent projects. It supports interactive chat sessions, semi-autonomous agents (human-in-the-loop), and fully autonomous agents that run on a schedule or in response to events. The system is designed for extensibility: agent types are defined declaratively, AI harnesses are pluggable, and runtimes can be local processes, Docker containers, or remote agents.

---

## Project Modes

| Mode | Description | HITL Required |
|---|---|---|
| **Interactive** | Back-and-forth chat session with an agent | Always |
| **Semi-autonomous** | Agent executes steps, pauses at designated approval gates | For flagged steps |
| **Autonomous** | Agent runs unattended on a schedule or event trigger | No (optional notifications) |

HITL channels (phased):
- **v1**: Chat UI (inline approval cards in the conversation)
- **v2**: Slack, webhook callbacks
- **v3**: WhatsApp, Telegram, email

---

## Agent Definition Language (ADL)

ADL is a YAML format for declaring an agent type. It is a **static definition**, not a runtime protocol — it describes *what* an agent is and *what it can do*, not *how* it executes.

### Design principles
1. **Definition-time only**: every field must be settable at authoring time without knowing the runtime
2. **Runtime-portable**: an ADL file should be runnable on any compliant harness
3. **Describe WHAT, not HOW**: ADL is not code

### Three-layer architecture (do not collapse these)
- **ADL** — agent identity, capabilities, steps, tool access, and schedule (*this layer*)
- **MCP** — runtime tool protocol; ADL references MCP servers by URL, not by implementation
- **SKILL.md** — agent behavior/persona; ADL may reference a skill file for the system prompt body

### Schema (draft)

```yaml
adl: "1.0"

# ── Identity ─────────────────────────────────────────────────────────────────
name: string          # ≤64 chars
description: string   # ≤1024 chars
version: semver       # e.g. "1.0.0"

# ── Harness ──────────────────────────────────────────────────────────────────
harness:
  type: claude-code | docker | remote | pi
  model: string                     # e.g. "claude-sonnet-4-6"
  workingDir: string                # optional; defaults to server CWD
  # docker-only:
  image: string                     # container image
  containerPort: 9090               # port the container listens on
  # remote-only:
  host: string                      # hostname or IP of the remote agent
  port: 9090                        # port of the remote agent

# ── Schedule (autonomous mode) ────────────────────────────────────────────────
schedule:
  cron: "0 9 * * 1-5"              # standard cron expression
  timezone: America/Los_Angeles     # IANA timezone; defaults to UTC

# ── System prompt ─────────────────────────────────────────────────────────────
systemPrompt: |
  You are ...
# OR reference a SKILL.md file:
skill: ./skills/my-agent.md

# ── Tools / dependencies ──────────────────────────────────────────────────────
tools:
  mcp:
    - url: "http://localhost:3001"   # MCP server
      name: filesystem
  plugins: []                       # future
  skills: []                        # sub-agent skills (SKILL.md references)

# ── Steps ─────────────────────────────────────────────────────────────────────
# Each step can override the top-level harness, model, and tools.
steps:
  - name: research
    policy: react                   # react | sequential | parallel | loop | batch | conditional
    harness:                        # optional per-step override
      type: claude-code
      model: claude-opus-4-8
    systemPrompt: |                 # overrides top-level for this step
      Focus only on research.
    tools:
      mcp:
        - url: "http://localhost:3002"
    outputs:
      - name: report                # named output referenced by downstream steps
        type: text                  # text | json
    approval: none                  # none | required | reversibility-based (future)

  - name: write
    policy: sequential
    dependsOn: [research]
    inputs:
      - from: research.report       # <stepName>.<outputName>
        as: researchReport          # optional alias injected into the prompt context
    approval: required              # pause and ask human before executing
    approvalTimeout: "30m"          # deny on timeout (safest default for irreversible actions)

# ── Constraints ───────────────────────────────────────────────────────────────
constraints:
  maxTokens: 100000
  timeout: "30m"
  retries: 3
  maxConcurrency: 4                 # max parallel steps (applies to parallel/batch policies)
```

### Execution policies

| Policy | Semantics |
|---|---|
| `react` | Think → Act → Observe loop until done (default for interactive) |
| `sequential` | Steps A → B → C in order, output of each feeds next |
| `parallel` | Fan-out all sub-steps, merge results |
| `loop` | Repeat until a condition is met |
| `batch` | Map agent over an array of inputs |
| `conditional` | Route to a step based on a condition |

Vendor extensions via `x-<vendor>.*` namespace (ignored by other runtimes).

---

## Agent Runtime Extension Interface

The `Agent` Go interface is the extension point:

```go
type Agent interface {
    Name() string
    Run(ctx context.Context, req RunRequest, events chan<- Event) error
}
```

### Built-in harnesses

| Harness | Implementation |
|---|---|
| `claude-code` | Shells out to `claude` CLI; streams `stream-json` output |
| `pi` | TCP JSON-RPC 2.0 to a managed Python extension process |
| `docker` | Launches container via `docker run`; communicates over HTTP/SSE |
| `remote` | Connects to a user-configured host:port via HTTP/SSE |

### Docker harness

Loop manages the full container lifecycle. On first use, `Manager.launchDocker` runs:

```
docker run -d -p 127.0.0.1::<containerPort> <image>
docker port <containerID> <containerPort>   # → resolve random host port
GET http://127.0.0.1:<hostPort>/info        # wait for HTTP readiness
```

Each chat message calls `POST /run` on the container and reads the SSE stream. On project delete or server shutdown, Loop calls `docker stop <containerID>`.

The container image must implement the HTTP/SSE extension protocol — see `dev/extension-examples/docker/`.

### Remote harness

Stores the user-configured `host:port` in `Project.AgentConfig`. On project create, Loop calls `GET http://<host>:<port>/info` to verify reachability. Each chat message calls `POST /run` and reads the SSE stream. Loop owns no process or container.

The remote server must implement the HTTP/SSE extension protocol — see `dev/extension-examples/remote/`.

### HTTP/SSE extension protocol

Both docker and remote harnesses use the same three endpoints:

| Endpoint | Description |
|---|---|
| `GET /info` | `{"name","version","capabilities"}` — used as health check |
| `POST /run` | Body: `{"message","sessionId"?,"workingDir"?}`; response: `text/event-stream` |
| `POST /cancel` | Body: `{"runId"}`; stop current run best-effort |

SSE events: `data: {"type":"text","content":"..."}`, `data: {"type":"done","sessionId":"..."}`, `data: {"type":"error","error":"..."}`


## UI/Backend Decoupling & Reconnection

**Problem**: agent inference is expensive; if the browser disconnects mid-run, the run must not restart.

**Solution: offset-based durable stream**

Every token written by an agent run is appended to a persistent log with a monotonic offset. On reconnect, the UI sends `Last-Event-ID: <offset>` and the server replays from that point — no inference is re-run.

Implementation options (in order of complexity):
1. **In-process ring buffer** (v1): append to an in-memory `[]Event` per run; replay on reconnect within the same process lifetime
2. **File-backed log** (v2): append to `~/.loop/runs/<runID>.jsonl`; survives backend restart
3. **Redis Streams** (v3): scale-out; multiple backend instances share the stream

**Transport**: SSE with `Last-Event-ID` (HTTP/1.1 compatible, simpler reconnection than WebSocket, sufficient for current unidirectional streaming). Reconsider WebSocket when bidirectional HITL channels (Slack relay, live approval push) are added.

---

## Human-in-the-Loop (HITL)

### v1: Chat UI approval gates

When a step has `approval: required`, the agent pauses and emits a special `approval_request` SSE event. The UI renders an inline approval card (Approve / Reject). The response is sent via `POST /api/projects/:id/approve` with `{runID, stepName, decision}`.

Pending questions to resolve before implementation:
- **Timeout policy**: what happens if no response arrives within N minutes? (deny / expire / escalate — no industry standard verified)
- **Reversibility-based gating**: auto-approve reversible actions (draft, summarize), require approval for irreversible ones (publish, email, spend) — promising pattern but specification is still in flux
- **State model**: approval state should be structured data, not parsed from chat messages

### v2+: External channels

Each approval request gets a stable `approvalURL` that can be opened outside the UI. External channels (Slack, webhook) POST to this URL. The backend resolves the pending approval regardless of which channel responds first.

---

## Persistence & State

| Store | Format | Location |
|---|---|---|
| Projects + session IDs | JSON | `~/.loop/data.json` |
| Settings (theme etc.) | JSON | `~/.loop/settings.json` |
| Run event log | JSONL | `~/.loop/runs/<runID>.jsonl` (v2) |
| Claude Code session | JSONL | `~/.claude/projects/<dirHash>/<sessionID>.jsonl` |
| ADL definitions | YAML | User-specified path per project, or `~/.loop/agents/<name>.yaml` |

---

## API Surface (planned additions)

| Method | Path | Purpose |
|---|---|---|
| `POST` | `/api/events/:name` | Emit a named event to trigger autonomous agents |
| `POST` | `/api/projects/:id/approve` | Respond to a HITL approval gate |
| `GET` | `/api/projects/:id/runs` | List runs for a project |
| `GET` | `/api/projects/:id/runs/:runID` | Get run status + event log |
| `GET` | `/api/agent-definitions` | List available ADL files |
| `POST` | `/api/agent-definitions` | Upload / register a new ADL file |

---

## Open Design Questions

1. **ADL versioning**: how should the system handle a step's tool or model requirements changing between runs of a long-running autonomous agent? Schema version + migration plan needed.
2. **HITL boundary in ADL**: should approval gates be a field on each step (`approval: required`), a separate top-level `approvalPolicy` block, or inferred from action reversibility? Current draft uses per-step field.
3. **SSE vs WebSocket at HITL scale**: SSE is sufficient for token streaming, but bidirectional HITL (Slack relay pushing approval responses back) may require a WebSocket or long-poll upgrade path.
4. ~~**Approval timeout default**~~ — resolved: deny-on-timeout confirmed as the standard across LangGraph, Temporal, Dapr, and Microsoft Agent Framework. `approvalTimeout` with deny-on-expiry is the ADL default.
5. **Docker harness security**: Docker alone is insufficient for untrusted agents (namespace escapes); gVisor or Firecracker micro-VMs should be evaluated for the sandboxing layer.

---

## Implementation Phases

### Phase 1 (current)
- [x] Interactive chat with Claude Code harness
- [x] Project CRUD with persistence (`~/.loop/data.json`)
- [x] Chat history from Claude session files
- [x] Theme settings (`~/.loop/settings.json`)
- [x] Extension agent framework (TCP JSON-RPC for built-in types)
- [x] Docker harness (container lifecycle + HTTP/SSE protocol)
- [x] Remote harness (user-configured host:port + HTTP/SSE protocol)

### Phase 2
- [ ] ADL file format (v1 schema above)
- [ ] Multi-step DAG execution (`dependsOn`, `parallel`, `sequential` policies)
- [ ] Named outputs / typed inputs between steps
- [ ] HITL approval gates in chat UI (`approval: required`, `approvalTimeout`)
- [ ] Step-level durable run log (`~/.loop/runs/`)
- [ ] SSE reconnection with `Last-Event-ID` replay

### Phase 3
- [ ] Autonomous mode: cron + event triggers (`schedule.cron`)
- [ ] `loop` and `batch` execution policies

### Phase 4
- [ ] External HITL channels (Slack, webhook)
- [ ] ADL skill references (SKILL.md)
- [ ] MCP server configuration in ADL
