// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryTab } from '@/components/customize/MemoryTab'
import { api } from '@/api'

vi.mock('@/api', () => ({
  api: {
    memory: {
      list: vi.fn(),
      getUser: vi.fn(),
      getAgent: vi.fn(),
      saveUser: vi.fn(),
      saveAgent: vi.fn(),
      deleteUser: vi.fn(),
      deleteAgent: vi.fn(),
    },
    agentTypes: { list: vi.fn() },
    settings: { get: vi.fn(), update: vi.fn() },
  },
}))

const mockedApi = vi.mocked(api)

describe('MemoryTab', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockedApi.memory.list.mockResolvedValue({
      user: { size: 0, mode: 'manual' },
      agents: [],
    })
    mockedApi.memory.getUser.mockResolvedValue({ content: '' })
    mockedApi.agentTypes.list.mockResolvedValue([
      {
        id: 'claude-code',
        label: 'Claude Code',
        harness: 'claude-code',
        isBuiltin: true,
        available: true,
      },
    ])
    mockedApi.settings.get.mockResolvedValue({ theme: 'light' })
  })

  it('renders after loading without violating hook order', async () => {
    render(<MemoryTab />)
    expect(screen.getByText('Loading memory…')).toBeInTheDocument()

    await waitFor(() => {
      expect(screen.getByText('User memory', { exact: true })).toBeInTheDocument()
    })
    expect(screen.getByText('Claude Code')).toBeInTheDocument()
  })
})
