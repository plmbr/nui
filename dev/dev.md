# nui — Product & Technical Specification

> **Status:** This document describes the product vision and the architecture **as implemented today**, with a roadmap for planned features. The Go code is the source of truth; sections marked *planned* are not yet enforced at runtime.

## Vision

nui is a self-hosted Go application with a bundled React web UI for creating and running AI agent sessions. Agent types are declared in ADL (Agent Definition Language); harnesses run as local subprocesses, Docker containers, or remote HTTP/SSE servers.

---

## Architecture (as built)

```mermaid
flowchart TB
  subgraph browser [Browser]
    UI[React UI]
  end

  subgraph nui_server [nui Go server]
    REST[REST API]
    AGUI[AG-UI endpoint]
    Store[(~/.nui/data.json)]
    ADLExec[ADLAgent]
    Mgr[Manager]
  end

  subgraph local [Local harnesses]
    Claude[claude CLI]
    Pi[pi CLI]
    Codex[codex CLI]
    OpenCode[opencode CLI]
  end

  subgraph http_agents [HTTP/SSE agents]
    UserDocker[User Docker image]
    RemoteSrv[Remote server]
    BuiltinDocker[nui-* images :8090]
  end

  UI -->|REST| REST
  UI -->|AG-UI SSE| AGUI
  REST <--> Store
  AGUI --> ADLExec
  ADLExec --> Mgr
  Mgr --> Claude & Pi & Codex & OpenCode
  Mgr --> UserDocker & RemoteSrv & BuiltinDocker
```

### Key design decisions

1. **Every session is an ADL agent.** Even the four built-in CLI harnesses are compiled-in ADL definitions (`builtinAgentDefs` in `api.go`). Selecting "Claude Code" in the UI stores `agentType: "claude-code"` (the ADL `id`), which resolves to `harness.type: claude-code`.

2. **Chat uses AG-UI, not raw SSE.** The UI (`useSessionChat.ts`) streams via `POST /api/sessions/:id/ag-ui` using the [AG-UI protocol](https://github.com/ag-ui-protocol/ag-ui). Tool calls, images, and MCP app frames are translated from agent `Event` types in `agui.go`. The legacy `POST /chat` endpoint still exists but the UI does not use it.

3. **Three production harness paths.**
   - **Go subprocess:** builtin harnesses (`claude-code`, `pi`, `codex`, `opencode`) managed directly in Go.
   - **HTTP/SSE:** docker, devcontainer, remote, and builtin `sandbox: docker` via `HTTPExtensionAgent`.
   - **Extension harnesses:** installed extensions contribute harnesses wired via `Manager.getExtensionHarnessAgent()` — stdio (default), TCP (`ExtensionAgent`), or HTTP (`HTTPExtensionAgent`). ADL references them as `harness.type: ext:<extension>/<harness-id>`.
   - **Reference only:** standalone examples in `dev/harness-examples/py|ts/` (no `extension.yaml`) demonstrate the TCP JSON-RPC protocol but are not registered as agent types.

4. **Docker/remote via custom ADL.** There is no built-in "Docker" or "Remote" picker in the UI. Users copy an ADL template from `dev/harness-examples/` into `~/.nui/agents/` (e.g. `docker-echo.yaml`), then select it under **Installed agents**. nui validates the connector on session create.

5. **CLI launch + UI preferences.** `nui ui -a <agent-id> --prompt --open` starts the HTTP server first, then creates a session via the same logic as `POST /api/launch` (shared with the warm-attach path when the server is already running). Session creation saves `lastAgentType` / `lastSessionId` to `settings.json` and exposes the prompt once via `GET /api/bootstrap`. `nui ui --open` (without `-a`) also creates a fresh session with the default agent. With `--open`, nui opens the browser to `/sessions/<id>` after the session is ready. If no sessions exist at startup and no launch flags were passed, nui auto-creates one with the default agent when the UI loads. Sidebar state and last-selected session/agent are also persisted in `settings.json`.

---

## Agent Definition Language (ADL)

ADL is a YAML format for declaring an agent type or multi-step workflow. It is a **static definition** — it describes *what* an agent is, not *how* nui executes it internally.

### Three-layer architecture

- **ADL** — agent identity, steps, harness, `aiAssets` (*this layer*)
- **MCP** — runtime tool protocol; ADL `aiAssets.mcpServers` is provisioned into per-session harness config; nui UI MCP tool frames also use `~/.nui/.mcp.json`
- **SKILL.md** — referenced via `aiAssets.skills` (path, ref, content, or git+path); copied into session harness config. Legacy top-level `skill:` is still supported.

### Schema

```yaml
adl: "1.0"

id: string                      # stable identifier; used by CLI (-a) and Session.agentType
name: string                    # display name in UI
description: string
version: semver
kind: agent | workflow          # workflow = multi-step orchestration; not selectable at session create

promptMode: user | auto         # auto hides input and runs default or launch prompt
defaultPrompt: string           # optional; auto mode when no launch prompt (default: built-in phrase)
workingDirInput: bool           # true = user picks working dir at session create; default uses ~/.nui/workspaces/<session-id>

promptSuggestions:              # quick-start pills above chat input (optional)
  - title: Review code
    prompt: Review the current changes and suggest improvements.
    icon: sparkles              # lucide icon name (optional)

harness:
  type: claude-code | pi | codex | opencode | docker | devcontainer | remote | ext:<extension>/<harness-id>
  model: string
  sandbox: none | bubblewrap | docker   # subprocess harnesses only; default: none
  innerHarness: claude-code         # harness.type:devcontainer
  image: string                   # optional devcontainer image override
  containerPort: 9090             # harness.type:docker only (user images; builtin sandbox images use 8090)
  host: string                    # harness.type:remote
  port: 9090                      # harness.type:remote
  env:                            # per-harness env (overrides top-level env on conflict)
    ANTHROPIC_API_KEY: string
  permissions: interactive | bypass # claude-code/codex native tool approval gate; default bypass

toolApprovals:                    # selective auto-approve when harness.permissions is interactive
  policy: default | all | allowlist | denylist
  tools:                          # required for allowlist or denylist
    - Read
    - Bash
    - mcp__my-server__*

hitl:
  mode: interactive | auto | off
  required: bool                  # semi-autonomous: human involvement mandatory (conflicts with promptMode auto)
  channels: [nui-ui, ext:...]
  ttlSeconds: 3600                # optional request TTL
  # approvals: [bash, write]      # deprecated; use toolApprovals with policy denylist instead

env:                              # global env for all harness subprocesses
  ANTHROPIC_BASE_URL: string

systemPrompt: |
  You are ...

skill: /path/to/skill-dir        # deprecated; use aiAssets.skills

aiAssets:
  mcpServers:
    - name: my-mcp-server        # required; used as the MCP server key in harness config
      ref: ext:corp-pack/tools   # or inline url/command (see extension-api.md)
      url: http://localhost:3000/mcp   # HTTP/SSE MCP (remote)
      type: http                 # http | sse (default: http when url is set)
    - name: local-mcp
      command: npx               # stdio MCP
      args: ["-y", "some-package"]
      type: stdio
  skills:
    - name: code-review          # required; install dir name in harness config
      path: ./skills/code-review # local dir or SKILL.md
    - name: commit-helper
      ref: commit-helper         # ~/.nui/skills/<ref>/ or builtin:* / ext:*
    - name: greeting
      content: |                 # inline SKILL.md (including frontmatter)
        ---
        name: greeting
        description: Brief greeting skill
        ---
        Keep responses to one sentence.
    - name: shared-style
      git: https://github.com/example/agent-skills.git
      path: skills/shared-style  # required with git; relative skill dir in repo
      version: v1.0.0            # optional tag/commit
  rules:
    - name: corp-guidelines
      ref: ext:corp-pack/corp-guidelines   # or path/content
  mentionProviders:
    - ref: ext:corp-pack/corp-refs         # @-mention autocomplete sources

subAgents:                        # orchestrator: route each user turn to a registry agent (mutually exclusive with steps)
  - hello-world
  - code-reviewer

steps:                            # omit for single-step agents
  - name: research
    type: agent                   # default; runs a harness step
    harness:                      # optional per-step override
      type: claude-code
      model: claude-opus-4-8
    dependsOn: []
    systemPrompt: |               # optional step override
      ...
    aiAssets:                     # optional step override
      mcpServers:
        - name: docs
          url: http://localhost:3040
          type: http
    outputs:
      - name: report
        type: text
    inputs:
      - from: other.report
        as: alias

  - name: approve-release
    type: hitl                    # orchestration gate — pauses workflow for human input
    dependsOn: [research]
    hitl:
      kind: approval              # approval | question | review
      title: Release approval
      message: Review the research report and approve release.
      actions:
        - id: approve
          label: Approve
        - id: reject
          label: Reject
      display:
        - from: research.report   # inject prior step output into HITL payload
      channels: [nui-ui]
```

#### Multi-step execution semantics

- Steps run **sequentially** in topological order (`dependsOn`). Independent branches are not executed in parallel.
- Each user chat turn **re-runs all steps** from the beginning. There is no cross-turn step state.
- Multi-step workflows do not persist harness session IDs in `agentSessions`; single-step agents do.

#### Orchestrator sub-agents

- `subAgents` lists canonical registry agent IDs only (builtins, `~/.nui/agents/`, `ext:…`). Names and descriptions are resolved from the registry at runtime — do not duplicate them in the orchestrator YAML.
- Mutually exclusive with `steps[]` / `kind: workflow`.
- Each user message triggers an ephemeral routing turn on the orchestrator harness, then delegates to the selected sub-agent with full event streaming (not tool-calling).
- Each sub-agent maintains its own harness session within the nui session (`agentSessions` key `{sessionId}#{subAgentId}`).

#### Named outputs and inputs

- When a step omits `outputs`, its full text is stored as an implicit default output.
- When `outputs` lists names (e.g. `brief`), the step's collected text is stored under each declared name (text-only; all names alias the same content).
- Downstream steps reference outputs with `inputs[].from: stepName.outputName` (e.g. `research.brief`) or `dependsOn` for the step's default output.
- `inputs[].filter` is reserved and not yet implemented.

#### HITL configuration

| Mechanism | When to use |
|---|---|
| `harness.permissions: interactive` + `toolApprovals` | Per-tool approve/deny during an agent run (Claude Code, Codex) |
| Top-level `hitl` block | Runtime ask-user via nui HITL MCP + skill injection |
| `steps[].type: hitl` | Orchestration gate between workflow steps |
| `subAgents` | Orchestrator routes user prompts to registry agents by id |

`hitl.approvals` is deprecated; use `toolApprovals` with `harness.permissions: interactive` instead.

#### MCP configuration

- **ADL `aiAssets.mcpServers`** — provisioned into per-session harness config (agent subprocess tools).
- **`~/.nui/.mcp.json`** — nui UI MCP tool frames only; not merged with ADL agent tools.

Per-step `aiAssets` **merges** with top-level entries by name (step entries override same name).

Example

```yaml
adl: "1.0"
id: example-agent
name: Example Agent
description: Example Agent
harness:
  type: claude-code
  model: claude-sonnet-4-6
  env:
    ANTHROPIC_API_KEY: your-api-key

env:
  ANTHROPIC_BASE_URL: https://api.anthropic.com

systemPrompt: |
  You are a helpful assistant.

aiAssets:
  mcpServers:
    - name: example-mcp-server
      url: https://example.org/mcp
      type: http
```

Place agent YAML files in `~/.nui/agents/` to make them selectable under **Installed agents** in the UI.

### Session harness config

For each nui session, ADL dependencies are materialized under `~/.nui/sessions/<session-id>/` and passed to harnesses via config-dir environment variables:

| Harness | Env var | Provisioned files (examples) |
|---|---|---|
| `claude-code` | `CLAUDE_CONFIG_DIR` | `CLAUDE.md`, `rules/…`, `.claude.json`, `skills/…` |
| `codex` | `CODEX_HOME` | `AGENTS.md`, `rules/…`, `config.toml`, `skills/…` |
| `pi` | `PI_CODING_AGENT_DIR` | `pi-agent/SYSTEM.md`, `pi-agent/rules/…`, `pi-agent/mcp.json`, `pi-agent/skills/…` |
| `opencode` | `OPENCODE_CONFIG_DIR` | `INSTRUCTIONS.md`, `rules/…`, `opencode.json`, `skills/…` |

ADL `env` (global) and `harness.env` are merged and set on harness subprocess environments. Harness keys override global keys. Host environment variables are inherited unless overridden.

`promptMode: auto` hides the chat input and automatically sends `defaultPrompt` (or `"Follow your system instructions and run."` when omitted). CLI `-m` and bootstrap prompts override the default for that launch.

### Harness types (implemented)

| Harness | Local (`sandbox: none`) | Bubblewrap (`sandbox: bubblewrap`) | Docker (`sandbox: docker`) |
|---|---|---|---|
| `claude-code` | `ClaudeCodeAgent` → `claude` CLI | bwrap + `~/.claude` | `nui-claude-code:latest` :8090 |
| `pi` | `PiAgent` → `pi --mode rpc` | bwrap + `~/.pi` | `nui-pi:latest` :8090 |
| `codex` | `CodexAgent` → `codex exec` | bwrap + `~/.codex` | `nui-codex:latest` :8090 |
| `opencode` | `OpenCodeAgent` → `opencode serve/run` | bwrap + `~/.local/share/opencode` | `nui-opencode:latest` :8090 |
| `docker` | — | — | User image at ADL `containerPort` (e.g. 9090) |
| `devcontainer` | — | — | nui-managed devcontainer + `innerHarness` CLI |
| `remote` | — | — | User `host:port` over HTTP/SSE |
| `ext:<ext>/<id>` | Extension host (stdio/tcp/http) | — | — |

Extension harnesses are contributed by installed extensions. See [extension-api.md](extension-api.md).

nui also auto-injects the `nui-viz` MCP server and `builtin:visualize` skill for inline chart rendering in chat (`internal/agent/harness_viz.go`).

### Persistent memory

Markdown memory files under `~/.nui/memory/`:

| File | Scope |
|---|---|
| `user.md` | Cross-agent user preferences and durable facts |
| `agents/<adl-agent-id>.md` | Per-agent learned context |

Memory is **not** ADL rules — it is mutable, agent-editable state re-read on every harness run. Configure modes in **Customize → Memory** (user + per-agent).

Each layer supports **auto**, **manual**, or **disabled**:

| Mode | Read (inject) | Write (save) |
|------|---------------|--------------|
| disabled | No | No |
| manual | Yes | On user request (`/remember`, "remember this") |
| auto | Yes | Agent proactively saves durable decisions |

Default is **manual**. The `remember` skill is attached when any layer is not disabled. In **auto** mode, a system-prompt appendix encourages proactive `update_memory` calls.

Agents update memory via the builtin `remember` skill, direct file writes (CLI harnesses), or the `update_memory` tool on the `nui-agent` MCP server. Writes to a **disabled** scope are rejected by the MCP tool.

Extensions may register **storage handlers** to replace built-in persistence for session history, agent memory, or user memory per agent type. See [extension-api.md](extension-api.md#storage-handlers). Memory **modes** (`auto` / `manual` / `disabled`) remain nui-owned; handlers control **where** data is stored.

Sandbox config flows: ADL `harness.sandbox` → `harnessBuiltinConfig()` → `Manager.getBuiltinAgent()` → agent struct `Sandbox` field.

### ADL executor

| Feature | Status |
|---|---|
| YAML parse into `ADLDefinition` | Done |
| Topological sort (`dependsOn`) | Done |
| Per-step harness/model/systemPrompt | Done |
| Named outputs → downstream inputs | Done |
| All six harness types + sandbox | Done |
| `aiAssets.mcpServers` → session harness config | Done |
| `skill` + `systemPrompt` → session harness config | Done (legacy `skill:`; prefer `aiAssets.skills`) |
| `aiAssets.skills` → resolve + install into session | Done |
| `env` / `harness.env` → subprocess environment | Done |
| `promptMode` / `defaultPrompt` → UI auto-run | Done |
| `promptSuggestions` → chat UI pills | Done |
| `aiAssets.rules` → harness rule files | Done |
| Persistent memory (`~/.nui/memory/`) → system prompt | Done |
| `aiAssets.mentionProviders` → @-mention menu | Done |
| `steps[].type: hitl` orchestration gates | Done |
| `subAgents` orchestrator routing | Done |
| Extension harness `ext:<ext>/<id>` | Done |

---

## Agent Runtime

### Go `Agent` interface

```go
type Agent interface {
    Name() string
    Run(ctx context.Context, req RunRequest, events chan<- Event) error
}
```

`ADLAgent` is the orchestrator. `Manager` caches one builtin agent per session ID **and harness type** and manages Docker container lifecycle (idle reaper at 30 min).

### HTTP/SSE harness protocol

Used by `docker`, `remote`, and builtin sandbox containers.

| Endpoint | Description |
|---|---|
| `GET /info` | `{"name","version","capabilities"}` — health check |
| `POST /run` | Body: `{message, sessionId?, workingDir?, systemPrompt?, model?}` → SSE |
| `POST /cancel` | Body: `{runId}` — cancel run best-effort |
| `POST /shutdown` | Stop subprocesses; nui calls this before `docker stop` |

SSE events (JSON in `data:` lines):

```
{"type":"text","content":"..."}
{"type":"done","sessionId":"..."}
{"type":"error","error":"..."}
```

Also supported: `tool_call_start`, `tool_call_args`, `tool_call_end`, `tool_call_result`, `image` (see `extension.go`).

Examples: `dev/harness-examples/docker/`, `dev/harness-examples/remote/`, `docker/http_nui_agent.py`.

### Extension harness protocol (stdio / TCP)

Used by installed extension harnesses (`ext:<extension>/<harness-id>`). See [extension-api.md](extension-api.md) and [harness-design.md](harness-design.md) §3.

| Method | Description |
|---|---|
| `harness.info` | Metadata |
| `harness.run` | Streams `harness.event` notifications |
| `harness.cancel` | Cancel run |
| `harness.shutdown` | Release resources |

Framework: `harness-sdk/nui_agent_stdio.py` (stdio), `harness-sdk/nui_agent.py` (TCP reference).

---

## UI / Backend

### Chat persistence

- UI messages (user + assistant text) saved to `~/.nui/data.json` → `sessionMessages` after each turn
- On session open: load `sessionMessages` if present, else fall back to agent history files
- Tool call bubbles and images are **not** persisted across restarts (AG-UI state is in-memory during the session)

### Reconnection

| Stream | Replay on disconnect? | Status |
|---|---|---|
| `GET /api/sessions/:id/runs/:runId/events` | Yes — `Last-Event-ID` replays from `~/.nui/runs/<runId>.jsonl` | Done |
| UI refresh during active headless run | Partial — `sessionChatStore.reconnectActiveRun()` re-attaches via runs API | Done |
| `POST /api/sessions/:id/ag-ui` (interactive chat) | No durable offset replay | *Planned* |

Headless runs persist events to JSONL and support SSE replay via the `Last-Event-ID` header (`runs_api.go`). The UI re-attaches to in-flight runs after page refresh by listing active runs and subscribing to their event streams.

Interactive AG-UI chat does not yet support mid-stream offset replay. A disconnect during an AG-UI turn loses the in-flight stream (persisted text messages survive via `sessionMessages`).

---

## Persistence

| Store | Format | Location | Status |
|---|---|---|---|
| Sessions + agent session IDs + UI messages | JSON | `~/.nui/data.json` | Done |
| Settings | JSON | `~/.nui/settings.json` | Done (`theme`, `lastAgentType`, `lastSessionId`, `sidebarOpen`, `disabledExtensions`) |
| ADL definitions | YAML | `~/.nui/agents/*.yaml` | Done |
| Run event log | JSONL | `~/.nui/runs/<runID>.jsonl` | Done |
| Schedules | JSON | `~/.nui/schedules.json` | Done |
| Claude Code sessions | JSONL | `~/.claude/projects/<dirHash>/` | External |
| pi / codex / opencode sessions | varies | Harness-specific paths | External |

Example ADL templates for docker/remote harness walkthroughs: `dev/harness-examples/docker/docker-echo.yaml`, `dev/harness-examples/remote/remote-echo.yaml`, `dev/harness-examples/docker/opencode-docker.yaml`.

---

## API Surface

### Implemented

| Method | Path | Purpose |
|---|---|---|
| `GET/POST` | `/api/sessions` | List / create (docker/remote config validated on create; agents start on first message) |
| `POST` | `/api/sessions/ensure-default` | Return last session or create one with the default agent |
| `GET` | `/api/sessions/events` | Global session list SSE (`changed`) |
| `GET/PATCH/DELETE` | `/api/sessions/:id` | Get / rename / delete |
| `GET/PUT` | `/api/sessions/:id/messages` | Persisted UI messages |
| `POST/GET` | `/api/sessions/:id/uploads[/:file]` | File uploads for chat attachments |
| `GET` | `/api/sessions/:id/mentions` | @-mention autocomplete |
| `POST` | `/api/sessions/:id/ag-ui` | AG-UI chat stream |
| `POST` | `/api/sessions/:id/chat` | Legacy agent-event SSE |
| `POST` | `/api/sessions/:id/runs` | Start async headless run (`202` + `runId`) |
| `GET` | `/api/sessions/:id/runs` | List runs for session |
| `GET` | `/api/sessions/:id/runs/:runId` | Run status and output |
| `GET` | `/api/sessions/:id/runs/:runId/events` | SSE event stream with `Last-Event-ID` replay |
| `POST` | `/api/sessions/:id/runs/:runId/hitl` | Create HITL request scoped to run |
| `POST` | `/api/sessions/:id/stop` | Cancel in-flight run (`?runId=` optional) |
| `GET` | `/api/sessions/:id/history` | Agent-side history |
| `GET` | `/api/agent-types` | Builtin + ADL + extension agent types |
| `GET` | `/api/directories` | Working-dir suggestions |
| `GET/PUT` | `/api/settings` | User preferences (partial PUT; includes memory toggles) |
| `GET` | `/api/bootstrap` | One-shot CLI bootstrap (`sessionId`, `initialPrompt`) |
| `POST` | `/api/launch` | Create session + optional initial prompt |
| `GET` | `/api/capabilities` | Bwrap availability |
| `GET` | `/api/extensions` | Installed extensions |
| `POST` | `/api/extensions/reload` | Rescan extensions |
| `GET/PUT` | `/api/mcp-servers` | User MCP server config |
| `GET/DELETE` | `/api/skills[/:name]` | Skill catalog |
| `GET` | `/api/memory` | Memory summary (user + agent files) |
| `GET/PUT` | `/api/memory/user` | User memory markdown |
| `GET/PUT/DELETE` | `/api/memory/agents/:id` | Per-agent memory markdown |
| `GET/POST/PUT/DELETE` | `/api/agents[/:file]` | User ADL agent CRUD |
| `POST` | `/api/agents/:id/deploy` | Deploy agent via extension deployer |
| `GET` | `/api/agent-deployers` | List deployers |
| `GET/POST/PATCH/DELETE` | `/api/schedules[/:id]` | Schedule CRUD |
| `POST` | `/api/schedules/:id/run-now` | Trigger schedule immediately |
| `POST/GET` | `/api/hitl/requests[/:id]` | HITL request CRUD + wait/respond |
| `GET` | `/api/hitl-channels` | Available HITL delivery channels |
| `POST/GET` | `/mcp-call-tool`, `/mcp-resource` | MCP proxy for UI tool frames |

### Planned

| Method | Path | Purpose |
|---|---|---|
| _(none)_ | | Headless runs API implemented — see Implemented table |

---

## Implementation Phases

### Phase 1 — Interactive chat ✅
- [x] Four builtin CLI harnesses (claude-code, pi, codex, opencode)
- [x] Session CRUD + persistence (`data.json` including UI messages)
- [x] AG-UI chat streaming with tool calls and images
- [x] HTTP/SSE docker + remote connectors
- [x] Builtin sandbox Docker images (`docker/`, port 8090)
- [x] CLI session launch (`nui ui --agent-type --prompt --working-dir`)
- [x] UI preferences (`lastAgentType`, `lastSessionId`, `sidebarOpen`)
- [x] Bubblewrap sandbox for all four CLI harnesses (Linux)
- [x] User ADL in `~/.nui/agents/*.yaml`
- [x] Docker/remote reachability check on session create

### Phase 2 — ADL workflows ✅
- [x] ADL YAML schema + parser
- [x] Multi-step DAG (`dependsOn`, topo sort)
- [x] Named outputs / inputs between steps
- [x] Per-step harness override + sandbox propagation
- [x] Durable run log + SSE reconnection

### Phase 3 — External integrations
- [x] ADL `skill` references (SKILL.md) → session harness config
- [x] ADL `aiAssets.skills` (path, ref, content, git+path) → catalog + session harness config
- [x] `nui skills add|list|remove` CLI
- [x] `nui memory list|show|edit` CLI
- [x] Persistent memory (`~/.nui/memory/`) with UI toggles and agent write path
- [x] `nui extension add|remove` CLI
- [x] ADL `aiAssets.mcpServers` → session harness config

### Phase 4 — Scheduled runs ✅
- [x] Interval schedules align to clock boundaries (e.g. `5m` at :00, :05, :10… UTC), not current time + interval
- [x] Server scheduler: each tick creates a new session + headless run
- [x] Only `promptMode: auto` ADL agents are schedulable
- [x] REST API + `nui schedule` CLI
- [x] Customize → Schedules UI
- [x] Session sidebar: scheduled indicator + relative last-run time

---

## Open Questions

1. **AG-UI chat replay** — add offset-based durable replay for interactive chat streams?
2. **Chat persistence scope** — persist tool calls/images in `sessionMessages` or separate store?
3. **Docker security** — gVisor/Firecracker for untrusted agents?

See also: [harness-design.md](harness-design.md), [adl/examples/README.md](adl/examples/README.md), [harness-examples/](harness-examples/).
