# nui Extension SDK

Programmatic extensions run as **external processes** and implement a `NuiExtension` base class by overriding methods. nui generates a minimal `extension.yaml` at install time (`runtime` + `install` only) and discovers capabilities via `extension.initialize` IPC.

Declarative packs (`extension.yaml` + static files + bash scripts) remain supported via `nui extension add ./pack`.

## Extension shapes

| Shape | Discovery | yaml |
|-------|-----------|------|
| **Programmatic package** | `extension.initialize` → `getHarnesses()`, … | `runtime` + `install` only |
| **Declarative pack** | Host reads yaml/files | full `contributions` |

## `runtime` vs `install`

- **`runtime`** — how nui spawns the extension subprocess (`transport`, `command`). Used at registry load and for harness/mention/HITL RPC.
- **`install`** — package provenance for `nui extension upgrade/remove` only. Not sent to the extension process.

## SDK API (override-based)

Subclass `NuiExtension` and override:

**Lifecycle:** `initialize()`, `shutdown()`

**Discovery:** `getHarnesses()`, `getAgents()`, `getMentionProviders()`, `getRules()`, `getHITLChannels()`, `getDeployers()`, …

**Runtime:** `runHarness()`, `listMentions()`, `resolveMention()`, `deliverHITL()`, `deploy()`

**Helpers:** `readBundled(path)` — load embedded static files inside `initialize()` (opaque to nui)

### Python

```python
from nui_extension import NuiExtension

class CorpExtension(NuiExtension):
    def get_harnesses(self):
        return [{"id": "echo", "displayName": "Echo"}]

    def run_harness(self, harness_id, message, ctx=None):
        if harness_id == "echo":
            yield message

if __name__ == "__main__":
    CorpExtension().serve()
```

Package: [`sdk/python/`](../sdk/python/) · auto-copied module: [`harness-sdk/nui_extension.py`](../harness-sdk/nui_extension.py)

### TypeScript

```typescript
import { NuiExtension } from "@nui/extension-sdk";

class CorpExtension extends NuiExtension {
  getHarnesses() { return [{ id: "echo", displayName: "Echo" }]; }
  async *runHarness(id, message) { if (id === "echo") yield message; }
}

new CorpExtension().serve();
```

Package: [`sdk/typescript/`](../sdk/typescript/)

### Go

```go
type ext struct{ nuiextension.Base }

func (e *ext) GetHarnesses() []map[string]any {
    return []map[string]any{{"id": "echo", "displayName": "Echo"}}
}

func main() { nuiextension.ServeStdio(&ext{}) }
```

Package: [`sdk/go/nuiextension/`](../sdk/go/nuiextension/)

## Install

```sh
nui extension add npm:@corp/nui-ext@1.0.0
nui extension add pip:corp-nui-ext==1.0.0
nui extension add ./my-python-ext          # detects pyproject.toml / package.json
nui extension add ./corp-pack              # declarative pack (unchanged)
```

## Scaffold

```sh
nui extension create my-ext --lang python
nui extension create my-ext --lang npm
nui extension create my-ext --lang go
```

## Protocol

- Schemas: [`sdk/protocol/`](../sdk/protocol/)
- Initialize: `extension.initialize` → `ContributionManifest`
- Runtime RPC: `harness.*`, `mention.*`, `hitl.deliver`, `extension.deploy`
- Shutdown: `extension.shutdown`

## Example

[`dev/extension-examples/programmatic-echo/`](../dev/extension-examples/programmatic-echo/)

```sh
nui extension add dev/extension-examples/programmatic-echo
```

See also: [`dev/extension-api.md`](extension-api.md) for declarative contributions and wire protocol details.
