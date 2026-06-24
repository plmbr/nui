// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useCallback, useEffect, useRef, useState } from 'react'
import { SidebarProvider, SidebarTrigger } from '@/components/ui/sidebar'
import { TooltipProvider } from '@/components/ui/tooltip'
import { AppSidebar } from '@/components/AppSidebar'
import { ConversationPanel } from '@/components/ConversationPanel'
import { CustomizePanel } from '@/components/customize/CustomizePanel'
import { NewSessionPanel } from '@/components/NewSessionPanel'
import { SessionsListPanel } from '@/components/SessionsListPanel'
import { ThemeProvider } from '@/contexts/theme'
import { api } from '@/api'
import { groupSessionsByAgentType } from '@/lib/sessionGroups'
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
  const [newSessionOpen, setNewSessionOpen] = useState(false)
  const [sessionListGroupId, setSessionListGroupId] = useState<string | null>(null)
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
    setNewSessionOpen(false)
    setSessionListGroupId(null)
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
  const sessionListGroup =
    sessionListGroupId != null
      ? groupSessionsByAgentType(sessions, agentTypes).find((group) => group.id === sessionListGroupId) ?? null
      : null

  const handleOpenCustomize = useCallback(() => {
    setNewSessionOpen(false)
    setSessionListGroupId(null)
    setCustomizeOpen(true)
  }, [])

  const handleCloseCustomize = useCallback(() => {
    setCustomizeOpen(false)
  }, [])

  const handleOpenNewSession = useCallback(() => {
    setCustomizeOpen(false)
    setSessionListGroupId(null)
    setNewSessionOpen(true)
  }, [])

  const handleCloseNewSession = useCallback(() => {
    setNewSessionOpen(false)
  }, [])

  const handleOpenSessionList = useCallback((groupId: string) => {
    setCustomizeOpen(false)
    setNewSessionOpen(false)
    setSessionListGroupId(groupId)
  }, [])

  const handleCloseSessionList = useCallback(() => {
    setSessionListGroupId(null)
  }, [])

  const handleSessionCreated = useCallback(async (session: Session) => {
    await loadSessions()
    handleSelect(session.id)
  }, [loadSessions, handleSelect])

  const handleAgentTypesChanged = useCallback(() => {
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

  const handleBulkDeleteSessions = useCallback(async (ids: string[]) => {
    await api.sessions.bulkDelete(ids)
    const list = await loadSessions()
    if (selectedId && ids.includes(selectedId)) {
      const nextId = list[0]?.id ?? null
      setSelectedId(nextId)
      if (nextId) {
        navigateToSession(nextId, true)
      } else {
        window.history.replaceState(null, '', '/')
      }
    }
    if (sessionListGroupId) {
      const groups = groupSessionsByAgentType(list, agentTypes)
      if (!groups.some((group) => group.id === sessionListGroupId && group.sessions.length > 0)) {
        setSessionListGroupId(null)
      }
    }
  }, [selectedId, loadSessions, sessionListGroupId, agentTypes])

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
          <span className="app-brand shrink-0">The Loop</span>
        </header>
        <div className="app-body">
          <AppSidebar
            sessions={sessions}
            agentTypes={agentTypes}
            selectedId={selectedId}
            customizeOpen={customizeOpen}
            newSessionOpen={newSessionOpen}
            sessionListGroupId={sessionListGroupId}
            onSelect={handleSelect}
            onOpenCustomize={handleOpenCustomize}
            onOpenNewSession={handleOpenNewSession}
            onOpenSessionList={handleOpenSessionList}
            onRename={handleRenameSession}
            onDelete={handleDeleteSession}
          />
          <main className="flex min-h-0 flex-1 overflow-hidden">
            {customizeOpen ? (
              <CustomizePanel
                onClose={handleCloseCustomize}
                onAgentTypesChanged={handleAgentTypesChanged}
              />
            ) : newSessionOpen ? (
              <NewSessionPanel
                agentTypes={agentTypes}
                onClose={handleCloseNewSession}
                onCreated={handleSessionCreated}
              />
            ) : sessionListGroup ? (
              <SessionsListPanel
                group={sessionListGroup}
                selectedId={selectedId}
                onSelect={handleSelect}
                onClose={handleCloseSessionList}
                onBulkDelete={handleBulkDeleteSessions}
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
