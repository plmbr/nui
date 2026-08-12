---
layout: page
title: Install
subtitle: Install the CLI, the desktop app, or both — then start chatting with your agents.
permalink: /install/
---

## Prerequisites

- **Linux, macOS, or Windows.** Release binaries are available for all three platforms.
- **An agent CLI or API key** — install at least one of the built-in agents you want to use (see [Agents](/agents/)).

## Install the CLI

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
NUI_VERSION=v0.6.2-beta curl -fsSL https://nui.plmbr.dev/install.sh | sh
```

```powershell
$env:NUI_VERSION = "v0.6.2-beta"; irm https://nui.plmbr.dev/install.ps1 | iex
```

### Manual CLI install

Download the archive for your platform from [GitHub Releases](https://github.com/plmbr/nui/releases), extract the `nui` binary (or `nui.exe` on Windows), and place it on your `PATH`.

Look for assets named `nui_<tag>_<os>_<arch>.tar.gz` (or `.zip` on Windows).

## Install the desktop app

Prefer a native window? Download the desktop package for your platform from [GitHub Releases](https://github.com/plmbr/nui/releases). Assets look like:

| Platform | Asset |
|---|---|
| macOS Apple Silicon | `nui-desktop_<tag>_darwin_arm64.zip` |
| macOS Intel | `nui-desktop_<tag>_darwin_amd64.zip` |
| Windows | `nui-desktop_<tag>_windows_amd64.zip` |
| Linux | `nui-desktop_<tag>_linux_amd64.tar.gz` / `_linux_arm64.tar.gz` |

### macOS

1. Download and unzip the darwin archive for your chip.
2. Clear Gatekeeper quarantine (builds are ad-hoc signed, not notarized):

```bash
xattr -cr ~/Downloads/nui.app
open ~/Downloads/nui.app
```

3. Optionally drag `nui.app` into **Applications**.

On first launch the app starts the local server and **installs the bundled CLI** to `~/.local/bin/nui` (and can update your shell profile so `nui` is on `PATH`). Open a new terminal after the first launch if you need the CLI elsewhere.

If macOS still blocks the app, right-click → **Open**, or allow it under **System Settings → Privacy & Security**.

### Windows

1. Download and unzip `nui-desktop_<tag>_windows_amd64.zip`.
2. Run `nui-desktop.exe` (WebView2 is required; Windows 11 usually has it).
3. On first launch the app installs the bundled CLI to `%LOCALAPPDATA%\nui` and adds that folder to your user `PATH` when needed.

### Linux

1. Download and extract the linux archive for your arch.
2. Run `./nui-desktop` (needs GTK3 + WebKitGTK 4.1).
3. On first launch the app installs the bundled CLI to `~/.local/bin/nui`.

You can still use the curl/irm CLI installers without the desktop app.

<div class="callout">
  <p class="callout__title">macOS note (CLI)</p>
  <p>The install script strips quarantine and ad-hoc codesigns the CLI so it runs on current macOS. If you install the CLI manually and see <code>zsh: killed</code>, run:</p>
  <pre><code>xattr -cr /path/to/nui
codesign -s - -f /path/to/nui</code></pre>
  <p>Then allow the binary under <strong>System Settings → Privacy &amp; Security</strong> if prompted.</p>
</div>

## First run

### CLI / browser

1. **Start the server.** <code>nui server</code> listens on port 8080 by default.
2. **Open the UI.** Navigate to <a href="http://localhost:8080">http://localhost:8080</a> or use <code>nui server --open</code>.
3. **Pick an agent.** Choose a built-in or installed agent and a working directory.
4. **Send a prompt.** nui creates a session automatically on first launch.

Launch with a specific agent and prompt:

```bash
nui server --agent-type claude-code --prompt "Review the README" --open
```

### Desktop

Open the app — it attaches to an existing `nui server` if one is already listening, otherwise it starts the server for you. Use the same agent picker and chat UI as the browser.

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

- `nui server` output in the terminal where you started the server (or the desktop app logs).
- That your agent CLI is on `PATH` and authenticated.
- On macOS desktop: you ran `xattr -cr` on `nui.app` after download.
- [GitHub Issues](https://github.com/plmbr/nui/issues) for known problems.

For deeper issues, see the [developer guide](https://github.com/plmbr/nui/blob/main/DEVELOPERS.md) in the repo.
