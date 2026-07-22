// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { sessionDisplayName } from '@/lib/sessionDisplay'
import type { AgentType, Session } from '@/types'

export function normalizeSearchQuery(query: string): string {
  return query.trim().toLowerCase()
}

export function matchesSearchQuery(text: string, query: string): boolean {
  const normalized = normalizeSearchQuery(query)
  if (!normalized) return true
  return text.toLowerCase().includes(normalized)
}

export function filterBySearchQuery<T>(
  items: T[],
  query: string,
  getSearchText: (item: T) => string,
): T[] {
  const normalized = normalizeSearchQuery(query)
  if (!normalized) return items
  return items.filter((item) => matchesSearchQuery(getSearchText(item), normalized))
}

export interface SearchableListItem {
  id: string
  label: string
  group?: string
  description?: string
  searchText?: string
}

export function searchableItemText(item: SearchableListItem): string {
  return [item.label, item.group, item.description, item.searchText].filter(Boolean).join(' ')
}

export function filterSearchableItems<T extends SearchableListItem>(items: T[], query: string): T[] {
  return filterBySearchQuery(items, query, searchableItemText)
}

export function agentTypeLabel(agentTypes: AgentType[], agentTypeId: string): string {
  return agentTypes.find((agent) => agent.id === agentTypeId)?.label ?? agentTypeId
}

export function filterSessionsByQuery(
  sessions: Session[],
  query: string,
  agentTypes: AgentType[],
): Session[] {
  const normalized = normalizeSearchQuery(query)
  if (!normalized) return sessions

  return sessions.filter((session) => {
    const displayName = sessionDisplayName(session)
    const haystack = [
      session.name,
      displayName,
      session.workingDir,
      session.agentType,
      agentTypeLabel(agentTypes, session.agentType),
      session.scheduleName,
      session.scheduleId,
    ]
      .filter(Boolean)
      .join(' ')
    return matchesSearchQuery(haystack, normalized)
  })
}
