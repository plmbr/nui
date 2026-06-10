// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useState } from 'react'
import { FolderOpen, Plus } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Sidebar,
  SidebarContent,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from '@/components/ui/sidebar'
import { NewProjectDialog } from '@/components/NewProjectDialog'
import type { Project } from '@/types'

interface Props {
  projects: Project[]
  selectedId: string | null
  onSelect: (id: string) => void
  onRefresh: () => void
}

export function AppSidebar({ projects, selectedId, onSelect, onRefresh }: Props) {
  const [dialogOpen, setDialogOpen] = useState(false)

  return (
    <>
      <Sidebar collapsible="icon" className="pt-12">
        <SidebarHeader className="p-3">
          <Button
            size="sm"
            className="w-full justify-start gap-2"
            onClick={() => setDialogOpen(true)}
          >
            <Plus className="size-4 shrink-0" />
            <span className="group-data-[collapsible=icon]:hidden">New Project</span>
          </Button>
        </SidebarHeader>
        <SidebarContent>
          <ScrollArea className="flex-1">
            <SidebarMenu className="px-2">
              {projects.map((p) => (
                <SidebarMenuItem key={p.id}>
                  <SidebarMenuButton
                    isActive={p.id === selectedId}
                    onClick={() => onSelect(p.id)}
                    tooltip={p.name}
                  >
                    <FolderOpen className="size-4 shrink-0" />
                    <span className="truncate">{p.name}</span>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              ))}
              {projects.length === 0 && (
                <p className="px-2 py-4 text-xs text-muted-foreground group-data-[collapsible=icon]:hidden">
                  No projects yet.
                </p>
              )}
            </SidebarMenu>
          </ScrollArea>
        </SidebarContent>
      </Sidebar>

      <NewProjectDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        onCreated={(project) => { onRefresh(); onSelect(project.id) }}
      />
    </>
  )
}
