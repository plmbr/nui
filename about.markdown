---
layout: page
title: About
subtitle: A self-hosted agent runtime with a platform-grade extension model.
permalink: /about/
---

nui is a self-hosted **agent runtime**: define agents declaratively in [ADL]({{ '/features/adl/' | relative_url }}), run them on any harness or model — Claude Code, Codex, OpenCode, pi, Antigravity, hosted APIs, or free local models via Ollama — locally, in Docker, or on a remote server, and reach them from a web UI, the CLI, or a REST/AG-UI/MCP API. Extensions add new harnesses, tools, and storage backends without touching nui's own code.

It is single-operator infrastructure you host yourself (everything under `~/.nui/`), not a multi-tenant SaaS platform. The bundled web UI is one client of the same backend as `nui run`, `nui mcp`, and (planned) IDE extensions for VS Code, Cursor, and JupyterLab.

Built-in agents cover the `nui` master/launcher agent, Claude Code, pi, codex, opencode, Antigravity, and in-process API providers (Anthropic, OpenAI, Gemini, OpenRouter, Ollama). Custom agents are defined in YAML and can run in sandboxes, Docker containers, dev containers, or on remote servers.

Extensions add harnesses, MCP servers, skills, HITL channels, storage backends, and deployers. Built-in harnesses use the same contribution path as third-party ones — nui is an open framework with agents pre-installed, not a fixed list of five.

nui is open source and MIT-licensed. It is maintained by [Mehmet Bektaş](https://github.com/mbektas). The codebase lives at [github.com/plmbr/nui](https://github.com/plmbr/nui).
