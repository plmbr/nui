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
  agents: [
    { id: 'hello-world', label: 'Hello World', description: 'Greeter', group: 'Installed' },
    { id: 'code-reviewer', label: 'Code Reviewer', description: 'Reviews code', group: 'Installed' },
  ],
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

  it('inlines extension MCP server config instead of writing a ref', () => {
    const options: AgentFormOptions = {
      ...emptyOptions,
      mcpServers: [
        {
          id: 'ext-mcp:ext:corp-pack/docs',
          label: 'docs (Corp Pack)',
          group: 'Extension MCP servers',
          name: 'docs',
          server: {
            name: 'docs',
            url: 'http://localhost:3040/mcp',
            type: 'http',
          },
        },
      ],
    }
    const form = {
      ...defaultAgentForm(),
      mcpServers: [{ optionId: 'ext-mcp:ext:corp-pack/docs', name: 'docs' }],
    }
    const merged = mergeFormIntoAgentYaml(
      `adl: "1.0"\nid: my-agent\nname: My Agent\nharness:\n  type: claude-code\n`,
      form,
      options,
    )
    expect(merged).toContain('url: http://localhost:3040/mcp')
    expect(merged).toContain('type: http')
    expect(merged).not.toContain('ref:')
  })

  it('writes ref for custom extension MCP servers without inline config', () => {
    const options: AgentFormOptions = {
      ...emptyOptions,
      mcpServers: [
        {
          id: 'ext-mcp:ext:corp-pack/corp-tools',
          label: 'corp-tools (Corp Pack)',
          group: 'Extension MCP servers',
          name: 'corp-tools',
          ref: 'ext:corp-pack/corp-tools',
        },
      ],
    }
    const form = {
      ...defaultAgentForm(),
      mcpServers: [{ optionId: 'ext-mcp:ext:corp-pack/corp-tools', name: 'corp-tools' }],
    }
    const merged = mergeFormIntoAgentYaml(
      `adl: "1.0"\nid: my-agent\nname: My Agent\nharness:\n  type: claude-code\n`,
      form,
      options,
    )
    expect(merged).toContain('ref: ext:corp-pack/corp-tools')
  })
})

describe('adlAgentForm evals', () => {
  it('parses single-turn eval with expect', () => {
    const yaml = `adl: "1.0"
id: eval-agent
name: Eval Agent
harness:
  type: claude-code
evals:
  - name: polite-greeting
    description: Greets politely
    input: Hello
    expect:
      type: contains
      value: assistant
    tags: [smoke, greeting]
    timeout: 60
`
    const { form } = parseAgentYaml(yaml, emptyOptions)
    expect(form.evals).toHaveLength(1)
    expect(form.evals[0].name).toBe('polite-greeting')
    expect(form.evals[0].inputMode).toBe('single')
    expect(form.evals[0].input).toBe('Hello')
    expect(form.evals[0].expectType).toBe('contains')
    expect(form.evals[0].expectValue).toBe('assistant')
    expect(form.evals[0].tags).toBe('smoke, greeting')
    expect(form.evals[0].timeout).toBe('60')
  })

  it('parses multi-turn eval messages', () => {
    const yaml = `adl: "1.0"
id: eval-agent
name: Eval Agent
harness:
  type: claude-code
evals:
  - name: follow-up
    messages:
      - role: user
        content: What is 2+2?
      - role: assistant
        content: "4"
      - role: user
        content: Are you sure?
    expect:
      type: contains
      value: "4"
`
    const { form } = parseAgentYaml(yaml, emptyOptions)
    expect(form.evals[0].inputMode).toBe('conversation')
    expect(form.evals[0].messages).toHaveLength(3)
    expect(form.evals[0].messages[2].role).toBe('user')
  })

  it('round-trips evals', () => {
    const original = `adl: "1.0"
id: eval-agent
name: Eval Agent
harness:
  type: claude-code
evals:
  - name: smoke
    input: hi
    expect:
      type: exact
      value: hello
`
    const { form } = parseAgentYaml(original, emptyOptions)
    const merged = mergeFormIntoAgentYaml(original, form, emptyOptions)
    const { form: reparsed } = parseAgentYaml(merged, emptyOptions)
    expect(reparsed.evals).toHaveLength(1)
    expect(reparsed.evals[0].name).toBe('smoke')
    expect(reparsed.evals[0].expectType).toBe('exact')
    expect(reparsed.evals[0].expectValue).toBe('hello')
  })

  it('removes evals when cleared in form', () => {
    const original = `adl: "1.0"
id: eval-agent
name: Eval Agent
harness:
  type: claude-code
evals:
  - name: smoke
    input: hi
`
    const { form } = parseAgentYaml(original, emptyOptions)
    const merged = mergeFormIntoAgentYaml(original, { ...form, evals: [] }, emptyOptions)
    expect(merged).not.toContain('evals:')
  })
})

describe('adlAgentForm subAgents', () => {
  it('parses subAgents from YAML', () => {
    const yaml = `adl: "1.0"
id: triage-bot
name: Triage Bot
harness:
  type: claude-code
subAgents:
  - hello-world
  - code-reviewer
`
    const { form, hasSubAgents } = parseAgentYaml(yaml, emptyOptions)
    expect(hasSubAgents).toBe(true)
    expect(form.subAgents).toEqual(['hello-world', 'code-reviewer'])
  })

  it('round-trips subAgents', () => {
    const original = `adl: "1.0"
id: triage-bot
name: Triage Bot
harness:
  type: claude-code
subAgents:
  - hello-world
`
    const { form } = parseAgentYaml(original, emptyOptions)
    const merged = mergeFormIntoAgentYaml(original, form, emptyOptions)
    const { form: reparsed } = parseAgentYaml(merged, emptyOptions)
    expect(reparsed.subAgents).toEqual(['hello-world'])
  })

  it('removes subAgents when cleared in form', () => {
    const original = `adl: "1.0"
id: triage-bot
name: Triage Bot
harness:
  type: claude-code
subAgents:
  - hello-world
`
    const { form } = parseAgentYaml(original, emptyOptions)
    const merged = mergeFormIntoAgentYaml(original, { ...form, subAgents: [] }, emptyOptions)
    expect(merged).not.toContain('subAgents:')
  })
})
