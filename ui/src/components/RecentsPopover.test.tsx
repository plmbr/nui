// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { RecentsPopover } from '@/components/RecentsPopover'
import { api } from '@/api'
import type { AgentType, RecentAgentEntry, Session } from '@/types'

vi.mock('@/api', () => ({
  api: {
    settings: { update: vi.fn() },
    state: { update: vi.fn() },
  },
}))

const agents: AgentType[] = Array.from({ length: 9 }, (_, index) => ({
  id: `agent-${index + 1}`,
  label: index === 8 ? 'Searchable Agent' : `Agent ${index + 1}`,
  harness: 'claude-code',
  available: true,
}))
const recentAgents: RecentAgentEntry[] = agents.map((agent) => ({ agentType: agent.id }))
const sessions: Session[] = agents.map((agent, index) => ({
  id: `session-${index + 1}`,
  name: index === 8 ? 'Searchable Session' : `Session ${index + 1}`,
  workingDir: '/tmp/project',
  agentType: agent.id,
  createdAt: '2026-01-01T00:00:00Z',
}))
const recentSessionIds = sessions.map((session) => session.id)

function renderPopover() {
  const onRecentAgentClick = vi.fn()
  const onRecentSessionClick = vi.fn()
  const onRecentsChange = vi.fn()
  render(
    <RecentsPopover
      sessions={sessions}
      agentTypes={agents}
      recentSessionIds={recentSessionIds}
      recentAgents={recentAgents}
      onRecentAgentClick={onRecentAgentClick}
      onRecentSessionClick={onRecentSessionClick}
      onRecentsChange={onRecentsChange}
    />,
  )
  return { onRecentAgentClick, onRecentSessionClick, onRecentsChange }
}

describe('RecentsPopover', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(api.settings.update).mockResolvedValue({})
    vi.mocked(api.state.update).mockResolvedValue({})
  })

  it('opens an overview and opens a recent item', async () => {
    const { onRecentAgentClick } = renderPopover()
    fireEvent.click(screen.getByRole('button', { name: 'Recents' }))

    expect(await screen.findByRole('heading', { name: 'Recents' })).toBeInTheDocument()
    expect(screen.getByText('Agents')).toBeInTheDocument()
    expect(screen.getByText('Sessions')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Recent agent: Agent 1' }))
    expect(onRecentAgentClick).toHaveBeenCalledWith(recentAgents[0])
    await waitFor(() => {
      expect(screen.queryByRole('heading', { name: 'Recents' })).not.toBeInTheDocument()
    })
  })

  it('replaces the overview with a searchable More view and returns back', async () => {
    renderPopover()
    fireEvent.click(screen.getByRole('button', { name: 'Recents' }))
    await screen.findByRole('heading', { name: 'Recents' })

    const moreButtons = screen.getAllByRole('button', { name: 'More…' })
    fireEvent.click(moreButtons[0])
    const search = screen.getByRole('textbox', { name: 'Search recent agents' })
    expect(screen.queryByText('Sessions')).not.toBeInTheDocument()

    fireEvent.change(search, { target: { value: 'Searchable' } })
    expect(screen.getByRole('button', { name: 'Recent agent: Searchable Agent' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Recent agent: Agent 1' })).not.toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Back to recents' }))
    expect(screen.getByText('Agents')).toBeInTheDocument()
    expect(screen.getByText('Sessions')).toBeInTheDocument()
  })

  it('filters sessions and removes one without closing the popover', async () => {
    const { onRecentsChange } = renderPopover()
    fireEvent.click(screen.getByRole('button', { name: 'Recents' }))
    await screen.findByRole('heading', { name: 'Recents' })

    fireEvent.click(screen.getAllByRole('button', { name: 'More…' })[1])
    fireEvent.change(screen.getByRole('textbox', { name: 'Search recent sessions' }), {
      target: { value: 'Searchable' },
    })
    fireEvent.click(screen.getByRole('button', {
      name: 'Remove Searchable Session from recent sessions',
    }))

    await waitFor(() => {
      expect(api.state.update).toHaveBeenCalledWith({
        recentSessionIds: recentSessionIds.slice(0, -1),
      })
      expect(onRecentsChange).toHaveBeenCalledWith({
        recentSessionIds: recentSessionIds.slice(0, -1),
      })
    })
    expect(screen.getByRole('textbox', { name: 'Search recent sessions' })).toBeInTheDocument()
  })
})
