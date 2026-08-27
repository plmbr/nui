// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useMemo, useState, type ReactNode } from 'react'
import { ChevronRight } from 'lucide-react'
import { HarnessIcon } from '@/components/HarnessIcon'
import { RecentsListRow } from '@/components/RecentsListRow'
import { RecentsMoreDialog, RECENTS_PREVIEW_LIMIT } from '@/components/RecentsMoreDialog'
import {
  resolveRecentAgents,
  resolveRecentSessions,
} from '@/lib/recents'
import { cn } from '@/lib/utils'
import type { AgentType, RecentAgentEntry, Session } from '@/types'

interface Props {
  sessions: Session[]
  agentTypes: AgentType[]
  recentSessionIds?: string[]
  recentAgents?: RecentAgentEntry[]
  open: boolean
  onOpenChange: (open: boolean) => void
  onRecentAgentClick: (entry: RecentAgentEntry) => void
  onRecentSessionClick: (sessionId: string) => void
  onRecentsChange: (patch: { recentSessionIds?: string[]; recentAgents?: RecentAgentEntry[] }) => void
}

export function RecentsSection({
  sessions,
  agentTypes,
  recentSessionIds,
  recentAgents,
  open,
  onOpenChange,
  onRecentAgentClick,
  onRecentSessionClick,
  onRecentsChange,
}: Props) {
  const [moreKind, setMoreKind] = useState<'agents' | 'sessions' | null>(null)

  const agentItems = useMemo(
    () => resolveRecentAgents(recentAgents, agentTypes),
    [recentAgents, agentTypes],
  )
  const sessionItems = useMemo(
    () => resolveRecentSessions(recentSessionIds, sessions, agentTypes),
    [recentSessionIds, sessions, agentTypes],
  )

  const previewAgents = agentItems.slice(0, RECENTS_PREVIEW_LIMIT)
  const previewSessions = sessionItems.slice(0, RECENTS_PREVIEW_LIMIT)

  return (
    <>
      <section className="recents-section" aria-label="Recents">
        <button
          type="button"
          className="recents-section__title"
          aria-expanded={open}
          onClick={() => onOpenChange(!open)}
        >
          <ChevronRight
            className={cn(
              'size-4 shrink-0 text-muted-foreground transition-transform duration-200',
              open && 'rotate-90',
            )}
            aria-hidden
          />
          <span>Recents</span>
        </button>

        {open && (
          <div className="recents-section__grid">
            <RecentsColumn
              title="Agents"
              emptyLabel="No recent agents"
              itemCount={previewAgents.length}
              showMore={agentItems.length > RECENTS_PREVIEW_LIMIT}
              onMore={() => setMoreKind('agents')}
            >
              {previewAgents.map((item) => (
                <RecentsListRow
                  key={item.entry.agentType}
                  label={item.label}
                  ariaLabel={`Recent agent: ${item.label}`}
                  onClick={() => onRecentAgentClick(item.entry)}
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
              showMore={sessionItems.length > RECENTS_PREVIEW_LIMIT}
              onMore={() => setMoreKind('sessions')}
            >
              {previewSessions.map((item) => (
                <RecentsListRow
                  key={item.id}
                  label={item.label}
                  ariaLabel={`Recent session: ${item.label}`}
                  onClick={() => onRecentSessionClick(item.id)}
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
        )}
      </section>

      {moreKind && (
        <RecentsMoreDialog
          kind={moreKind}
          open={moreKind !== null}
          onOpenChange={(openDialog) => {
            if (!openDialog) setMoreKind(null)
          }}
          sessions={sessions}
          agentTypes={agentTypes}
          recentSessionIds={recentSessionIds}
          recentAgents={recentAgents}
          onRecentAgentClick={onRecentAgentClick}
          onRecentSessionClick={onRecentSessionClick}
          onRecentsChange={onRecentsChange}
        />
      )}
    </>
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
        {itemCount > 0 ? children : (
          <p className="recents-section__empty">{emptyLabel}</p>
        )}
      </div>
      {showMore && (
        <button type="button" className="recents-section__more" onClick={onMore}>
          More…
        </button>
      )}
    </div>
  )
}
