// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useEffect, useState } from 'react'
import { MoreHorizontal, Pencil, Trash2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
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
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
} from '@/components/ui/dropdown-menu'
import type { Session } from '@/types'


interface Props {
  session: Session
  onRename: (newName: string) => Promise<void>
  onDelete: () => Promise<void>
}

export function SessionMenu({ session, onRename, onDelete }: Props) {
  const [editing, setEditing] = useState(false)
  const [nameValue, setNameValue] = useState(session.name)
  const [deleteOpen, setDeleteOpen] = useState(false)

  useEffect(() => {
    setNameValue(session.name)
    setEditing(false)
  }, [session.id, session.name])

  async function saveName() {
    const trimmed = nameValue.trim()
    setEditing(false)
    if (trimmed && trimmed !== session.name) {
      await onRename(trimmed)
    }
  }

  return (
    <>
      <div className="flex flex-1 items-center gap-2 min-w-0">
        {editing ? (
          <Input
            className="h-7 w-44 text-sm"
            value={nameValue}
            autoFocus
            onChange={(e) => setNameValue(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') saveName()
              if (e.key === 'Escape') { setEditing(false); setNameValue(session.name) }
            }}
            onBlur={saveName}
          />
        ) : (
          <span className="text-sm font-semibold truncate">{session.name}</span>
        )}
      </div>

      <DropdownMenu>
        <DropdownMenuTrigger
          render={
            <Button
              variant="ghost"
              size="icon-sm"
              className="shrink-0"
              aria-label="Session options"
            />
          }
        >
          <MoreHorizontal className="size-4" />
        </DropdownMenuTrigger>
        <DropdownMenuContent align="start" className="w-64">
          <DropdownMenuItem onClick={() => setEditing(true)}>
            <Pencil className="size-3.5 text-muted-foreground" />
            Rename
          </DropdownMenuItem>

          <DropdownMenuSeparator />

          {/* Session metadata */}
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
