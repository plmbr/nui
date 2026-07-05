// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { describe, expect, it } from 'vitest'
import {
  agentSupportsHarnessPermissions,
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
})
