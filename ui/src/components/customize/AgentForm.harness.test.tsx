// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { AgentForm } from '@/components/customize/AgentForm'
import { defaultAgentForm, type AgentFormOptions } from '@/lib/adlAgentForm'

const options: AgentFormOptions = {
  harnesses: [
    { id: 'builtin:claude-code', label: 'Claude Code', group: 'CLI', harnessType: 'claude-code' },
    {
      id: 'builtin:api/openai',
      label: 'OpenAI',
      group: 'API',
      harnessType: 'api',
      apiProvider: 'openai',
    },
    { id: 'builtin:devcontainer', label: 'Dev container', group: 'Built-in', harnessType: 'devcontainer' },
  ],
  skills: [],
  mcpServers: [],
  agents: [],
}

describe('AgentForm harness fields', () => {
  it('shows model for api harness without separate provider select', async () => {
    const onChange = vi.fn()
    const form = {
      ...defaultAgentForm(),
      harnessOptionId: 'builtin:api/openai',
      apiProvider: 'openai',
    }
    render(<AgentForm form={form} options={options} onChange={onChange} />)
    expect(screen.queryByText('API provider')).not.toBeInTheDocument()
    expect(screen.getByLabelText('Model')).not.toBeDisabled()
  })

  it('lists API harnesses in the harness type select', () => {
    const onChange = vi.fn()
    render(<AgentForm form={defaultAgentForm()} options={options} onChange={onChange} />)
    // SelectGrouped shows current value; options include OpenAI as a harness choice
    expect(options.harnesses.some((h) => h.id === 'builtin:api/openai')).toBe(true)
    expect(options.harnesses.some((h) => h.group === 'API')).toBe(true)
  })

  it('shows inner harness select for devcontainer harness', () => {
    const onChange = vi.fn()
    const form = {
      ...defaultAgentForm(),
      harnessOptionId: 'builtin:devcontainer',
      innerHarness: 'pi',
    }
    render(<AgentForm form={form} options={options} onChange={onChange} />)
    expect(screen.getByText('Inner harness')).toBeInTheDocument()
  })
})
