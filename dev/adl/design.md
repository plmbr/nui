# Agent Definition Language (ADL) 1.0

ADL is the YAML format nui uses to define agents. Install agent files to `~/.nui/agents/` or ship them via extensions.

**Canonical specification:** [github.com/plmbr/ADL](https://github.com/plmbr/ADL) — full [design.md](https://github.com/plmbr/ADL/blob/main/design.md), [JSON Schema](https://github.com/plmbr/ADL/blob/main/schema/adl.schema.json), and [examples](https://github.com/plmbr/ADL/tree/main/examples). With a sibling checkout, see `../ADL/`.

The notes below are a nui-focused summary. When in doubt, prefer the ADL repo.

## Top-level fields

| Field | Required | Description |
|---|---|---|
| `adl` | yes | Must be `"1.0"` |
| `id` | yes | Stable agent identifier (used in sessions and CLI) |
| `name` | yes | Display name in the UI |
| `description` | no | Short description |
| `tags` | no | Labels for filtering in the new-session UI |
| `harness` | yes | How the agent runs (see below) |
| `allowedHarnesses` | no | CLI harness whitelist for session/CLI override; omit = any CLI harness when default is CLI |
| `systemPrompt` | no | System prompt for single-step agents |
| `promptMode` | no | `user` (default) or `auto` (for scheduled agents) |
| `defaultPrompt` | no | Prompt used in `auto` mode |
| `workingDirInput` | no | When true, user picks working directory at session create |
| `aiAssets` | no | Skills and MCP servers |
| `hitl` | no | Human-in-the-loop mode and channels |
| `toolApprovals` | no | Allow/deny tool policies |
| `evals` | no | Test cases for `nui agent eval run` |
| `orchestration` | no | Multi-agent modes: `subAgents`, `council`, or `workflow` |

Legacy top-level `steps`, `subAgents`, and `council` are **rejected**. Use `orchestration` instead.

## Harness types

| `harness.type` | Description |
|---|---|
| `claude-code`, `pi`, `codex`, `opencode` | Host subprocess (optional `sandbox`: `none`, `bubblewrap`, `docker`) |
| `api` | In-process LLM API (`provider`, `model`, `baseUrl`, `apiKeyEnv`, `disableTools`) |
| `docker` | HTTP/SSE container (`image`, `containerPort`) |
| `devcontainer` | nui-managed dev container (`innerHarness`: CLI type above) |
| `remote` | Pre-running HTTP/SSE server (`host`, `port`) |
| `ext:<extension>/<harness-id>` | Extension harness (stdio/tcp/http) |

See [harness-design.md](../harness-design.md) for wire protocols. Sample YAML: [examples/](examples/) (nui-local) or [ADL/examples](https://github.com/plmbr/ADL/tree/main/examples).

## Orchestration

All multi-agent execution is configured under `orchestration`:

| `orchestration.type` | Behavior |
|---|---|
| `subAgents` | Adaptive chair — picks **one** member at a time via `run_sub_agent`, then finishes or re-delegates |
| `council` | Deliberation — members run scheduled rounds; this agent synthesizes the verdict |
| `workflow` | Step DAG — topological `dependsOn` pipeline (optionally with `type: hitl` gates) |

### Shared member fields (`subAgents` / `council`)

```yaml
orchestration:
  type: council          # or subAgents
  members:
    - agent: hello-world
    - agent: code-reviewer
  sessionMode: persistent   # persistent | fresh
  memberTimeout: 8m
```

- `members[].agent` — registry agent ids only (builtins, `~/.nui/agents/`, `ext:…`)
- Orchestration agents cannot nest as members or workflow step agents of another orchestration agent

### `subAgents`

```yaml
orchestration:
  type: subAgents
  members:
    - agent: claude-code
    - agent: codex
  maxTurns: 20
```

**Execution (nui):**

1. The chair (this agent) chooses **exactly one** member best suited to start (or continue) the work.
2. That member runs alone — there is no fan-out to all members.
3. From the result, the chair either **finishes** with the final answer or **delegates again** to the same or a different member.
4. Repeat until the goal is done or `maxTurns` is exhausted (default 20).

Member sessions are created lazily on first delegation. The UI shows only the currently running member (overview lists members that have participated).

API harnesses use the `run_sub_agent` tool; CLI harnesses use a one-action-per-turn JSON protocol (`delegate` / `finish`).

See [examples/subagents-orchestrator.yaml](examples/subagents-orchestrator.yaml).

### `council`

```yaml
orchestration:
  type: council
  members:
    - agent: claude-code
    - agent: codex
  rounds: independent+rebuttal   # independent | independent+rebuttal | independent+rebuttal+adjudication
  quorum: 2
  failurePolicy: continue-with-quorum   # or fail
```

See [examples/council-agent.yaml](examples/council-agent.yaml).

### `workflow`

```yaml
orchestration:
  type: workflow
  steps:
    - name: review
      harness:
        type: claude-code
      systemPrompt: Review the codebase and list issues.
    - name: fix
      dependsOn: [review]
      systemPrompt: Apply fixes for the issues found in the review step.
```

Steps run sequentially in topological order. Each user chat turn re-runs the full DAG. See [examples/workflow-agent.yaml](examples/workflow-agent.yaml).

## Validation

nui validates ADL on load via `internal/model/adl_validate.go`. Invalid files are skipped with a warning in server logs.

## Further reading

- [plmbr/ADL](https://github.com/plmbr/ADL) — canonical ADL spec, schema, and examples
- [harness-design.md](../harness-design.md) — harness protocols
- [extension-api.md](../extension-api.md) — extension manifests
- [dev.md](../dev.md) — nui product spec
