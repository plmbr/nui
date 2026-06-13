# ADL Examples

ADL (Agent Definition Language) is a YAML format for declaring agent pipelines.
These examples range from trivial to production-complex — all are designed to be
runnable once the ADL executor is implemented.

## Examples

| File | Complexity | Demonstrates |
|---|---|---|
| `01-hello-world.yaml` | Trivial | Single step, default harness |
| `02-sequential-research-write.yaml` | Basic | `dependsOn`, named outputs, per-step `systemPrompt` |
| `03-docker-code-runner.yaml` | Basic | Docker harness |
| `04-remote-agent.yaml` | Basic | Remote harness |
| `05-parallel-research-fan-out.yaml` | Intermediate | Parallel steps, fan-in, multiple `inputs` |
| `06-hitl-approval-gate.yaml` | Intermediate | `approval: required`, `approvalTimeout` |
| `07-loop-policy.yaml` | Intermediate | `loop` policy, `maxIterations`, self-improvement |
| `08-batch-processing.yaml` | Intermediate | `batch` policy, `maxConcurrency`, array inputs |
| `09-multi-harness-pipeline.yaml` | Advanced | Per-step harness/model override, 4 harness types |
| `10-autonomous-scheduled.yaml` | Advanced | `schedule.cron`, MCP tools, no HITL |
| `11-complex-research-pipeline.yaml` | Complex | All features: parallel fan-out, multi-harness, adversarial verify loop, HITL, batch, MCP |

## Key ADL Concepts

### Harness override per step
```yaml
harness:
  type: claude-code
  model: claude-sonnet-4-6      # top-level default

steps:
  - name: plan
    harness:
      type: claude-code
      model: claude-opus-4-8    # this step uses a different model
```

### Named outputs and typed inputs
```yaml
steps:
  - name: research
    outputs:
      - name: brief
        type: text              # text | json

  - name: write
    dependsOn: [research]
    inputs:
      - from: research.brief    # <stepName>.<outputName>
        as: researchBrief       # optional alias in the prompt context
```

### Execution policies

| Policy | Semantics |
|---|---|
| `react` | Think → Act → Observe loop (standard LLM agent) |
| `sequential` | Fixed ordered steps, output feeds next |
| `parallel` | Steps in the same `dependsOn` level run concurrently |
| `loop` | Repeat until the step signals `DONE` or `maxIterations` is reached |
| `batch` | Map step over an array input with bounded concurrency |
| `conditional` | Route based on a prior step's output (TBD) |

### Approval gates
```yaml
- name: review
  approval: required
  approvalTimeout: 30m    # deny on timeout — safest default for irreversible actions
```

The orchestrator pauses execution, persists gate state in `~/.loop/data.json`,
and waits for `POST /api/projects/:id/approve`. On approval the next step runs;
on rejection or timeout the run ends.

## Implementation Notes

These examples define the target state. The ADL executor (Phase 2) needs to:

1. Parse YAML into a `StepGraph` (nodes + `dependsOn` edges)
2. Topologically schedule steps using `Azure/go-workflow` or equivalent
3. Resolve harness per step, falling back to the top-level harness
4. Forward the previous step's named outputs as inputs to dependents
5. Implement `approval: required` as a persisted channel block
6. Support `loop` and `batch` as special execution modes outside the DAG runner

See `dev/adl/orchestration-research.md` for the research backing these decisions.
