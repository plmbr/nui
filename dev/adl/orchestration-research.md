# ADL Multi-Step Orchestration — Research Findings [AI generated]

---

## Summary

Production multi-agent orchestration systems converge on two execution models for
heterogeneous-harness DAGs:

1. **Goroutine-per-ready-step** with typed closure-based data flow (`go-workflow`, `natessilva/dag`)
2. **Pregel-style bulk-synchronous parallel super-steps** with typed state channels (LangGraph)

For human-in-the-loop approval gates, all surveyed systems (LangGraph, Temporal, Dapr,
Microsoft Agent Framework) model pausing as a first-class primitive that suspends a specific
branch or the entire graph while persisting state, with resumption triggered by an external
signal carrying the approval value.

Inter-step data flow uniformly avoids generic message buses in favor of typed, direct handoffs.

No surveyed open spec (Agent Spec v1, OpenAI Agents SDK, LangGraph) provides a Docker/
container-specific harness abstraction — container isolation is treated as an infrastructure
concern layered beneath the orchestration protocol.

---

## Q1 — How do production systems handle heterogeneous-harness DAGs?

**Convergent answer: explicit `dependsOn` DAG + goroutine-per-ready-step.**

Every surveyed system (Argo, LangGraph, go-workflow, Dapr) uses the same structural
primitive: declare edges explicitly, run all nodes whose dependencies are satisfied in
parallel automatically. The harness type is treated as an infrastructure detail *below*
the orchestration layer.

LangGraph uses a **Pregel bulk-synchronous parallel (BSP)** model:
- Parallel-policy steps share a "super-step"
- Sequential/conditional steps each form their own super-step with a state flush between them
- `react` policy = one step per super-step

**Refuted (0-3):** A2A Agent Cards were proposed as a per-step harness-routing mechanism.
Rejected — harness selection belongs in ADL config, not runtime agent discovery.

---

## Q2 — Right Go-side execution model?

**Recommendation: [`Azure/go-workflow`](https://github.com/Azure/go-workflow)**

| Library | Verdict |
|---|---|
| `Azure/go-workflow` | Best fit — goroutines per ready step, `MaxConcurrency` cap, typed Input/Output callbacks, explicit dependency declarations mapping directly to ADL `dependsOn` |
| `natessilva/dag` | Simpler but lacks built-in output passing; requires shared closure capture (fragile for multi-harness step outputs) |
| Embedded Temporal/Dapr | Overkill for a self-hosted lightweight app |

The key differentiator is **typed Input callbacks** — each step reads upstream outputs
directly from the prior step's struct after its `Do()` completes:

```go
workflow.Add(save,
    workflow.DependOn(fetch),
    workflow.Input(func(_ context.Context, s *Save) error {
        s.Body = fetch.Body  // typed; no message bus
        return nil
    }),
)
```

Sources: [Azure/go-workflow](https://github.com/Azure/go-workflow) · [natessilva/dag](https://pkg.go.dev/github.com/natessilva/dag) · [dagor](https://dev.to/will_zhang_598824ef87a46c/introducing-dagor-a-high-performance-dag-execution-engine-in-go-2m5e) (3-0 verified)

---

## Q3 — Inter-step data flow?

**Pattern: typed direct handoffs, not a message bus.** All surveyed systems agree:

| System | Mechanism |
|---|---|
| go-workflow | Typed `Input` callback reads upstream step struct fields |
| Argo | `{{tasks.stepname.outputs.parameters.value}}` template refs (scalars) + artifact refs (large/binary) |
| OpenAI Agents SDK | `HandoffInputData` struct — `input_history`, `pre_handoff_items`, `new_items`, optional `input_filter` to strip/summarize |
| LangGraph | Typed state channels with per-key reducer functions (default: last-write-wins; `add_messages` = append with dedup by ID) |

**For ADL:** each step should produce a structured `StepOutput` (text, tool calls, metadata).
Steps receiving it via `dependsOn` get the prior step's `StepOutput` injected at start.

Per-step model switching = new agent instantiation with explicit context forwarding. From
OpenAI Agents SDK: *"it's as though the new agent takes over the conversation and gets to
see the entire previous conversation history."*

Sources: [OpenAI Agents SDK handoffs](https://openai.github.io/openai-agents-python/handoffs/) · [LangGraph graph API](https://docs.langchain.com/oss/python/langgraph/graph-api) · [Argo artifacts](https://argo-workflows.readthedocs.io/en/latest/walk-through/artifacts/) (3-0 verified)

---

## Q4 — Approval gates (HITL)?

**All systems model pause as a first-class primitive.** Convergent pattern:

| System | Pause | Resume |
|---|---|---|
| LangGraph | `interrupt()` suspends node | `Command(resume=value)` — value becomes `interrupt()` return |
| Temporal | `workflow.wait_condition()` | Signal arrival satisfying condition |
| Dapr | `WaitForExternalEvent` | `POST raise-event` API; optional `when_any([approval, timeout])` race |
| Microsoft Agent Framework | `RequestPort/RequestInfoEvent` checkpoint | Re-emits pending requests on restore |

**Key finding for Loop:** pending approval state must be **persisted** (in `~/.loop/data.json`)
so a server restart can re-emit the pending gate. The Microsoft Agent Framework pattern is most
directly applicable:
1. Persist gate state at checkpoint
2. Re-emit pending requests as events on restore
3. Route human response back via the same channel as normal step outputs

**Refuted (0-3):** Temporal's pause is "effectively free." It occupies a slot but doesn't burn
CPU. Matters for Loop's resource model if many projects have pending approvals simultaneously.

Sources: [LangGraph](https://docs.langchain.com/oss/python/langgraph/graph-api) · [Temporal HITL tutorial](https://learn.temporal.io/tutorials/ai/building-durable-ai-applications/human-in-the-loop/) · [Dapr workflow patterns](https://docs.dapr.io/developing-applications/building-blocks/workflow/workflow-patterns/) · [Microsoft Agent Framework HITL](https://learn.microsoft.com/en-us/agent-framework/workflows/human-in-the-loop) (3-0 on all four)

---

## Q5 — Open specs and multi-harness?

**No open spec covers Docker/container harnesses at the protocol level.**

Agent Spec v1 defines three harnesses:
- **ServerTools** — executed in the same runtime environment as the agent
- **ClientTools** — not executed by the runtime; caller runs and returns results
- **RemoteTools** — run in an external environment via RPC or REST

Zero mentions of Docker or containers. Container isolation is infrastructure below the spec.

**Implication:** Loop's Docker harness is correctly modeled as RemoteTools (container exposes
HTTP/SSE) — no spec alignment work needed. The existing `HTTPExtensionAgent` already fits.

Source: [Agent Spec v1](https://arxiv.org/html/2510.04173v1) (3-0 verified)

---

## Recommended ADL Execution Design for Loop

```
ADL YAML parse → StepGraph (nodes + dependsOn edges)
       ↓
go-workflow DAG runner (goroutine per ready step)
  ├── each step resolves its harness from ADL config:
  │     claude-code → ClaudeCodeAgent
  │     docker      → HTTPExtensionAgent (existing)
  │     remote      → HTTPExtensionAgent (existing)
  │     pi          → ExtensionAgent (existing)
  ├── typed Input callback injects prior step's StepOutput
  ├── step emits SSE events to browser (existing pipeline)
  └── approval: required → emit needs_approval event
                         → persist gate in ~/.loop/data.json
                         → block goroutine on channel
                         → POST /api/projects/:id/approve resumes channel
```

### ADL step schema additions implied by research

```yaml
steps:
  - name: research
    harness:                    # per-step override of top-level harness
      type: claude-code
      model: claude-opus-4-8
    policy: react
    outputs:
      - name: report            # named outputs for downstream reference
        type: text
    approval: none

  - name: publish
    harness:
      type: docker
      image: my-publisher
    dependsOn: [research]
    inputs:
      - from: research.report   # typed reference to upstream named output
    approval: required
    approvalTimeout: 30m        # deny on timeout (safest default for irreversible actions)
```

---

## Open Questions

1. **`batch` and `loop` policies** — dynamic step registration in a running workflow, or
   fixed unrolled DAG generated at ADL parse time? `go-workflow` docs don't address cyclic
   DAGs; may need special-casing for `react`/`loop` outside the DAG runner.

2. **Checkpoint depth for approval gates** — full step output snapshots (enables resume after
   server restart) vs. gate state only (simpler, loses intermediate outputs on restart)?

3. **Docker harness security** — RemoteTools-compatible HTTP adapter (container exposes
   HTTP/SSE, already implemented) vs. stdio-bridged ServerTools. Security implications differ
   for a self-hosted deployment.

4. **`go-workflow` cycle support** — does it support cyclic DAGs needed for `react`/`loop`
   policies, or does Loop need to handle those as a separate execution path?

---

## Refuted Claims (excluded from findings)

| Claim | Vote | Source |
|---|---|---|
| Argo emissary reads/writes artifacts file-based by default | 0-3 | [Argo executors](https://argo-workflows.readthedocs.io/en/latest/workflow-executors/) |
| Temporal workflow pause is "effectively free" (no slot consumed) | 0-3 | [Temporal HITL](https://learn.temporal.io/tutorials/ai/building-durable-ai-applications/human-in-the-loop/) |
| MCP stdio vs ACP REST makes ACP better for container orchestration | 0-3 | [Agent protocol comparison](https://arxiv.org/pdf/2505.02279) |
| A2A Agent Cards drive per-step harness/model routing | 0-3 | [Agent protocol comparison](https://arxiv.org/pdf/2505.02279) |
| ACP is the only protocol with native pause/resume for approval gates | 0-3 | [Agent protocol comparison](https://arxiv.org/pdf/2505.02279) |
