// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { detectMentionTrigger, type MentionTrigger } from '@/hooks/useMentionMenu'
import { isNuiAgent, selectableAgentTypes } from '@/lib/agentTypes'
import type { AgentType, MentionBreadcrumb, MentionItem } from '@/types'

export const LAUNCHER_AGENTS_MENTION_ROOT = 'builtin:agents'
const MAX_LAUNCHER_MENTION_ITEMS = 20

export function launchableAgentsForMention(types: AgentType[]): AgentType[] {
  return selectableAgentTypes(types).filter((agent) => !isNuiAgent(agent))
}

function agentMatchesMentionQuery(agent: AgentType, query: string): boolean {
  const haystack = [
    agent.label,
    agent.id,
    agent.description ?? '',
  ].join(' ').toLowerCase()
  return haystack.includes(query)
}

export function formatLauncherAgentMentionToken(agentId: string, label: string): string {
  const safeLabel = label.replace(/]/g, '')
  return `@${agentId}:[${safeLabel}]`
}

/** True when the prompt already has a selected launcher agent mention at the start. */
export function hasCompleteLauncherAgentMention(prompt: string): boolean {
  const trimmed = prompt.trimStart()
  if (!trimmed.startsWith('@')) return false
  const rest = trimmed.slice(1)
  if (rest.includes(':[')) {
    return rest.includes(']')
  }
  return /\s/.test(rest)
}

export function detectLauncherMentionTrigger(value: string, cursor: number): MentionTrigger | null {
  if (hasCompleteLauncherAgentMention(value)) return null
  const trigger = detectMentionTrigger(value, cursor)
  if (!trigger) return null
  if (trigger.query.includes(':[')) return null
  return trigger
}

export function listLauncherMentionItems(
  agents: AgentType[],
  query: string,
): { items: MentionItem[]; breadcrumb: MentionBreadcrumb[] } {
  const launchable = launchableAgentsForMention(agents)
  const normalizedQuery = query.trim().toLowerCase()
  const items = launchable
    .filter((agent) => !normalizedQuery || agentMatchesMentionQuery(agent, normalizedQuery))
    .sort((a, b) => a.label.localeCompare(b.label, undefined, { sensitivity: 'base' }))
    .slice(0, MAX_LAUNCHER_MENTION_ITEMS)
    .map((agent) => ({
      label: agent.label,
      value: agent.id,
      hasChildren: false,
      icon: 'agent',
    }))
  return {
    items,
    breadcrumb: [{ label: 'Agents', parent: LAUNCHER_AGENTS_MENTION_ROOT }],
  }
}

export function parseLauncherAgentMention(prompt: string): { agentId: string; delegated: string } | null {
  const trimmed = prompt.trimStart()
  if (!trimmed.startsWith('@')) return null
  const rest = trimmed.slice(1)
  if (!rest) return null

  const labelIdx = rest.indexOf(':[')
  if (labelIdx >= 0) {
    const closeIdx = rest.indexOf(']', labelIdx)
    if (closeIdx < 0) return null
    const agentId = rest.slice(0, labelIdx)
    if (!agentId) return null
    const after = rest.slice(closeIdx + 1)
    if (after.length > 0 && !after.startsWith(' ')) return null
    return { agentId, delegated: after.trimStart() }
  }

  const spaceIndex = rest.search(/\s/)
  if (spaceIndex < 0) {
    const agentId = rest
    if (!agentId) return null
    return { agentId, delegated: '' }
  }
  const agentId = rest.slice(0, spaceIndex)
  if (!agentId) return null
  return { agentId, delegated: rest.slice(spaceIndex + 1).trim() }
}

export function isLauncherAgentOnlyMention(prompt: string): boolean {
  const mention = parseLauncherAgentMention(prompt)
  return mention !== null && mention.delegated === ''
}
