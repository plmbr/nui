// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { describe, expect, it } from 'vitest'
import {
  applyRecentAgentToForm,
  buildCreateRequestFromRecent,
  resolveRecentAgents,
  resolveRecentSessions,
  touchRecentSessionIds,
} from '@/lib/recents'
import type { AgentType, RecentAgentEntry, Session } from '@/types'

const claudeAgent: AgentType = {
  id: 'claude-code',
  label: 'Claude Code',
  harness: 'claude-code',
  available: true,
  isBuiltin: true,
}

const session: Session = {
  id: 's1',
  name: 'My session',
  workingDir: '/Users/me/project',
  agentType: 'claude-code',
  createdAt: '2026-01-01T00:00:00Z',
}

describe('recents', () => {
  it('buildCreateRequestFromRecent maps config fields', () => {
    const entry: RecentAgentEntry = {
      agentType: 'claude-code',
      workingDir: '/tmp/work',
      agentConfig: {
        harnessType: 'pi',
        userScopeHarnessConfig: true,
        hitlMode: 'off',
        harnessPermissions: 'bypass',
      },
    }
    expect(buildCreateRequestFromRecent(entry)).toEqual({
      agentType: 'claude-code',
      workingDir: '/tmp/work',
      agentConfig: {
        harnessType: 'pi',
        userScopeHarnessConfig: true,
        hitlMode: 'off',
        harnessPermissions: 'bypass',
      },
    })
  })

  it('applyRecentAgentToForm restores harness and permission flags', () => {
    const entry: RecentAgentEntry = {
      agentType: 'claude-code',
      workingDir: '/tmp/work',
      agentConfig: {
        harnessType: 'pi',
        userScopeHarnessConfig: true,
        hitlMode: 'off',
        harnessPermissions: 'bypass',
      },
    }
    const agent: AgentType = {
      ...claudeAgent,
      allowedHarnesses: ['claude-code', 'codex'],
    }
    const entryWithCodex: RecentAgentEntry = {
      ...entry,
      agentConfig: {
        ...entry.agentConfig,
        harnessType: 'codex',
      },
    }
    expect(applyRecentAgentToForm(entryWithCodex, [agent])).toEqual({
      selectedId: 'claude-code',
      workingDir: '/tmp/work',
      harnessOverride: 'codex',
      userScopeHarnessConfig: true,
      harnessPermissionsEnabled: false,
    })
  })

  it('touchRecentSessionIds dedupes and moves to front', () => {
    expect(touchRecentSessionIds(['a', 'b', 'c'], 'b')).toEqual(['b', 'a', 'c'])
  })

  it('resolveRecentSessions drops missing sessions', () => {
    const resolved = resolveRecentSessions(['s1', 'missing'], [session], [claudeAgent])
    expect(resolved).toHaveLength(1)
    expect(resolved[0]?.id).toBe('s1')
  })

  it('resolveRecentAgents drops unavailable agents', () => {
    const resolved = resolveRecentAgents(
      [{ agentType: 'missing' }, { agentType: 'claude-code' }],
      [claudeAgent],
    )
    expect(resolved).toHaveLength(1)
    expect(resolved[0]?.agentType.id).toBe('claude-code')
  })
})
