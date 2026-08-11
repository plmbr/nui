---
layout: post
title: Introducing nui
subtitle: A self-hosted web UI for AI agents — built for cost control and open-weight models.
description: nui is now available. One self-hosted interface for coding agents and API models, with first-class support for OpenRouter, Ollama, and custom open-weight workflows.
date: 2026-08-10
---

Frontier coding agents are extraordinary. They are also expensive. Token bills climb, hosted UIs lock you into one vendor, and the moment you want to try an open-weight model for the same workflow, you are back in a different terminal with a different UX.

**nui** is a self-hosted web UI for interactive AI agent sessions. Install a single binary, run `nui server`, and open the browser. Use Claude Code, pi, codex, or opencode when the task needs a premium CLI. Switch to OpenRouter or local Ollama models when it does not. Same sessions, same chat, same extensions — pick the model that fits the job, not the invoice.

nui is open source (MIT) and available today.

<figure>
  <img src="{{ '/assets/images/hero/nui.png' | relative_url }}"
       alt="nui web UI showing a chat session with an AI agent."
       width="2227" height="1369" loading="eager">
</figure>

## Why now

The industry is shifting. Teams are routing more work to open-weight models, local inference, and cheaper multi-provider APIs. Cost is a first-class constraint again — not just latency or quality.

Most agent tooling still assumes the opposite: one cloud, one model family, one bill. nui starts from the opposite bet:

- **Self-hosted** — your machine, your server, your data path.
- **Multi-agent** — CLI harnesses and API providers in one UI.
- **Open-weight ready** — Ollama is a peer, not a side quest. OpenRouter makes cheaper and open models one click away.
- **Composable** — custom agents, extensions, and MCP so you can build the stack you actually want.

Use frontier models when they earn their keep. Use open weights when they do not. Keep the interface.

## Sessions in the browser

Type a task on the home screen and the built-in `nui` master agent routes it to a specialist — or open a session with any built-in or installed agent. Chat, attach files, use `@` mentions for context, and switch between past sessions from the sidebar. Preferences persist across reloads.

No juggling terminals. No “which window was that agent in?” Same session model whether the backend is Claude Code or a local Ollama model.

## Built-in agents

**CLI agents**:

- Claude Code
- pi
- codex
- opencode

**API agents** (in-process; no separate CLI):

- Anthropic, OpenAI, Gemini
- **OpenRouter** — route across providers and price points from one agent
- **Ollama** — local and self-hosted open-weight models

That split is the cost story: keep premium CLIs for hard work, and keep OpenRouter / Ollama for everything else — drafts, triage, summarization, internal tools — without leaving nui.

## Custom agents with ADL

Define your own agents in YAML with the Agent Definition Language (ADL). Pick a harness, set a system prompt, add skills and MCP servers, and optionally run in a sandbox, Docker, a dev container, or on a remote host.

Multi-step workflows, per-step harness overrides, and eval test cases are part of the schema. Install with `nui agent add`, or build them in the UI. Portable agents can target different CLI harnesses so the same definition can ride a frontier CLI today and a cheaper path tomorrow.

## Extensions

Extensions add harnesses, MCP servers, skills, and agents. Install from a local directory, zip file, or git URL:

```bash
nui extension add ./my-extension
nui extension add https://github.com/example/my-extension.git
```

Manage them from **Settings → Extensions**, or disable individually without uninstalling. This is how teams plug in internal tools without forking nui.

## MCP in both directions

nui exposes itself as an MCP server — wire it into Cursor, Claude Desktop, or any MCP host and drive sessions programmatically (`list_agents`, `create_session`, `run_agent`, and more).

It also injects built-in MCP servers into harness subprocesses:

- **Human-in-the-loop** — approvals and `ask_user` prompts
- **Visualization** — inline charts in chat
- **Agent memory** — persistent memory updates
- **Orchestrator** — home-launcher routing for the `nui` master agent

Remote HTTP MCP servers can authenticate with OAuth from Settings.

## Sandboxing and isolation

Run agents unsandboxed on the host, or isolate them with bubblewrap (Linux), Docker, or dev containers. Pair isolation with cheaper or open-weight agents when you want more automation without expanding blast radius.

## Headless, schedules, and evals

Not everything needs a browser.

```bash
# Headless run (server must be running)
nui run -a claude-code -m "Review README" -w . --wait

# Auto-start server if needed
nui run -m "Summarize changes" -w . --spawn --wait
```

Use [`nui schedule`](/cli/#schedules) for recurring jobs and [`nui agent eval`](/cli/#agent-evaluation) to validate ADL agents against test cases in CI. Script the expensive path when you must; automate the cheap path when you can.

## Install

**Linux and macOS:**

```bash
curl -fsSL https://nui.plmbr.dev/install.sh | sh
nui server --open
```

**Windows:**

```powershell
irm https://nui.plmbr.dev/install.ps1 | iex
```

Or download a release binary from [GitHub Releases](https://github.com/plmbr/nui/releases). Full platform notes and agent prerequisites are in the [install guide](/install/).

## What's next

nui is intentionally a thin, self-hosted surface over the agents you already use — and the open-weight ones you are about to use more. Star or watch [github.com/plmbr/nui](https://github.com/plmbr/nui), file issues, and tell us what cost-sensitive or open-weight workflows you want first-class support for.
