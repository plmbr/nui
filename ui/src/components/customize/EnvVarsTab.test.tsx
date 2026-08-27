// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { EnvVarsTab } from '@/components/customize/EnvVarsTab'
import { api } from '@/api'

vi.mock('@/api', () => ({
  api: {
    env: {
      get: vi.fn(),
      update: vi.fn(),
    },
  },
}))

const mockedApi = vi.mocked(api)

describe('EnvVarsTab', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockedApi.env.get.mockResolvedValue({
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
      custom: [{ key: 'MY_TOKEN', value: 'tok' }],
    })
    mockedApi.env.update.mockImplementation(async (body) => {
      const env = body.env ?? {}
      const custom = body.custom ?? {}
      return {
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
        custom: Object.entries(custom).map(([key, value]) => ({ key, value })),
      }
    })
  })

  it('masks secret values by default and toggles visibility', async () => {
    render(<EnvVarsTab />)

    const keyInput = await screen.findByLabelText('Anthropic API key')
    expect(keyInput).toHaveAttribute('type', 'password')

    fireEvent.click(screen.getByRole('button', { name: 'Show Anthropic API key' }))
    expect(keyInput).toHaveAttribute('type', 'text')

    fireEvent.click(screen.getByRole('button', { name: 'Hide Anthropic API key' }))
    expect(keyInput).toHaveAttribute('type', 'password')
  })

  it('saves credentials and custom env and notifies parent', async () => {
    const onChanged = vi.fn()
    render(<EnvVarsTab onChanged={onChanged} />)

    const keyInput = await screen.findByLabelText('Anthropic API key')
    fireEvent.change(keyInput, { target: { value: 'sk-new' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save env vars' }))

    await waitFor(() => {
      expect(mockedApi.env.update).toHaveBeenCalledWith({
        env: {
          ANTHROPIC_API_KEY: 'sk-new',
          ANTHROPIC_BASE_URL: '',
        },
        custom: {
          MY_TOKEN: 'tok',
        },
      })
    })
    expect(onChanged).toHaveBeenCalled()
  })
})
