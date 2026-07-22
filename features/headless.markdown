---
layout: page
title: Headless & scheduled runs
subtitle: Script agent runs from the terminal, schedule recurring jobs, and evaluate ADL agents.
permalink: /features/headless/
---

nui supports running agents without opening the browser — useful for scripts, CI pipelines, and scheduled jobs.

## Headless runs

```bash
# Server must be running
nui run -m "Review README" -w .
nui run -a claude-code -m "Review README" -w . --wait

# Auto-start server if not running
nui run -m "Summarize changes" -w . --spawn --wait
```

Set `NUI_URL` or pass `--url` if the server is not on `http://127.0.0.1:8080`.

## Scheduled runs

```bash
nui schedule list
nui schedule add --agent-type "Claude Code" --prompt "Daily standup summary" --cron "0 9 * * 1-5"
nui schedule enable <id>
nui schedule run-now <id>
nui schedule delete <id>
```

Schedules require a running `nui server` instance.

## Agent evaluation

```bash
nui agent eval run -a my-agent
```

Run ADL eval cases against a running server to validate agent behavior.

## CLI launch flags

| Flag | Short | Description |
|---|---|---|
| `--open` | | Open the web UI in your browser with a new session |
| `--agent-type` | `-a` | Agent to use (e.g. `Claude Code`, `pi`, `codex`) |
| `--prompt` | `-m` | Initial prompt sent automatically |
| `--hide-input` | | Hide the chat input (use with `--prompt`) |
| `--working-dir` | `-w` | Working directory for the session |
| `--theme` | | UI theme: `light` or `dark` |
| `--default-agent` | | Default agent for new sessions |
