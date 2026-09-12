// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import type { MentionTrigger } from '@/hooks/useMentionMenu'
import { isNuiAgent, selectableAgentTypes } from '@/lib/agentTypes'
import { fuzzyMatchScore } from '@/lib/fuzzyMatch'
import type { AgentType, MentionBreadcrumb, MentionItem } from '@/types'

export const LAUNCHER_AGENTS_MENTION_ROOT = 'builtin:agents'
const MAX_LAUNCHER_MENTION_ITEMS = 20

export function launchableAgentsForMention(types: AgentType[]): AgentType[] {
  return selectableAgentTypes(types).filter((agent) => !isNuiAgent(agent))
}

function agentMentionHaystack(agent: AgentType): string {
  return [agent.label, agent.id, agent.description ?? ''].join(' ')
}

function agentMentionScore(agent: AgentType, query: string): number {
  const labelScore = fuzzyMatchScore(query, agent.label)
  const idScore = fuzzyMatchScore(query, agent.id)
  const fullScore = fuzzyMatchScore(query, agentMentionHaystack(agent))
  // Prefer label/id hits over description-only matches.
  return Math.max(labelScore * 1.25, idScore * 1.1, fullScore)
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
  // Only a selected `@id:[label]` token closes autocomplete. Spaces may appear in agent names
  // while the user is still filtering suggestions.
  if (!rest.includes(':[')) return false
  return rest.includes(']')
}

export function detectLauncherMentionTrigger(value: string, cursor: number): MentionTrigger | null {
  if (hasCompleteLauncherAgentMention(value)) return null
  const before = value.slice(0, cursor)
  const at = before.lastIndexOf('@')
  if (at < 0) return null
  if (at > 0 && !/\s/.test(before[at - 1] ?? '')) return null
  const query = before.slice(at + 1)
  // Allow spaces in the query so multi-word agent labels keep filtering; newlines end the trigger.
  if (query.includes('\n')) return null
  if (query.includes(':[')) return null
  return { triggerStart: at, query }
}

export function listLauncherMentionItems(
  agents: AgentType[],
  query: string,
): { items: MentionItem[]; breadcrumb: MentionBreadcrumb[] } {
  const launchable = launchableAgentsForMention(agents)
  const normalizedQuery = query.trim().toLowerCase()
  const ranked = launchable
    .map((agent) => ({
      agent,
      score: normalizedQuery ? agentMentionScore(agent, normalizedQuery) : 1,
    }))
    .filter((entry) => entry.score > 0)
    .sort((a, b) => {
      if (b.score !== a.score) return b.score - a.score
      return a.agent.label.localeCompare(b.agent.label, undefined, { sensitivity: 'base' })
    })
    .slice(0, MAX_LAUNCHER_MENTION_ITEMS)

  const items = ranked.map(({ agent }) => ({
    label: agent.label,
    value: agent.id,
    hasChildren: false,
    icon: 'agent' as const,
  }))
  return {
    items,
    breadcrumb: [{ label: 'Agents', parent: LAUNCHER_AGENTS_MENTION_ROOT }],
  }
}

export function parseLauncherAgentMention(
  prompt: string,
  agents: AgentType[] = [],
): { agentId: string; delegated: string } | null {
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

  const launchable = launchableAgentsForMention(agents)
  if (launchable.length > 0) {
    const matched = matchLeadingLauncherAgent(rest, launchable)
    if (matched) return matched
    return null
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

function matchLeadingLauncherAgent(
  rest: string,
  agents: AgentType[],
): { agentId: string; delegated: string } | null {
  const restLower = rest.toLowerCase()
  let bestLen = -1
  let bestId = ''
  for (const agent of agents) {
    for (const key of [agent.id, agent.label].filter(Boolean)) {
      const keyLower = key.toLowerCase()
      if (!restLower.startsWith(keyLower)) continue
      if (rest.length > key.length && !/\s/.test(rest[key.length]!)) continue
      if (key.length < bestLen) continue
      if (key.length === bestLen && bestId) continue
      bestLen = key.length
      bestId = agent.id
    }
  }
  if (bestLen < 0) return null
  return { agentId: bestId, delegated: rest.slice(bestLen).trim() }
}

export function isLauncherAgentOnlyMention(prompt: string, agents: AgentType[] = []): boolean {
  const mention = parseLauncherAgentMention(prompt, agents)
  return mention !== null && mention.delegated === ''
}
