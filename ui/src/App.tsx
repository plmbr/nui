// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useCallback, useEffect, useState } from 'react'
import { SidebarProvider, SidebarTrigger } from '@/components/ui/sidebar'
import { TooltipProvider } from '@/components/ui/tooltip'
import { AppSidebar } from '@/components/AppSidebar'
import { ConversationPanel } from '@/components/ConversationPanel'
import { ThemeProvider } from '@/contexts/theme'
import { api } from '@/api'
import type { Session } from '@/types'

export default function App() {
  const [sessions, setSessions] = useState<Session[]>([])
  const [selectedId, setSelectedId] = useState<string | null>(null)

  const loadSessions = useCallback(async () => {
    try {
      const list = await api.sessions.list()
      setSessions(list)
    } catch {
      // ignore — backend may not be up during dev
    }
  }, [])

  useEffect(() => {
    loadSessions()
  }, [loadSessions])

  const selected = sessions.find((s) => s.id === selectedId) ?? null

  const handleDeleteSession = useCallback(async () => {
    if (!selectedId) return
    await api.sessions.delete(selectedId)
    setSelectedId(null)
    await loadSessions()
  }, [selectedId, loadSessions])

  const handleRenameSession = useCallback(async (newName: string) => {
    if (!selectedId) return
    await api.sessions.rename(selectedId, newName)
    await loadSessions()
  }, [selectedId, loadSessions])

  return (
    <ThemeProvider>
    <TooltipProvider>
      <SidebarProvider>
        <header className="app-header">
          <SidebarTrigger />
          <span className="font-semibold text-sm shrink-0">The Loop</span>
        </header>
        <div className="app-body">
          <AppSidebar
            sessions={sessions}
            selectedId={selectedId}
            onSelect={setSelectedId}
            onRefresh={loadSessions}
          />
          <main className="flex flex-1 overflow-hidden">
            {selected ? (
              <ConversationPanel
                session={selected}
                key={selected.id}
                onRename={handleRenameSession}
                onDelete={handleDeleteSession}
              />
            ) : (
              <div className="empty-state">
                Select a session or create a new one.
              </div>
            )}
          </main>
        </div>
      </SidebarProvider>
    </TooltipProvider>
    </ThemeProvider>
  )
}
