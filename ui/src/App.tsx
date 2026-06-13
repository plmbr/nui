// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useCallback, useEffect, useState } from 'react'
import { SidebarProvider, SidebarTrigger } from '@/components/ui/sidebar'
import { TooltipProvider } from '@/components/ui/tooltip'
import { AppSidebar } from '@/components/AppSidebar'
import { ProjectDetails } from '@/components/ProjectDetails'
import { ConversationPanel } from '@/components/ConversationPanel'
import { ThemeProvider } from '@/contexts/theme'
import { api } from '@/api'
import type { Project } from '@/types'

export default function App() {
  const [projects, setProjects] = useState<Project[]>([])
  const [selectedId, setSelectedId] = useState<string | null>(null)

  const loadProjects = useCallback(async () => {
    try {
      const list = await api.projects.list()
      setProjects(list)
    } catch {
      // ignore — backend may not be up during dev
    }
  }, [])

  useEffect(() => {
    loadProjects()
  }, [loadProjects])

  const selected = projects.find((p) => p.id === selectedId) ?? null

  const handleDeleteProject = useCallback(async () => {
    if (!selectedId) return
    await api.projects.delete(selectedId)
    setSelectedId(null)
    await loadProjects()
  }, [selectedId, loadProjects])

  const handleRenameProject = useCallback(async (newName: string) => {
    if (!selectedId) return
    await api.projects.rename(selectedId, newName)
    await loadProjects()
  }, [selectedId, loadProjects])

  return (
    <ThemeProvider>
    <TooltipProvider>
      <SidebarProvider>
        <header className="app-header">
          <SidebarTrigger />
          <span className="font-semibold text-sm">The Loop</span>
          {selected && (
            <>
              <span className="text-muted-foreground text-sm">/</span>
              <span className="text-sm">{selected.name}</span>
            </>
          )}
        </header>
        <div className="app-body">
          <AppSidebar
            projects={projects}
            selectedId={selectedId}
            onSelect={setSelectedId}
            onRefresh={loadProjects}
          />
          <main className="flex-1 overflow-hidden">
            {selected ? (
              <div className="project-layout">
                <div className="project-details-panel">
                  <ProjectDetails
                    project={selected}
                    onRename={handleRenameProject}
                    onDelete={handleDeleteProject}
                  />
                </div>
                <div className="project-chat-panel" key={selected.id}>
                  <ConversationPanel project={selected} />
                </div>
              </div>
            ) : (
              <div className="empty-state">
                Select a project or create a new one.
              </div>
            )}
          </main>
        </div>
      </SidebarProvider>
    </TooltipProvider>
    </ThemeProvider>
  )
}
