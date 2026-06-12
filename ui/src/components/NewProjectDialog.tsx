// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useEffect, useState } from 'react'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { api } from '@/api'
import type { AgentType, CreateProjectRequest, Project } from '@/types'

interface Props {
  open: boolean
  onOpenChange: (open: boolean) => void
  onCreated: (project: Project) => void
}

export function NewProjectDialog({ open, onOpenChange, onCreated }: Props) {
  const [name, setName] = useState('')
  const [workingDir, setWorkingDir] = useState('')
  const [agentType, setAgentType] = useState('')
  const [agentTypes, setAgentTypes] = useState<AgentType[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    api.agentTypes.list().then((types) => {
      setAgentTypes(types)
      if (types.length > 0) setAgentType(types[0].id)
    }).catch(() => {})
  }, [])

  function reset() {
    setName('')
    setWorkingDir('')
    setAgentType(agentTypes[0]?.id ?? '')
    setError('')
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!name.trim() || !agentType) {
      setError('Name and agent type are required.')
      return
    }
    setLoading(true)
    setError('')
    try {
      const req: CreateProjectRequest = { name: name.trim(), workingDir: workingDir.trim(), agentType }
      const project = await api.projects.create(req)
      reset()
      onOpenChange(false)
      onCreated(project)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create project.')
    } finally {
      setLoading(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={(o) => { if (!o) reset(); onOpenChange(o) }}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>New Project</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4 py-2">
          <div className="space-y-1.5">
            <Label htmlFor="name">Name</Label>
            <Input
              id="name"
              placeholder="my-project"
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="workingDir">Working Directory</Label>
            <Input
              id="workingDir"
              placeholder="/path/to/project (optional)"
              value={workingDir}
              onChange={(e) => setWorkingDir(e.target.value)}
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="agentType">Agent Type</Label>
            <Select value={agentType} onValueChange={setAgentType}>
              <SelectTrigger id="agentType" className="w-full">
                <SelectValue placeholder="Select agent type" />
              </SelectTrigger>
              <SelectContent>
                {agentTypes.map((a) => (
                  <SelectItem key={a.id} value={a.id}>
                    {a.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          {error && <p className="text-sm text-destructive">{error}</p>}
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => { reset(); onOpenChange(false) }}>
              Cancel
            </Button>
            <Button type="submit" disabled={loading}>
              {loading ? 'Creating…' : 'Create'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
