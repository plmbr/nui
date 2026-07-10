# ADL Examples

ADL (Agent Definition Language) is a YAML format for declaring agent types and multi-step workflows. Place files in `~/.loop/agents/` to make them selectable in the Loop UI under **Installed agents**.

Echo-specific docker/remote walkthroughs (with runnable harness servers) live under [`dev/harness-examples/`](../harness-examples/).

Each agent requires an `id` (stable identifier used by the CLI and sessions) and a `name` (display label). The UI shows the name with the description below it.

The executor in `internal/agent/adl.go` runs multi-step pipelines in **sequential** topological order (no parallel fan-out).

## Examples

| File | Complexity | Demonstrates |
|---|---|---|
| `01-hello-world.yaml` | Trivial | Single step, default harness |
| `02-sequential-research-write.yaml` | Basic | `dependsOn`, named outputs, per-step `systemPrompt` |
| `03-docker-code-runner.yaml` | Basic | Docker harness |
| `04-remote-agent.yaml` | Basic | Remote harness |
| `06-devcontainer-agent.yaml` | Basic | Devcontainer harness |
| `05-parallel-research-fan-out.yaml` | Intermediate | Fan-in via `dependsOn` (steps still run sequentially) |
| `09-multi-harness-pipeline.yaml` | Advanced | Per-step harness/model override |
| `11-complex-research-pipeline.yaml` | Complex | Multi-harness pipeline with MCP and verification |
| `12-codex-sandbox-variants.yaml` | Basic | codex: none / bubblewrap / docker |
| `13-opencode-sandbox-variants.yaml` | Basic | opencode: none / bubblewrap / docker |
| `14-ai-assets-mcp.yaml` | Basic | `aiAssets.mcpServers` (HTTP + stdio) |
| `15-env-vars.yaml` | Basic | Global `env` + `harness.env` |
| `16-ai-assets-skills.yaml` | Basic | `aiAssets.skills` (path, ref, content, git) |
| `17-auto-scheduled-agent.yaml` | Basic | `promptMode: auto` agent for Customize → Schedules |
| `18-tool-approvals.yaml` | Basic | Top-level `toolApprovals` with `harness.permissions: interactive` |
| `19-hitl-workflow-gate.yaml` | Intermediate | Workflow step `type: hitl` orchestration gate |
| `20-evals.yaml` | Basic | `evals` test cases for `loop agent eval run` |
| `21-orchestrator-sub-agents.yaml` | Intermediate | `subAgents` orchestrator routing to registry agents |

## Harness types

Builtin and connector types supported by the executor:

| Type | How it runs |
|---|---|
| `claude-code` | `claude` CLI subprocess |
| `pi` | `pi --mode rpc` subprocess |
| `codex` | `codex exec` subprocess |
| `opencode` | `opencode serve` + `opencode run` |
| `docker` | HTTP/SSE in user-managed Docker container |
| `devcontainer` | Loop-managed devcontainer sandbox (`innerHarness` CLI via devcontainer exec) |
| `remote` | HTTP/SSE at configured `host:port` |
| `ext:<extension>/<harness-id>` | Installed extension harness (stdio/tcp/http) |

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

Per-step overrides merge with top-level `aiAssets` by name (step entries override same name):

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

### Rules and mention providers

```yaml
aiAssets:
  rules:
    - ref: ext:corp-pack/corp-guidelines
  mentionProviders:
    - ref: ext:corp-pack/corp-refs
```

Rules materialize into harness-specific rule files. Mention providers power `@`-mention autocomplete when opted in via ADL.

### Prompt suggestions

Quick-start pills above the chat input:

```yaml
promptSuggestions:
  - title: Review code
    prompt: Review the current changes and suggest improvements.
    icon: sparkles
```

### HITL workflow gates

Workflow steps with `type: hitl` pause the DAG for human approval, questions, or review:

```yaml
steps:
  - name: draft
    outputs:
      - name: summary
        type: text

  - name: review-gate
    type: hitl
    dependsOn: [draft]
    hitl:
      kind: approval
      title: Approve summary
      message: Review the draft before publishing.
      actions:
        - id: approve
          label: Approve
        - id: reject
          label: Reject
      display:
        - from: draft.summary
      channels: [loop-ui]
```

See `19-hitl-workflow-gate.yaml` for a full example.

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

### Tool approvals

When `harness.permissions: interactive` (Claude Code or Codex), Loop routes native tool permission requests through the UI. Top-level `toolApprovals` controls which tools auto-approve vs prompt:

| Policy | Behavior |
|---|---|
| `default` | Prompt for every tool |
| `all` | Auto-approve all tools |
| `allowlist` | Auto-approve only listed tools |
| `denylist` | Prompt only for listed tools |

Tool names include native harness tools (`Bash`, `Write`) and MCP tools (`mcp__server-name__tool-name`). Glob patterns such as `mcp__corp__*` are supported. Selective policies are enforced for Claude Code in v1; Codex supports binary bypass only.

Session overrides: `agentConfig.toolApprovalPolicy` and `agentConfig.toolApprovalTools`.

```yaml
harness:
  permissions: interactive
toolApprovals:
  policy: denylist
  tools: [Bash, Write, Edit]
```

```yaml
promptMode: auto
defaultPrompt: Follow your system instructions and run.
```

### Evals

Define test cases on an agent to verify behavior with `loop agent eval run -a <agent-id>`:

```yaml
evals:
  - name: polite-greeting
    description: Agent introduces itself politely
    input: |
      Hello, who are you?
    expect:
      type: contains      # contains | exact | regex | llm | none
      value: assistant
    tags: [smoke]
    timeout: 120          # seconds (optional; default 120, 300 for devcontainer)
    workingDir: ./fixtures   # optional per-case override
    disabled: false
```

Multi-turn evals use `messages` instead of `input` (must end with a `user` turn). Graders: `contains`, `exact`, `regex`, `llm` (criteria rubric), or `none` (informational). Use `hitl.mode: off` and non-interactive tool approvals for unattended eval runs.

See `20-evals.yaml` for a full example.

### Named outputs and inputs

Each step's collected text is stored under declared `outputs` names (or an implicit default when omitted). Reference downstream with `from: stepName.outputName`:

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

Each user chat turn re-runs all workflow steps from the beginning.

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

## Executor status

| Feature | Status |
|---|---|
| YAML parse | Done |
| Topological scheduling | Done |
| Per-step harness / model / systemPrompt | Done |
| Named outputs → inputs | Done |
| Builtin + connector harness types + sandbox | Done |
| Extension harness `ext:<ext>/<id>` | Done |
| `aiAssets.mcpServers` → harness config | Done |
| `aiAssets.skills` → harness config | Done |
| `aiAssets.rules` → harness config | Done |
| `aiAssets.mentionProviders` → @-mention menu | Done |
| `skill` + `systemPrompt` → harness config | Done |
| `env` / `harness.env` → subprocess env | Done |
| `promptMode` / `defaultPrompt` | Done |
| `promptSuggestions` → chat UI pills | Done |
| `workingDirInput` → session create dialog | Done |
| `toolApprovals` selective auto-approve | Done (Claude; requires `harness.permissions: interactive`) |
| `steps[].type: hitl` orchestration gates | Done |
| `promptMode: auto` → schedules | Done |
| `evals` → `loop agent eval run` | Done |

See [dev/dev.md](../../dev.md) for the full architecture and [orchestration-research.md](../orchestration-research.md) for design research.
