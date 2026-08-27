// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useCallback, useLayoutEffect, useRef, useState } from 'react'
import { ArrowUp, Loader2, Plus, Settings } from 'lucide-react'
import { LandingTitle } from '@/components/LandingTitle'
import { PlumeriaFlower } from '@/components/PlumeriaFlower'
import { PlumeriaRandomBackdrop } from '@/components/PlumeriaBackdrop'
import { RecentsSection } from '@/components/RecentsSection'
import { Button } from '@/components/ui/button'
import { useTheme } from '@/contexts/theme'
import type { AgentType, RecentAgentEntry, Session } from '@/types'

interface Props {
  active: boolean
  focusToken?: number
  sessions: Session[]
  agentTypes: AgentType[]
  recentSessionIds?: string[]
  recentAgents?: RecentAgentEntry[]
  onLaunchWithPrompt: (prompt: string) => Promise<void>
  onNewSession: () => void
  onCustomize: () => void
  onOpenSession: (sessionId: string) => void
  onCreateFromRecentAgent: (entry: RecentAgentEntry) => Promise<void>
  onRecentsChange: (patch: { recentSessionIds?: string[]; recentAgents?: RecentAgentEntry[] }) => void
}

export function LandingPage({
  active,
  focusToken = 0,
  sessions,
  agentTypes,
  recentSessionIds,
  recentAgents,
  onLaunchWithPrompt,
  onNewSession,
  onCustomize,
  onOpenSession,
  onCreateFromRecentAgent,
  onRecentsChange,
}: Props) {
  const promptRef = useRef<HTMLTextAreaElement>(null)
  const { uiThemeDef } = useTheme()
  const showFlowers = uiThemeDef.flowers
  const [layoutKey] = useState(() => Date.now())
  const [prompt, setPrompt] = useState('')
  const [loading, setLoading] = useState(false)
  const [creatingAgent, setCreatingAgent] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  const submit = useCallback(async () => {
    const trimmed = prompt.trim()
    if (!trimmed || loading) return
    setError(null)
    setLoading(true)
    try {
      await onLaunchWithPrompt(trimmed)
      setPrompt('')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to launch session')
    } finally {
      setLoading(false)
    }
  }, [loading, onLaunchWithPrompt, prompt])

  useLayoutEffect(() => {
    if (!active) return

    let cancelled = false
    const focusPrompt = () => {
      if (!cancelled) {
        promptRef.current?.focus({ preventScroll: true })
      }
    }

    const frame = requestAnimationFrame(() => {
      requestAnimationFrame(focusPrompt)
    })

    return () => {
      cancelled = true
      cancelAnimationFrame(frame)
    }
  }, [active, focusToken])

  const onKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      void submit()
    }
  }

  async function handleRecentAgentClick(entry: RecentAgentEntry) {
    if (creatingAgent) return
    setError(null)
    setCreatingAgent(entry.agentType)
    try {
      await onCreateFromRecentAgent(entry)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create session')
    } finally {
      setCreatingAgent(null)
    }
  }

  return (
    <div className="landing-page">
      {showFlowers && (
        <PlumeriaRandomBackdrop count={6} opacityVariant="landing" layoutKey={layoutKey} />
      )}
      <div className="landing-page__hero">
        <div className="landing-page__content">
          <LandingTitle />

          <div className="landing-page__prompt">
            <textarea
              ref={promptRef}
              className="landing-page__prompt-input"
              value={prompt}
              onChange={(e) => setPrompt(e.target.value)}
              onKeyDown={onKeyDown}
              placeholder="Aloha! Drop a task here — nui will route the mahi so you can kick back and enjoy the breeze."
              rows={4}
              spellCheck={false}
              autoCorrect="off"
              autoCapitalize="off"
              disabled={loading || creatingAgent !== null}
              aria-label="Launch prompt"
            />
            <button
              type="button"
              className="landing-page__prompt-send"
              onClick={() => void submit()}
              disabled={!prompt.trim() || loading || creatingAgent !== null}
              aria-label="Submit prompt"
              title="Submit prompt"
            >
              {loading ? (
                <Loader2 className="size-4 animate-spin" aria-hidden />
              ) : (
                <>
                  {showFlowers ? (
                    <PlumeriaFlower size={18} className="landing-page__prompt-send-flower" />
                  ) : (
                    <ArrowUp className="size-4 shrink-0" aria-hidden />
                  )}
                  <span className="landing-page__prompt-send-label">Submit</span>
                </>
              )}
            </button>
          </div>
          {error && (
            <p className="landing-page__error" role="alert">
              {error}
            </p>
          )}

          <div className="landing-page__actions">
            <Button size="default" variant="ghost" className="gap-2 pl-3 pr-5" onClick={onNewSession}>
              <Plus className="size-4" />
              New Session
            </Button>
            <Button size="default" variant="ghost" className="gap-2 px-5" onClick={onCustomize}>
              <Settings className="size-4" />
              Customize
            </Button>
          </div>
        </div>
      </div>

      <div className="landing-page__recents">
        <RecentsSection
          sessions={sessions}
          agentTypes={agentTypes}
          recentSessionIds={recentSessionIds}
          recentAgents={recentAgents}
          onRecentAgentClick={(entry) => void handleRecentAgentClick(entry)}
          onRecentSessionClick={onOpenSession}
          onRecentsChange={onRecentsChange}
        />
      </div>
    </div>
  )
}
