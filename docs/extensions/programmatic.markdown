---
layout: docs
title: Programmatic extension SDK
subtitle: Python, TypeScript, and Go NuiExtension packages.
permalink: /docs/extensions/programmatic/
---

Programmatic extensions run as **external processes** and implement a `NuiExtension` base class. nui generates a minimal `extension.yaml` at install time (`runtime` + `install` only) and discovers capabilities via `extension.initialize` IPC.

Declarative packs (`extension.yaml` + static files) remain fully supported — use whichever fits your deployment model.

## Extension shapes compared

| Shape | Discovery | `extension.yaml` |
|-------|-----------|------------------|
| **Declarative pack** | nui reads yaml/files | Full `contributions` block |
| **Programmatic package** | `extension.initialize` → `getHarnesses()`, … | `runtime` + `install` only |

## `runtime` vs `install`

- **`runtime`** — how nui spawns the subprocess (`transport`, `command`). Used at registry load and for harness/mention/HITL RPC.
- **`install`** — package provenance for `nui extension upgrade/remove` only. Not sent to the extension process.

## Install programmatic packages

```bash
nui extension add npm:@corp/nui-ext@1.0.0
nui extension add pip:corp-nui-ext==1.0.0
nui extension add ./my-python-ext          # detects pyproject.toml
nui extension add ./my-ts-ext              # detects package.json

nui extension create my-ext --lang python
nui extension create my-ext --lang npm
nui extension create my-ext --lang go
```

## Python

Package: [`sdk/python/`](https://github.com/plmbr/nui/tree/main/sdk/python/)  
Runtime module (auto-copied): [`harness-sdk/nui_extension.py`](https://github.com/plmbr/nui/blob/main/harness-sdk/nui_extension.py)

```python
from nui_extension import NuiExtension


class CorpExtension(NuiExtension):
    def initialize(self):
        # load config from read_bundled("config.yaml")
        pass

    def get_harnesses(self):
        return [
            {"id": "echo", "displayName": "Echo", "description": "Repeats input"},
            {"id": "reverse", "displayName": "Reverse"},
        ]

    def run_harness(self, harness_id, message, ctx=None):
        if harness_id == "reverse":
            yield message[::-1]
        else:
            yield message

    def get_custom_mcp_servers(self):
        return [{
            "name": "tools",
            "tools": [{
                "name": "ping",
                "description": "Ping",
                "command": ["python3", f"{self.extension_dir}/ping.py"],
            }],
        }]

    def get_mention_providers(self):
        return [{"id": "refs", "displayName": "References"}]

    def list_mentions(self, provider_id, parent="", query="", limit=20, ctx=None):
        return {"items": [{"label": "Doc", "value": "doc:1", "hasChildren": False}]}

    def resolve_mention(self, provider_id, value, ctx=None):
        return {"text": f"Content for {value}"}


if __name__ == "__main__":
    CorpExtension().serve()
```

### Override reference

| Category | Methods |
|----------|---------|
| Lifecycle | `initialize()`, `shutdown()` |
| Discovery | `get_harnesses()`, `get_agents()`, `get_mcp_servers()`, `get_custom_mcp_servers()`, `get_skills()`, `get_custom_skills()`, `get_rules()`, `get_mention_providers()`, `get_hitl_channels()`, `get_storage_handlers()`, `get_deployers()` |
| Runtime | `run_harness()`, `cancel_harness()`, `list_mentions()`, `resolve_mention()`, `deliver_hitl()`, storage read/write/delete methods, `deploy()` |
| Helpers | `read_bundled(path)` — load files from the package directory |

## TypeScript

Package: [`sdk/typescript/`](https://github.com/plmbr/nui/tree/main/sdk/typescript/)

```typescript
import { NuiExtension } from "@nui/extension-sdk";

class CorpExtension extends NuiExtension {
  getHarnesses() {
    return [{ id: "echo", displayName: "Echo" }];
  }

  async *runHarness(id: string, message: string) {
    if (id === "echo") yield message;
  }
}

new CorpExtension().serve();
```

CLI entry: `npx nui-ext` (see `sdk/typescript/bin/nui-ext.js`)

## Go

Package: [`sdk/go/nuiextension/`](https://github.com/plmbr/nui/tree/main/sdk/go/nuiextension/)

```go
package main

import "github.com/plmbr/nui/sdk/go/nuiextension"

type ext struct{ nuiextension.Base }

func (e *ext) GetHarnesses() []map[string]any {
    return []map[string]any{
        {"id": "echo", "displayName": "Echo"},
    }
}

func (e *ext) RunHarness(id, message string, ctx map[string]any) nuiextension.RunResult {
    return nuiextension.TextStream(func(yield func(string) bool) {
        yield(message)
    })
}

func main() {
    nuiextension.ServeStdio(&ext{})
}
```

## Wire protocol

### Initialize

Request: `extension.initialize`  
Response: `ContributionManifest` with discovered lists

Schemas: [`sdk/protocol/`](https://github.com/plmbr/nui/tree/main/sdk/protocol/)

- [`extension-v1.schema.json`](https://github.com/plmbr/nui/blob/main/sdk/protocol/extension-v1.schema.json)
- [`initialization.schema.json`](https://github.com/plmbr/nui/blob/main/sdk/protocol/initialization.schema.json)

### Runtime RPC namespaces

| Namespace | Methods |
|-----------|---------|
| `harness.*` | `info`, `run`, `cancel`, `shutdown` |
| `mention.*` | `info`, `list`, `resolve`, `shutdown` |
| `hitl.*` | `info`, `deliver`, `shutdown` |
| `storage.*` | session/memory read/write/delete, `shutdown` |
| `extension.deploy` | Deploy user agent |
| `extension.shutdown` | Cleanup |

### Catalog RPC (dynamic lists)

When using `extension.listHarnesses` etc. from a catalog provider process — see [Dynamic catalog]({{ '/docs/extensions/catalog/' | relative_url }}).

## Environment variables

| Variable | Description |
|----------|-------------|
| `NUI_EXTENSION_DIR` | Package install directory |
| `NUI_EXTENSION_NAME` | Extension id |
| `NUI_API_URL` | nui REST base URL |

## Example extension

[`dev/extension-examples/programmatic-echo/`](https://github.com/plmbr/nui/tree/main/dev/extension-examples/programmatic-echo/)

```bash
nui extension add dev/extension-examples/programmatic-echo
```

## When to use programmatic vs declarative

| Use declarative when… | Use programmatic when… |
|-----------------------|----------------------|
| Assets are static yaml/json/md files | Lists depend on runtime config or APIs |
| Authors prefer no build step | You ship a pip/npm/go package |
| Simple harness hosts in Python | You need typed SDKs and unit tests in-process |

Both can coexist — many teams use declarative packs for internal catalogs and programmatic packages for published extensions.
