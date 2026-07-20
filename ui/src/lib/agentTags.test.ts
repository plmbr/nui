// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { describe, expect, it } from 'vitest'
import {
  collectAgentTags,
  filterAgentsByTags,
  filterTagSuggestions,
} from '@/lib/agentTags'
import type { AgentType } from '@/types'

function agent(id: string, tags?: string[]): AgentType {
  return {
    id,
    label: id,
    harness: 'claude-code',
    available: true,
    isBuiltin: false,
    tags,
  }
}

describe('agentTags', () => {
  it('collects unique sorted tags', () => {
    const tags = collectAgentTags([
      agent('a', ['research', 'coding']),
      agent('b', ['coding', 'local']),
    ])
    expect(tags).toEqual(['coding', 'local', 'research'])
  })

  it('filters agents by all selected tags', () => {
    const agents = [
      agent('a', ['coding', 'research']),
      agent('b', ['coding']),
      agent('c', ['research']),
    ]
    const filtered = filterAgentsByTags(agents, new Set(['coding', 'research']))
    expect(filtered.map((item) => item.id)).toEqual(['a'])
  })

  it('filters tag suggestions by query and selection', () => {
    const suggestions = filterTagSuggestions(
      ['coding', 'local', 'research'],
      new Set(['coding']),
      're',
    )
    expect(suggestions).toEqual(['research'])
  })
})
