#!/usr/bin/env python3
"""Example mention provider for corp-pack."""

import os
import sys


def _find_sdk_dir() -> str | None:
    candidates = []
    if sdk := os.environ.get("NUI_MENTION_SDK_DIR"):
        candidates.append(sdk)
    ext_dir = os.path.dirname(os.path.abspath(__file__))
    candidates.extend([
        ext_dir,
        os.path.join(ext_dir, "..", "..", "..", "harness-sdk"),
        os.path.expanduser("~/.nui/harness-sdk"),
    ])
    seen: set[str] = set()
    for candidate in candidates:
        if not candidate:
            continue
        norm = os.path.abspath(candidate)
        if norm in seen:
            continue
        seen.add(norm)
        if os.path.isfile(os.path.join(norm, "nui_mention.py")):
            return norm
    return None


_sdk_dir = _find_sdk_dir()
if _sdk_dir is None:
    sys.stderr.write("nui_mention.py not found (set NUI_MENTION_SDK_DIR)\n")
    sys.exit(1)
sys.path.insert(0, _sdk_dir)

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

        if parent == f"{root}:runbooks" or parent.endswith(":runbooks"):
            items = [
                {
                    "label": "Deploy checklist",
                    "value": f"{root}:runbooks/deploy",
                    "hasChildren": False,
                },
                {
                    "label": "Rollback plan",
                    "value": f"{root}:runbooks/rollback",
                    "hasChildren": False,
                },
            ]
            if query:
                q = query.lower()
                items = [i for i in items if q in i["label"].lower()]
            return {
                "items": items[:limit],
                "breadcrumb": [
                    {"label": "Root", "parent": ""},
                    {"label": "Runbooks", "parent": f"{root}:runbooks"},
                ],
            }

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
