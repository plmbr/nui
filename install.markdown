---
layout: page
title: Install
subtitle: Download a release binary or use the install script — then start the server and open the UI.
permalink: /install/
---

## Prerequisites

- **Linux, macOS, or Windows.** Release binaries are available for all three platforms.
- **An agent CLI or API key** — install at least one of the built-in agents you want to use (see [Agents](/agents/)).

## Install with the script

**Linux and macOS** (installs to `~/.local/bin`):

```bash
curl -fsSL https://nui.plmbr.dev/install.sh | sh
```

**Windows** (installs to `%LOCALAPPDATA%\nui` and adds it to your user `PATH`):

```powershell
irm https://nui.plmbr.dev/install.ps1 | iex
```

Install a specific version:

```bash
NUI_VERSION=v0.4.0-alpha curl -fsSL https://nui.plmbr.dev/install.sh | sh
```

```powershell
$env:NUI_VERSION = "v0.4.0-alpha"; irm https://nui.plmbr.dev/install.ps1 | iex
```

## Manual install

Download the archive for your platform from [GitHub Releases](https://github.com/plmbr/nui/releases), extract the `nui` binary (or `nui.exe` on Windows), and place it on your `PATH`.

<div class="callout">
  <p class="callout__title">macOS note</p>
  <p>Release binaries are currently unsigned. The install script strips the download quarantine attribute automatically. If you install manually, run <code>xattr -d com.apple.quarantine /path/to/nui</code>. If macOS still blocks the binary, allow it under <strong>System Settings → Privacy &amp; Security</strong>.</p>
</div>

## First run

1. **Start the server.** <code>nui server</code> listens on port 8080 by default.
2. **Open the UI.** Navigate to <a href="http://localhost:8080">http://localhost:8080</a> or use <code>nui server --open</code>.
3. **Pick an agent.** Choose a built-in or installed agent and a working directory.
4. **Send a prompt.** nui creates a session automatically on first launch.

Launch with a specific agent and prompt:

```bash
nui server --agent-type claude-code --prompt "Review the README" --open
```

## Agent prerequisites

Install the agent CLI you want to use and make sure it is on your `PATH`:

| Agent (ADL id) | CLI command |
|---|---|
| `claude-code` | `claude` |
| `pi` | `pi` |
| `codex` | `codex` |
| `opencode` | `opencode` |

**API agents** (no CLI required) use provider API keys instead:

| Agent | API key environment variable |
|---|---|
| Anthropic | `ANTHROPIC_API_KEY` (or `ANTHROPIC_AUTH_TOKEN`) |
| OpenAI | `OPENAI_API_KEY` |
| Gemini | `GEMINI_API_KEY` or `GOOGLE_API_KEY` |
| OpenRouter | `OPENROUTER_API_KEY` |
| Ollama | none (local; optional `OLLAMA_HOST`) |

**Optional:**

- **Docker** — for sandboxed built-in agents and custom Docker-based agents
- **Dev Container CLI** — for devcontainer harness agents (`npm install -g @devcontainers/cli`)

## Troubleshooting

If something doesn't work, check:

- `nui server` output in the terminal where you started the server.
- That your agent CLI is on `PATH` and authenticated.
- [GitHub Issues](https://github.com/plmbr/nui/issues) for known problems.

For deeper issues, see the [developer guide](https://github.com/plmbr/nui/blob/main/DEVELOPERS.md) in the repo.
