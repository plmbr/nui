// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { AgentForm } from '@/components/customize/AgentForm'
import { defaultAgentForm, type AgentFormOptions } from '@/lib/adlAgentForm'

const options: AgentFormOptions = {
  harnesses: [
    { id: 'builtin:claude-code', label: 'Claude Code', group: 'Built-in', harnessType: 'claude-code' },
    { id: 'builtin:api', label: 'API', group: 'Built-in', harnessType: 'api' },
    { id: 'builtin:devcontainer', label: 'Dev container', group: 'Built-in', harnessType: 'devcontainer' },
  ],
  skills: [],
  mcpServers: [],
  agents: [],
}

describe('AgentForm harness fields', () => {
  it('shows API provider select for api harness', async () => {
    const onChange = vi.fn()
    const form = {
      ...defaultAgentForm(),
      harnessOptionId: 'builtin:api',
      apiProvider: 'openai',
    }
    render(<AgentForm form={form} options={options} onChange={onChange} />)
    expect(screen.getByText('API provider')).toBeInTheDocument()
    expect(screen.getByLabelText('Model')).not.toBeDisabled()
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
