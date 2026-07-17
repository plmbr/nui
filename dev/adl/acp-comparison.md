# ACP vs nui — Research Findings [AI generated]

> **Status:** Historical research (non-normative, partially outdated). API paths may reference `/api/projects/` — the current nui API uses `/api/sessions/`. Named ADL outputs and HITL workflow gates were added after this document. See [`dev/dev.md`](../dev.md) for the implemented architecture.

---

## Summary

There are two distinct protocols both called "ACP":

1. **Zed ACP** — [agentclientprotocol.com](https://agentclientprotocol.com/get-started/introduction): JSON-RPC 2.0 over stdio (local) or HTTP/WebSocket (draft) between code editors and coding agents, modeled on LSP. August 2025.
2. **IBM BeeAI ACP** — [agentcommunicationprotocol.dev](https://agentcommunicationprotocol.dev/introduction/welcome): REST/HTTP with SSE streaming and MIME-typed multipart messages for general agent communication. Archived August 27, 2025.

**The research question URL points to Zed ACP.** Both were included because they represent the current landscape.

**Core finding:** nui's ADL is architecturally orthogonal to both ACPs. ADL is a declarative workflow language (steps, DAG edges, harness types). ACP is a communication contract. They address different layers and could coexist — ACP could serve as the wire protocol for nui's docker/remote harnesses.

---

## Finding 1 — ACP is a communication protocol, not an execution framework (3-0)

Neither ACP spec defines agent execution models, DAG orchestration, step policies, or inter-step data flow.

- **Zed ACP**: JSON-RPC 2.0 over stdio. *"Local agents run as sub-processes of the code editor, communicating via JSON-RPC over stdio."*
- **IBM BeeAI ACP**: REST/HTTP with OpenAPI spec. *"Remains agnostic to internal implementations, requiring only minimal specifications for compatibility."*
- **Arxiv 2505.02279**: *"ACP uses RESTful HTTP interfaces... stateless by default."*

No `dependsOn`, `parallel`, `loop`, `batch`, or `conditional` constructs appear in either spec.

**Implication for nui:** nui's ADL step policies (react/sequential/parallel/nui/batch/conditional) and DAG execution are explicitly out of ACP's scope. Adopting ACP does not require rethinking ADL.

---

## Finding 2 — Session lifecycle and streaming (3-0)

**Zed ACP** defines a linear session lifecycle:

```
initialize → authenticate (optional) → session/new or session/load → prompt turns
```

Streaming progress is delivered via `session/update` — one-way JSON-RPC notifications (no response expected) carrying:
- `ContentChunk` with `messageId` (a change in `messageId` signals a new message)
- Tool call updates
- Plan updates
- Mode changes

**IBM BeeAI ACP** delivers streaming via SSE with typed run events: `run.created`, `run.completed`, `message.part`. This is structurally similar to nui's SSE event stream.

**nui comparison:**

| | nui | Zed ACP | IBM BeeAI ACP |
|---|---|---|---|
| Transport | HTTP/SSE | stdio JSON-RPC 2.0 | REST/HTTP + SSE |
| Streaming | `data: {type,content}` | `session/update` notifications | `run.X` SSE events |
| Session model | per-project in `data.json` | `session/new` / `session/load` | per-run |

nui's session model (per-project, persisted) is more analogous to Zed ACP's `session/load` than to IBM BeeAI ACP's stateless runs.

---

## Finding 3 — ACP has no DAG orchestration (3-0)

ACP's `Plan` object is a **visibility/reporting construct**, not an execution scheduler.

```
PlanEntry {
  content: string     // human-readable
  priority: high|medium|low
  status: pending|in_progress|completed
}
```

No `dependsOn` edges, no parallel lanes, no nui/batch/conditional policies. *"The client replaces the entire plan with each update"* — confirming snapshot-reporting semantics.

Multi-agent coordination is described only conceptually: *"specialized agents operate as coordinated teams with standardized handoffs"* — no formal orchestration primitives.

**This is the sharpest divergence from nui's ADL**, which defines `dependsOn` DAG edges and six step policies.

---

## Finding 4 — Human-in-the-loop (3-0)

**Zed ACP HITL:** per-tool-call permission request (`session/request_permission`):
- Blocks synchronously until user selects `allow_once`, `allow_always`, `reject_once`, or `reject_always`
- No timeout field
- *"On cancellation: client MUST respond to all pending session/request_permission requests with Cancelled outcome"*

**IBM BeeAI ACP HITL:** `Await` primitive with formal run state machine:
```
in_progress → awaiting → in_progress
```
REST resume endpoint: `POST /runs/{run_id}`. No configurable timeout.

**nui ADL** does not define HITL approval gates or scheduled runs in the current schema. Inter-step orchestration uses `dependsOn`, named outputs/inputs, and per-step harness overrides.

The functional analogy with ACP pause/resume (all pause execution awaiting external input) applies at the transport layer only; nui's chat UI drives interactive sessions today.

---

## Finding 5 — Alignment and divergence with nui (2-1)

### Where nui aligns with ACP

| nui component | ACP analogue |
|---|---|
| `pi` harness (TCP JSON-RPC 2.0 to local Python process) | Zed ACP stdio JSON-RPC 2.0 transport |
| `docker`/`remote` harness (HTTP/SSE) | IBM BeeAI ACP REST/SSE transport |
| Per-project session ID persisted in `data.json` | Zed ACP `session/new` / `session/load` lifecycle |
| SSE events (`text`, `done`, `error`) | IBM BeeAI ACP run events (`message.part`, `run.completed`, `run.failed`) |
| `GET /info` health check | Both ACPs: protocol-level handshake/initialization |

### Where nui diverges from ACP

| nui feature | ACP coverage |
|---|---|
| `dependsOn` DAG edges | Not in scope for either ACP |
| Named outputs / typed inputs between steps | Not in scope for either ACP |
| Per-step harness/model override | Not in either ACP spec |

### ACP compatibility path

Making nui's docker/remote harnesses ACP-compatible would require:

1. **IBM BeeAI ACP target**: Wrap `POST /run → SSE` in ACP's `POST /runs` envelope; map `EventText → message.part`, `EventDone → run.completed`, `EventError → run.failed`; expose `POST /runs/{id}` for resume (maps to existing `POST /api/projects/:id/approve`).
2. **Zed ACP target**: Replace TCP JSON-RPC in the `pi` harness with Zed ACP's stdio JSON-RPC 2.0, gaining `session/new` / `session/load` semantics. For docker/remote, wait for Zed ACP's HTTP/WebSocket transport to stabilize.

In both cases, **ADL's DAG execution layer remains above ACP's scope** — no ACP changes are needed to implement `dependsOn` or inter-step data flow.

---

## Protocol Landscape Comparison

| Protocol | Transport | Orchestration | HITL | Primary use case |
|---|---|---|---|---|
| **Zed ACP** | JSON-RPC 2.0 / stdio | None (communication only) | Per-tool permission gate | Code editor ↔ coding agent |
| **IBM BeeAI ACP** | REST/HTTP + SSE | None (run-level only) | Await/resume | General agent communication |
| **A2A** | HTTP + SSE | None (agent-to-agent calls) | None | Agent-to-agent task delegation |
| **MCP** | JSON-RPC 2.0 / stdio or HTTP | None (tool calls only) | None | LLM ↔ tool/data source |
| **OpenAI Agents SDK** | Library (Python) | Handoffs, typed inputs | Interrupt/resume | Python multi-agent workflows |
| **LangGraph** | Library (Python) | Pregel BSP DAG | interrupt() | Python stateful agent graphs |
| **nui ADL** | YAML + Go executor | DAG + multi-harness steps | Interactive chat | Self-hosted multi-harness pipelines |

**Key takeaway:** No open protocol covers what ADL covers. ACP, A2A, and MCP are all communication/transport protocols. nui's ADL sits at the orchestration layer above all of them.

---

## Open Questions

1. Has Zed ACP's Streamable HTTP/WebSocket transport ([PR #721](https://github.com/agentclientprotocol/agent-client-protocol)) shipped as of mid-2026? If so, does it converge with IBM BeeAI ACP's REST/SSE model enough to unify the two variants?
2. nui's `pi` harness uses TCP JSON-RPC 2.0 — could it be replaced with Zed ACP's stdio JSON-RPC 2.0 to gain ACP compatibility with minimal changes?
3. IBM BeeAI ACP was archived August 27, 2025 — is it superseded by another IBM protocol, or simply abandoned?

---

## Refuted Claims

| Claim | Vote | Source |
|---|---|---|
| ACP uses JSON-RPC 2.0 as its sole transport/format (IBM BeeAI ACP is also REST/HTTP) | 1-2 | agentclientprotocol.com/protocol/schema |
| ACP defines a session model comparable to nui's per-project sessions | 1-2 | agentclientprotocol.com/protocol/schema |
| ACP uses only standard HTTP REST endpoints, no JSON-RPC | 1-2 | agentcommunicationprotocol.dev |
| ACP uses HTTP with incremental streaming, no persistent connection | 0-3 | arxiv.org/html/2505.02279v1 |
| ACP uses a brokered architecture (registry/router between client and agent) | 0-3 | arxiv.org/html/2505.02279v1 |

---

## Sources

Primary: agentclientprotocol.com · agentcommunicationprotocol.dev · github.com/agentclientprotocol · github.com/i-am-bee/acp · arxiv.org/pdf/2505.02279  
Secondary: arxiv.org/html/2505.02279v1/v2 · docker.com/blog/docker-jetbrains-zed-acp · workos.com/guide/understanding-mcp-acp-a2a
