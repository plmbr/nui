// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { CredentialsTab } from '@/components/customize/CredentialsTab'
import { api } from '@/api'

vi.mock('@/api', () => ({
  api: {
    credentials: {
      get: vi.fn(),
      update: vi.fn(),
    },
  },
}))

const mockedApi = vi.mocked(api)

describe('CredentialsTab', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockedApi.credentials.get.mockResolvedValue({
      fields: [
        {
          key: 'ANTHROPIC_API_KEY',
          label: 'Anthropic API key',
          description: 'Claude API',
          group: 'Anthropic',
          secret: true,
          value: 'sk-secret',
          fromEnv: false,
          configured: true,
        },
        {
          key: 'ANTHROPIC_BASE_URL',
          label: 'Anthropic base URL',
          group: 'Anthropic',
          secret: false,
          value: '',
          fromEnv: false,
          configured: false,
        },
      ],
    })
    mockedApi.credentials.update.mockImplementation(async (env: Record<string, string>) => ({
      fields: [
        {
          key: 'ANTHROPIC_API_KEY',
          label: 'Anthropic API key',
          description: 'Claude API',
          group: 'Anthropic',
          secret: true,
          value: env.ANTHROPIC_API_KEY ?? '',
          fromEnv: false,
          configured: !!(env.ANTHROPIC_API_KEY ?? ''),
        },
        {
          key: 'ANTHROPIC_BASE_URL',
          label: 'Anthropic base URL',
          group: 'Anthropic',
          secret: false,
          value: env.ANTHROPIC_BASE_URL ?? '',
          fromEnv: false,
          configured: !!(env.ANTHROPIC_BASE_URL ?? ''),
        },
      ],
    }))
  })

  it('masks secret values by default and toggles visibility', async () => {
    render(<CredentialsTab />)

    const keyInput = await screen.findByLabelText('Anthropic API key')
    expect(keyInput).toHaveAttribute('type', 'password')

    fireEvent.click(screen.getByRole('button', { name: 'Show Anthropic API key' }))
    expect(keyInput).toHaveAttribute('type', 'text')

    fireEvent.click(screen.getByRole('button', { name: 'Hide Anthropic API key' }))
    expect(keyInput).toHaveAttribute('type', 'password')
  })

  it('saves credentials and notifies parent', async () => {
    const onChanged = vi.fn()
    render(<CredentialsTab onChanged={onChanged} />)

    const keyInput = await screen.findByLabelText('Anthropic API key')
    fireEvent.change(keyInput, { target: { value: 'sk-new' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save credentials' }))

    await waitFor(() => {
      expect(mockedApi.credentials.update).toHaveBeenCalledWith({
        ANTHROPIC_API_KEY: 'sk-new',
        ANTHROPIC_BASE_URL: '',
      })
    })
    expect(onChanged).toHaveBeenCalled()
  })
})
