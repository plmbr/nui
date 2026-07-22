# nui Extension API

Extensions live under `~/.nui/extensions/<name>/` and contribute backend capabilities to nui: **harnesses**, **catalog** (MCP servers and skills), **custom MCP tool servers**, **rules**, **agent deployers**, and **agents**.

## Layout

```
~/.nui/extensions/
  corp-pack/
    extension.yaml       # manifest (required)
    harnesses.yaml       # list of harnesses (optional)
    mcp-servers.json     # catalog MCP servers (optional)
    skills.yaml          # catalog skills (optional)
    agents.yaml          # list of ADL agents (optional)
    tools/               # CLI scripts for custom MCP tools (optional)
    harness_host.py      # harness runtime process (when using harnesses)
    hitl-channels.yaml   # HITL delivery channels (optional)
    hitl_channel_host.py # HITL channel runtime (optional)
```

Copy the example from [`dev/extension-examples/corp-pack/`](extension-examples/corp-pack/) into `~/.nui/extensions/corp-pack/` to try it, or install it with the CLI:

```sh
nui extension add dev/extension-examples/corp-pack
nui extension add dev/extension-examples/hitl-demo   # HITL demo
nui extension add dev/extension-examples/storage-demo  # persistence demo
```

## Install

```sh
nui extension add <url-or-path>     # install from git URL, directory, or .zip
nui extension list                  # list installed extensions
nui extension remove <ext-id>       # remove by extension name (manifest id)
```

**Sources:**

| Source | Example |
|--------|---------|
| Local directory | `nui extension add ./my-extension` |
| Zip archive | `nui extension add ./corp-pack.zip` |
| Git repository | `nui extension add https://github.com/example/my-extension.git` |

For a **directory** or **zip**, nui looks for `extension.yaml` at the root or in exactly one immediate subdirectory.

For a **git URL**, nui shallow-clones the repo to a temporary directory, copies the extension into `~/.nui/extensions/<name>/`, then deletes the clone. The repository root (or a single top-level subdirectory containing `extension.yaml`) must be the extension package.

Re-installing replaces the existing copy under `~/.nui/extensions/<name>/`. Restart the UI or call `POST /api/extensions/reload` to pick up changes without restarting.

```sh
nui extension remove corp-pack
```

## Manifest

```yaml
apiVersion: nui.plmbr.dev/extension/v1
name: corp-pack              # must match directory name
version: 1.0.0
displayName: Corp Pack

contributions:
  aiAssets:
    mcpServers:
      - name: corp-tools
        tools:
          - name: echo
            description: Echo a message back
            command: ["python3", "${NUI_EXTENSION_DIR}/tools/echo.py"]
          - name: reverse
            description: Reverse a message
            command: ["python3", "${NUI_EXTENSION_DIR}/tools/reverse.py"]
    skills:
      - name: deploy-checklist
        content: |
          ---
          name: deploy-checklist
          description: Pre-deploy verification steps
          ---
          Run through the checklist before merging.
    rules:
      - name: corp-guidelines
        content: |
          Follow corporate security guidelines. Never commit secrets.
    agentDeployers:
      - name: docker
        description: Build and deploy agents as Docker images
        command: ["python3", "${NUI_EXTENSION_DIR}/deploy.py"]

  catalog:
    mcpServers:
      source:
        file: mcp-servers.json
    skills:
      source:
        file: skills.yaml
    command: ["python3", "catalog.py"]   # optional shared list provider

  harnesses:
    source:
      file: harnesses.yaml
    runtime:
      transport: stdio       # stdio | tcp | http
      command: ["python3", "harness_host.py"]

  agents:
    source:
      file: agents.yaml

  mentionProviders:
    source:
      file: mention-providers.yaml
    runtime:
      transport: stdio
      command: ["python3", "mention_host.py"]

  hitlChannels:
    source:
      file: hitl-channels.yaml
    runtime:
      transport: stdio
      command: ["python3", "hitl_channel_host.py"]
```

**Catalog source resolution** per list type: `source.file` → `source.command` → `catalog.command`.

Legacy root-level `mcpServers` and top-level `contributions.mcpServers` / `contributions.skills` are still supported but deprecated; use `contributions.aiAssets` and `contributions.catalog` instead.

## Contribution lists

Catalog list files may be JSON or YAML with a top-level array key matching the type (`mcpServers`, `skills`). Harnesses and agents use their own contribution keys.

### Harnesses

Registered as agent types `ext:<extension>/<harness-id>`. Execution uses the harness wire protocol (`harness.info`, `harness.run`, `harness.cancel`, `harness.shutdown`) documented in [`harness-design.md`](harness-design.md).

Framework: [`harness-sdk/nui_agent_stdio.py`](../harness-sdk/nui_agent_stdio.py)

### Catalog MCP servers

Standard MCP servers (stdio/http/sse). Same schema as ADL `aiAssets.mcpServers`. Referenced in ADL as:

```yaml
aiAssets:
  mcpServers:
    - ref: ext:corp-pack/echo-tool
```

### Custom MCP servers (aiAssets)

Command-tool MCP servers declared under `contributions.aiAssets.mcpServers`. Each server groups multiple tools; each tool runs a CLI command with MCP tool arguments passed as **JSON on stdin**. Referenced in agent ADL as:

```yaml
aiAssets:
  mcpServers:
    - ref: ext:corp-pack/corp-tools
```

| Field | Description |
|-------|-------------|
| `name` | Server name used in `ref: ext:<extension>/<name>` |
| `tools` | List of tools with `name`, `description`, `command`, and optional `inputSchema` |

Each tool may define `inputSchema` as a JSON Schema object (MCP `tools/list` exposes it to the harness). When omitted, the proxy defaults to a single required `message` string parameter.

nui materializes these as stdio MCP servers using [`harness-sdk/nui_mcp_tools.py`](../harness-sdk/nui_mcp_tools.py) (copied to `~/.nui/harness-sdk/` on first use). Harness config is written when a session is created and refreshed on each run. Tool scripts read JSON from stdin:

```python
import json, sys
args = json.load(sys.stdin)
print(args.get("message", ""))
```

Set `NUI_MCP_TOOLS_PATH` to override the proxy script location.

### Custom skills (aiAssets)

Skills declared under `contributions.aiAssets.skills`. Use `name`, `path`, and/or `content` (same schema as ADL `aiAssets.skills`). Referenced as:

```yaml
aiAssets:
  skills:
    - ref: ext:corp-pack/deploy-checklist
```

### Rules (aiAssets)

Rules are markdown instruction files declared under `contributions.aiAssets.rules`. Use `name`, `path`, and/or `content` — exactly one source is required. Referenced in agent ADL as:

```yaml
aiAssets:
  rules:
    - ref: ext:corp-pack/corp-guidelines
```

nui materializes rules into harness-specific rule files under the session config directory:

| Harness | Rule files | Registration |
|---------|------------|--------------|
| `claude-code` | `rules/<name>.md` | Auto-discovered by Claude Code |
| `codex` | `rules/<name>.md` | Listed in `config.toml` `instructions` |
| `pi` | `pi-agent/rules/<name>.md` | Claude-compatible layout under agent dir |
| `opencode` | `rules/<name>.md` | Listed in `opencode.json` `instructions` |

| Field | Description |
|-------|-------------|
| `name` | Rule identifier used in `ref: ext:<extension>/<name>` and the output filename |
| `content` | Inline rule markdown |
| `path` | File path relative to the extension directory |

### Catalog skills

Catalog skills are listed under `contributions.catalog.skills` for ADL `ref:` discovery. Same schema as ADL `aiAssets.skills`. Paths resolve relative to the extension directory. Referenced as:

```yaml
aiAssets:
  skills:
    - ref: ext:corp-pack/code-review
```

### Agents

Full ADL agent definitions. IDs are namespaced as `ext:<extension>/<agent-id>`. Agents reference extension MCP servers, skills, and rules explicitly via `ref:` in `aiAssets`.

### Mention providers

Extensions can contribute `@`-mention autocomplete sources for the chat input. Declared under `contributions.mentionProviders` with a list file and stdio runtime (same pattern as harnesses). Mention providers are **not** active globally — agents opt in via `aiAssets.mentionProviders`:

```yaml
aiAssets:
  mentionProviders:
    - ref: ext:corp-pack/corp-refs
```

When a session uses an agent with one or more mention provider refs, only those providers appear in the `@` menu (in addition to the built-in **Files & folders** provider).

`mention-providers.yaml`:

```yaml
mentionProviders:
  - id: corp-refs
    displayName: Corp References
```

Wire protocol (`mention.*` namespace):

| Method | Params | Result |
|--------|--------|--------|
| `mention.info` | `{}` | `{id, name, version, capabilities}` |
| `mention.list` | `{providerId, parent?, query?, limit?, workingDir?, sessionId?}` | `{items, breadcrumb?}` |
| `mention.resolve` | `{providerId, value, workingDir?, sessionId?}` | `{text}` |
| `mention.shutdown` | `{}` | `{ok: true}` |

Each item: `{label, value, hasChildren?, icon?}`. Selecting a leaf inserts `@value` into chat; nui resolves mentions server-side before sending to the harness.

Built-in provider: **Files & folders** (`builtin:files`) lists files under the session working directory and resolves `file:<relative-path>` to the full absolute path.

SDK: [`harness-sdk/nui_mention.py`](../harness-sdk/nui_mention.py) (auto-installed to `~/.nui/harness-sdk/` on first use, same as `nui_mcp_tools.py`). Example: [`dev/extension-examples/corp-pack/mention_host.py`](extension-examples/corp-pack/mention_host.py).

Chat API: `GET /api/sessions/:id/mentions?parent=&query=`

### HITL channels

Extensions can contribute **delivery channels** for human-in-the-loop prompts. Declared under `contributions.hitlChannels` with a list file and optional stdio runtime (same pattern as harnesses and mention providers). Channels are **not** active globally — agents opt in via ADL `hitl.channels`:

```yaml
hitl:
  mode: interactive
  channels:
    - nui-ui
    - ext:hitl-demo/demo-slack
```

Built-in channel: **nui UI** (`nui-ui`) renders prompt cards in chat.

`hitl-channels.yaml`:

```yaml
hitlChannels:
  - id: demo-slack
    displayName: Demo Slack Channel
    description: Forwards HITL prompts to Slack (example)
```

Channel ids use `ext:<extension>/<channel-id>` when referenced in agent ADL.

Wire protocol (`hitl.*` namespace) for stdio channel hosts:

| Method | Params | Result |
|--------|--------|--------|
| `hitl.info` | `{}` | `{id, name, version, capabilities}` |
| `hitl.deliver` | `{channelId, request, workingDir?, sessionId?}` | `{ok: true, ...}` |
| `hitl.shutdown` | `{}` | `{ok: true}` |

The `request` object is the canonical HITL envelope (`requestId`, `kind`, `payload`, `routing`, `status`, …).

SDK: [`harness-sdk/nui_hitl_channel.py`](../harness-sdk/nui_hitl_channel.py). Example: [`dev/extension-examples/hitl-demo/hitl_channel_host.py`](extension-examples/hitl-demo/hitl_channel_host.py).

#### Harness-agnostic HITL (REST)

Any harness — extension TCP/stdio hosts, remote HTTP agents, or custom MCP tool scripts — can create and wait on HITL requests via the nui REST API. nui sets `NUI_API_URL`, `NUI_SESSION_ID`, and `NUI_RUN_ID` on harness subprocesses.

| Step | HTTP |
|------|------|
| Create | `POST /api/hitl/requests` |
| Wait | `GET /api/hitl/requests/:id/wait` |
| Respond | `POST /api/hitl/requests/:id/respond` |
| List pending | `GET /api/hitl/requests?pending=true` |

Create body (minimal):

```json
{
  "sessionId": "...",
  "runId": "...",
  "kind": "question",
  "payload": {
    "title": "Confirm",
    "message": "Proceed?",
    "questions": []
  },
  "routing": { "channels": ["nui-ui", "ext:hitl-demo/demo-slack"] }
}
```

Python SDK helpers:

| Module | Use |
|--------|-----|
| [`harness-sdk/nui_hitl.py`](../harness-sdk/nui_hitl.py) | `ask_user()`, `request_approval()`, `create_request()`, `wait_response()` |
| [`harness-sdk/nui_agent.py`](../harness-sdk/nui_agent.py) | `NuiAgent.ask_user()` on TCP harnesses |
| [`harness-sdk/nui_agent_stdio.py`](../harness-sdk/nui_agent_stdio.py) | `NuiAgent.ask_user()` on stdio extension harnesses; emits `harness.event` with `type: hitl_request` |

Builtin harnesses (Claude, Codex, Pi, OpenCode) receive an injected **`nui-hitl`** MCP server (`nui hitl-mcp`) with `ask_user` and `request_approval` tools when `hitl.mode` is `interactive`.

Example extension: [`dev/extension-examples/hitl-demo/`](extension-examples/hitl-demo/).

#### REST-only origin bridge

For event-bus integrations (Kafka, webhooks, Slack), implement a channel **without** a stdio runtime. Declare the channel in `hitl-channels.yaml` for discovery (`GET /api/hitl-channels`), then run a sidecar process that polls and responds over REST:

1. `GET /api/hitl/requests?pending=true` — filter by `routing.channels` containing your channel id (e.g. `ext:hitl-demo/demo-webhook`).
2. Deliver the prompt to your external system (Slack message, Kafka event, etc.).
3. When the human answers, `POST /api/hitl/requests/:id/respond` with `{status, answers, respondedBy: {channel}}`.

Example sidecar: [`dev/extension-examples/hitl-demo/origin_bridge.py`](extension-examples/hitl-demo/origin_bridge.py).

HITL API:

| Endpoint | Description |
|----------|-------------|
| `GET /api/hitl-channels` | Built-in and extension channel ids |
| `POST /api/hitl/requests` | Create a HITL request |
| `GET /api/hitl/requests/:id/wait` | Block until answered |
| `POST /api/hitl/requests/:id/respond` | Submit an answer |
| `GET /api/hitl/requests?pending=true` | List pending requests |

## Storage handlers

Extensions can own persistence for three data domains: **session history** (chat messages + harness resume ids), **agent memory**, and **user memory**. Declared under `contributions.storage` with a handler list file and stdio runtime (same pattern as mention providers and HITL channels).

When a matching handler is installed for an agent type or memory scope, nui **skips built-in storage** for that scope (`data.json` session rows or `~/.nui/memory/` files). With no handler, behavior is unchanged.

| Kind | Built-in fallback | Match rule |
|------|-------------------|------------|
| `sessionHistory` | `data.json` (`sessionMessages`, `agentSessions`) | `agentTypes` on handler |
| `agentMemory` | `~/.nui/memory/agents/<id>.md` | `agentTypes` on handler |
| `userMemory` | `~/.nui/memory/user.md` | global (no `agentTypes`) |

**Read semantics:** session history uses the **first successful handler**; agent/user memory **merges** non-empty content from all handlers (`\n\n` between blocks).

**Write/delete semantics:** nui updates in-memory session state immediately, then **async fan-outs** to all matching handlers. Extension errors are logged to stderr only — API callers still succeed.

`extension.yaml`:

```yaml
contributions:
  storage:
    source:
      file: storage-handlers.yaml
    runtime:
      command: ["python3", "storage_host.py"]
```

`storage-handlers.yaml`:

```yaml
storageHandlers:
  - id: postgres-sessions
    kind: sessionHistory
    agentTypes: ["ext:corp/reviewer", "claude-code"]
  - id: agent-notes
    kind: agentMemory
    agentTypes: ["ext:corp/reviewer"]
  - id: user-cloud
    kind: userMemory
```

Wire protocol (`storage.*` namespace):

| Method | Params | Result |
|--------|--------|--------|
| `storage.info` | `{}` | `{id}` |
| `storage.session.read` | `{handlerId, sessionId, agentType, workingDir}` | `{messages, agentSessionId}` |
| `storage.session.write` | `{handlerId, sessionId, agentType, messages, agentSessionId, workingDir}` | `{ok}` |
| `storage.session.delete` | `{handlerId, sessionId, agentType, agentSessionId, workingDir}` | `{ok}` |
| `storage.agentMemory.read` | `{handlerId, agentId}` | `{content}` |
| `storage.agentMemory.write` | `{handlerId, agentId, content, writeMode}` | `{ok}` |
| `storage.agentMemory.delete` | `{handlerId, agentId}` | `{ok}` |
| `storage.userMemory.read` | `{handlerId}` | `{content}` |
| `storage.userMemory.write` | `{handlerId, content, writeMode}` | `{ok}` |
| `storage.userMemory.delete` | `{handlerId}` | `{ok}` |
| `storage.shutdown` | `{}` | `{ok: true}` |

`messages` use the same shape as nui chat messages. `writeMode` is `replace` or `append` (distinct from nui memory mode settings — modes remain nui-owned and gate **when** reads/writes happen).

Programmatic extensions: override `get_storage_handlers()` and `read_session` / `write_session` / memory methods on [`sdk/python/nui_extension/extension.py`](../sdk/python/nui_extension/extension.py) or [`sdk/go/nuiextension/extension.go`](../sdk/go/nuiextension/extension.go).

Declarative stdio host SDK: [`harness-sdk/nui_storage.py`](../harness-sdk/nui_storage.py) (auto-installed to `~/.nui/harness-sdk/` on first use).

Example extension: [`dev/extension-examples/storage-demo/`](extension-examples/storage-demo/).

Install:

```sh
nui extension add dev/extension-examples/storage-demo
```

## Agent deployers

Extensions may declare `contributions.aiAssets.agentDeployers` — named commands that deploy user ADL agents to a remote platform. Registry URLs, image tags, and auth are **extension-owned** (config files, env vars); nui only passes the agent definition and bundled assets.

Deployer ids use the `ext:<extension>/<name>` convention, for example `ext:docker-deployer/docker`.

**CLI:**

```sh
nui agent deployers
nui agent deploy ext:docker-deployer/docker my-agent
```

**Invocation:** nui spawns the deployer `command`, writes one JSON line to stdin, reads one JSON line from stdout.

Request:

```json
{
  "action": "deploy",
  "deployerId": "ext:docker-deployer/docker",
  "agentId": "my-agent",
  "definition": { "...ADLDefinition..." },
  "assets": { "skills": [], "mcpServers": [], "rules": [] }
}
```

Response:

```json
{
  "ok": true,
  "deploymentId": "nui-my-agent-1.0.0",
  "status": "ready",
  "message": "built image nui-my-agent:1.0.0",
  "endpoint": { "host": "127.0.0.1", "port": 9090 }
}
```

Example extension: [`dev/extension-examples/docker-deployer/`](extension-examples/docker-deployer/).

Install:

```sh
nui extension add dev/extension-examples/docker-deployer
```

## Catalog provider (dynamic lists)

When `source.command` or `contributions.catalog.command` is set, nui spawns a stdio JSON-RPC process:

| Method | Result |
|--------|--------|
| `extension.initialize` | `{apiVersion, extensionName, capabilities}` |
| `extension.listHarnesses` | `{harnesses: [...]}` |
| `extension.listMCPServers` | `{mcpServers: [...]}` |
| `extension.listSkills` | `{skills: [...]}` |
| `extension.listAgents` | `{agents: [...]}` |
| `extension.shutdown` | cleanup |

Framework: [`harness-sdk/nui_catalog.py`](../harness-sdk/nui_catalog.py)

## API

| Endpoint | Description |
|----------|-------------|
| `GET /api/extensions` | Installed extensions and contribution item ids |
| `POST /api/extensions/reload` | Rescan `~/.nui/extensions/` |
| `GET /api/agent-deployers` | Installed extension agent deployers |
| `POST /api/agents/:id/deploy` | Deploy user agent; body `{"deployerId":"ext:..."}` |
| `GET /api/hitl-channels` | HITL delivery channels (builtin + extensions) |
| `POST /api/hitl/requests` | Create a HITL request |
| `GET /api/hitl/requests/:id/wait` | Wait for HITL response |
| `POST /api/hitl/requests/:id/respond` | Respond to a HITL request |

Extension harnesses and agents appear in `GET /api/agent-types`.

## Security

Extensions run as the nui user with full host access — equivalent to shell scripts and MCP server commands. Only install extensions you trust.

## Connection files (TCP/HTTP harnesses)

Harness processes that bind a TCP or HTTP port write handshake metadata to `~/.nui/connections/<id>.json` (not under `extensions/`). nui reads these when `transport` is `tcp` or `http`. The connection id defaults to the harness name or `--project-id`; extension harnesses use `NUI_CONNECTION_ID` (derived from `ext:<extension>/<harness-id>`).
