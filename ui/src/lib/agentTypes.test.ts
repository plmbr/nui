// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { describe, expect, it } from 'vitest'
import {
  agentSupportsHarnessPermissions,
  partitionBuiltinAgents,
  showToolApprovalsOption,
  selectableAgentTypes,
} from '@/lib/agentTypes'
import type { AgentType } from '@/types'

const claudeAgent: AgentType = {
  id: 'claude-code',
  label: 'Claude Code',
  harness: 'claude-code',
  available: true,
  isBuiltin: true,
}

const autoApproveAgent: AgentType = {
  ...claudeAgent,
  id: 'auto-agent',
  toolApprovalPolicy: 'all',
}

describe('agentTypes', () => {
  it('filters selectable agents by availability', () => {
    const agents: AgentType[] = [
      claudeAgent,
      { ...claudeAgent, id: 'missing', available: false },
    ]
    expect(selectableAgentTypes(agents)).toHaveLength(1)
  })

  it('detects harness permission support', () => {
    expect(agentSupportsHarnessPermissions(claudeAgent)).toBe(true)
    expect(agentSupportsHarnessPermissions({ ...claudeAgent, harness: 'pi' })).toBe(false)
  })

  it('hides tool approval toggle when policy is all', () => {
    expect(showToolApprovalsOption(claudeAgent)).toBe(true)
    expect(showToolApprovalsOption(autoApproveAgent)).toBe(false)
    expect(showToolApprovalsOption(undefined)).toBe(false)
  })

  it('partitions built-in agents into API and CLI groups', () => {
    const agents: AgentType[] = [
      claudeAgent,
      { id: 'ollama', label: 'Ollama', harness: 'api', provider: 'ollama', available: true, isBuiltin: true },
      { id: 'anthropic', label: 'Anthropic', harness: 'api', provider: 'anthropic', available: true, isBuiltin: true },
      { id: 'pi', label: 'Pi', harness: 'pi', available: true, isBuiltin: true },
    ]
    const { api, cli } = partitionBuiltinAgents(agents)
    expect(api.map((a) => a.id)).toEqual(['anthropic', 'ollama'])
    expect(cli.map((a) => a.id)).toEqual(['claude-code', 'pi'])
  })
})
