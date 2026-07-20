// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { describe, expect, it } from 'vitest'
import {
  BUILTIN_GROUP_ID,
  BUILTIN_AGENTS_LABEL,
  groupSessionsByAgentType,
  findOrCreateSessionGroup,
  defaultAgentTypeForGroup,
} from '@/lib/sessionGroups'
import type { AgentType, Session } from '@/types'

const agentTypes: AgentType[] = [
  { id: 'claude-code', label: 'Claude Code', harness: 'claude-code', available: true, isBuiltin: true },
  { id: 'my-agent', label: 'My Agent', harness: 'claude-code', available: true, isBuiltin: false, source: 'user' },
]

const sessions: Session[] = [
  { id: 's1', name: 'One', agentType: 'claude-code', workingDir: '/tmp', createdAt: '2026-01-02T00:00:00Z' },
  { id: 's2', name: 'Two', agentType: 'my-agent', workingDir: '/tmp', createdAt: '2026-01-01T00:00:00Z' },
]

describe('groupSessionsByAgentType', () => {
  it('groups builtin and installed sessions separately', () => {
    const groups = groupSessionsByAgentType(sessions, agentTypes)
    expect(groups).toHaveLength(2)
    expect(groups[0].id).toBe(BUILTIN_GROUP_ID)
    expect(groups[0].label).toBe(BUILTIN_AGENTS_LABEL)
    expect(groups[0].sessions.map((s) => s.id)).toEqual(['s1'])
    const installed = groups.find((g) => g.id === 'my-agent')
    expect(installed?.sessions.map((s) => s.id)).toEqual(['s2'])
  })
})

describe('findOrCreateSessionGroup', () => {
  it('creates empty builtin group when missing', () => {
    const group = findOrCreateSessionGroup(BUILTIN_GROUP_ID, [], agentTypes)
    expect(group.id).toBe(BUILTIN_GROUP_ID)
    expect(group.sessions).toEqual([])
  })
})

describe('defaultAgentTypeForGroup', () => {
  it('returns first available builtin for builtin group', () => {
    const group = findOrCreateSessionGroup(BUILTIN_GROUP_ID, sessions, agentTypes)
    expect(defaultAgentTypeForGroup(group, agentTypes)).toBe('claude-code')
  })

  it('returns installed agent id for custom group', () => {
    const groups = groupSessionsByAgentType(sessions, agentTypes)
    const installed = groups.find((g) => g.id === 'my-agent')!
    expect(defaultAgentTypeForGroup(installed, agentTypes)).toBe('my-agent')
  })
})
