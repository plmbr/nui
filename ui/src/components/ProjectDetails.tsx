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
import type { Project } from '@/types'

function formatAgentType(id: string) {
  return id.split('-').map((w) => w.charAt(0).toUpperCase() + w.slice(1)).join(' ')
}

interface Props {
  project: Project
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

export function ProjectDetails({ project, onRename, onDelete }: Props) {
  const [editingName, setEditingName] = useState(false)
  const [nameValue, setNameValue] = useState(project.name)
  const [deleteOpen, setDeleteOpen] = useState(false)
  const created = new Date(project.createdAt).toLocaleString()

  function startEdit() {
    setNameValue(project.name)
    setEditingName(true)
  }

  async function saveName() {
    const trimmed = nameValue.trim()
    setEditingName(false)
    if (trimmed && trimmed !== project.name) {
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
            <h2 className="text-xl font-semibold">{project.name}</h2>
            <button
              className="rename-btn"
              onClick={startEdit}
              aria-label="Rename project"
            >
              <Pencil className="size-4" />
            </button>
          </div>
        )}
        <p className="text-sm text-muted-foreground mt-1">Project details</p>
      </div>
      <Separator />
      <div className="flex flex-col gap-5">
        <Field
          icon={<FolderOpen className="size-4" />}
          label="Working Directory"
          value={project.workingDir || '(server working directory)'}
        />
        <Field
          icon={<Bot className="size-4" />}
          label="Agent Type"
          value={formatAgentType(project.agentType)}
        />
        {project.agentType === 'docker' && project.agentConfig && (
          <Field
            icon={<Container className="size-4" />}
            label="Docker Image"
            value={`${project.agentConfig.image as string} (port ${project.agentConfig.containerPort as number})`}
          />
        )}
        {project.agentType === 'remote' && project.agentConfig && (
          <Field
            icon={<Globe className="size-4" />}
            label="Remote Address"
            value={`${project.agentConfig.host as string}:${project.agentConfig.port as number}`}
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
        Delete Project
      </Button>

      <Dialog open={deleteOpen} onOpenChange={(open) => setDeleteOpen(open)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete project?</DialogTitle>
            <DialogDescription>
              This will permanently delete <strong>{project.name}</strong> and its associated chat
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
