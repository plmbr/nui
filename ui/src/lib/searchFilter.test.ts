// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { describe, expect, it } from 'vitest'
import {
  filterBySearchQuery,
  filterSearchableItems,
  filterSessionsByQuery,
  matchesSearchQuery,
} from '@/lib/searchFilter'
import type { AgentType, Session } from '@/types'

const agentTypes: AgentType[] = [
  {
    id: 'claude-code',
    label: 'Claude Code',
    harness: 'claude-code',
    isBuiltin: true,
  },
]

const sessions: Session[] = [
  {
    id: '1',
    name: 'Refactor auth',
    workingDir: '/projects/app',
    agentType: 'claude-code',
    createdAt: '2026-01-01T00:00:00Z',
  },
  {
    id: '2',
    name: 'Daily report',
    workingDir: '/tmp',
    agentType: 'claude-code',
    scheduleId: 'sched-1',
    scheduleName: 'Morning digest',
    createdAt: '2026-01-02T00:00:00Z',
  },
]

describe('searchFilter', () => {
  it('matches case-insensitive substrings', () => {
    expect(matchesSearchQuery('Claude Code', 'code')).toBe(true)
    expect(matchesSearchQuery('Claude Code', 'gemini')).toBe(false)
  })

  it('filters generic items by search text', () => {
    const items = [
      { id: 'a', label: 'Alpha' },
      { id: 'b', label: 'Beta' },
    ]
    expect(filterBySearchQuery(items, 'alp', (item) => item.label)).toEqual([items[0]])
  })

  it('filters searchable list items across label, group, and description', () => {
    const items = [
      { id: 'a', label: 'Server A', group: 'Extensions', description: 'stdio transport' },
      { id: 'b', label: 'Server B', group: 'User', description: 'http transport' },
    ]
    expect(filterSearchableItems(items, 'stdio')).toEqual([items[0]])
    expect(filterSearchableItems(items, 'extensions')).toEqual([items[0]])
  })

  it('filters sessions by name, working dir, agent label, and schedule', () => {
    expect(filterSessionsByQuery(sessions, 'refactor', agentTypes).map((s) => s.id)).toEqual(['1'])
    expect(filterSessionsByQuery(sessions, '/projects/app', agentTypes).map((s) => s.id)).toEqual(['1'])
    expect(filterSessionsByQuery(sessions, 'claude code', agentTypes).map((s) => s.id)).toEqual(['1', '2'])
    expect(filterSessionsByQuery(sessions, 'morning', agentTypes).map((s) => s.id)).toEqual(['2'])
  })
})
