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

`nui run` and `nui agent run` are equivalent. Set `NUI_URL` or pass `--url` if the server is not on `http://127.0.0.1:8080`. Use `--session-id` to continue an existing session.

See the [CLI reference]({{ '/cli/#headless-runs' | relative_url }}) for all flags.

## Scheduled runs

```bash
nui schedule list
nui schedule add --agent-type claude-code --prompt "Daily standup summary" --cron "0 9 * * 1-5"
nui schedule enable <id>
nui schedule run-now <id>
nui schedule delete <id>
```

Schedules require a running `nui server` instance. Use `--every` instead of `--cron` for interval-based schedules (e.g. `--every 1h`).

## Agent evaluation

Define test cases in an agent's ADL `evals:` block — in **Customize → Agents → Evals** or directly in YAML — then run them from the UI or terminal.

### From the UI

Open **Customize → Agents**, select an agent, and use the **Evals** section to add cases (name, prompt, expected text). Run a single case with **Run**, or all enabled cases with **Run evals**. See [ADL agents — eval UI]({{ '/features/adl/#define-and-run-evals-in-the-ui' | relative_url }}) for the full form reference.

### From the CLI

```bash
nui agent eval run -a my-agent
nui agent eval run -a my-agent --case smoke --case regression
nui agent eval run -a my-agent --parallel 2 --json --spawn
```

Each case sends a prompt (or multi-turn conversation) to the agent and grades the output. Graders support `contains`, `exact`, `regex`, `llm` (LLM judge), and `none` (run only).

```yaml
evals:
  - name: smoke
    input: Say hello in one sentence.
    expect:
      type: contains
      value: hello
```

Filter cases with `--case`, set a default working directory with `-w`, and get machine-readable output with `--json`. The command exits non-zero when any case fails.

Full schema, grader types, and a multi-turn example: [CLI reference — Agent evaluation]({{ '/cli/#agent-evaluation-schema' | relative_url }}).

## Server launch flags

These flags apply to `nui server` when starting the web UI with a pre-created session:

| Flag | Short | Description |
|---|---|---|
| `--open` | | Open the web UI in your browser with a new session |
| `--agent-type` | `-a` | ADL agent id (e.g. `claude-code`, `pi`, `codex`) |
| `--prompt` | `-m` | Initial prompt sent automatically |
| `--hide-input` | | Hide the chat input (use with `--prompt`) |
| `--working-dir` | `-w` | Working directory for the session |
| `--theme` | | UI theme: `light` or `dark` |
| `--default-agent` | | Default ADL agent id for new sessions |
| `--default-harness` | | Default harness for internal agents (e.g. `api/anthropic`) |
