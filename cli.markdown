---
layout: page
title: CLI reference
subtitle: Every nui command for the terminal, CI, and scheduled jobs.
permalink: /cli/
---

Most commands talk to a running `nui server`. Set `NUI_URL` or pass `--url` when the server is not on `http://127.0.0.1:8080`. Several commands accept `--spawn` to start the server in the background if it is unreachable.

## Server

Start the web UI and REST API.

```bash
nui server                    # listen on :8080
nui server --port 3000        # custom port
nui server --open             # open browser with a new session
nui server -a claude-code -m "Review README" -w . --open
```

| Flag | Short | Description |
|---|---|---|
| `--port` | `-p` | Port to listen on (default `8080`) |
| `--open` | | Open the web UI in the system default browser |
| `--no-browser` | | Do not open a browser (daemon mode) |
| `--agent-type` | `-a` | Agent id for a session created on startup |
| `--prompt` | `-m` | Initial prompt for the new session |
| `--working-dir` | `-w` | Working directory for the new session |
| `--hide-input` | | Hide the chat input (use with `--prompt`) |
| `--theme` | | UI theme: `light` or `dark` |
| `--default-agent` | | Default agent for new sessions (saved to `~/.nui/settings.json`) |
| `--default-harness` | | Default harness for internal agents (saved to settings) |

If a server is already running on the target port, `nui server` attaches to it and can still create a session when launch flags are passed.

## Headless runs

Run an agent without opening the browser. `nui run` and `nui agent run` are equivalent.

```bash
nui run -m "Review README" -w .
nui run -a claude-code -m "Review README" -w . --wait
nui run -m "Summarize changes" -w . --spawn --wait
nui run --session-id <id> -m "Follow up" --wait
```

| Flag | Short | Description |
|---|---|---|
| `--agent-type` | `-a` | ADL agent id for a new session (default: settings `defaultAgentType`) |
| `--message` | `-m` | Prompt message |
| `--working-dir` | `-w` | Working directory for a new session |
| `--session-id` | | Existing session id (skips session create) |
| `--wait` | | Wait for the run to finish and stream text to stdout (default `true`) |
| `--no-wait` | | Return immediately after starting the run |
| `--spawn` | | Start `nui server` in the background if unreachable |
| `--url` | | Server base URL |

## Agents

Manage ADL agent definitions in `~/.nui/agents/`.

```bash
nui agent list
nui agent add ./my-agent.yaml
nui agent add https://github.com/example/repo/blob/main/agents/watchdog.yaml
nui agent remove my-agent
nui agent deployers
nui agent deploy ext:docker-deployer/docker my-agent
```

| Command | Description |
|---|---|
| `list` | List available agent types (requires server) |
| `add` | Install an ADL YAML from a local file or git URL |
| `remove` | Remove a user-installed agent |
| `deployers` | List extension agent deployers |
| `deploy` | Deploy an agent via an extension deployer |

## Agent evaluation

Run test cases defined in an agent's ADL `evals:` block against a running server. Useful in CI to validate agent behavior.

```bash
nui agent eval run -a my-agent
nui agent eval run -a my-agent --case smoke --case regression
nui agent eval run -a my-agent -w ./fixtures --json
nui agent eval run -a my-agent --parallel 2 --spawn
```

| Flag | Short | Description |
|---|---|---|
| `--agent-type` | `-a` | ADL agent id (required) |
| `--case` | | Run only evals with this name (repeatable) |
| `--working-dir` | `-w` | Default working directory for eval cases |
| `--parallel` | | Number of eval cases to run concurrently (default `1`) |
| `--json` | | Output machine-readable JSON results |
| `--spawn` | | Start server in the background if unreachable |
| `--url` | | Server base URL |

Exit code is non-zero when any case fails or errors. See [Agent evaluation](#agent-evaluation-schema) below for the ADL schema.

## Schedules

Manage recurring autonomous agent runs. Requires a running server.

```bash
nui schedule list
nui schedule add --agent-type claude-code --prompt "Daily summary" --cron "0 9 * * 1-5"
nui schedule add --agent-type my-agent --prompt "Check logs" --every 1h --name hourly
nui schedule enable <id>
nui schedule disable <id>
nui schedule run-now <id>
nui schedule delete <id>
```

`schedule add` requires `--agent-type` and `--prompt`, plus either `--cron` or `--every`.

## Extensions

```bash
nui extension add ./my-extension/          # local directory
nui extension add https://github.com/...   # git URL or zip
nui extension list
nui extension remove my-extension
nui extension create my-extension          # scaffold a new extension
```

## Skills

```bash
nui skills add ./my-skill/
nui skills list
nui skills remove my-skill
```

## Memory

Manage persistent memory files used by agents.

```bash
nui memory list
nui memory show user
nui memory show my-agent
nui memory edit user                      # opens $EDITOR
```

## MCP servers

Expose nui to external MCP hosts or run built-in MCP servers injected into harnesses.

| Command | Purpose |
|---|---|
| `nui mcp` | Main MCP server — `list_agents`, `create_session`, `run_agent`, etc. |
| `nui hitl-mcp` | Human-in-the-loop prompts (`ask_user`, approvals) |
| `nui viz-mcp` | Inline chart/visualization rendering |
| `nui agent-mcp` | Save ADL agents and update memory |
| `nui orchestrator-mcp` | Orchestrator launcher routing |

See [MCP integration]({{ '/features/mcp/' | relative_url }}) for config examples.

## Harness SDK

```bash
nui harness-sdk reinstall    # copy SDK modules to ~/.nui/harness-sdk/
```

## Agent evaluation schema

Add an `evals:` list to any ADL agent definition. Each case sends a prompt (or multi-turn conversation) to the agent and grades the response.

```yaml
adl: "1.0"
id: my-agent
name: My Agent
harness:
  type: claude-code
  sandbox: none
systemPrompt: You are a helpful assistant.
evals:
  - name: smoke
    description: Basic greeting check
    input: Say hello in one sentence.
    expect:
      type: contains
      value: hello
  - name: regex-check
    input: What is 2 + 2?
    expect:
      type: regex
      value: '\b4\b'
  - name: multi-turn
    messages:
      - role: user
        content: My name is Alex.
      - role: assistant
        content: Nice to meet you, Alex!
      - role: user
        content: What is my name?
    expect:
      type: contains
      value: Alex
  - name: llm-judge
    input: Explain recursion briefly.
    expect:
      type: llm
      criteria: Response mentions base case and self-reference.
    timeout: 180
```

### Eval fields

| Field | Required | Description |
|---|---|---|
| `name` | yes | Unique case name (used with `--case`) |
| `description` | no | Human-readable note |
| `input` | one of | Single user prompt |
| `messages` | one of | Multi-turn conversation (must end with a `user` turn) |
| `expect` | no | How to grade output (see below) |
| `timeout` | no | Timeout in seconds (default `120`, `300` for devcontainer harnesses) |
| `workingDir` | no | Override working directory for this case |
| `tags` | no | Labels for filtering |
| `disabled` | no | Skip this case when `true` |

### Grader types (`expect.type`)

| Type | Fields | Description |
|---|---|---|
| `contains` | `value` | Output contains `value` (case-insensitive) |
| `exact` | `value` | Output matches `value` exactly (trimmed) |
| `regex` | `value` | Output matches the regular expression |
| `llm` | `criteria` | LLM judge grades output against natural-language criteria |
| `none` | | Run only — no automatic grading |

### HTTP API

Evals can also be triggered over REST:

```
POST /api/agents/:id/evals/run
```

Body: `{ "caseNames": ["smoke"], "workingDir": "./fixtures", "parallel": 2 }`

## Further reading

- [ADL agents]({{ '/features/adl/' | relative_url }}) — custom agent definitions
- [Headless & scheduled runs]({{ '/features/headless/' | relative_url }}) — CI and cron workflows
- [ADL design doc](https://github.com/plmbr/nui/blob/main/dev/adl/design.md) — full schema reference
- [Developer guide](https://github.com/plmbr/nui/blob/main/DEVELOPERS.md) — build from source, API details
