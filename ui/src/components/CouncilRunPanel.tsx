// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useEffect, useMemo, useState } from 'react'
import { api } from '@/api'
import { ChatPanel } from '@/components/ChatPanel'
import { attachCouncilMemberSession } from '@/lib/sessionChatStore'
import { cn } from '@/lib/utils'
import type { CouncilProgressState, Session } from '@/types'

function formatElapsed(ms?: number): string {
  if (ms == null || ms < 0) return ''
  const sec = Math.round(ms / 1000)
  if (sec < 60) return `${sec}s`
  const m = Math.floor(sec / 60)
  const s = sec % 60
  return `${m}:${String(s).padStart(2, '0')}`
}

type TabId = 'overview' | `member:${string}`

function stubSession(id: string, name: string, agentType: string): Session {
  return {
    id,
    name,
    workingDir: '',
    agentType,
    createdAt: '',
  }
}

export function CouncilRunPanel({ progress }: { progress: CouncilProgressState }) {
  const [activeTab, setActiveTab] = useState<TabId>('overview')
  const [childSessions, setChildSessions] = useState<Record<string, Session>>({})

  const isSubAgents = progress.round === 'subAgents'
  const allMembers = progress.members
  // Sub-agents: while a member is running, show only that agent. Otherwise show
  // agents that have participated (never the unused pool).
  const members = useMemo(() => {
    if (!isSubAgents) return allMembers
    if (progress.phase === 'complete') return allMembers
    const running = allMembers.filter((m) => m.status === 'running')
    if (running.length > 0) return running
    return allMembers
  }, [allMembers, isSubAgents, progress.phase])

  const roundLabel = progress.round
    ? progress.round === 'subAgents'
      ? 'Sub-agents'
      : progress.round.charAt(0).toUpperCase() + progress.round.slice(1)
    : 'Orchestration'
  const stage =
    progress.phase === 'synthesizing'
      ? 'Synthesizing'
      : progress.phase === 'delegating'
        ? isSubAgents
          ? 'Choosing agent…'
          : 'Delegating'
        : progress.phase === 'complete'
          ? 'Complete'
          : progress.phase === 'member_started'
            ? isSubAgents
              ? 'Running'
              : roundLabel
            : roundLabel

  useEffect(() => {
    if (!isSubAgents) return
    const running = allMembers.find((m) => m.status === 'running')
    if (running) setActiveTab(`member:${running.id}`)
  }, [allMembers, isSubAgents])

  useEffect(() => {
    for (const m of members) {
      if (!m.sessionId) continue
      // Only attach live run streaming while the member is still running;
      // completed members load persisted messages via ensureSessionChatLoaded.
      void attachCouncilMemberSession(
        m.sessionId,
        m.status === 'running' ? m.runId : undefined,
      )
    }
  }, [members])

  useEffect(() => {
    let cancelled = false
    const missing = members.filter((m) => m.sessionId && !childSessions[m.sessionId])
    if (missing.length === 0) return
    void Promise.all(
      missing.map(async (m) => {
        const sid = m.sessionId!
        try {
          const s = await api.sessions.get(sid)
          return [sid, s] as const
        } catch {
          return [sid, stubSession(sid, m.label, m.id)] as const
        }
      }),
    ).then((pairs) => {
      if (cancelled) return
      setChildSessions((prev) => {
        const next = { ...prev }
        for (const [sid, s] of pairs) next[sid] = s
        return next
      })
    })
    return () => {
      cancelled = true
    }
  }, [members, childSessions])

  // When a member gets a new session id (fresh mode), drop stale cache entries.
  useEffect(() => {
    const live = new Set(allMembers.map((m) => m.sessionId).filter(Boolean) as string[])
    setChildSessions((prev) => {
      let changed = false
      const next: Record<string, Session> = {}
      for (const [id, s] of Object.entries(prev)) {
        if (live.has(id)) next[id] = s
        else changed = true
      }
      return changed ? next : prev
    })
  }, [allMembers])

  const tabs = useMemo(() => {
    const items: { id: TabId; label: string; status?: string }[] = [
      { id: 'overview', label: 'Overview' },
    ]
    for (const m of members) {
      items.push({
        id: `member:${m.id}`,
        label: m.label || m.id,
        status: m.status,
      })
    }
    return items
  }, [members])

  const activeMember = allMembers.find((m) => activeTab === `member:${m.id}`)
  const activeSession =
    activeMember?.sessionId != null ? childSessions[activeMember.sessionId] : undefined

  const finishedSteps = allMembers.filter(
    (m) => m.status === 'completed' || m.status === 'failed',
  ).length

  return (
    <div className="mb-3 overflow-hidden rounded-md border border-border/60 bg-muted/20">
      <div className="flex flex-wrap items-center gap-1 border-b border-border/50 bg-muted/40 px-2 py-1.5">
        {tabs.map((tab) => (
          <button
            key={tab.id}
            type="button"
            className={cn(
              'rounded px-2 py-1 text-xs transition-colors',
              activeTab === tab.id
                ? 'bg-background text-foreground shadow-sm'
                : 'text-muted-foreground hover:text-foreground',
            )}
            onClick={() => setActiveTab(tab.id)}
          >
            {tab.label}
            {tab.status && tab.status !== 'queued' ? (
              <span className="ml-1 opacity-70">· {tab.status}</span>
            ) : null}
          </button>
        ))}
      </div>

      {activeTab === 'overview' ? (
        <div className="space-y-2 px-3 py-2 text-xs">
          <div className="flex flex-wrap items-baseline justify-between gap-2">
            <p className="font-medium text-foreground">
              {stage}
              {progress.roundIndex && progress.roundsTotal
                ? ` · round ${progress.roundIndex}/${progress.roundsTotal}`
                : ''}
              {!isSubAgents && progress.membersDone != null && progress.membersTotal
                ? ` · ${progress.membersDone}/${progress.membersTotal} members`
                : ''}
              {isSubAgents && finishedSteps > 0
                ? ` · ${finishedSteps} step${finishedSteps === 1 ? '' : 's'}`
                : ''}
            </p>
            {progress.estimatedCost ? (
              <p className="text-muted-foreground">{progress.estimatedCost}</p>
            ) : null}
          </div>
          {allMembers.length > 0 ? (
            <ul className="space-y-1">
              {allMembers.map((m) => (
                <li key={`${m.id}:${m.runId ?? m.sessionId ?? ''}`}>
                  <button
                    type="button"
                    className="flex w-full items-center justify-between gap-2 rounded px-1 py-1 text-left text-muted-foreground hover:bg-muted/50 hover:text-foreground"
                    onClick={() => setActiveTab(`member:${m.id}`)}
                  >
                    <span className="truncate">
                      <span className="text-foreground">{m.label}</span>
                      {' · '}
                      {m.status}
                      {m.error ? ` (${m.error})` : ''}
                    </span>
                    {m.elapsedMs != null ? (
                      <span className="shrink-0 tabular-nums">{formatElapsed(m.elapsedMs)}</span>
                    ) : null}
                  </button>
                </li>
              ))}
            </ul>
          ) : (
            <p className="text-muted-foreground">
              {isSubAgents ? 'Waiting for the orchestrator to pick an agent…' : 'Waiting for council members…'}
            </p>
          )}
        </div>
      ) : activeMember?.sessionId && activeSession ? (
        <div className="h-[min(420px,50vh)] min-h-[240px] flex flex-col">
          <ChatPanel
            key={`${activeMember.sessionId}:${activeMember.runId ?? ''}`}
            session={activeSession}
            hideInput
            skipBootstrap
            embedded
          />
        </div>
      ) : (
        <div className="px-3 py-4 text-xs text-muted-foreground">
          {activeMember
            ? `Waiting for ${activeMember.label} session…`
            : isSubAgents
              ? 'Select a sub-agent.'
              : 'Select a council member.'}
        </div>
      )}
    </div>
  )
}
