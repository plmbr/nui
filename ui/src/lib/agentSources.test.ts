// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { describe, expect, it } from 'vitest'
import {
  buildCustomAgentSourceOptions,
  customAgentSourceKey,
  filterCustomAgentsBySources,
  parseExtensionNameFromAgentId,
} from '@/lib/agentSources'
import type { AgentType } from '@/types'

describe('agentSources', () => {
  it('parses extension name from agent id', () => {
    expect(parseExtensionNameFromAgentId('ext:corp-pack/echo-bot')).toBe('corp-pack')
    expect(parseExtensionNameFromAgentId('local-agent')).toBeNull()
  })

  it('builds source keys for local and extension agents', () => {
    const local: AgentType = {
      id: 'my-agent',
      label: 'Mine',
      harness: 'claude-code',
      available: true,
      isBuiltin: false,
    }
    const ext: AgentType = {
      id: 'ext:corp-pack/echo-bot',
      label: 'Echo',
      harness: 'extension',
      available: true,
      isBuiltin: false,
      source: 'extension',
    }
    expect(customAgentSourceKey(local)).toBe('local')
    expect(customAgentSourceKey(ext)).toBe('ext:corp-pack')
    const options = buildCustomAgentSourceOptions([local, ext], [])
    expect(options.map((o) => o.key)).toEqual(['local', 'ext:corp-pack'])
  })

  it('filters agents by selected source keys', () => {
    const agents: AgentType[] = [
      { id: 'a', label: 'A', harness: 'claude-code', available: true, isBuiltin: false },
      {
        id: 'ext:corp-pack/b',
        label: 'B',
        harness: 'extension',
        available: true,
        isBuiltin: false,
        source: 'extension',
      },
    ]
    const filtered = filterCustomAgentsBySources(agents, new Set(['ext:corp-pack']))
    expect(filtered).toHaveLength(1)
    expect(filtered[0].id).toBe('ext:corp-pack/b')
  })
})
