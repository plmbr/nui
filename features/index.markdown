---
layout: page
title: Features
subtitle: Define, run, access, and extend agents — six capabilities nui ships today.
permalink: /features/
---

<div class="feature-grid">
  <a class="card" href="{{ '/features/agents/' | relative_url }}">
    <h3 class="card__title">Built-in agents</h3>
    <p class="card__body">The <code>nui</code> master agent (home launcher), Claude Code, pi, codex, opencode, Antigravity, and in-process API agents — Anthropic, OpenAI, Gemini, OpenRouter, and Ollama.</p>
  </a>
  <a class="card" href="{{ '/features/adl/' | relative_url }}">
    <h3 class="card__title">ADL agents</h3>
    <p class="card__body">Define custom agents in YAML or the form editor — with eval test cases, harness, system prompt, workflows, sub-agents, councils, and optional sandbox.</p>
  </a>
  <a class="card" href="{{ '/features/extensions/' | relative_url }}">
    <h3 class="card__title">Extensions</h3>
    <p class="card__body">Platform-grade extensibility: harnesses, MCP servers, skills, HITL channels, storage, and deployers from a local dir, zip, or git URL — same path built-ins use.</p>
  </a>
  <a class="card" href="{{ '/features/mcp/' | relative_url }}">
    <h3 class="card__title">MCP integration</h3>
    <p class="card__body">Expose nui to external MCP hosts. Built-in HITL, visualization, agent-memory, and orchestrator servers for harnesses — plus OAuth for remote MCP.</p>
  </a>
  <a class="card" href="{{ '/features/sandbox/' | relative_url }}">
    <h3 class="card__title">Sandboxing</h3>
    <p class="card__body">Run agents in bubblewrap sandboxes, Docker containers, or dev containers — or unsandboxed on the host. Trust level is an ADL field.</p>
  </a>
  <a class="card" href="{{ '/features/headless/' | relative_url }}">
    <h3 class="card__title">Headless &amp; scheduled runs</h3>
    <p class="card__body">Script agent runs from the terminal, schedule recurring jobs, and evaluate ADL agents without a browser.</p>
  </a>
</div>
