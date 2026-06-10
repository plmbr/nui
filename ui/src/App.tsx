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

  return (
    <ThemeProvider>
    <TooltipProvider>
      <SidebarProvider>
        <header className="fixed top-0 left-0 right-0 h-12 z-50 flex items-center px-4 border-b bg-background gap-3 shrink-0">
          <SidebarTrigger />
<span className="font-semibold text-sm">The Loop</span>
          {selected && (
            <>
              <span className="text-muted-foreground text-sm">/</span>
              <span className="text-sm">{selected.name}</span>
            </>
          )}
        </header>
        <div className="flex h-screen w-full overflow-hidden pt-12">
          <AppSidebar
            projects={projects}
            selectedId={selectedId}
            onSelect={setSelectedId}
            onRefresh={loadProjects}
          />
          <main className="flex-1 overflow-hidden">
            {selected ? (
              <div className="flex h-full overflow-hidden">
                <div className="w-72 shrink-0 border-r overflow-auto">
                  <ProjectDetails project={selected} />
                </div>
                <div className="flex-1 overflow-hidden" key={selected.id}>
                  <ConversationPanel project={selected} />
                </div>
              </div>
            ) : (
              <div className="flex items-center justify-center h-full text-muted-foreground text-sm">
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
