// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { describe, expect, it } from 'vitest'
import {
  defaultAgentForm,
  mergeFormIntoAgentYaml,
  parseAgentYaml,
  syncYamlFromForm,
  type AgentFormOptions,
} from '@/lib/adlAgentForm'

const emptyOptions: AgentFormOptions = {
  harnesses: [
    { id: 'builtin:claude-code', label: 'Claude Code', group: 'Built-in', harnessType: 'claude-code' },
  ],
  skills: [],
  mcpServers: [],
}

describe('adlAgentForm toolApprovals and hitl', () => {
  it('parses toolApprovals and hitl.mode from YAML', () => {
    const yaml = `adl: "1.0"
id: approvals-agent
name: Approvals Agent
harness:
  type: claude-code
toolApprovals:
  policy: denylist
  tools:
    - Bash
    - Write
hitl:
  mode: interactive
`
    const { form } = parseAgentYaml(yaml, emptyOptions)
    expect(form.toolApprovalPolicy).toBe('denylist')
    expect(form.toolApprovalTools).toEqual(['Bash', 'Write'])
    expect(form.hitlMode).toBe('interactive')
  })

  it('round-trips toolApprovals denylist and hitl.mode', () => {
    const original = `adl: "1.0"
id: approvals-agent
name: Approvals Agent
harness:
  type: claude-code
  model: claude-sonnet-4-6
toolApprovals:
  policy: denylist
  tools:
    - Bash
    - Write
hitl:
  mode: interactive
  channels:
    - loop-ui
`
    const { form } = parseAgentYaml(original, emptyOptions)
    const merged = mergeFormIntoAgentYaml(original, form, emptyOptions)
    const { form: reparsed } = parseAgentYaml(merged, emptyOptions)

    expect(reparsed.toolApprovalPolicy).toBe('denylist')
    expect(reparsed.toolApprovalTools).toEqual(['Bash', 'Write'])
    expect(reparsed.hitlMode).toBe('interactive')
    expect(merged).toContain('channels:\n    - loop-ui')
  })

  it('writes toolApprovals all policy without tools', () => {
    const form = {
      ...defaultAgentForm(),
      toolApprovalPolicy: 'all' as const,
      hitlMode: 'off' as const,
    }
    const merged = mergeFormIntoAgentYaml(
      `adl: "1.0"\nid: my-agent\nname: My Agent\nharness:\n  type: claude-code\n`,
      form,
      emptyOptions,
    )
    expect(merged).toContain('policy: all')
    expect(merged).not.toContain('tools:')
    expect(merged).toContain('mode: off')
  })

  it('removes toolApprovals and hitl.mode when cleared in form', () => {
    const original = `adl: "1.0"
id: approvals-agent
name: Approvals Agent
harness:
  type: claude-code
toolApprovals:
  policy: allowlist
  tools:
    - Read
hitl:
  mode: auto
  ttlSeconds: 300
`
    const { form } = parseAgentYaml(original, emptyOptions)
    const cleared = {
      ...form,
      toolApprovalPolicy: '' as const,
      toolApprovalTools: [],
      hitlMode: '' as const,
    }
    const merged = mergeFormIntoAgentYaml(original, cleared, emptyOptions)

    expect(merged).not.toContain('toolApprovals')
    expect(merged).not.toContain('mode:')
    expect(merged).toContain('ttlSeconds: 300')
  })

  it('writes allowlist with tools from form edits', () => {
    const form = {
      ...defaultAgentForm(),
      toolApprovalPolicy: 'allowlist' as const,
      toolApprovalTools: ['Read', 'Grep', 'Glob'],
    }
    const merged = mergeFormIntoAgentYaml(
      `adl: "1.0"\nid: my-agent\nname: My Agent\nharness:\n  type: claude-code\n`,
      form,
      emptyOptions,
    )
    const { form: reparsed } = parseAgentYaml(merged, emptyOptions)
    expect(reparsed.toolApprovalPolicy).toBe('allowlist')
    expect(reparsed.toolApprovalTools).toEqual(['Read', 'Grep', 'Glob'])
  })

  it('syncYamlFromForm merges form edits while preserving yaml-only fields', () => {
    const original = `adl: "1.0"
id: my-agent
name: My Agent
harness:
  type: claude-code
  sandbox: bubblewrap
steps:
  - name: review
    type: agent
`
    const form = {
      ...defaultAgentForm(),
      id: 'my-agent',
      name: 'Renamed Agent',
      description: 'Updated from form',
    }
    const synced = syncYamlFromForm(original, form, emptyOptions)
    expect(synced).toContain('name: Renamed Agent')
    expect(synced).toContain('description: Updated from form')
    expect(synced).toContain('sandbox: bubblewrap')
    expect(synced).toContain('steps:')
  })

  it('parseAgentYaml reports parseError for invalid YAML', () => {
    const result = parseAgentYaml('id: [unclosed', emptyOptions)
    expect(result.parseError).toBe(true)
    expect(result.form).toEqual(defaultAgentForm())
  })
})
