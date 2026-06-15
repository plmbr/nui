// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useState } from 'react'
import { FolderOpen, Plus } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from '@/components/ui/sidebar'
import { NewSessionDialog } from '@/components/NewSessionDialog'
import { SettingsSheet } from '@/components/SettingsSheet'
import type { Session } from '@/types'

interface Props {
  sessions: Session[]
  selectedId: string | null
  onSelect: (id: string) => void
  onRefresh: () => void
}

export function AppSidebar({ sessions, selectedId, onSelect, onRefresh }: Props) {
  const [dialogOpen, setDialogOpen] = useState(false)

  return (
    <>
      <Sidebar collapsible="icon" className="pt-12">
        <SidebarHeader className="p-3">
          <Button
            variant="secondary"
            size="sm"
            className="w-full justify-start gap-2"
            onClick={() => setDialogOpen(true)}
          >
            <Plus className="size-4 shrink-0" />
            <span className="group-data-[collapsible=icon]:hidden">New Session</span>
          </Button>
        </SidebarHeader>
        <SidebarContent>
          <ScrollArea className="flex-1">
            <SidebarMenu className="px-2">
              {sessions.map((s) => (
                <SidebarMenuItem key={s.id}>
                  <SidebarMenuButton
                    isActive={s.id === selectedId}
                    onClick={() => onSelect(s.id)}
                    tooltip={s.name}
                  >
                    <FolderOpen className="size-4 shrink-0" />
                    <span className="truncate">{s.name}</span>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              ))}
              {sessions.length === 0 && (
                <p className="px-2 py-4 text-xs text-muted-foreground group-data-[collapsible=icon]:hidden">
                  No sessions yet.
                </p>
              )}
            </SidebarMenu>
          </ScrollArea>
        </SidebarContent>
        <SidebarFooter className="p-2">
          <SettingsSheet />
        </SidebarFooter>
      </Sidebar>

      <NewSessionDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        onCreated={(session) => { onRefresh(); onSelect(session.id) }}
      />
    </>
  )
}
