---
layout: page
title: MCP integration
subtitle: Expose nui to external MCP hosts and inject built-in MCP servers into agent harnesses.
permalink: /features/mcp/
---

nui integrates with the [Model Context Protocol](https://modelcontextprotocol.io/) in two directions: as an MCP server for external hosts, and as an MCP client that injects servers into agent harnesses.

## nui as an MCP server

Expose nui agents to MCP hosts (Cursor, Claude Desktop, etc.) by adding this to your MCP config:

```json
{
  "mcpServers": {
    "nui": {
      "command": "nui",
      "args": ["mcp"],
      "env": { "NUI_URL": "http://127.0.0.1:8080" }
    }
  }
}
```

Available tools: `list_agents`, `list_sessions`, `create_session`, `run_agent`, `get_run`, `get_run_events`, `stop_run`.

The server must be running (`nui server`) for MCP tools to work.

## Built-in MCP servers

nui injects these MCP servers into agent harnesses when configured:

| MCP server | Command | Purpose |
|---|---|---|
| `nui-hitl` | `nui hitl-mcp` | Human-in-the-loop prompts (`ask_user`, approvals) |
| `nui-viz` | `nui viz-mcp` | Inline chart/visualization rendering in chat |
| `nui-agent` | `nui agent-mcp` | Save ADL agents (`save_agent`) and update memory (`update_memory`) |

## Extension MCP servers

Installed extensions can contribute MCP server definitions via `contributions.mcpServers` in `extension.yaml`. These are provisioned to harness subprocesses alongside built-in servers.
