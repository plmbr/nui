# Loop vs. Omnigent — Research Report

## Summary

Loop and Omnigent occupy different positions in the stack. Loop is a **thin local wrapper** around Claude Code; Omnigent is a **meta-harness coordination layer** that sits above multiple agent runtimes. They are more complementary than competing.

The Databricks blog post is a **launch announcement for Omnigent** (a separate project), not a Databricks product. Databricks Agent Bricks is a distinct enterprise platform and not a direct competitor to Loop.

## What Omnigent is

Omnigent wraps existing runtimes — Claude Code, Codex, Pi, custom YAML-defined agents — and exposes each session via a uniform interface: messages and files in, text streams and tool calls out. It does not replace the underlying agent; it coordinates above it. (Confirmed 3-0.)

## Feature comparison

| Capability | Loop | Omnigent |
|---|---|---|
| Agent runtimes | Claude Code only | Claude Code, Codex, Pi, custom YAML agents |
| Multi-device access | Local :8080 only | Terminal, browser, phone (LAN or cloud deploy) |
| Session sharing | Single-user, no sharing | Share URL → teammates watch & steer in real time |
| Governance / policies | ADL schema has fields, **silently ignored at runtime** | Pause-for-approval, spend caps, tool-access limits per server/agent/chat |
| Cloud sandboxes | Docker on 127.0.0.1 or manual remote host | Modal & Daytona provisioning, no laptop required |
| UI | Browser SPA (React/Tailwind) | Web + terminal + desktop |

## Verified findings

**Omnigent is a meta-harness (3-0).** Wraps Claude Code, Codex, Pi, and custom YAML agents via a runner/server architecture. Uniform interface across all runtimes. Loop has no concept of pluggable harnesses — it spawns one `claude` CLI process per project.

**Multi-device session continuity (2-1).** Omnigent: start in terminal, continue in browser, pick up on phone (LAN or cloud deploy). Messages, sub-agents, terminals, and files stay in sync. Loop: single-machine local server, no mobile UI, no LAN discovery, no sync.

**Real-time collaboration (2-1).** Omnigent: share a URL, teammates watch and send commands in real time. Loop: single-user, one SSE stream per chat, no session sharing.

**Governance policies (2-1).** Omnigent ships pause-for-approval gates, spend caps, and tool-access limits scoped per server/agent/chat. Loop's `ADLConstraints` struct defines `Policy`, `Approval`, `ApprovalTimeout`, and `MaxTokens` fields but `internal/agent/adl.go` never reads or enforces them — they are parsed from YAML and discarded.

**Cloud sandboxes (3-0).** Omnigent: first-class Modal and Daytona provisioning, no local machine required. Loop: Docker containers bound to `127.0.0.1`, or a manually provisioned remote host — no built-in cloud sandbox integration.

## Refuted claims

| Claim | Vote | Note |
|---|---|---|
| "Omnibox" OS-level sandbox | 1-2 | Not confirmed in GitHub README; may be future/paid |
| Omnigent fans out tasks to parallel agent workers | 0-3 | Routes sessions to one agent at a time |
| File commenting in agent workspaces | unverified | Only in Databricks marketing blog, not in README |

## Key architectural note

Loop already ships `HTTPExtensionAgent` for docker/remote connectors. Builtin CLI harnesses run as Go-managed subprocesses with optional bubblewrap (`sandbox: bubblewrap` in ADL). The gap versus Omnigent is **session sharing** and **governance enforcement** (ADL constraints/approval parsed but not enforced), not fundamental architecture.

## Open questions

1. Is Loop intentionally a thin local wrapper, or is the plan to grow toward session sharing, governance, and cloud sandboxes?
2. Are the ADL policy fields (`Approval`, `MaxTokens`) intentionally deferred, or worth implementing now given Omnigent ships this?
3. Databricks Agent Bricks (enterprise, Unity Catalog, LangChain/LangGraph) is a separate product — not a direct Loop competitor.

## Sources

- https://github.com/omnigent-ai/omnigent
- https://www.databricks.com/blog/introducing-omnigent-meta-harness-combine-control-and-share-your-agents
- https://www.databricks.com/product/artificial-intelligence/agent-bricks
- https://docs.databricks.com/aws/en/generative-ai/agent-framework/multi-agent-apps
