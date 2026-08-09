// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import type { AgentType, Session } from '@/types'
import { isNuiAgent, NUI_AGENT_ID } from '@/lib/agentTypes'

export const BUILTIN_AGENTS_LABEL = 'Built-in agents'
export const INSTALLED_AGENTS_LABEL = 'Installed agents'

/** Session sidebar group id for the nui master agent. */
export const NUI_GROUP_ID = NUI_AGENT_ID

/** Session sidebar group label for the nui master agent. */
export const NUI_GROUP_LABEL = 'nui'

/** Session sidebar group id for compiled-in harness agents (Claude Code, Pi, …). */
export const BUILTIN_GROUP_ID = '__builtin__'

/** @deprecated Use BUILTIN_GROUP_ID */
export const STANDARD_GROUP_ID = BUILTIN_GROUP_ID

const LEGACY_AGENT_TYPE_IDS: Record<string, string> = {
  'claude-code': 'claude-code',
  'pi': 'pi',
  'codex': 'codex',
  'opencode': 'opencode',
  'docker-claude': 'claude-code',
  'docker-pi': 'pi',
  'docker-opencode': 'opencode',
  'Claude Code': 'claude-code',
  'nui-orchestrator': NUI_AGENT_ID,
}

const BUILTIN_AGENT_IDS = new Set(['claude-code', 'pi', 'codex', 'opencode'])

export interface SessionGroup {
  id: string
  label: string
  sessions: Session[]
}

function sessionLastActivityAt(session: Session): string {
  return session.lastRunAt ?? session.createdAt
}

function resolveAgentTypeId(agentType: string): string {
  return LEGACY_AGENT_TYPE_IDS[agentType] ?? agentType.replace(/^adl:/, '')
}

function isBuiltinAgent(agentTypeId: string, agent: AgentType | undefined): boolean {
  if (isNuiAgent(agentTypeId) || isNuiAgent(agent?.id)) return false
  if (agent?.isBuiltin) return true
  return BUILTIN_AGENT_IDS.has(agentTypeId)
}

function groupLabel(agentTypeId: string, agent: AgentType | undefined): string {
  if (isNuiAgent(agentTypeId) || isNuiAgent(agent?.id)) return NUI_GROUP_LABEL
  if (isBuiltinAgent(agentTypeId, agent)) return BUILTIN_AGENTS_LABEL
  return agent?.label ?? agentTypeId
}

function groupId(agentTypeId: string, agent: AgentType | undefined): string {
  if (isNuiAgent(agentTypeId) || isNuiAgent(agent?.id)) return NUI_GROUP_ID
  if (isBuiltinAgent(agentTypeId, agent)) return BUILTIN_GROUP_ID
  return agentTypeId
}

export function defaultAgentTypeForGroup(
  group: SessionGroup,
  agentTypes: AgentType[],
): string | undefined {
  if (group.id === NUI_GROUP_ID) {
    return agentTypes.some((t) => t.id === NUI_AGENT_ID) ? NUI_AGENT_ID : undefined
  }
  if (group.id !== BUILTIN_GROUP_ID) {
    return agentTypes.some((t) => t.id === group.id) ? group.id : undefined
  }
  if (group.sessions.length > 0) {
    const agentTypeId = resolveAgentTypeId(group.sessions[0].agentType)
    if (agentTypes.some((t) => t.id === agentTypeId)) return agentTypeId
  }
  return agentTypes.find((t) => t.isBuiltin && t.available && !isNuiAgent(t))?.id
}

export function groupSessionsByAgentType(
  sessions: Session[],
  agentTypes: AgentType[],
): SessionGroup[] {
  const typeById = new Map(agentTypes.map((t) => [t.id, t]))
  const groups = new Map<string, SessionGroup>()

  for (const session of sessions) {
    const agentTypeId = resolveAgentTypeId(session.agentType)
    const agent = typeById.get(agentTypeId) ?? typeById.get(session.agentType)
    const id = groupId(agentTypeId, agent)
    const label = groupLabel(agentTypeId, agent)

    let group = groups.get(id)
    if (!group) {
      group = { id, label, sessions: [] }
      groups.set(id, group)
    }
    group.sessions.push(session)
  }

  const result = Array.from(groups.values())

  for (const group of result) {
    group.sessions.sort(
      (a, b) =>
        new Date(sessionLastActivityAt(b)).getTime() - new Date(sessionLastActivityAt(a)).getTime(),
    )
  }

  result.sort((a, b) => {
    if (a.id === NUI_GROUP_ID) return -1
    if (b.id === NUI_GROUP_ID) return 1
    if (a.id === BUILTIN_GROUP_ID) return -1
    if (b.id === BUILTIN_GROUP_ID) return 1
    return a.label.localeCompare(b.label)
  })

  return result
}

export function findOrCreateSessionGroup(
  groupId: string,
  sessions: Session[],
  agentTypes: AgentType[],
): SessionGroup {
  const found = groupSessionsByAgentType(sessions, agentTypes).find((group) => group.id === groupId)
  if (found) return found

  if (groupId === NUI_GROUP_ID) {
    return { id: groupId, label: NUI_GROUP_LABEL, sessions: [] }
  }
  if (groupId === BUILTIN_GROUP_ID) {
    return { id: groupId, label: BUILTIN_AGENTS_LABEL, sessions: [] }
  }

  const agent = agentTypes.find((type) => type.id === groupId)
  return { id: groupId, label: agent?.label ?? groupId, sessions: [] }
}
