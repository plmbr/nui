// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useCallback, useEffect, useRef, useState } from 'react'
import { SidebarProvider, SidebarTrigger } from '@/components/ui/sidebar'
import { TooltipProvider } from '@/components/ui/tooltip'
import { AppSidebar } from '@/components/AppSidebar'
import { ConversationPanel } from '@/components/ConversationPanel'
import { ThemeProvider } from '@/contexts/theme'
import { api } from '@/api'
import type { Session, AgentType } from '@/types'

export default function App() {
  const [sessions, setSessions] = useState<Session[]>([])
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [sidebarOpen, setSidebarOpen] = useState(true)
  const [initialPrompt, setInitialPrompt] = useState<string | undefined>()
  const [hideInput, setHideInput] = useState(false)
  const [agentTypes, setAgentTypes] = useState<AgentType[]>([])
  const [appReady, setAppReady] = useState(false)
  const initializedRef = useRef(false)

  const loadSessions = useCallback(async () => {
    try {
      const list = await api.sessions.list()
      setSessions(list)
      return list
    } catch {
      return []
    }
  }, [])

  useEffect(() => {
    if (initializedRef.current) return
    initializedRef.current = true

    async function init() {
      let [list, settings, bootstrap, types] = await Promise.all([
        loadSessions(),
        api.settings.get().catch(() => ({ theme: 'light' as const })),
        api.bootstrap.get().catch(() => ({})),
        api.agentTypes.list().catch(() => []),
      ])

      setAgentTypes(types)

      if (bootstrap.sidebarOpen !== undefined) {
        setSidebarOpen(bootstrap.sidebarOpen)
        api.settings.update({ sidebarOpen: bootstrap.sidebarOpen }).catch(() => {})
      } else if (settings.sidebarOpen !== undefined) {
        setSidebarOpen(settings.sidebarOpen)
      }

      let nextId = bootstrap.sessionId ?? settings.lastSessionId ?? null
      if (nextId && !list.some((s) => s.id === nextId)) {
        nextId = null
      }

      if (!nextId) {
        try {
          const session = await api.sessions.ensureDefault()
          list = await loadSessions()
          nextId = session.id
        } catch {
          if (list.length > 0) {
            nextId = list[0].id
          }
        }
      }

      setSessions(list)
      setSelectedId(nextId)

      if (bootstrap.initialPrompt && nextId) {
        setInitialPrompt(bootstrap.initialPrompt)
      }
      if (bootstrap.hideInput) {
        setHideInput(true)
      }

      setAppReady(true)
    }

    void init()
  }, [loadSessions])

  const handleSelect = useCallback((id: string) => {
    setSelectedId(id)
    setInitialPrompt(undefined)
    setHideInput(false)
    api.settings.update({ lastSessionId: id }).catch(() => {})
  }, [])

  const handleSidebarOpenChange = useCallback((open: boolean) => {
    setSidebarOpen(open)
    api.settings.update({ sidebarOpen: open }).catch(() => {})
  }, [])

  const selected = sessions.find((s) => s.id === selectedId) ?? null
  const selectedAgent = agentTypes.find((a) => a.id === selected?.agentType)
  const promptMode = selectedAgent?.promptMode ?? 'user'
  const effectiveHideInput = hideInput || promptMode === 'auto'

  const handleDeleteSession = useCallback(async (id: string) => {
    await api.sessions.delete(id)
    if (selectedId === id) setSelectedId(null)
    await loadSessions()
  }, [selectedId, loadSessions])

  const handleRenameSession = useCallback(async (id: string, newName: string) => {
    await api.sessions.rename(id, newName)
    await loadSessions()
  }, [loadSessions])

  return (
    <ThemeProvider>
    <TooltipProvider>
      {!appReady ? (
        <div className="h-screen bg-background" />
      ) : (
      <SidebarProvider open={sidebarOpen} onOpenChange={handleSidebarOpenChange}>
        <header className="app-header">
          <SidebarTrigger />
          <span className="font-semibold text-sm shrink-0">The Loop</span>
        </header>
        <div className="app-body">
          <AppSidebar
            sessions={sessions}
            selectedId={selectedId}
            onSelect={handleSelect}
            onRefresh={loadSessions}
            onRename={handleRenameSession}
            onDelete={handleDeleteSession}
          />
          <main className="flex flex-1 overflow-hidden">
            {selected ? (
              <ConversationPanel
                session={selected}
                initialPrompt={initialPrompt}
                hideInput={effectiveHideInput}
                promptMode={promptMode}
                defaultPrompt={selectedAgent?.defaultPrompt}
                agentLabel={selectedAgent?.label}
                agentDescription={selectedAgent?.description}
                key={selected.id}
              />
            ) : (
              <div className="empty-state">
                Select a session or create a new one.
              </div>
            )}
          </main>
        </div>
      </SidebarProvider>
      )}
    </TooltipProvider>
    </ThemeProvider>
  )
}
