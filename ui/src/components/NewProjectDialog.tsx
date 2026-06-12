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
  const [dockerImage, setDockerImage] = useState('')
  const [dockerContainerPort, setDockerContainerPort] = useState('')
  const [remoteHost, setRemoteHost] = useState('')
  const [remotePort, setRemotePort] = useState('')
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
    setDockerImage('')
    setDockerContainerPort('')
    setRemoteHost('')
    setRemotePort('')
    setError('')
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!name.trim() || !agentType) {
      setError('Name and agent type are required.')
      return
    }
    if (agentType === 'docker' && (!dockerImage.trim() || !dockerContainerPort.trim())) {
      setError('Docker agent requires an image and container port.')
      return
    }
    if (agentType === 'remote' && (!remoteHost.trim() || !remotePort.trim())) {
      setError('Remote agent requires a host and port.')
      return
    }

    let agentConfig: Record<string, unknown> | undefined
    if (agentType === 'docker') {
      agentConfig = { image: dockerImage.trim(), containerPort: Number(dockerContainerPort) }
    } else if (agentType === 'remote') {
      agentConfig = { host: remoteHost.trim(), port: Number(remotePort) }
    }

    setLoading(true)
    setError('')
    try {
      const req: CreateProjectRequest = { name: name.trim(), workingDir: workingDir.trim(), agentType, agentConfig }
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

          {agentType === 'docker' && (
            <>
              <div className="space-y-1.5">
                <Label htmlFor="dockerImage">Docker Image</Label>
                <Input
                  id="dockerImage"
                  placeholder="my-agent:latest"
                  value={dockerImage}
                  onChange={(e) => setDockerImage(e.target.value)}
                />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="dockerContainerPort">Container Port</Label>
                <Input
                  id="dockerContainerPort"
                  type="number"
                  placeholder="9090"
                  value={dockerContainerPort}
                  onChange={(e) => setDockerContainerPort(e.target.value)}
                />
              </div>
            </>
          )}

          {agentType === 'remote' && (
            <div className="flex gap-3">
              <div className="flex-1 space-y-1.5">
                <Label htmlFor="remoteHost">Host</Label>
                <Input
                  id="remoteHost"
                  placeholder="127.0.0.1"
                  value={remoteHost}
                  onChange={(e) => setRemoteHost(e.target.value)}
                />
              </div>
              <div className="w-28 space-y-1.5">
                <Label htmlFor="remotePort">Port</Label>
                <Input
                  id="remotePort"
                  type="number"
                  placeholder="9000"
                  value={remotePort}
                  onChange={(e) => setRemotePort(e.target.value)}
                />
              </div>
            </div>
          )}

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
