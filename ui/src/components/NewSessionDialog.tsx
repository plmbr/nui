// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useEffect, useRef, useState } from 'react'
import { Check, Folder } from 'lucide-react'
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

const HARNESS_LABELS: Record<string, string> = {
  'claude-code': 'Claude Code',
  'pi': 'Pi',
  'codex': 'Codex',
  'opencode': 'OpenCode',
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
  const [directorySuggestions, setDirectorySuggestions] = useState<string[]>([])
  const [directoryInputFocused, setDirectoryInputFocused] = useState(false)
  const [activeDirectoryIndex, setActiveDirectoryIndex] = useState(0)
  const suppressDirectoryLookupForValue = useRef<string | null>(null)

  useEffect(() => {
    if (!open) return
    Promise.all([
      api.agentTypes.list(),
      api.settings.get().catch(() => ({ theme: 'light' as const })),
    ]).then(([types, settings]) => {
      setAgentTypes(types)
      if (types.length === 0) return
      const preferred = settings.lastAgentType
        ? types.find((t) => t.id === settings.lastAgentType)
        : undefined
      setSelectedId((current) => {
        if (current && types.some((t) => t.id === current)) return current
        return preferred?.id ?? types[0].id
      })
    }).catch(() => {})
  }, [open])

  useEffect(() => {
    if (!open || !directoryInputFocused || !workingDir.trim()) {
      setDirectorySuggestions([])
      return
    }
    if (suppressDirectoryLookupForValue.current === workingDir) {
      suppressDirectoryLookupForValue.current = null
      return
    }
    suppressDirectoryLookupForValue.current = null

    const controller = new AbortController()
    const timeout = window.setTimeout(() => {
      api.directories.suggest(workingDir, controller.signal).then(({ directories }) => {
        setDirectorySuggestions(directories)
        setActiveDirectoryIndex(0)
      }).catch((err) => {
        if (!(err instanceof DOMException && err.name === 'AbortError')) {
          setDirectorySuggestions([])
        }
      })
    }, 150)

    return () => {
      window.clearTimeout(timeout)
      controller.abort()
    }
  }, [directoryInputFocused, open, workingDir])

  function reset() {
    setName('')
    setWorkingDir('')
    setError('')
    setDirectorySuggestions([])
    setDirectoryInputFocused(false)
    // Keep selectedId so the user's last pick persists across opens.
  }

  const selected = agentTypes.find((a) => a.id === selectedId)
  const builtins = agentTypes.filter((a) => a.isBuiltin && a.available)
  const userDefined = agentTypes.filter((a) => !a.isBuiltin)
  const isBasicLoopSelected = builtins.some((a) => a.id === selectedId)
  const directoryListOpen = directoryInputFocused && directorySuggestions.length > 0

  function selectDirectory(path: string) {
    suppressDirectoryLookupForValue.current = path
    setWorkingDir(path)
    setDirectorySuggestions([])
    setActiveDirectoryIndex(0)
  }

  function handleDirectoryKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (!directoryListOpen) return
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      setActiveDirectoryIndex((current) => (current + 1) % directorySuggestions.length)
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      setActiveDirectoryIndex((current) => (current - 1 + directorySuggestions.length) % directorySuggestions.length)
    } else if (e.key === 'Enter' || e.key === 'Tab') {
      e.preventDefault()
      selectDirectory(directorySuggestions[activeDirectoryIndex])
    } else if (e.key === 'Escape') {
      e.preventDefault()
      setDirectorySuggestions([])
    }
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!selectedId) {
      setError('Select an agent type.')
      return
    }
    const displayLabel = selected?.label ?? selectedId
    const sessionName = name.trim() || displayLabel
    setLoading(true)
    setError('')
    try {
      const req: CreateSessionRequest = {
        name: sessionName,
        workingDir: workingDir.trim(),
        agentType: selectedId,
      }
      const session = await api.sessions.create(req)
      api.settings.update({ lastAgentType: selectedId }).catch(() => {})
      reset()
      onOpenChange(false)
      onCreated(session)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create session.')
    } finally {
      setLoading(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={(o) => { if (!o) reset(); onOpenChange(o) }}>
      <DialogContent className="sm:max-w-2xl flex flex-col max-h-[85vh]">
        <DialogHeader>
          <DialogTitle>New Session</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="flex flex-col min-h-0 flex-1">
          <div className="flex flex-col gap-5 py-1 overflow-y-auto flex-1 px-0.5">

            {/* ── Agent picker ─────────────────────────────────────────── */}
            <div className="space-y-2">
              <Label>Built-in Agents</Label>
              <div className="grid grid-cols-1 gap-1.5">

                {/* Basic Loop — single card with 4 harness pills */}
                {builtins.length > 0 && (
                  <div className={[
                    'rounded-lg border px-3 py-2.5 transition-colors',
                    isBasicLoopSelected
                      ? 'border-primary bg-primary/5'
                      : 'border-border bg-background',
                  ].join(' ')}>
                    <span className="block text-sm font-medium leading-tight mb-2">Standard</span>
                    <div className="flex gap-1.5 flex-wrap">
                      {builtins.map((a) => (
                        <button
                          key={a.id}
                          type="button"
                          onClick={() => setSelectedId(a.id)}
                          className={[
                            'rounded-md px-2.5 py-1 text-xs font-medium transition-colors border',
                            selectedId === a.id
                              ? 'border-primary bg-primary text-primary-foreground'
                              : 'border-border bg-background text-muted-foreground hover:bg-muted',
                          ].join(' ')}
                        >
                          {a.label}
                        </button>
                      ))}
                    </div>
                  </div>
                )}

                {/* Custom agents */}
                {userDefined.length > 0 && (
                  <>
                    <Label className="mt-2">Custom Agents</Label>
                    <div className="flex flex-col gap-1.5 max-h-48 overflow-y-auto">
                      {userDefined.map((a) => (
                        <AgentCard
                          key={a.id}
                          agent={a}
                          selected={selectedId === a.id}
                          onSelect={() => setSelectedId(a.id)}
                        />
                      ))}
                    </div>
                  </>
                )}
              </div>
              {selected && (
                <div className="rounded-lg border px-3 py-2 bg-muted/30">
                  <span className="block text-sm font-medium leading-tight">{selected.label}</span>
                  {selected.description && (
                    <span className="block text-xs text-muted-foreground mt-0.5 leading-snug">
                      {selected.description}
                    </span>
                  )}
                </div>
              )}
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
            {selected?.workingDirInput && (
            <div className="space-y-1.5">
              <Label htmlFor="workingDir">
                Working Directory <span className="text-muted-foreground font-normal">(optional)</span>
              </Label>
              <Input
                id="workingDir"
                placeholder="/path/to/project"
                value={workingDir}
                onChange={(e) => {
                  setWorkingDir(e.target.value)
                  setActiveDirectoryIndex(0)
                }}
                onFocus={() => setDirectoryInputFocused(true)}
                onBlur={() => {
                  setDirectoryInputFocused(false)
                  setDirectorySuggestions([])
                }}
                onKeyDown={handleDirectoryKeyDown}
                role="combobox"
                aria-autocomplete="list"
                aria-expanded={directoryListOpen}
                aria-controls="workingDir-suggestions"
                aria-activedescendant={directoryListOpen ? `workingDir-option-${activeDirectoryIndex}` : undefined}
              />
              {directoryListOpen && (
                <div
                  id="workingDir-suggestions"
                  role="listbox"
                  aria-label="Working directory suggestions"
                  className="max-h-44 overflow-y-auto rounded-lg border bg-popover p-1 text-popover-foreground shadow-md"
                >
                  {directorySuggestions.map((path, index) => (
                    <div
                      id={`workingDir-option-${index}`}
                      key={path}
                      role="option"
                      aria-selected={index === activeDirectoryIndex}
                      title={path}
                      className={[
                        'flex cursor-default items-center gap-2 rounded-md px-2 py-1.5 text-sm outline-none',
                        index === activeDirectoryIndex ? 'bg-accent text-accent-foreground' : '',
                      ].join(' ')}
                      onMouseDown={(e) => {
                        e.preventDefault()
                        selectDirectory(path)
                      }}
                      onMouseEnter={() => setActiveDirectoryIndex(index)}
                    >
                      <Folder className="size-4 shrink-0 text-muted-foreground" />
                      <span className="truncate font-mono text-xs">{path}</span>
                    </div>
                  ))}
                </div>
              )}
            </div>
            )}

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
