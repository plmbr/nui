// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useEffect, useState } from 'react'
import { Check } from 'lucide-react'
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
import { api } from '@/api'
import type { AgentType, CreateSessionRequest, Session } from '@/types'

interface Props {
  open: boolean
  onOpenChange: (open: boolean) => void
  onCreated: (session: Session) => void
}

function harnessLabel(harness: string, sandbox?: string): string {
  if (sandbox === 'docker') return `${harness} · docker`
  if (sandbox === 'bubblewrap') return `${harness} · bwrap`
  return harness
}

export function NewSessionDialog({ open, onOpenChange, onCreated }: Props) {
  const [name, setName] = useState('')
  const [workingDir, setWorkingDir] = useState('')
  const [selectedId, setSelectedId] = useState('')
  const [agentTypes, setAgentTypes] = useState<AgentType[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!open) return
    api.agentTypes.list().then((types) => {
      setAgentTypes(types)
      if (types.length > 0 && !selectedId) {
        setSelectedId(types[0].id)
      }
    }).catch(() => {})
  }, [open])

  function reset() {
    setName('')
    setWorkingDir('')
    setError('')
    // Keep selectedId so the user's last pick persists across opens.
  }

  const selected = agentTypes.find((a) => a.id === selectedId)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!selectedId) {
      setError('Select an agent type.')
      return
    }
    const sessionName = name.trim() || (selected?.label ?? selectedId)
    setLoading(true)
    setError('')
    try {
      const req: CreateSessionRequest = {
        name: sessionName,
        workingDir: workingDir.trim(),
        agentType: selectedId,
      }
      const session = await api.sessions.create(req)
      reset()
      onOpenChange(false)
      onCreated(session)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create session.')
    } finally {
      setLoading(false)
    }
  }

  const builtins = agentTypes.filter((a) => a.isBuiltin)
  const userDefined = agentTypes.filter((a) => !a.isBuiltin)

  return (
    <Dialog open={open} onOpenChange={(o) => { if (!o) reset(); onOpenChange(o) }}>
      <DialogContent className="sm:max-w-lg flex flex-col max-h-[85vh]">
        <DialogHeader>
          <DialogTitle>New Session</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="flex flex-col min-h-0 flex-1">
          <div className="flex flex-col gap-5 py-1 overflow-y-auto flex-1 px-0.5">

            {/* ── Agent picker ─────────────────────────────────────────── */}
            <div className="space-y-2">
              <Label>Agent</Label>
              <div className="grid grid-cols-1 gap-1.5">
                {builtins.map((a) => (
                  <AgentCard
                    key={a.id}
                    agent={a}
                    selected={selectedId === a.id}
                    onSelect={() => setSelectedId(a.id)}
                  />
                ))}
                {userDefined.length > 0 && (
                  <>
                    <p className="pt-1 text-[10px] uppercase tracking-wide text-muted-foreground font-medium">
                      User-defined
                    </p>
                    {userDefined.map((a) => (
                      <AgentCard
                        key={a.id}
                        agent={a}
                        selected={selectedId === a.id}
                        onSelect={() => setSelectedId(a.id)}
                      />
                    ))}
                  </>
                )}
              </div>
            </div>

            {/* ── Session name ─────────────────────────────────────────── */}
            <div className="space-y-1.5">
              <Label htmlFor="name">
                Name <span className="text-muted-foreground font-normal">(optional)</span>
              </Label>
              <Input
                id="name"
                placeholder={selected?.label ?? 'my-session'}
                value={name}
                onChange={(e) => setName(e.target.value)}
              />
            </div>

            {/* ── Working directory ─────────────────────────────────────── */}
            <div className="space-y-1.5">
              <Label htmlFor="workingDir">
                Working Directory <span className="text-muted-foreground font-normal">(optional)</span>
              </Label>
              <Input
                id="workingDir"
                placeholder="/path/to/project"
                value={workingDir}
                onChange={(e) => setWorkingDir(e.target.value)}
              />
            </div>

            {error && <p className="text-sm text-destructive">{error}</p>}
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => { reset(); onOpenChange(false) }}>
              Cancel
            </Button>
            <Button type="submit" disabled={loading || !selectedId}>
              {loading ? 'Creating…' : 'Create'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

interface AgentCardProps {
  agent: AgentType
  selected: boolean
  onSelect: () => void
}

function AgentCard({ agent, selected, onSelect }: AgentCardProps) {
  return (
    <button
      type="button"
      onClick={onSelect}
      className={[
        'flex items-start gap-3 rounded-lg border px-3 py-2.5 text-left transition-colors',
        selected
          ? 'border-primary bg-primary/5 text-foreground'
          : 'border-border bg-background hover:bg-muted/60',
      ].join(' ')}
    >
      <span className={[
        'mt-0.5 flex size-4 shrink-0 items-center justify-center rounded-full border',
        selected ? 'border-primary bg-primary' : 'border-muted-foreground/40',
      ].join(' ')}>
        {selected && <Check className="size-2.5 text-primary-foreground stroke-[3]" />}
      </span>
      <span className="flex-1 min-w-0">
        <span className="block text-sm font-medium leading-tight">{agent.label}</span>
        {agent.description && (
          <span className="block text-xs text-muted-foreground mt-0.5 leading-snug">
            {agent.description}
          </span>
        )}
      </span>
      <span className="mt-0.5 shrink-0 rounded px-1.5 py-0.5 text-[10px] font-mono text-muted-foreground bg-muted">
        {harnessLabel(agent.harness, agent.sandbox)}
      </span>
    </button>
  )
}
