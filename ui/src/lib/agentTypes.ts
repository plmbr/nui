// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import type { AgentType } from '@/types'

export const NUI_AGENT_ID = 'nui'

/** The nui master agent (launcher orchestrator). */
export function isNuiAgent(agentOrId: AgentType | string | undefined | null): boolean {
  if (!agentOrId) return false
  const id = typeof agentOrId === 'string' ? agentOrId : agentOrId.id
  return id === NUI_AGENT_ID || id === 'nui-orchestrator'
}

/** Agent types that can be selected for a new session or as the default agent. */
export function selectableAgentTypes(types: AgentType[]): AgentType[] {
  return types.filter((t) => t.available)
}

const API_BUILTIN_ORDER = ['anthropic', 'openai', 'gemini', 'openrouter', 'ollama'] as const
const CLI_BUILTIN_ORDER = ['claude-code', 'pi', 'codex', 'opencode'] as const

function sortAgentsByIdOrder(agents: AgentType[], order: readonly string[]): AgentType[] {
  const rank = new Map(order.map((id, index) => [id, index]))
  return [...agents].sort(
    (a, b) => (rank.get(a.id) ?? Number.MAX_SAFE_INTEGER) - (rank.get(b.id) ?? Number.MAX_SAFE_INTEGER),
  )
}

/** Built-in agents that use the in-process API harness (internal/llm HTTP clients). */
export function isApiBuiltinAgent(agent: AgentType): boolean {
  return agent.isBuiltin && agent.harness === 'api' && !isNuiAgent(agent)
}

/** Built-in agents that shell out to a CLI harness. */
export function isCliBuiltinAgent(agent: AgentType): boolean {
  return agent.isBuiltin && agent.harness !== 'api' && !isNuiAgent(agent)
}

export function partitionBuiltinAgents(builtins: AgentType[]): {
  api: AgentType[]
  cli: AgentType[]
} {
  const api: AgentType[] = []
  const cli: AgentType[] = []
  for (const agent of builtins) {
    if (isApiBuiltinAgent(agent)) api.push(agent)
    else if (isCliBuiltinAgent(agent)) cli.push(agent)
  }
  return {
    api: sortAgentsByIdOrder(api, API_BUILTIN_ORDER),
    cli: sortAgentsByIdOrder(cli, CLI_BUILTIN_ORDER),
  }
}

/**
 * Built-in agents for the new-session picker: nui, then API, then CLI.
 * Unavailable agents are omitted.
 */
export function orderedBuiltinAgentsForPicker(types: AgentType[]): AgentType[] {
  const all = types.filter((a) => a.isBuiltin && a.available)
  const nui = all.find((a) => isNuiAgent(a))
  const { api, cli } = partitionBuiltinAgents(all.filter((a) => !isNuiAgent(a)))
  return [...(nui ? [nui] : []), ...api, ...cli]
}

export function pickDefaultAgentTypeId(
  types: AgentType[],
  preferredId?: string | null,
): string {
  const selectable = selectableAgentTypes(types)
  if (preferredId) {
    const preferred = selectable.find((t) => t.id === preferredId)
    if (preferred) return preferred.id
  }
  const apiOrder = ['anthropic', 'openai', 'gemini', 'openrouter', 'ollama']
  for (const id of apiOrder) {
    const match = selectable.find((t) => t.id === id)
    if (match) return match.id
  }
  const builtin = selectable.find((t) => t.isBuiltin && !isNuiAgent(t))
  return builtin?.id ?? selectable[0]?.id ?? ''
}

/** Default agent for the new-session UI: explicit id, else nui, else first available. */
export function pickNewSessionAgentTypeId(
  types: AgentType[],
  initialId?: string | null,
): string {
  const selectable = selectableAgentTypes(types)
  if (initialId) {
    const match = selectable.find((t) => t.id === initialId)
    if (match) return match.id
  }
  const nui = selectable.find((t) => isNuiAgent(t))
  return nui?.id ?? pickDefaultAgentTypeId(types)
}

/** Harnesses that can load user/project settings via native CLI flags. */
export function harnessSupportsUserScope(harness: AgentType['harness']): boolean {
  return harness === 'claude-code' || harness === 'codex'
}

/** Harnesses that support interactive tool approval via harness permissions. */
export function harnessSupportsPermissions(harness: AgentType['harness']): boolean {
  return harness === 'claude-code' || harness === 'codex'
}

export function agentSupportsHarnessPermissions(agent: AgentType | undefined): boolean {
  if (!agent) return false
  if (agent.supportsHarnessPermissions != null) return agent.supportsHarnessPermissions
  return harnessSupportsPermissions(agent.harness)
}

/** Whether the new-session UI should offer the user-scope harness config toggle. */
export function showUserScopeOption(agent: AgentType | undefined): boolean {
  // nui is a launcher; session options do not apply to agents it launches.
  if (!agent || isNuiAgent(agent)) return false
  return harnessSupportsUserScope(agent.harness)
}

/** Whether the new-session UI should offer the tool-approvals toggle. */
export function showToolApprovalsOption(agent: AgentType | undefined): boolean {
  // nui is a launcher; session options do not apply to agents it launches.
  if (!agent || isNuiAgent(agent)) return false
  if (!agentSupportsHarnessPermissions(agent)) return false
  // ADL toolApprovals.policy: all auto-approves every tool; no session override needed.
  if (agent.toolApprovalPolicy === 'all') return false
  return true
}

/** Default user-scope checkbox state for a newly selected agent type. */
export function defaultUserScopeHarnessConfig(agent: AgentType | undefined): boolean {
  if (!agent || !showUserScopeOption(agent)) return false
  return agent.isBuiltin
}

export interface HarnessRefOption {
  ref: string
  label: string
  group: 'API' | 'CLI'
}

/** Build selectable default harness refs from available agent types. */
export function selectableHarnessRefs(types: AgentType[]): HarnessRefOption[] {
  const out: HarnessRefOption[] = []
  const { api, cli } = partitionBuiltinAgents(types.filter((t) => t.available))
  for (const agent of api) {
    const provider = agent.provider?.trim() || agent.id
    out.push({ ref: `api/${provider}`, label: agent.label, group: 'API' })
  }
  for (const agent of cli) {
    out.push({ ref: agent.harness, label: agent.label, group: 'CLI' })
  }
  return out
}

export function pickDefaultHarnessRef(
  types: AgentType[],
  preferredRef?: string | null,
): string {
  const selectable = selectableHarnessRefs(types)
  if (preferredRef) {
    const preferred = selectable.find((h) => h.ref === preferredRef)
    if (preferred) return preferred.ref
  }
  return selectable[0]?.ref ?? ''
}
