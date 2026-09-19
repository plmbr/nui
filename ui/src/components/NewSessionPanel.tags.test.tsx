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
      onClose={() => {}}
      onCreated={() => {}}
      initialAgentTypeId="local-writer"
    />,
  )
}

function tagInput(): HTMLElement {
  return screen.getByPlaceholderText('Filter by tag…')
}

async function openFilters() {
  fireEvent.click(screen.getByRole('button', { name: /Filters/i }))
  await screen.findByPlaceholderText('Filter by tag…')
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
    await openFilters()

    fireEvent.focus(tagInput())
    expect(tagSuggestions()).toEqual(['review', 'writing'])
  })

  it('limits tag suggestions to the selected source', async () => {
    renderPanel()
    await openFilters()
    fireEvent.click(await screen.findByRole('button', { name: 'Sample Pack' }))

    fireEvent.focus(tagInput())
    expect(tagSuggestions()).toEqual(['review'])
  })

  it('drops selected tags that no longer apply after a source change', async () => {
    renderPanel()
    await openFilters()

    fireEvent.focus(tagInput())
    fireEvent.mouseDown(screen.getByRole('option', { name: 'writing' }))
    expect(screen.getByRole('button', { name: 'Remove tag writing' })).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Sample Pack' }))
    await waitFor(() => {
      expect(screen.queryByRole('button', { name: 'Remove tag writing' })).not.toBeInTheDocument()
    })
    expect(screen.getByText('Reviewer')).toBeInTheDocument()
  })

  it('searches built-in and installed agents together and selects the first result', async () => {
    renderPanel()

    expect(screen.getByRole('button', { name: 'Nui' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Local Writer' })).toBeInTheDocument()

    fireEvent.change(screen.getByRole('searchbox', { name: 'Search agents' }), {
      target: { value: 'review' },
    })
    expect(screen.queryByRole('button', { name: 'Nui' })).not.toBeInTheDocument()
    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Reviewer' })).toHaveAttribute('aria-pressed', 'true')
    })
  })

  it('selects the first visible agent when search hides the previous selection', async () => {
    renderPanel()
    expect(screen.getByRole('button', { name: 'Local Writer' })).toHaveAttribute('aria-pressed', 'true')

    fireEvent.change(screen.getByRole('searchbox', { name: 'Search agents' }), {
      target: { value: 'Nui' },
    })
    expect(screen.queryByRole('button', { name: 'Local Writer' })).not.toBeInTheDocument()
    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Nui' })).toHaveAttribute('aria-pressed', 'true')
    })
    expect(screen.getByRole('button', { name: 'Create Session' })).toBeEnabled()
  })

  it('creates a session with the first search result when Enter is pressed', async () => {
    mockedApi.sessions.create.mockResolvedValue({} as never)
    renderPanel()
    const search = screen.getByRole('searchbox', { name: 'Search agents' })

    fireEvent.change(search, { target: { value: 'review' } })
    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Reviewer' })).toHaveAttribute('aria-pressed', 'true')
    })
    fireEvent.keyDown(search, { key: 'Enter' })

    await waitFor(() => {
      expect(mockedApi.sessions.create).toHaveBeenCalledWith({
        agentType: 'ext:sample-pack/reviewer',
      })
    })
  })

  it('creates a session with the preselected agent when Enter is pressed without a search', async () => {
    mockedApi.sessions.create.mockResolvedValue({} as never)
    renderPanel()

    fireEvent.keyDown(screen.getByRole('searchbox', { name: 'Search agents' }), { key: 'Enter' })

    await waitFor(() => {
      expect(mockedApi.sessions.create).toHaveBeenCalledWith(
        expect.objectContaining({ agentType: 'local-writer' }),
      )
    })
  })

  it.each([
    ['Nui', 'nui'],
    ['Reviewer', 'ext:sample-pack/reviewer'],
  ])('creates a session after selecting %s with the mouse and pressing Enter', async (label, id) => {
    mockedApi.sessions.create.mockResolvedValue({} as never)
    renderPanel()
    const agent = screen.getByRole('button', { name: label })

    fireEvent.click(agent)
    fireEvent.keyDown(agent, { key: 'Enter' })

    await waitFor(() => {
      expect(mockedApi.sessions.create).toHaveBeenCalledWith(
        expect.objectContaining({ agentType: id }),
      )
    })
  })
})
