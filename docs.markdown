---
layout: page
title: Documentation
subtitle: The deep docs live in the repo. This page is a router.
permalink: /docs/
---

nui's reference documentation is maintained alongside the code in the [main repository](https://github.com/plmbr/nui). Keeping one source of truth means the docs and the code can't drift.

## Reference

- [README](https://github.com/plmbr/nui/blob/main/README.md) — install, quick start, CLI reference.
- [Developer guide](https://github.com/plmbr/nui/blob/main/DEVELOPERS.md) — build from source, API reference, contributing, releasing.
- [Product & technical spec](https://github.com/plmbr/nui/blob/main/dev/dev.md) — architecture and roadmap.
- [Extension API](https://github.com/plmbr/nui/blob/main/dev/extension-api.md) — extension manifest, HITL, deployers.
- [Harness protocols](https://github.com/plmbr/nui/blob/main/dev/harness-design.md) — custom harness HTTP/SSE and JSON-RPC.
- [ADL](https://github.com/plmbr/nui/blob/main/dev/adl/design.md) — Agent Definition Language schema and examples.
- [ADL examples](https://github.com/plmbr/nui/tree/main/dev/adl/examples/) — sample agent definitions.
- [Harness examples](https://github.com/plmbr/nui/tree/main/dev/harness-examples/) — Python and TypeScript reference harnesses.

## CLI quick reference

```
nui server              # start web server on :8080
nui server --port 3000  # custom port
nui server --open       # open browser with a new session
nui run -a claude-code -m "Review README" --wait  # headless run
nui agent list          # list agent types (requires nui server)
nui agent add ./my-agent.yaml  # install custom agent
nui extension add       # install extension from git URL, directory, or zip
nui skills add|list|remove  # manage skills catalog
nui schedule list|add|enable|disable|delete|run-now  # recurring runs
```
