// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import type { AgentType } from '@/types'

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
  return agent.isBuiltin && agent.harness === 'api'
}

/** Built-in agents that shell out to a CLI harness. */
export function isCliBuiltinAgent(agent: AgentType): boolean {
  return agent.isBuiltin && agent.harness !== 'api'
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
  const builtin = selectable.find((t) => t.isBuiltin)
  return builtin?.id ?? selectable[0]?.id ?? ''
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

/** Whether the new-session UI should offer the tool-approvals toggle. */
export function showToolApprovalsOption(agent: AgentType | undefined): boolean {
  if (!agentSupportsHarnessPermissions(agent)) return false
  // ADL toolApprovals.policy: all auto-approves every tool; no session override needed.
  if (agent?.toolApprovalPolicy === 'all') return false
  return true
}

/** Default user-scope checkbox state for a newly selected agent type. */
export function defaultUserScopeHarnessConfig(agent: AgentType | undefined): boolean {
  if (!agent || !harnessSupportsUserScope(agent.harness)) return false
  return agent.isBuiltin
}
