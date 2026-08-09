// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { describe, expect, it } from 'vitest'
import {
  agentSupportsHarnessPermissions,
  defaultUserScopeHarnessConfig,
  orderedBuiltinAgentsForPicker,
  partitionBuiltinAgents,
  pickNewSessionAgentTypeId,
  showToolApprovalsOption,
  showUserScopeOption,
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

const nuiAgent: AgentType = {
  id: 'nui',
  label: 'nui',
  harness: 'claude-code',
  available: true,
  isBuiltin: true,
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

  it('hides user-scope and tool approval toggles for the nui orchestrator', () => {
    expect(showUserScopeOption(nuiAgent)).toBe(false)
    expect(showToolApprovalsOption(nuiAgent)).toBe(false)
    expect(defaultUserScopeHarnessConfig(nuiAgent)).toBe(false)
    expect(showUserScopeOption({ ...nuiAgent, id: 'nui-orchestrator' })).toBe(false)
    expect(showUserScopeOption(claudeAgent)).toBe(true)
    expect(defaultUserScopeHarnessConfig(claudeAgent)).toBe(true)
  })

  it('partitions built-in agents into API and CLI groups', () => {
    const agents: AgentType[] = [
      claudeAgent,
      { id: 'ollama', label: 'Ollama', harness: 'api', provider: 'ollama', available: true, isBuiltin: true },
      { id: 'anthropic', label: 'Claude API', harness: 'api', provider: 'anthropic', available: true, isBuiltin: true },
      { id: 'pi', label: 'Pi', harness: 'pi', available: true, isBuiltin: true },
    ]
    const { api, cli } = partitionBuiltinAgents(agents)
    expect(api.map((a) => a.id)).toEqual(['anthropic', 'ollama'])
    expect(cli.map((a) => a.id)).toEqual(['claude-code', 'pi'])
  })

  it('orders built-in picker agents and hides unavailable', () => {
    const agents: AgentType[] = [
      { id: 'pi', label: 'Pi', harness: 'pi', available: false, isBuiltin: true },
      { id: 'anthropic', label: 'Claude API', harness: 'api', provider: 'anthropic', available: true, isBuiltin: true },
      nuiAgent,
      { id: 'openai', label: 'OpenAI', harness: 'api', provider: 'openai', available: false, isBuiltin: true },
      claudeAgent,
    ]
    expect(orderedBuiltinAgentsForPicker(agents).map((a) => a.id)).toEqual([
      'nui',
      'anthropic',
      'claude-code',
    ])
  })

  it('defaults new session agent to nui unless an explicit id is provided', () => {
    const agents: AgentType[] = [
      claudeAgent,
      nuiAgent,
      { id: 'custom', label: 'Custom', harness: 'claude-code', available: true, isBuiltin: false },
    ]
    expect(pickNewSessionAgentTypeId(agents)).toBe('nui')
    expect(pickNewSessionAgentTypeId(agents, 'custom')).toBe('custom')
    expect(pickNewSessionAgentTypeId(agents, 'missing')).toBe('nui')
  })
})
