// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useCallback, useEffect, useRef, useState } from 'react'
import { SidebarProvider, SidebarTrigger } from '@/components/ui/sidebar'
import { TooltipProvider } from '@/components/ui/tooltip'
import { AppSidebar } from '@/components/AppSidebar'
import { ConversationPanel } from '@/components/ConversationPanel'
import { CustomizePanel } from '@/components/customize/CustomizePanel'
import { ThemeProvider } from '@/contexts/theme'
import { api } from '@/api'
import { navigateToSession, sessionIdFromPath } from '@/lib/sessionUrl'
import type { Session, AgentType } from '@/types'

export default function App() {
  const [sessions, setSessions] = useState<Session[]>([])
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [sidebarOpen, setSidebarOpen] = useState(true)
  const [initialPrompt, setInitialPrompt] = useState<string | undefined>()
  const [hideInput, setHideInput] = useState(false)
  const [agentTypes, setAgentTypes] = useState<AgentType[]>([])
  const [customizeOpen, setCustomizeOpen] = useState(false)
  const [appReady, setAppReady] = useState(false)
  const initializedRef = useRef(false)
  const sessionsRef = useRef(sessions)
  sessionsRef.current = sessions

  const loadAgentTypes = useCallback(async () => {
    try {
      const types = await api.agentTypes.list()
      setAgentTypes(types)
      return types
    } catch {
      return []
    }
  }, [])

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
        loadAgentTypes(),
      ])

      setAgentTypes(types)

      if (bootstrap.sidebarOpen !== undefined) {
        setSidebarOpen(bootstrap.sidebarOpen)
        api.settings.update({ sidebarOpen: bootstrap.sidebarOpen }).catch(() => {})
      } else if (settings.sidebarOpen !== undefined) {
        setSidebarOpen(settings.sidebarOpen)
      }

      let nextId = sessionIdFromPath() ?? bootstrap.sessionId ?? settings.lastSessionId ?? null
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

      if (nextId) {
        navigateToSession(nextId, true)
      }

      if (bootstrap.initialPrompt && nextId) {
        setInitialPrompt(bootstrap.initialPrompt)
      }
      if (bootstrap.hideInput) {
        setHideInput(true)
      }

      setAppReady(true)
    }

    void init()
  }, [loadSessions, loadAgentTypes])

  useEffect(() => {
    function onPopState() {
      const id = sessionIdFromPath()
      if (!id || !sessionsRef.current.some((s) => s.id === id)) return
      setSelectedId(id)
      setInitialPrompt(undefined)
      setHideInput(false)
      api.settings.update({ lastSessionId: id }).catch(() => {})
    }

    window.addEventListener('popstate', onPopState)
    return () => window.removeEventListener('popstate', onPopState)
  }, [])

  const handleSelect = useCallback((id: string) => {
    setCustomizeOpen(false)
    setSelectedId(id)
    setInitialPrompt(undefined)
    setHideInput(false)
    navigateToSession(id)
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

  const handleOpenCustomize = useCallback(() => {
    setCustomizeOpen(true)
  }, [])

  const handleCloseCustomize = useCallback(() => {
    setCustomizeOpen(false)
  }, [])

  const handleExtensionsChanged = useCallback(() => {
    void loadAgentTypes()
  }, [loadAgentTypes])

  const handleDeleteSession = useCallback(async (id: string) => {
    await api.sessions.delete(id)
    const list = await loadSessions()
    if (selectedId === id) {
      const nextId = list[0]?.id ?? null
      setSelectedId(nextId)
      if (nextId) {
        navigateToSession(nextId, true)
      } else {
        window.history.replaceState(null, '', '/')
      }
    }
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
            agentTypes={agentTypes}
            selectedId={selectedId}
            customizeOpen={customizeOpen}
            onSelect={handleSelect}
            onOpenCustomize={handleOpenCustomize}
            onRefresh={loadSessions}
            onRename={handleRenameSession}
            onDelete={handleDeleteSession}
          />
          <main className="flex flex-1 overflow-hidden">
            {customizeOpen ? (
              <CustomizePanel
                onClose={handleCloseCustomize}
                onExtensionsChanged={handleExtensionsChanged}
              />
            ) : selected ? (
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
