// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useEffect, useState } from 'react'
import { FolderOpen, MoreHorizontal, Pencil, Plus, Trash2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuAction,
  SidebarMenuButton,
  SidebarMenuItem,
} from '@/components/ui/sidebar'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { NewSessionDialog } from '@/components/NewSessionDialog'
import { SettingsSheet } from '@/components/SettingsSheet'
import type { Session } from '@/types'

interface Props {
  sessions: Session[]
  selectedId: string | null
  onSelect: (id: string) => void
  onRefresh: () => void
  onRename: (id: string, newName: string) => Promise<void>
  onDelete: (id: string) => Promise<void>
}

interface SessionListItemProps {
  session: Session
  isActive: boolean
  onSelect: () => void
  onRename: (newName: string) => Promise<void>
  onDelete: () => Promise<void>
}

function SessionListItem({ session, isActive, onSelect, onRename, onDelete }: SessionListItemProps) {
  const [renameOpen, setRenameOpen] = useState(false)
  const [nameValue, setNameValue] = useState(session.name)
  const [deleteOpen, setDeleteOpen] = useState(false)

  useEffect(() => {
    setNameValue(session.name)
  }, [session.name])

  async function saveRename() {
    const trimmed = nameValue.trim()
    setRenameOpen(false)
    if (trimmed && trimmed !== session.name) {
      await onRename(trimmed)
    }
  }

  return (
    <>
      <SidebarMenuItem>
        <SidebarMenuButton isActive={isActive} onClick={onSelect} tooltip={session.name}>
          <FolderOpen className="size-4 shrink-0" />
          <span className="truncate">{session.name}</span>
        </SidebarMenuButton>
        <DropdownMenu>
          <DropdownMenuTrigger
            render={<SidebarMenuAction showOnHover aria-label="Session options" />}
          >
            <MoreHorizontal className="size-4" />
          </DropdownMenuTrigger>
          <DropdownMenuContent align="start" className="w-64">
            <DropdownMenuItem onClick={() => { setNameValue(session.name); setRenameOpen(true) }}>
              <Pencil className="size-3.5 text-muted-foreground" />
              Rename
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <div className="px-2 py-1.5 space-y-2.5">
              <div>
                <p className="text-[10px] uppercase tracking-wide text-muted-foreground font-medium mb-0.5">Working Directory</p>
                <p className="text-xs break-all leading-snug">{session.workingDir || '(server working directory)'}</p>
              </div>
              <div>
                <p className="text-[10px] uppercase tracking-wide text-muted-foreground font-medium mb-0.5">Agent</p>
                <p className="text-xs">{session.agentType}</p>
              </div>
              <div>
                <p className="text-[10px] uppercase tracking-wide text-muted-foreground font-medium mb-0.5">Created</p>
                <p className="text-xs">{new Date(session.createdAt).toLocaleString()}</p>
              </div>
            </div>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              className="text-destructive data-highlighted:bg-destructive/10 data-highlighted:text-destructive"
              onClick={() => setDeleteOpen(true)}
            >
              <Trash2 className="size-3.5" />
              Delete Session
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </SidebarMenuItem>

      <Dialog open={renameOpen} onOpenChange={setRenameOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Rename session</DialogTitle>
          </DialogHeader>
          <Input
            value={nameValue}
            autoFocus
            onChange={(e) => setNameValue(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') saveRename()
              if (e.key === 'Escape') setRenameOpen(false)
            }}
          />
          <DialogFooter>
            <Button variant="outline" onClick={() => setRenameOpen(false)}>Cancel</Button>
            <Button onClick={saveRename}>Rename</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={deleteOpen} onOpenChange={setDeleteOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete session?</DialogTitle>
            <DialogDescription>
              This will permanently delete <strong>{session.name}</strong> and its associated chat
              history. This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteOpen(false)}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={async () => {
                setDeleteOpen(false)
                await onDelete()
              }}
            >
              Delete
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}

export function AppSidebar({ sessions, selectedId, onSelect, onRefresh, onRename, onDelete }: Props) {
  const [dialogOpen, setDialogOpen] = useState(false)

  return (
    <>
      <Sidebar collapsible="offcanvas" className="pt-12">
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
                <SessionListItem
                  key={s.id}
                  session={s}
                  isActive={s.id === selectedId}
                  onSelect={() => onSelect(s.id)}
                  onRename={(newName) => onRename(s.id, newName)}
                  onDelete={() => onDelete(s.id)}
                />
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
