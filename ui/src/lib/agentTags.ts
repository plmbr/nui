// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import type { AgentType } from '@/types'

/** Collect unique tags from agents, sorted alphabetically. */
export function collectAgentTags(agents: AgentType[]): string[] {
  const tags = new Set<string>()
  for (const agent of agents) {
    for (const tag of agent.tags ?? []) {
      const trimmed = tag.trim()
      if (trimmed) tags.add(trimmed)
    }
  }
  return [...tags].sort((a, b) => a.localeCompare(b, undefined, { sensitivity: 'base' }))
}

/** Keep agents that have every selected tag. Empty selection shows all agents. */
export function filterAgentsByTags(
  agents: AgentType[],
  selectedTags: ReadonlySet<string>,
): AgentType[] {
  if (selectedTags.size === 0) return agents
  return agents.filter((agent) => {
    const agentTags = new Set((agent.tags ?? []).map((tag) => tag.trim()).filter(Boolean))
    for (const tag of selectedTags) {
      if (!agentTags.has(tag)) return false
    }
    return true
  })
}

/** Filter tag options by query, excluding already-selected tags. */
export function filterTagSuggestions(
  availableTags: string[],
  selectedTags: ReadonlySet<string>,
  query: string,
): string[] {
  const normalizedQuery = query.trim().toLowerCase()
  return availableTags.filter((tag) => {
    if (selectedTags.has(tag)) return false
    if (!normalizedQuery) return true
    return tag.toLowerCase().includes(normalizedQuery)
  })
}
