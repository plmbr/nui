// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { describe, expect, it } from 'vitest'
import {
  BUILTIN_GROUP_ID,
  BUILTIN_AGENTS_LABEL,
  NUI_GROUP_ID,
  NUI_GROUP_LABEL,
  groupSessionsByAgentType,
  findOrCreateSessionGroup,
  defaultAgentTypeForGroup,
} from '@/lib/sessionGroups'
import type { AgentType, Session } from '@/types'

const agentTypes: AgentType[] = [
  { id: 'nui', label: 'nui', harness: 'api', provider: 'anthropic', available: true, isBuiltin: true },
  { id: 'claude-code', label: 'Claude Code', harness: 'claude-code', available: true, isBuiltin: true },
  { id: 'my-agent', label: 'My Agent', harness: 'claude-code', available: true, isBuiltin: false, source: 'user' },
]

const sessions: Session[] = [
  { id: 's0', name: 'Nui', agentType: 'nui', workingDir: '/tmp', createdAt: '2026-01-03T00:00:00Z' },
  { id: 's1', name: 'One', agentType: 'claude-code', workingDir: '/tmp', createdAt: '2026-01-02T00:00:00Z' },
  { id: 's2', name: 'Two', agentType: 'my-agent', workingDir: '/tmp', createdAt: '2026-01-01T00:00:00Z' },
]

describe('groupSessionsByAgentType', () => {
  it('groups nui, builtin, and installed sessions separately', () => {
    const groups = groupSessionsByAgentType(sessions, agentTypes)
    expect(groups).toHaveLength(3)
    expect(groups[0].id).toBe(NUI_GROUP_ID)
    expect(groups[0].label).toBe(NUI_GROUP_LABEL)
    expect(groups[0].sessions.map((s) => s.id)).toEqual(['s0'])
    expect(groups[1].id).toBe(BUILTIN_GROUP_ID)
    expect(groups[1].label).toBe(BUILTIN_AGENTS_LABEL)
    expect(groups[1].sessions.map((s) => s.id)).toEqual(['s1'])
    const installed = groups.find((g) => g.id === 'my-agent')
    expect(installed?.sessions.map((s) => s.id)).toEqual(['s2'])
  })

  it('maps legacy nui-orchestrator sessions to the nui group', () => {
    const legacy: Session[] = [
      { id: 'legacy', name: 'Legacy', agentType: 'nui-orchestrator', workingDir: '/tmp', createdAt: '2026-01-01T00:00:00Z' },
    ]
    const groups = groupSessionsByAgentType(legacy, agentTypes)
    expect(groups).toHaveLength(1)
    expect(groups[0].id).toBe(NUI_GROUP_ID)
    expect(groups[0].label).toBe(NUI_GROUP_LABEL)
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
  it('returns nui for nui group', () => {
    const group = findOrCreateSessionGroup(NUI_GROUP_ID, sessions, agentTypes)
    expect(defaultAgentTypeForGroup(group, agentTypes)).toBe('nui')
  })

  it('returns first available non-nui builtin for builtin group', () => {
    const group = findOrCreateSessionGroup(BUILTIN_GROUP_ID, sessions, agentTypes)
    expect(defaultAgentTypeForGroup(group, agentTypes)).toBe('claude-code')
  })

  it('returns installed agent id for custom group', () => {
    const groups = groupSessionsByAgentType(sessions, agentTypes)
    const installed = groups.find((g) => g.id === 'my-agent')!
    expect(defaultAgentTypeForGroup(installed, agentTypes)).toBe('my-agent')
  })
})
