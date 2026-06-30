// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useCallback, useEffect, useRef, useState } from 'react'
import { SidebarProvider, SidebarTrigger } from '@/components/ui/sidebar'
import { TooltipProvider } from '@/components/ui/tooltip'
import { AgentHeader } from '@/components/AgentHeader'
import { AppSidebar } from '@/components/AppSidebar'
import { ConversationPanel } from '@/components/ConversationPanel'
import { CustomizePanel, CustomizeTrigger } from '@/components/customize/CustomizePanel'
import { ThemeSwitch } from '@/components/ThemeSwitch'
import { LandingPage } from '@/components/LandingPage'
import { NewSessionPanel } from '@/components/NewSessionPanel'
import { SessionsListPanel } from '@/components/SessionsListPanel'
import { ThemeProvider } from '@/contexts/theme'
import { api } from '@/api'
import { groupSessionsByAgentType, defaultAgentTypeForGroup } from '@/lib/sessionGroups'
import {
  agentFromNewSessionSearch,
  cwdFromNewSessionSearch,
  isCreateSessionPath,
  isCustomizePath,
  isLaunchPath,
  isNewSessionPath,
  navigateToCustomize,
  navigateToHome,
  navigateToLaunch,
  navigateToNewSession,
  navigateToSession,
  sessionIdFromPath,
} from '@/lib/appUrl'
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
  const [landingOpen, setLandingOpen] = useState(false)
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

  const createSessionFromUrl = useCallback(async (): Promise<string | null> => {
    try {
      const session = await api.sessions.createFromUrl({
        agent: agentFromNewSessionSearch() ?? undefined,
        cwd: cwdFromNewSessionSearch() ?? undefined,
      })
      await loadSessions()
      setSelectedId(session.id)
      setNewSessionOpen(false)
      setCustomizeOpen(false)
      setLandingOpen(false)
      setSessionListGroupId(null)
      setInitialPrompt(undefined)
      setHideInput(false)
      navigateToSession(session.id, true)
      api.settings.update({ lastSessionId: session.id }).catch(() => {})
      return session.id
    } catch {
      navigateToHome(true)
      return null
    }
  }, [loadSessions])

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

      const openCustomize = isCustomizePath()
      const openNewSession = isNewSessionPath()
      const openCreateSession = isCreateSessionPath()
      const openLaunch = isLaunchPath()
      if (openCustomize) {
        setCustomizeOpen(true)
      }
      if (openNewSession) {
        setNewSessionOpen(true)
      }
      if (openLaunch) {
        setLandingOpen(true)
      }

      let nextId: string | null = null

      if (openCreateSession) {
        nextId = await createSessionFromUrl()
        list = await loadSessions()
      } else {
        nextId = sessionIdFromPath() ?? bootstrap.sessionId ?? null
        if (nextId && !list.some((s) => s.id === nextId)) {
          nextId = null
        }

        if (nextId && !openCustomize && !openNewSession && !openLaunch) {
          navigateToSession(nextId, true)
        } else if (!nextId && !openCustomize && !openNewSession && !openCreateSession && !openLaunch) {
          navigateToLaunch(true)
          setLandingOpen(true)
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
  }, [loadSessions, loadAgentTypes, createSessionFromUrl])

  useEffect(() => {
    function onPopState() {
      if (isLaunchPath()) {
        setLandingOpen(true)
        setCustomizeOpen(false)
        setNewSessionOpen(false)
        setSessionListGroupId(null)
        return
      }

      setLandingOpen(false)

      if (isCustomizePath()) {
        setCustomizeOpen(true)
        setNewSessionOpen(false)
        setSessionListGroupId(null)
        return
      }

      setCustomizeOpen(false)

      if (isNewSessionPath()) {
        setNewSessionOpen(true)
        setSessionListGroupId(null)
        return
      }

      if (isCreateSessionPath()) {
        void createSessionFromUrl()
        return
      }

      setNewSessionOpen(false)

      const id = sessionIdFromPath()
      if (!id || !sessionsRef.current.some((s) => s.id === id)) return
      setSelectedId(id)
      setSessionListGroupId(null)
      setInitialPrompt(undefined)
      setHideInput(false)
      api.settings.update({ lastSessionId: id }).catch(() => {})
    }

    window.addEventListener('popstate', onPopState)
    return () => window.removeEventListener('popstate', onPopState)
  }, [createSessionFromUrl])

  const handleSelect = useCallback((id: string) => {
    setCustomizeOpen(false)
    setNewSessionOpen(false)
    setLandingOpen(false)
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
    setLandingOpen(false)
    setSessionListGroupId(null)
    setCustomizeOpen(true)
    navigateToCustomize()
  }, [])

  const handleOpenLaunch = useCallback(() => {
    setCustomizeOpen(false)
    setNewSessionOpen(false)
    setSessionListGroupId(null)
    setLandingOpen(true)
    navigateToLaunch()
  }, [])

  const handleCloseCustomize = useCallback(() => {
    setCustomizeOpen(false)
    if (selectedId) {
      navigateToSession(selectedId, true)
    } else {
      navigateToHome(true)
    }
  }, [selectedId])

  const handleOpenNewSession = useCallback(() => {
    setCustomizeOpen(false)
    setLandingOpen(false)
    setSessionListGroupId(null)
    setNewSessionOpen(true)
    navigateToNewSession()
  }, [])

  const handleCloseNewSession = useCallback(() => {
    setNewSessionOpen(false)
    if (selectedId) {
      navigateToSession(selectedId, true)
    } else {
      navigateToHome(true)
    }
  }, [selectedId])

  const handleOpenNewSessionForGroup = useCallback((groupId: string) => {
    const group = groupSessionsByAgentType(sessions, agentTypes).find((g) => g.id === groupId)
    if (!group) return
    const agentId = defaultAgentTypeForGroup(group, agentTypes)
    setCustomizeOpen(false)
    setLandingOpen(false)
    setSessionListGroupId(null)
    setNewSessionOpen(true)
    navigateToNewSession(agentId ? { agent: agentId } : undefined)
  }, [sessions, agentTypes])

  const handleOpenSessionList = useCallback((groupId: string) => {
    setCustomizeOpen(false)
    setNewSessionOpen(false)
    setLandingOpen(false)
    setSessionListGroupId(groupId)
  }, [])

  const handleCloseSessionList = useCallback(() => {
    setSessionListGroupId(null)
    if (!selectedId) {
      setLandingOpen(true)
      navigateToHome(true)
    }
  }, [selectedId])

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
        navigateToHome(true)
        setLandingOpen(true)
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
        navigateToHome(true)
        setLandingOpen(true)
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
          <button type="button" className="app-brand shrink-0" onClick={handleOpenLaunch}>
            The Loop
          </button>
          {selected && selectedAgent && !customizeOpen && !newSessionOpen && !sessionListGroup && !landingOpen && (
            <>
              <span className="text-muted-foreground/35 shrink-0 select-none" aria-hidden="true">/</span>
              <AgentHeader name={selected.name} agent={selectedAgent} />
            </>
          )}
          <div className="app-header__actions">
            <ThemeSwitch />
            <CustomizeTrigger active={customizeOpen} onOpen={handleOpenCustomize} compact />
          </div>
        </header>
        <div className="app-body">
          <AppSidebar
            sessions={sessions}
            agentTypes={agentTypes}
            selectedId={selectedId}
            newSessionOpen={newSessionOpen}
            sessionListGroupId={sessionListGroupId}
            onSelect={handleSelect}
            onOpenNewSession={handleOpenNewSession}
            onOpenNewSessionForGroup={handleOpenNewSessionForGroup}
            onOpenSessionList={handleOpenSessionList}
            onRename={handleRenameSession}
            onDelete={handleDeleteSession}
          />
          <main className="flex min-h-0 flex-1 overflow-hidden">
            {landingOpen ? (
              <LandingPage
                onNewSession={handleOpenNewSession}
                onCustomize={handleOpenCustomize}
              />
            ) : customizeOpen ? (
              <CustomizePanel
                onClose={handleCloseCustomize}
                onAgentTypesChanged={handleAgentTypesChanged}
              />
            ) : newSessionOpen ? (
              <NewSessionPanel
                agentTypes={agentTypes}
                initialAgentTypeId={agentFromNewSessionSearch()}
                initialWorkingDir={cwdFromNewSessionSearch()}
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
                promptSuggestions={selectedAgent?.promptSuggestions}
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
