# Loop Extension API

Extensions live under `~/.loop/extensions/<name>/` and contribute backend capabilities to Loop: **harnesses**, **catalog** (MCP servers and skills), **custom MCP tool servers**, **rules**, and **agents**.

## Layout

```
~/.loop/extensions/
  corp-pack/
    extension.yaml       # manifest (required)
    harnesses.yaml       # list of harnesses (optional)
    mcp-servers.json     # catalog MCP servers (optional)
    skills.yaml          # catalog skills (optional)
    agents.yaml          # list of ADL agents (optional)
    tools/               # CLI scripts for custom MCP tools (optional)
    harness_host.py      # harness runtime process (when using harnesses)
```

Copy the example from [`dev/extension-examples/corp-pack/`](extension-examples/corp-pack/) into `~/.loop/extensions/corp-pack/` to try it.

## Manifest

```yaml
apiVersion: loop.dev/extension/v1
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
            command: ["python3", "${LOOP_EXTENSION_DIR}/tools/echo.py"]
          - name: reverse
            description: Reverse a message
            command: ["python3", "${LOOP_EXTENSION_DIR}/tools/reverse.py"]
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
```

**Catalog source resolution** per list type: `source.file` → `source.command` → `catalog.command`.

Legacy root-level `mcpServers` and top-level `contributions.mcpServers` / `contributions.skills` are still supported but deprecated; use `contributions.aiAssets` and `contributions.catalog` instead.

## Contribution lists

Catalog list files may be JSON or YAML with a top-level array key matching the type (`mcpServers`, `skills`). Harnesses and agents use their own contribution keys.

### Harnesses

Registered as agent types `ext:<extension>/<harness-id>`. Execution uses the harness wire protocol (`harness.info`, `harness.run`, `harness.cancel`, `harness.shutdown`) documented in [`harness-design.md`](harness-design.md).

Framework: [`harness-sdk/loop_agent_stdio.py`](../harness-sdk/loop_agent_stdio.py)

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

Loop materializes these as stdio MCP servers using [`harness-sdk/loop_mcp_tools.py`](../harness-sdk/loop_mcp_tools.py) (copied to `~/.loop/harness-sdk/` on first use). Harness config is written when a session is created and refreshed on each run. Tool scripts read JSON from stdin:

```python
import json, sys
args = json.load(sys.stdin)
print(args.get("message", ""))
```

Set `LOOP_MCP_TOOLS_PATH` to override the proxy script location.

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

Loop materializes rules into harness-specific rule files under the session config directory:

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

Each item: `{label, value, hasChildren?, icon?}`. Selecting a leaf inserts `@value` into chat; Loop resolves mentions server-side before sending to the harness.

Built-in provider: **Files & folders** (`builtin:files`) lists files under the session working directory and resolves `file:<relative-path>` to the full absolute path.

SDK: [`harness-sdk/loop_mention.py`](../harness-sdk/loop_mention.py) (auto-installed to `~/.loop/harness-sdk/` on first use, same as `loop_mcp_tools.py`). Example: [`dev/extension-examples/corp-pack/mention_host.py`](extension-examples/corp-pack/mention_host.py).

Chat API: `GET /api/sessions/:id/mentions?parent=&query=`

## Catalog provider (dynamic lists)

When `source.command` or `contributions.catalog.command` is set, Loop spawns a stdio JSON-RPC process:

| Method | Result |
|--------|--------|
| `extension.initialize` | `{apiVersion, extensionName, capabilities}` |
| `extension.listHarnesses` | `{harnesses: [...]}` |
| `extension.listMCPServers` | `{mcpServers: [...]}` |
| `extension.listSkills` | `{skills: [...]}` |
| `extension.listAgents` | `{agents: [...]}` |
| `extension.shutdown` | cleanup |

Framework: [`harness-sdk/loop_catalog.py`](../harness-sdk/loop_catalog.py)

## API

| Endpoint | Description |
|----------|-------------|
| `GET /api/extensions` | Installed extensions and contribution item ids |
| `POST /api/extensions/reload` | Rescan `~/.loop/extensions/` |

Extension harnesses and agents appear in `GET /api/agent-types`.

## Security

Extensions run as the Loop user with full host access — equivalent to shell scripts and MCP server commands. Only install extensions you trust.

## Connection files (TCP/HTTP harnesses)

Harness processes that bind a TCP or HTTP port write handshake metadata to `~/.loop/connections/<id>.json` (not under `extensions/`). Loop reads these when `transport` is `tcp` or `http`. The connection id defaults to the harness name or `--project-id`; extension harnesses use `LOOP_CONNECTION_ID` (derived from `ext:<extension>/<harness-id>`).
