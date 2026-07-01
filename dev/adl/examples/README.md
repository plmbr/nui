# ADL Examples

ADL (Agent Definition Language) is a YAML format for declaring agent types and multi-step workflows. Place files in `~/.loop/agents/` to make them selectable in the Loop UI under **Custom Agents**.

Each agent requires an `id` (stable identifier used by the CLI and sessions) and a `name` (display label). The UI shows the name with the description below it.

The executor in `internal/agent/adl.go` runs multi-step pipelines today. Examples marked *planned* in the table below have ADL fields that parse correctly but are **not enforced** at runtime.

## Examples

| File | Complexity | Demonstrates | Runnable today? |
|---|---|---|---|
| `01-hello-world.yaml` | Trivial | Single step, default harness | Yes |
| `02-sequential-research-write.yaml` | Basic | `dependsOn`, named outputs, per-step `systemPrompt` | Yes |
| `03-docker-code-runner.yaml` | Basic | Docker harness | Yes (build image first) |
| `04-remote-agent.yaml` | Basic | Remote harness | Yes (start remote agent first) |
| `05-parallel-research-fan-out.yaml` | Intermediate | Parallel steps, fan-in | Steps run sequentially* |
| `06-hitl-approval-gate.yaml` | Intermediate | `approval: required` | No — HITL not implemented |
| `07-loop-policy.yaml` | Intermediate | `loop` policy | No — policy not enforced |
| `08-batch-processing.yaml` | Intermediate | `batch` policy | No — policy not enforced |
| `09-multi-harness-pipeline.yaml` | Advanced | Per-step harness/model override | Yes |
| `10-autonomous-scheduled.yaml` | Advanced | `schedule.cron`, `aiAssets.mcpServers` | Schedule not enforced |
| `11-complex-research-pipeline.yaml` | Complex | All features combined | Partial |
| `12-codex-sandbox-variants.yaml` | Basic | codex: none / bubblewrap / docker | Yes |
| `13-opencode-sandbox-variants.yaml` | Basic | opencode: none / bubblewrap / docker | Yes |
| `14-ai-assets-mcp.yaml` | Basic | `aiAssets.mcpServers` (HTTP + stdio) | Yes |
| `15-env-vars.yaml` | Basic | Global `env` + `harness.env` | Yes |
| `16-ai-assets-skills.yaml` | Basic | `aiAssets.skills` (path, ref, content, git) | Yes |

\*All steps execute in topological order regardless of `policy` field today.

## Harness types

All six types are supported by the executor:

| Type | How it runs |
|---|---|
| `claude-code` | `claude` CLI subprocess |
| `pi` | `pi --mode rpc` subprocess |
| `codex` | `codex exec` subprocess |
| `opencode` | `opencode serve` + `opencode run` |
| `docker` | HTTP/SSE in user-managed Docker container |
| `remote` | HTTP/SSE at configured `host:port` |

## Key concepts

### Harness override per step

```yaml
harness:
  type: claude-code
  model: claude-sonnet-4-6

steps:
  - name: plan
    harness:
      type: claude-code
      model: claude-opus-4-8
```

### AI assets (MCP servers)

MCP servers are declared under `aiAssets.mcpServers`. Each entry requires `name`. Use `url` + `type` for HTTP/SSE servers, or `command` + `args` + `type` for stdio servers.

```yaml
aiAssets:
  mcpServers:
    - name: test-mcp-server
      url: http://localhost:3000/mcp
      type: http
    - name: local-tool
      command: npx
      args: ["-y", "some-mcp-package"]
      type: stdio
```

Loop provisions these into `~/.loop/sessions/<session-id>/` and sets the harness config-dir env var (`CLAUDE_CONFIG_DIR`, `CODEX_HOME`, `PI_CODING_AGENT_DIR`, or `OPENCODE_CONFIG_DIR`) before each run.

Per-step overrides:

```yaml
steps:
  - name: doc-search
    aiAssets:
      mcpServers:
        - name: internal-docs
          url: http://localhost:3040
          type: http
```

### System prompt and skills

```yaml
systemPrompt: |
  You are a helpful assistant.

aiAssets:
  skills:
    - name: code-review
      path: ./skills/code-review
    - name: commit-helper
      ref: commit-helper
    - name: greeting
      content: |
        ---
        name: greeting
        description: Brief greeting skill
        ---
        Keep responses to one sentence.
```

Legacy top-level `skill:` still works (mapped to a single `aiAssets.skills` entry).

Install catalog skills ahead of time:

```sh
loop skills add ./skills/code-review
loop skills add https://github.com/example/agent-skills/tree/main/skills/shared-style
loop skills add --git https://github.com/example/agent-skills.git --path skills/shared-style
loop skills list
```

`systemPrompt` is written as harness-native markdown (`CLAUDE.md`, `AGENTS.md`, etc.). Skills are copied into the session harness skills directory (full directory tree, not just `SKILL.md`).

### Environment variables

Global `env` applies to every harness subprocess. `harness.env` overrides global keys for that harness (and per-step harness overrides).

```yaml
env:
  ANTHROPIC_BASE_URL: https://api.anthropic.com

harness:
  type: claude-code
  model: claude-sonnet-4-6
  env:
    ANTHROPIC_API_KEY: your-api-key
```

### Prompt mode

`promptMode: user` (default) waits for the user to type a message. `promptMode: auto` hides the input and runs on session open with a launch prompt, ADL `defaultPrompt`, or the built-in phrase `"Follow your system instructions and run."`.

```yaml
promptMode: auto
defaultPrompt: Follow your system instructions and run.
```

### Named outputs and inputs

```yaml
steps:
  - name: research
    outputs:
      - name: brief
        type: text

  - name: write
    dependsOn: [research]
    inputs:
      - from: research.brief
        as: researchBrief
```

### Sandbox variants

Applies to `claude-code`, `pi`, `codex`, and `opencode`:

```yaml
harness:
  type: codex
  sandbox: none        # host subprocess (default)

harness:
  type: codex
  sandbox: bubblewrap  # Linux only; bwrap wrapper

harness:
  type: codex
  sandbox: docker
  image: loop-codex:latest   # builtin image, port 8090
```

### Approval gates (*planned*)

```yaml
- name: review
  approval: required
  approvalTimeout: 30m
```

When implemented, the executor will pause and wait for `POST /api/sessions/:id/approve`.

## Executor status

| Feature | Status |
|---|---|
| YAML parse | Done |
| Topological scheduling | Done |
| Per-step harness / model / systemPrompt | Done |
| Named outputs → inputs | Done |
| Six harness types + sandbox | Done |
| `aiAssets.mcpServers` → harness config | Done |
| `aiAssets.skills` → harness config | Done |
| `skill` + `systemPrompt` → harness config | Done |
| `env` / `harness.env` → subprocess env | Done |
| `promptMode` / `defaultPrompt` | Done |
| Step `policy` (parallel/loop/batch) | Parsed only |
| `approval` / `approvalTimeout` | Parsed only |
| `constraints` | Parsed only |
| `schedule.cron` | Parsed only |

See [dev/dev.md](../../dev.md) for the full architecture and [orchestration-research.md](../orchestration-research.md) for design research.
