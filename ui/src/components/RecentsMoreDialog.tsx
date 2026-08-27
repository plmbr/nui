// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useState, type ReactNode } from 'react'
import { Trash2 } from 'lucide-react'
import { HarnessIcon } from '@/components/HarnessIcon'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { api } from '@/api'
import {
  RECENTS_PREVIEW_LIMIT,
  removeRecentAgent,
  removeRecentSessionId,
  resolveRecentAgents,
  resolveRecentSessions,
  type ResolvedRecentAgent,
  type ResolvedRecentSession,
} from '@/lib/recents'
import type { AgentType, RecentAgentEntry, Session } from '@/types'

type RecentsKind = 'agents' | 'sessions'

interface Props {
  kind: RecentsKind
  open: boolean
  onOpenChange: (open: boolean) => void
  sessions: Session[]
  agentTypes: AgentType[]
  recentSessionIds?: string[]
  recentAgents?: RecentAgentEntry[]
  onRecentAgentClick: (entry: RecentAgentEntry) => void
  onRecentSessionClick: (sessionId: string) => void
  onRecentsChange: (patch: { recentSessionIds?: string[]; recentAgents?: RecentAgentEntry[] }) => void
}

export function RecentsMoreDialog({
  kind,
  open,
  onOpenChange,
  sessions,
  agentTypes,
  recentSessionIds,
  recentAgents,
  onRecentAgentClick,
  onRecentSessionClick,
  onRecentsChange,
}: Props) {
  const [removingKey, setRemovingKey] = useState<string | null>(null)
  const agentItems = resolveRecentAgents(recentAgents, agentTypes)
  const sessionItems = resolveRecentSessions(recentSessionIds, sessions, agentTypes)
  const title = kind === 'agents' ? 'Recent agents' : 'Recent sessions'

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
      await api.settings.update({ recentSessionIds: next })
      onRecentsChange({ recentSessionIds: next })
    } finally {
      setRemovingKey(null)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>
            {kind === 'agents'
              ? 'Open a new session with the last settings used for each agent.'
              : 'Reopen a recent session. Removing an item only clears it from this list.'}
          </DialogDescription>
        </DialogHeader>
        <div className="recents-dialog__list">
          <ul className="flex flex-col gap-0.5">
            {kind === 'agents' ? (
              agentItems.length === 0 ? (
                <li className="px-2 py-3 text-sm text-muted-foreground">No recent agents.</li>
              ) : (
                agentItems.map((item) => (
                  <RecentsDialogRow
                    key={item.entry.agentType}
                    label={item.label}
                    openLabel={`Recent agent: ${item.label}`}
                    removing={removingKey === item.entry.agentType}
                    removeLabel={`Remove ${item.label} from recent agents`}
                    onOpen={() => {
                      onOpenChange(false)
                      onRecentAgentClick(item.entry)
                    }}
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
                ))
              )
            ) : sessionItems.length === 0 ? (
              <li className="px-2 py-3 text-sm text-muted-foreground">No recent sessions.</li>
            ) : (
              sessionItems.map((item) => (
                <RecentsDialogRow
                  key={item.id}
                  label={item.label}
                  openLabel={`Recent session: ${item.label}`}
                  removing={removingKey === item.id}
                  removeLabel={`Remove ${item.label} from recent sessions`}
                  onOpen={() => {
                    onOpenChange(false)
                    onRecentSessionClick(item.id)
                  }}
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
              ))
            )}
          </ul>
        </div>
      </DialogContent>
    </Dialog>
  )
}

function RecentsDialogRow({
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

export { RECENTS_PREVIEW_LIMIT }
