// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { describe, expect, it, vi, beforeEach } from 'vitest'
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { NewSessionPanel } from '@/components/NewSessionPanel'
import { api } from '@/api'
import type { AgentType } from '@/types'

vi.mock('@/api', () => ({
  api: {
    agentTypes: { list: vi.fn() },
    extensions: { list: vi.fn() },
    directories: { suggest: vi.fn() },
    sessions: { create: vi.fn() },
    settings: { update: vi.fn() },
  },
}))

const mockedApi = vi.mocked(api)

const agentTypes: AgentType[] = [
  {
    id: 'nui',
    label: 'Nui',
    harness: 'api',
    available: true,
    isBuiltin: true,
    tags: ['builtin'],
  },
  {
    id: 'local-writer',
    label: 'Local Writer',
    harness: 'claude-code',
    available: true,
    isBuiltin: false,
    tags: ['writing'],
  },
  {
    id: 'ext:sample-pack/reviewer',
    label: 'Reviewer',
    harness: 'extension',
    available: true,
    isBuiltin: false,
    source: 'extension',
    tags: ['review'],
  },
]

function renderPanel() {
  return render(
    <NewSessionPanel
      agentTypes={agentTypes}
      initialAgentTypeId="local-writer"
      onClose={() => {}}
      onCreated={() => {}}
    />,
  )
}

function tagInput(): HTMLElement {
  return screen.getByPlaceholderText('Filter by tag…')
}

function tagSuggestions(): string[] {
  const list = screen.getByRole('listbox', { name: 'Tag suggestions' })
  return within(list).getAllByRole('option').map((option) => option.textContent ?? '')
}

describe('NewSessionPanel tag filter', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    // jsdom has no layout engine, so the panel's scroll-into-view needs a stub.
    Element.prototype.scrollIntoView = vi.fn()
    mockedApi.extensions.list.mockResolvedValue([
      { name: 'sample-pack', displayName: 'Sample Pack', disabled: false },
    ])
    mockedApi.directories.suggest.mockResolvedValue({ directories: [] })
  })

  it('suggests tags from every source when no source filter is active', async () => {
    renderPanel()
    await screen.findByRole('button', { name: 'Sample Pack' })

    fireEvent.focus(tagInput())
    expect(tagSuggestions()).toEqual(['review', 'writing'])
  })

  it('limits tag suggestions to the selected source', async () => {
    renderPanel()
    fireEvent.click(await screen.findByRole('button', { name: 'Sample Pack' }))

    fireEvent.focus(tagInput())
    expect(tagSuggestions()).toEqual(['review'])
  })

  it('drops selected tags that no longer apply after a source change', async () => {
    renderPanel()
    await screen.findByRole('button', { name: 'Sample Pack' })

    fireEvent.focus(tagInput())
    fireEvent.mouseDown(screen.getByRole('option', { name: 'writing' }))
    expect(screen.getByRole('button', { name: 'Remove tag writing' })).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Sample Pack' }))
    await waitFor(() => {
      expect(screen.queryByRole('button', { name: 'Remove tag writing' })).not.toBeInTheDocument()
    })
    expect(screen.getByText('Reviewer')).toBeInTheDocument()
  })
})
