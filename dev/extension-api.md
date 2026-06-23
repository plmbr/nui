# Loop Extension API

Extensions live under `~/.loop/extensions/<name>/` and contribute backend capabilities to Loop: **harnesses**, **mcpServers**, **skills**, and **agents**.

## Layout

```
~/.loop/extensions/
  corp-pack/
    extension.yaml       # manifest (required)
    harnesses.yaml       # list of harnesses (optional)
    mcp-servers.json     # list of MCP servers (optional)
    skills.yaml          # list of skills (optional)
    agents.yaml          # list of ADL agents (optional)
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
  harnesses:
    source:
      file: harnesses.yaml   # or command: ["python3", "catalog.py"]
    runtime:
      transport: stdio       # stdio | tcp | http
      command: ["python3", "harness_host.py"]

  mcpServers:
    source:
      file: mcp-servers.json

  skills:
    source:
      file: skills.yaml

  agents:
    source:
      file: agents.yaml

  catalog:
    command: ["python3", "catalog.py"]   # optional shared list provider
```

**Source resolution** per contribution type: `source.file` → `source.command` → `catalog.command`.

## Contribution lists

All contributions are **lists**. Files may be JSON or YAML with a top-level array key matching the type (`harnesses`, `mcpServers`, `skills`, `agents`).

### Harnesses

Registered as agent types `ext:<extension>/<harness-id>`. Execution uses the harness wire protocol (`harness.info`, `harness.run`, `harness.cancel`, `harness.shutdown`) documented in [`harness-design.md`](harness-design.md).

Framework: [`harness-sdk/loop_agent_stdio.py`](../harness-sdk/loop_agent_stdio.py)

### MCP servers

Same schema as ADL `aiAssets.mcpServers`. Referenced in ADL as:

```yaml
aiAssets:
  mcpServers:
    - ref: ext:corp-pack/echo-tool
```

### Skills

Same schema as ADL `aiAssets.skills`. Paths resolve relative to the extension directory. Referenced as:

```yaml
aiAssets:
  skills:
    - ref: ext:corp-pack/code-review
```

### Agents

Full ADL agent definitions. IDs are namespaced as `ext:<extension>/<agent-id>`.

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
