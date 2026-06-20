# ADL Examples

ADL (Agent Definition Language) is a YAML format for declaring agent types and multi-step workflows. Place files in `~/.loop/agents/` to make them selectable in the Loop UI under **Custom Agents**.

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
| `10-autonomous-scheduled.yaml` | Advanced | `schedule.cron`, MCP tools | No — schedule/MCP not enforced |
| `11-complex-research-pipeline.yaml` | Complex | All features combined | Partial |
| `12-codex-sandbox-variants.yaml` | Basic | codex: none / bubblewrap / docker | Yes |
| `13-opencode-sandbox-variants.yaml` | Basic | opencode: none / bubblewrap / docker | Yes |

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
| Step `policy` (parallel/loop/batch) | Parsed only |
| `approval` / `approvalTimeout` | Parsed only |
| `constraints` | Parsed only |
| `schedule.cron` | Parsed only |
| `tools.mcp` | Parsed only |

See [dev/dev.md](../dev.md) for the full architecture and [orchestration-research.md](orchestration-research.md) for design research.
