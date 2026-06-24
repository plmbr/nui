// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import type { AgentType, Session } from '@/types'

export const STANDARD_GROUP_ID = '__standard__'

const LEGACY_AGENT_TYPE_IDS: Record<string, string> = {
  'claude-code': 'claude-code',
  'pi': 'pi',
  'codex': 'codex',
  'opencode': 'opencode',
  'docker-claude': 'claude-code',
  'docker-pi': 'pi',
  'docker-opencode': 'opencode',
  'Claude Code': 'claude-code',
}

const BUILTIN_AGENT_IDS = new Set(['claude-code', 'pi', 'codex', 'opencode'])

export interface SessionGroup {
  id: string
  label: string
  sessions: Session[]
}

function resolveAgentTypeId(agentType: string): string {
  return LEGACY_AGENT_TYPE_IDS[agentType] ?? agentType.replace(/^adl:/, '')
}

function isStandardAgent(agentTypeId: string, agent: AgentType | undefined): boolean {
  if (agent?.isBuiltin) return true
  return BUILTIN_AGENT_IDS.has(agentTypeId)
}

function groupLabel(agentTypeId: string, agent: AgentType | undefined): string {
  if (isStandardAgent(agentTypeId, agent)) return 'Standard'
  return agent?.label ?? agentTypeId
}

function groupId(agentTypeId: string, agent: AgentType | undefined): string {
  if (isStandardAgent(agentTypeId, agent)) return STANDARD_GROUP_ID
  return agentTypeId
}

export function defaultAgentTypeForGroup(
  group: SessionGroup,
  agentTypes: AgentType[],
): string | undefined {
  if (group.id !== STANDARD_GROUP_ID) {
    return agentTypes.some((t) => t.id === group.id) ? group.id : undefined
  }
  if (group.sessions.length > 0) {
    const agentTypeId = resolveAgentTypeId(group.sessions[0].agentType)
    if (agentTypes.some((t) => t.id === agentTypeId)) return agentTypeId
  }
  return agentTypes.find((t) => t.isBuiltin && t.available)?.id
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
      (a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime(),
    )
  }

  result.sort((a, b) => {
    if (a.id === STANDARD_GROUP_ID) return -1
    if (b.id === STANDARD_GROUP_ID) return 1
    return a.label.localeCompare(b.label)
  })

  return result
}
