// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { beforeEach, describe, expect, it, vi } from 'vitest'
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { PageFindBar } from '@/components/PageFindBar'
import {
  clearPageFindHighlights,
  findPageMatches,
  scrollToPageFindMatch,
} from '@/lib/pageFind'

vi.mock('@/lib/pageFind', () => ({
  clearPageFindHighlights: vi.fn(),
  findPageMatches: vi.fn(),
  renderPageFindHighlights: vi.fn(),
  scrollToPageFindMatch: vi.fn(),
}))

const eventHandlers = new Map<string, () => void>()
const firstMatch = { range: {} as Range, element: document.body }
const secondMatch = { range: {} as Range, element: document.body }

describe('PageFindBar', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    eventHandlers.clear()
    window.__NUI_DESKTOP__ = true
    window.runtime = {
      EventsOn: vi.fn((name: string, callback: () => void) => {
        eventHandlers.set(name, callback)
        return () => eventHandlers.delete(name)
      }),
    }
    vi.mocked(findPageMatches).mockReturnValue([firstMatch, secondMatch])
  })

  it('opens with Cmd/Ctrl+F, searches, and navigates matches', async () => {
    render(<PageFindBar />)

    fireEvent.keyDown(window, { key: 'f', metaKey: true })
    const input = await screen.findByRole('searchbox', { name: 'Find in page' })
    expect(input).toHaveFocus()

    fireEvent.change(input, { target: { value: 'agent' } })
    await waitFor(() => expect(findPageMatches).toHaveBeenCalledWith(document.body, 'agent'))
    expect(screen.getByText('1 of 2')).toBeInTheDocument()

    fireEvent.keyDown(input, { key: 'Enter' })
    expect(screen.getByText('2 of 2')).toBeInTheDocument()
    expect(scrollToPageFindMatch).toHaveBeenLastCalledWith(secondMatch)

    fireEvent.keyDown(input, { key: 'Enter', shiftKey: true })
    expect(screen.getByText('1 of 2')).toBeInTheDocument()
  })

  it('responds to desktop menu events and clears on Escape', async () => {
    render(<PageFindBar />)

    act(() => eventHandlers.get('nui:show-page-find')?.())
    const input = await screen.findByRole('searchbox', { name: 'Find in page' })
    fireEvent.change(input, { target: { value: 'agent' } })
    await screen.findByText('1 of 2')

    act(() => eventHandlers.get('nui:find-next-page')?.())
    expect(screen.getByText('2 of 2')).toBeInTheDocument()

    fireEvent.keyDown(input, { key: 'Escape' })
    expect(screen.queryByRole('searchbox', { name: 'Find in page' })).not.toBeInTheDocument()
    expect(clearPageFindHighlights).toHaveBeenCalled()
  })
})
