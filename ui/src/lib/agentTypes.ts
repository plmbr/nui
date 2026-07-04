// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import type { AgentType } from '@/types'

/** Agent types that can be selected for a new session or as the default agent. */
export function selectableAgentTypes(types: AgentType[]): AgentType[] {
  return types.filter((t) => t.available)
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

/** Default user-scope checkbox state for a newly selected agent type. */
export function defaultUserScopeHarnessConfig(agent: AgentType | undefined): boolean {
  if (!agent || !harnessSupportsUserScope(agent.harness)) return false
  return agent.isBuiltin
}
