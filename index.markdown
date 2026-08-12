---
layout: home
title: nui
description: A self-hosted web UI for interactive AI agent sessions. Run agents locally, in Docker, or on a remote server — all from one interface.
---

<section class="hero">
  <div class="container two-col">
    <div>
      <p class="hero__eyebrow">Self-hosted agent UI</p>
      <h1 class="hero__headline">One interface for every agent.</h1>
      <p class="hero__sub">
        nui is a web UI for interactive AI agent sessions. Ask the home launcher to route
        work, or run Claude Code, pi, codex, opencode, and API agents on your machine,
        in Docker, or on a remote server — with sessions, extensions, MCP, and custom ADL
        agents in one place.
      </p>
      <div class="hero__cta">
        <a class="btn btn--primary" href="#install">Install</a>
        <a class="btn btn--ghost" href="https://github.com/plmbr/nui" rel="noopener">View on GitHub</a>
      </div>
    </div>
    <div>
      <figure class="hero__media">
        <img src="{{ '/assets/images/hero/nui.png' | relative_url }}"
             alt="nui web UI showing a chat session with an AI agent."
             width="2227" height="1369" loading="eager" fetchpriority="high">
      </figure>
    </div>
  </div>
</section>

<section class="providers">
  <div class="container">
    <p class="providers__label">Built-in agents</p>
    <div class="providers__row" aria-label="Supported agent harnesses">
      <span>nui</span>
      <span>Claude&nbsp;Code</span>
      <span>pi</span>
      <span>codex</span>
      <span>opencode</span>
      <span>Anthropic&nbsp;API</span>
      <span>OpenAI</span>
      <span>Gemini</span>
      <span>OpenRouter</span>
      <span>Ollama</span>
    </div>
  </div>
</section>

<section class="stripe" id="sessions">
  <div class="container stripe__grid">
    <div>
      <p class="stripe__eyebrow">Sessions</p>
      <h2 class="stripe__title">Chat, switch, and resume — without leaving the browser.</h2>
      <p class="stripe__body">
        Type a task on the home screen and the <code>nui</code> master agent routes it to
        the right specialist — or open a session with any built-in or installed agent.
        Send prompts, attach files, and use <code>@</code> mentions for context. Switch
        between past sessions from the sidebar; preferences persist across reloads.
      </p>
      <a class="stripe__link" href="{{ '/features/' | relative_url }}">Explore features →</a>
    </div>
    <div class="stripe__media" style="padding: var(--space-5);">
{% highlight bash %}
# Start the server and open the UI
nui server --open

# Launch with a specific agent and prompt
nui server -a claude-code -m "Review the README" -w . --open
{% endhighlight %}
    </div>
  </div>
</section>

<section class="stripe stripe--reverse">
  <div class="container stripe__grid">
    <div>
      <p class="stripe__eyebrow">Custom agents</p>
      <h2 class="stripe__title">ADL agents, Docker harnesses, and extensions.</h2>
      <p class="stripe__body">
        Define custom agents in YAML with the Agent Definition Language — pick a harness,
        set a system prompt, and run locally, in sandboxes, Docker containers, or on remote
        servers. Install extensions that contribute harnesses, MCP servers, skills, and agents.
      </p>
      <a class="stripe__link" href="{{ '/features/adl/' | relative_url }}">ADL and custom agents →</a>
    </div>
    <div class="stripe__media" style="padding: var(--space-5);">
{% highlight yaml %}
name: Review Agent
harness:
  type: claude-code
  sandbox: none
systemPrompt: |
  You are a code reviewer. List issues by severity and suggest fixes.
{% endhighlight %}
    </div>
  </div>
</section>

<section class="stripe">
  <div class="container stripe__grid">
    <div>
      <p class="stripe__eyebrow">MCP integration</p>
      <h2 class="stripe__title">Expose nui to Cursor, Claude Desktop, and other MCP hosts.</h2>
      <p class="stripe__body">
        Run <code>nui mcp</code> to expose agent tools to external MCP clients. nui also
        injects built-in MCP servers into harnesses for human-in-the-loop prompts,
        inline visualizations, agent memory, and home-launcher routing.
      </p>
      <a class="stripe__link" href="{{ '/features/mcp/' | relative_url }}">MCP integration →</a>
    </div>
    <div class="stripe__media" style="padding: var(--space-5);">
{% highlight json %}
{
  "mcpServers": {
    "nui": {
      "command": "nui",
      "args": ["mcp"],
      "env": { "NUI_URL": "http://127.0.0.1:8080" }
    }
  }
}
{% endhighlight %}
    </div>
  </div>
</section>

<section class="stripe stripe--reverse">
  <div class="container stripe__grid">
    <div>
      <p class="stripe__eyebrow">Headless runs</p>
      <h2 class="stripe__title">Script agent runs from the terminal or CI.</h2>
      <p class="stripe__body">
        Use <code>nui run</code> for headless agent execution against a running server.
        Schedule recurring runs with <code>nui schedule</code>. Evaluate ADL agents with
        <code>nui agent eval</code> — no browser required.
      </p>
      <a class="stripe__link" href="{{ '/cli/' | relative_url }}">CLI reference →</a>
    </div>
    <div class="stripe__media" style="padding: var(--space-5);">
{% highlight bash %}
# Headless run (server must be running)
nui run -a claude-code -m "Review README" -w . --wait

# Auto-start server if not running
nui run -m "Summarize changes" -w . --spawn --wait
{% endhighlight %}
    </div>
  </div>
</section>

<section class="section" id="install">
  <div class="container-narrow">
    <p class="eyebrow">First run</p>
    <h2>Install and try it in two minutes.</h2>
    <p class="lede">
      Install the CLI with the one-liner below, or download the
      <a href="{{ '/install/#install-the-desktop-app' | relative_url }}">desktop app</a>
      from GitHub Releases (it bundles the CLI and installs it on first launch).
    </p>

    <p>
      <span class="install-pill">
        <span class="install-pill__prompt" aria-hidden="true">$</span>
        <span class="install-pill__cmd" id="install-cmd">curl -fsSL https://nui.plmbr.dev/install.sh | sh</span>
        <button class="install-pill__copy" type="button" data-copy="#install-cmd" aria-label="Copy install command">
          {% include copy-icon.html %}
        </button>
      </span>
    </p>

    <ol class="steps">
      <li><strong>Install.</strong> Run the curl one-liner above, or grab the desktop package / CLI archive from GitHub Releases.</li>
      <li><strong>Start.</strong> Run <code>nui server</code> (listens on <code>:8080</code>), or open the desktop app.</li>
      <li><strong>Open the UI.</strong> Pick an agent and a working directory. nui creates a session on first launch.</li>
      <li><strong>Chat.</strong> Send a prompt, attach files, or use <code>@</code> mentions for context.</li>
    </ol>

    <p style="margin-top: var(--space-8);">
      <a class="btn btn--ghost" href="{{ '/install/' | relative_url }}">Full install guide →</a>
    </p>
  </div>
</section>
