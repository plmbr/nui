---
layout: docs
title: Getting started with extensions
subtitle: Install corp-pack, understand the layout, and create your first extension agent.
permalink: /docs/extensions/getting-started/
---

This walkthrough uses the **corp-pack** example from the nui repository — a complete declarative extension with harnesses, custom MCP tools, catalog entries, skills, rules, agents, and mention providers.

## 1. Install corp-pack

From a nui source checkout:

```bash
nui extension add dev/extension-examples/corp-pack
nui extension list
```

You should see `corp-pack` with contributed harness ids `echo`, `reverse`, agents `echo-bot` and `reviewer`, and more.

Reload after edits during development:

```bash
curl -X POST http://127.0.0.1:8080/api/extensions/reload
```

## 2. Directory layout

After install, files live at `~/.nui/extensions/corp-pack/`:

```
corp-pack/
  extension.yaml          # manifest
  harnesses.yaml          # echo + reverse harnesses
  harness_host.py         # multiplex stdio host
  agents.yaml             # echo-bot + reviewer agents
  mcp-servers.json        # catalog MCP server
  skills.yaml             # catalog skill
  tools/
    echo.py               # custom MCP tool script
    reverse.py
  mention-providers.yaml
  mention_host.py
```

## 3. Try an extension harness

Create a session with agent type **`ext:corp-pack/echo-bot`** (or pick **Echo Bot** in the new-session UI). The agent uses harness `ext:corp-pack/echo`, which repeats your message.

The harness host dispatches by `NUI_HARNESS_ID`:

```python
#!/usr/bin/env python3
import os
import sys

sys.path.insert(0, os.path.expanduser("~/.nui/harness-sdk"))

from nui_agent_stdio import NuiAgent


class HarnessHost(NuiAgent):
    name = "corp-pack-host"
    version = "1.0.0"

    def run(self, message: str, run_id: str, **kwargs):
        harness_id = kwargs.get("harnessId") or os.environ.get("NUI_HARNESS_ID", "echo")
        if harness_id == "reverse":
            yield message[::-1]
        else:
            yield f"You said: {message}"


if __name__ == "__main__":
    HarnessHost().serve_stdio()
```

## 4. harnesses.yaml

```yaml
harnesses:
  - id: echo
    displayName: Echo
    description: Repeats your message
  - id: reverse
    displayName: Reverse
    description: Reverses your message
```

Registered agent types: `ext:corp-pack/echo`, `ext:corp-pack/reverse`.

## 5. agents.yaml

```yaml
agents:
  - id: echo-bot
    name: Echo Bot
    description: Uses the extension echo harness
    harness:
      type: ext:corp-pack/echo

  - id: reviewer
    name: Code Reviewer
    description: Claude with corp extension assets
    harness:
      type: claude-code
      model: claude-sonnet-4-6
    aiAssets:
      skills:
        - name: code-review
          ref: ext:corp-pack/code-review
      mcpServers:
        - name: echo-tool
          ref: ext:corp-pack/echo-tool
        - name: corp-tools
          ref: ext:corp-pack/corp-tools
      rules:
        - name: corp-guidelines
          ref: ext:corp-pack/corp-guidelines
```

Agent types: `ext:corp-pack/echo-bot`, `ext:corp-pack/reviewer`.

## 6. Custom MCP tool

`extension.yaml` declares a command-tool MCP server:

```yaml
aiAssets:
  mcpServers:
    - name: corp-tools
      tools:
        - name: echo
          description: Echo a message back
          command: ["python3", "${NUI_EXTENSION_DIR}/tools/echo.py"]
          inputSchema:
            type: object
            properties:
              message:
                type: string
            required: [message]
```

Tool script — arguments arrive as **JSON on stdin**:

```python
#!/usr/bin/env python3
import json
import sys

args = json.load(sys.stdin)
message = args.get("message", "")
print(f"echo: {message}")
```

nui materializes this as a stdio MCP server via `nui_mcp_tools.py`. Reference it in ADL with `ref: ext:corp-pack/corp-tools`.

## 7. Scaffold a new extension

```bash
nui extension create my-ext --lang python
nui extension create my-ext --lang npm
nui extension create my-ext --lang go
```

This generates a programmatic extension skeleton. For a declarative pack, copy `corp-pack` and edit `extension.yaml`.

## 8. Next steps

- [Manifest reference]({{ '/docs/extensions/manifest/' | relative_url }}) — every `contributions` field
- [Harnesses]({{ '/docs/extensions/harnesses/' | relative_url }}) — TCP and HTTP transports
- [HITL demo](https://github.com/plmbr/nui/tree/main/dev/extension-examples/hitl-demo/) — interactive prompts
- [Storage demo](https://github.com/plmbr/nui/tree/main/dev/extension-examples/storage-demo/) — custom persistence
