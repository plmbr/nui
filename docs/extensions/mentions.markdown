---
layout: docs
title: Mention providers
subtitle: "At-mention autocomplete sources for the chat input."
permalink: /docs/extensions/mentions/
---

Mention providers power the `@` menu in chat. Extensions contribute hierarchical autocomplete sources; selecting a leaf inserts `@value` into the message, and nui resolves mentions server-side before sending to the harness.

## Opt-in per agent

Mention providers are **not** global. Agents declare which providers to use:

```yaml
aiAssets:
  mentionProviders:
    - ref: ext:corp-pack/corp-refs
```

When a session uses an agent with mention provider refs, only those providers appear in the `@` menu (plus the built-in **Files & folders** provider).

## Manifest

```yaml
contributions:
  mentionProviders:
    source:
      file: mention-providers.yaml
    runtime:
      transport: stdio
      command: ["python3", "mention_host.py"]
```

`mention-providers.yaml`:

```yaml
mentionProviders:
  - id: corp-refs
    displayName: Corp References
```

Provider id in ADL refs: `ext:<extension>/corp-refs` (the `id` field from the list file).

## Built-in provider

**Files & folders** (`builtin:files`) — lists files under the session working directory. Resolves `file:<relative-path>` to the full absolute path. Always available unless an agent restricts providers.

## Wire protocol (`mention.*`)

| Method | Params | Result |
|--------|--------|--------|
| `mention.info` | `{}` | `{id, name, version, capabilities}` |
| `mention.list` | `{providerId, parent?, query?, limit?, workingDir?, sessionId?}` | `{items, breadcrumb?}` |
| `mention.resolve` | `{providerId, value, workingDir?, sessionId?}` | `{text}` |
| `mention.shutdown` | `{}` | `{ok: true}` |

### Item shape

```json
{
  "label": "Deploy checklist",
  "value": "ext:corp-pack:corp-refs:runbooks/deploy",
  "hasChildren": false,
  "icon": "optional-icon-id"
}
```

- `hasChildren: true` — drilling into `value` as `parent` on the next `mention.list`
- Leaf items — inserted as `@value` in chat

### Breadcrumb (optional)

```json
{
  "breadcrumb": [
    {"label": "Root", "parent": ""},
    {"label": "Runbooks", "parent": "ext:corp-pack:corp-refs:runbooks"}
  ]
}
```

## Python SDK

Framework: [`harness-sdk/nui_mention.py`](https://github.com/plmbr/nui/blob/main/harness-sdk/nui_mention.py) (auto-installed to `~/.nui/harness-sdk/`).

### Example host

```python
#!/usr/bin/env python3
import os
import sys

sys.path.insert(0, os.path.expanduser("~/.nui/harness-sdk"))
from nui_mention import NuiMentionProvider


class CorpMentionHost(NuiMentionProvider):
    name = "corp-pack-mentions"
    version = "1.0.0"

    def list_items(self, parent="", query="", limit=20, **kwargs):
        ext = os.environ.get("NUI_EXTENSION_NAME", "corp-pack")
        provider = os.environ.get("NUI_MENTION_PROVIDER_ID", "corp-refs")
        root = f"ext:{ext}:{provider}"

        if not parent or parent == root:
            return {
                "items": [{
                    "label": "Runbooks",
                    "value": f"{root}:runbooks",
                    "hasChildren": True,
                }],
                "breadcrumb": [
                    {"label": "Root", "parent": ""},
                    {"label": "Corp References", "parent": root},
                ],
            }

        if parent.endswith(":runbooks"):
            items = [
                {"label": "Deploy checklist", "value": f"{root}:runbooks/deploy", "hasChildren": False},
                {"label": "Rollback plan", "value": f"{root}:runbooks/rollback", "hasChildren": False},
            ]
            if query:
                q = query.lower()
                items = [i for i in items if q in i["label"].lower()]
            return {"items": items[:limit], "breadcrumb": []}

        return {"items": [], "breadcrumb": []}

    def resolve_value(self, value="", **kwargs):
        if value.endswith("/deploy"):
            text = "Follow the deploy checklist before merging to main."
        elif value.endswith("/rollback"):
            text = "Follow the rollback plan if the deploy fails."
        else:
            text = value
        return {"text": text}


if __name__ == "__main__":
    CorpMentionHost().serve()
```

Override SDK path: `NUI_MENTION_SDK_DIR`

## REST API

```
GET /api/sessions/:id/mentions?parent=&query=
```

Query params mirror `mention.list`. Used by the chat UI; extension hosts are called server-side.

## Programmatic extensions

Override on `NuiExtension`:

```python
def get_mention_providers(self):
    return [{"id": "corp-refs", "displayName": "Corp References"}]

def list_mentions(self, provider_id, parent="", query="", limit=20, ctx=None):
    return {"items": [...], "breadcrumb": []}

def resolve_mention(self, provider_id, value, ctx=None):
    return {"text": "resolved content"}
```

See [Programmatic SDK]({{ '/docs/extensions/programmatic/' | relative_url }}).

## Full example

[`dev/extension-examples/corp-pack/`](https://github.com/plmbr/nui/tree/main/dev/extension-examples/corp-pack/) — `mention_host.py` + `mention-providers.yaml`
