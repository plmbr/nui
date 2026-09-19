// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useMemo, useState, type ReactNode } from 'react'
import { Popover } from '@base-ui/react/popover'
import { ArrowLeft, History, Search, Trash2, X } from 'lucide-react'
import { HarnessIcon } from '@/components/HarnessIcon'
import { RecentsListRow } from '@/components/RecentsListRow'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { api } from '@/api'
import {
  RECENTS_PREVIEW_LIMIT,
  removeRecentAgent,
  removeRecentSessionId,
  resolveRecentAgents,
  resolveRecentSessions,
} from '@/lib/recents'
import type { AgentType, RecentAgentEntry, Session } from '@/types'

type RecentsView = 'overview' | 'agents' | 'sessions'

interface Props {
  sessions: Session[]
  agentTypes: AgentType[]
  recentSessionIds?: string[]
  recentAgents?: RecentAgentEntry[]
  onRecentAgentClick: (entry: RecentAgentEntry) => void
  onRecentSessionClick: (sessionId: string) => void
  onRecentsChange: (patch: { recentSessionIds?: string[]; recentAgents?: RecentAgentEntry[] }) => void
}

export function RecentsPopover({
  sessions,
  agentTypes,
  recentSessionIds,
  recentAgents,
  onRecentAgentClick,
  onRecentSessionClick,
  onRecentsChange,
}: Props) {
  const [open, setOpen] = useState(false)
  const [view, setView] = useState<RecentsView>('overview')
  const [query, setQuery] = useState('')
  const [removingKey, setRemovingKey] = useState<string | null>(null)

  const agentItems = useMemo(
    () => resolveRecentAgents(recentAgents, agentTypes),
    [recentAgents, agentTypes],
  )
  const sessionItems = useMemo(
    () => resolveRecentSessions(recentSessionIds, sessions, agentTypes),
    [recentSessionIds, sessions, agentTypes],
  )
  const normalizedQuery = query.trim().toLocaleLowerCase()
  const filteredAgents = normalizedQuery
    ? agentItems.filter((item) => item.label.toLocaleLowerCase().includes(normalizedQuery))
    : agentItems
  const filteredSessions = normalizedQuery
    ? sessionItems.filter((item) => item.label.toLocaleLowerCase().includes(normalizedQuery))
    : sessionItems

  function handleOpenChange(nextOpen: boolean) {
    setOpen(nextOpen)
    if (!nextOpen) {
      setView('overview')
      setQuery('')
    }
  }

  function showMore(nextView: Exclude<RecentsView, 'overview'>) {
    setQuery('')
    setView(nextView)
  }

  function openAgent(entry: RecentAgentEntry) {
    handleOpenChange(false)
    onRecentAgentClick(entry)
  }

  function openSession(sessionId: string) {
    handleOpenChange(false)
    onRecentSessionClick(sessionId)
  }

  async function removeAgent(agentType: string) {
    setRemovingKey(agentType)
    try {
      const next = removeRecentAgent(recentAgents, agentType)
      await api.settings.update({ recentAgents: next })
      onRecentsChange({ recentAgents: next })
    } finally {
      setRemovingKey(null)
    }
  }

  async function removeSession(sessionId: string) {
    setRemovingKey(sessionId)
    try {
      const next = removeRecentSessionId(recentSessionIds, sessionId)
      await api.state.update({ recentSessionIds: next })
      onRecentsChange({ recentSessionIds: next })
    } finally {
      setRemovingKey(null)
    }
  }

  return (
    <Popover.Root open={open} onOpenChange={handleOpenChange}>
      <Popover.Trigger
        className="recents-popover__trigger"
        title="Recent agents and sessions"
      >
        <History className="size-4" aria-hidden />
        <span>Recents</span>
      </Popover.Trigger>
      <Popover.Portal>
        <Popover.Backdrop className="recents-popover__backdrop" />
        <Popover.Positioner side="bottom" align="center" sideOffset={8} className="z-50">
          <Popover.Popup className="recents-popover">
            <Popover.Close className="recents-popover__close" aria-label="Close recents">
              <X className="size-4" aria-hidden />
            </Popover.Close>
            {view === 'overview' ? (
              <RecentsOverview
                agents={agentItems}
                sessions={sessionItems}
                onAgentClick={openAgent}
                onSessionClick={openSession}
                onMore={showMore}
              />
            ) : (
              <div className="recents-popover__detail">
                <div className="recents-popover__detail-header">
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon-sm"
                    aria-label="Back to recents"
                    onClick={() => {
                      setView('overview')
                      setQuery('')
                    }}
                  >
                    <ArrowLeft className="size-4" />
                  </Button>
                  <Popover.Title className="recents-popover__title">
                    Recent {view === 'agents' ? 'agents' : 'sessions'}
                  </Popover.Title>
                </div>
                <div className="recents-popover__search">
                  <Search className="size-4" aria-hidden />
                  <Input
                    className="!pl-9"
                    value={query}
                    onChange={(event) => setQuery(event.target.value)}
                    placeholder={`Search recent ${view}`}
                    aria-label={`Search recent ${view}`}
                    autoFocus
                  />
                </div>
                <ul className="recents-popover__detail-list">
                  {view === 'agents' ? (
                    filteredAgents.length > 0 ? filteredAgents.map((item) => (
                      <DetailRow
                        key={item.entry.agentType}
                        label={item.label}
                        openLabel={`Recent agent: ${item.label}`}
                        removing={removingKey === item.entry.agentType}
                        removeLabel={`Remove ${item.label} from recent agents`}
                        onOpen={() => openAgent(item.entry)}
                        onRemove={() => void removeAgent(item.entry.agentType)}
                        icon={(
                          <HarnessIcon
                            harness={item.agentType.harness}
                            provider={item.agentType.provider}
                            agentId={item.agentType.id}
                            size="sm"
                            className="shrink-0"
                          />
                        )}
                      />
                    )) : <EmptyDetail query={query} kind="agents" />
                  ) : (
                    filteredSessions.length > 0 ? filteredSessions.map((item) => (
                      <DetailRow
                        key={item.id}
                        label={item.label}
                        openLabel={`Recent session: ${item.label}`}
                        removing={removingKey === item.id}
                        removeLabel={`Remove ${item.label} from recent sessions`}
                        onOpen={() => openSession(item.id)}
                        onRemove={() => void removeSession(item.id)}
                        icon={(
                          <HarnessIcon
                            harness={item.agentType?.harness ?? 'api'}
                            provider={item.agentType?.provider}
                            agentId={item.session.agentType}
                            size="sm"
                            className="shrink-0"
                          />
                        )}
                      />
                    )) : <EmptyDetail query={query} kind="sessions" />
                  )}
                </ul>
              </div>
            )}
          </Popover.Popup>
        </Popover.Positioner>
      </Popover.Portal>
    </Popover.Root>
  )
}

function RecentsOverview({
  agents,
  sessions,
  onAgentClick,
  onSessionClick,
  onMore,
}: {
  agents: ReturnType<typeof resolveRecentAgents>
  sessions: ReturnType<typeof resolveRecentSessions>
  onAgentClick: (entry: RecentAgentEntry) => void
  onSessionClick: (sessionId: string) => void
  onMore: (view: 'agents' | 'sessions') => void
}) {
  const previewAgents = agents.slice(0, RECENTS_PREVIEW_LIMIT)
  const previewSessions = sessions.slice(0, RECENTS_PREVIEW_LIMIT)

  return (
    <div className="recents-popover__overview">
      <Popover.Title className="recents-popover__title recents-popover__overview-title">
        Recents
      </Popover.Title>
      <div className="recents-popover__grid">
        <RecentsColumn
          title="Agents"
          emptyLabel="No recent agents"
          itemCount={previewAgents.length}
          showMore={agents.length > RECENTS_PREVIEW_LIMIT}
          onMore={() => onMore('agents')}
        >
          {previewAgents.map((item) => (
            <RecentsListRow
              key={item.entry.agentType}
              label={item.label}
              ariaLabel={`Recent agent: ${item.label}`}
              onClick={() => onAgentClick(item.entry)}
              icon={(
                <HarnessIcon
                  harness={item.agentType.harness}
                  provider={item.agentType.provider}
                  agentId={item.agentType.id}
                  size="sm"
                  className="shrink-0"
                />
              )}
            />
          ))}
        </RecentsColumn>
        <RecentsColumn
          title="Sessions"
          emptyLabel="No recent sessions"
          itemCount={previewSessions.length}
          showMore={sessions.length > RECENTS_PREVIEW_LIMIT}
          onMore={() => onMore('sessions')}
        >
          {previewSessions.map((item) => (
            <RecentsListRow
              key={item.id}
              label={item.label}
              ariaLabel={`Recent session: ${item.label}`}
              onClick={() => onSessionClick(item.id)}
              icon={(
                <HarnessIcon
                  harness={item.agentType?.harness ?? 'api'}
                  provider={item.agentType?.provider}
                  agentId={item.session.agentType}
                  size="sm"
                  className="shrink-0"
                />
              )}
            />
          ))}
        </RecentsColumn>
      </div>
    </div>
  )
}

function RecentsColumn({
  title,
  emptyLabel,
  itemCount,
  showMore,
  onMore,
  children,
}: {
  title: string
  emptyLabel: string
  itemCount: number
  showMore: boolean
  onMore: () => void
  children: ReactNode
}) {
  return (
    <div className="recents-section__column">
      <h3 className="recents-section__heading">{title}</h3>
      <div className="recents-section__list">
        {itemCount > 0 ? children : <p className="recents-section__empty">{emptyLabel}</p>}
      </div>
      {showMore && (
        <button type="button" className="recents-section__more" onClick={onMore}>
          More…
        </button>
      )}
    </div>
  )
}

function DetailRow({
  label,
  openLabel,
  icon,
  removing,
  removeLabel,
  onOpen,
  onRemove,
}: {
  label: string
  openLabel: string
  icon: ReactNode
  removing: boolean
  removeLabel: string
  onOpen: () => void
  onRemove: () => void
}) {
  return (
    <li className="recents-dialog__row">
      <button type="button" className="recents-dialog__row-open" aria-label={openLabel} onClick={onOpen}>
        {icon}
        <span className="recents-dialog__row-label" aria-hidden="true">{label}</span>
      </button>
      <Button
        type="button"
        variant="ghost"
        size="icon-sm"
        className="recents-dialog__row-remove"
        disabled={removing}
        aria-label={removeLabel}
        onClick={onRemove}
      >
        <Trash2 className="size-3.5" />
      </Button>
    </li>
  )
}

function EmptyDetail({ query, kind }: { query: string; kind: 'agents' | 'sessions' }) {
  return (
    <li className="recents-popover__empty">
      {query.trim() ? `No matching recent ${kind}.` : `No recent ${kind}.`}
    </li>
  )
}
