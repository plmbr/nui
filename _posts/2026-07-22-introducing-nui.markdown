---
layout: post
title: Introducing nui
subtitle: A self-hosted web UI for interactive AI agent sessions.
description: nui brings Claude Code, pi, codex, opencode, and custom ADL agents into one web interface — with extensions, MCP, and headless runs.
date: 2026-07-22
---

If you've been juggling multiple agent CLIs, Docker containers, and remote servers, nui is the single interface for all of it.

## What is nui?

nui is a self-hosted web UI for interactive AI agent sessions. Install a single binary, run `nui server`, and open the browser. Pick an agent, choose a working directory, and start chatting.

Built-in agents cover the major coding-agent CLIs — Claude Code, pi, codex, and opencode — plus in-process API agents for Anthropic, OpenAI, Gemini, OpenRouter, and Ollama.

## Custom agents and extensions

Beyond the built-ins, nui supports custom agents defined in YAML with the Agent Definition Language (ADL). Multi-step workflows, per-step harness overrides, Docker and remote harnesses, and topological scheduling are all part of the schema.

Extensions add harnesses, MCP servers, skills, and agents. Install from a local directory, zip file, or git URL with `nui extension add`.

## MCP in both directions

nui exposes itself as an MCP server — wire it into Cursor or Claude Desktop and drive agent sessions programmatically. It also injects built-in MCP servers (human-in-the-loop, visualization, agent memory) into harness subprocesses.

## Headless and scheduled

Not everything needs a browser. `nui run` executes agents from the terminal or CI. `nui schedule` sets up recurring runs. `nui agent eval` validates ADL agents against test cases.

## Get started

```bash
curl -fsSL https://nui.plmbr.dev/install.sh | sh
nui server --open
```

See the [install guide](/install/) for platform-specific instructions and agent prerequisites.
