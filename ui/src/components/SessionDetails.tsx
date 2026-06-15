// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useState } from 'react'
import { FolderOpen, Bot, Calendar, Pencil, Trash2, Container, Globe } from 'lucide-react'
import { Separator } from '@/components/ui/separator'
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
import type { Session } from '@/types'

function formatAgentType(id: string) {
  return id.split('-').map((w) => w.charAt(0).toUpperCase() + w.slice(1)).join(' ')
}

interface Props {
  session: Session
  onRename: (newName: string) => Promise<void>
  onDelete: () => Promise<void>
}

function Field({ icon, label, value }: { icon: React.ReactNode; label: string; value: string }) {
  return (
    <div className="project-field">
      <div className="project-field-icon">{icon}</div>
      <div>
        <p className="project-field-label">{label}</p>
        <p className="project-field-value">{value}</p>
      </div>
    </div>
  )
}

export function SessionDetails({ session, onRename, onDelete }: Props) {
  const [editingName, setEditingName] = useState(false)
  const [nameValue, setNameValue] = useState(session.name)
  const [deleteOpen, setDeleteOpen] = useState(false)
  const created = new Date(session.createdAt).toLocaleString()

  function startEdit() {
    setNameValue(session.name)
    setEditingName(true)
  }

  async function saveName() {
    const trimmed = nameValue.trim()
    setEditingName(false)
    if (trimmed && trimmed !== session.name) {
      await onRename(trimmed)
    }
  }

  return (
    <div className="project-details">
      <div>
        {editingName ? (
          <Input
            className="text-base font-semibold"
            value={nameValue}
            autoFocus
            onChange={(e) => setNameValue(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') saveName()
              if (e.key === 'Escape') setEditingName(false)
            }}
            onBlur={saveName}
          />
        ) : (
          <div className="project-name-row">
            <h2 className="text-xl font-semibold">{session.name}</h2>
            <button
              className="rename-btn"
              onClick={startEdit}
              aria-label="Rename session"
            >
              <Pencil className="size-4" />
            </button>
          </div>
        )}
        <p className="text-sm text-muted-foreground mt-1">Session details</p>
      </div>
      <Separator />
      <div className="flex flex-col gap-5">
        <Field
          icon={<FolderOpen className="size-4" />}
          label="Working Directory"
          value={session.workingDir || '(server working directory)'}
        />
        <Field
          icon={<Bot className="size-4" />}
          label="Agent Type"
          value={formatAgentType(session.agentType)}
        />
        {session.agentType === 'docker' && session.agentConfig && (
          <Field
            icon={<Container className="size-4" />}
            label="Docker Image"
            value={`${session.agentConfig!.image as string} (port ${session.agentConfig!.containerPort as number})`}
          />
        )}
        {session.agentType === 'remote' && session.agentConfig && (
          <Field
            icon={<Globe className="size-4" />}
            label="Remote Address"
            value={`${session.agentConfig!.host as string}:${session.agentConfig!.port as number}`}
          />
        )}
        <Field
          icon={<Calendar className="size-4" />}
          label="Created"
          value={created}
        />
      </div>
      <Separator />
      <Button
        variant="destructive"
        size="sm"
        className="w-fit"
        onClick={() => setDeleteOpen(true)}
      >
        <Trash2 className="size-4 mr-2" />
        Delete Session
      </Button>

      <Dialog open={deleteOpen} onOpenChange={(open) => setDeleteOpen(open)}>
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
    </div>
  )
}
