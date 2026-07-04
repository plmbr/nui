// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react'
import { Check, Folder, Plus, Search, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { api } from '@/api'
import { harnessLabel } from '@/lib/agentDisplay'
import {
  buildCustomAgentSourceOptions,
  filterCustomAgentsBySources,
  sortCustomAgentsByName,
} from '@/lib/agentSources'
import { pickDefaultAgentTypeId, selectableAgentTypes, harnessSupportsUserScope, defaultUserScopeHarnessConfig, agentSupportsHarnessPermissions } from '@/lib/agentTypes'
import type { AgentType, CreateSessionRequest, ExtensionInfo, Session } from '@/types'

interface Props {
  agentTypes: AgentType[]
  initialAgentTypeId?: string | null
  initialWorkingDir?: string | null
  onClose: () => void
  onCreated: (session: Session) => void
}

function agentMatchesSearch(agent: AgentType, query: string): boolean {
  const haystack = [
    agent.label,
    agent.id,
    agent.description ?? '',
    agent.harness,
    agent.sandbox ?? '',
  ].join(' ').toLowerCase()
  return haystack.includes(query)
}

export function NewSessionPanel({ agentTypes, initialAgentTypeId, initialWorkingDir, onClose, onCreated }: Props) {
  const [name, setName] = useState('')
  const [workingDir, setWorkingDir] = useState(initialWorkingDir ?? '')
  const [selectedId, setSelectedId] = useState('')
  const [customSearch, setCustomSearch] = useState('')
  const [selectedSourceKeys, setSelectedSourceKeys] = useState<Set<string>>(() => new Set())
  const [extensions, setExtensions] = useState<ExtensionInfo[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [userScopeHarnessConfig, setUserScopeHarnessConfig] = useState(false)
  const [harnessPermissionsEnabled, setHarnessPermissionsEnabled] = useState(true)
  const [directorySuggestions, setDirectorySuggestions] = useState<string[]>([])
  const [directoryInputFocused, setDirectoryInputFocused] = useState(false)
  const [activeDirectoryIndex, setActiveDirectoryIndex] = useState(0)
  const suppressDirectoryLookupForValue = useRef<string | null>(null)
  const customAgentsScrollRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (agentTypes.length === 0) return
    const selectable = selectableAgentTypes(agentTypes)
    if (selectable.length === 0) return
    if (initialAgentTypeId && selectable.some((t) => t.id === initialAgentTypeId)) {
      setSelectedId(initialAgentTypeId)
      return
    }
    api.settings.get()
      .then((settings) => {
        setSelectedId((current) => {
          if (current && selectable.some((t) => t.id === current)) return current
          return pickDefaultAgentTypeId(agentTypes, settings.lastAgentType)
        })
      })
      .catch(() => {
        setSelectedId((current) => current || pickDefaultAgentTypeId(agentTypes))
      })
  }, [agentTypes, initialAgentTypeId])

  useEffect(() => {
    api.extensions.list()
      .then(setExtensions)
      .catch(() => setExtensions([]))
  }, [])

  useEffect(() => {
    if (initialWorkingDir) {
      setWorkingDir(initialWorkingDir)
    }
  }, [initialWorkingDir])

  useEffect(() => {
    if (!directoryInputFocused || !workingDir.trim()) {
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
  }, [directoryInputFocused, workingDir])

  function reset() {
    setName('')
    setWorkingDir('')
    setCustomSearch('')
    setSelectedSourceKeys(new Set())
    setError('')
    setUserScopeHarnessConfig(false)
    setHarnessPermissionsEnabled(true)
    setDirectorySuggestions([])
    setDirectoryInputFocused(false)
  }

  function handleClose() {
    reset()
    onClose()
  }

  const selected = agentTypes.find((a) => a.id === selectedId)
  const builtins = selectableAgentTypes(agentTypes).filter((a) => a.isBuiltin)
  const userDefined = useMemo(
    () => sortCustomAgentsByName(
      selectableAgentTypes(agentTypes).filter((a) => !a.isBuiltin),
    ),
    [agentTypes],
  )
  const customSourceOptions = useMemo(
    () => buildCustomAgentSourceOptions(userDefined, extensions),
    [userDefined, extensions],
  )
  const allSourceKeys = useMemo(
    () => customSourceOptions.map((option) => option.key),
    [customSourceOptions],
  )
  const allSourcesActive = selectedSourceKeys.size === 0
    || (allSourceKeys.length > 0 && allSourceKeys.every((key) => selectedSourceKeys.has(key)))
  const searchQuery = customSearch.trim().toLowerCase()
  const filteredCustom = useMemo(() => {
    const bySource = filterCustomAgentsBySources(userDefined, selectedSourceKeys)
    if (!searchQuery) return bySource
    return bySource.filter((a) => agentMatchesSearch(a, searchQuery))
  }, [searchQuery, userDefined, selectedSourceKeys])

  useEffect(() => {
    const validKeys = new Set(customSourceOptions.map((option) => option.key))
    setSelectedSourceKeys((current) => {
      const next = new Set([...current].filter((key) => validKeys.has(key)))
      return next.size === current.size ? current : next
    })
  }, [customSourceOptions])

  function toggleSourceFilter(sourceKey: string) {
    setSelectedSourceKeys((current) => {
      const next = new Set(current)
      if (next.has(sourceKey)) next.delete(sourceKey)
      else next.add(sourceKey)
      return next
    })
  }

  function toggleAllSourceFilters() {
    const allOn = allSourceKeys.length > 0 && allSourceKeys.every((key) => selectedSourceKeys.has(key))
    if (allOn) {
      setSelectedSourceKeys(new Set())
      return
    }
    setSelectedSourceKeys(new Set(allSourceKeys))
  }
  const isBasicLoopSelected = builtins.some((a) => a.id === selectedId)
  const directoryListOpen = directoryInputFocused && directorySuggestions.length > 0
  const showUserScopeOption = selected ? harnessSupportsUserScope(selected.harness) : false
  const showHarnessPermissionsOption = selected ? agentSupportsHarnessPermissions(selected) : false

  useEffect(() => {
    const agent = agentTypes.find((a) => a.id === selectedId)
    setUserScopeHarnessConfig(defaultUserScopeHarnessConfig(agent))
  }, [selectedId, agentTypes])

  useLayoutEffect(() => {
    if (!selectedId) return
    if (!userDefined.some((a) => a.id === selectedId)) return
    if (!filteredCustom.some((a) => a.id === selectedId)) return
    const container = customAgentsScrollRef.current
    if (!container) return
    const el = container.querySelector(`[data-agent-id="${CSS.escape(selectedId)}"]`)
    el?.scrollIntoView({ block: 'nearest' })
  }, [selectedId, userDefined, filteredCustom])

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
      const agentConfig: NonNullable<CreateSessionRequest['agentConfig']> = {}
      if (userScopeHarnessConfig) {
        agentConfig.userScopeHarnessConfig = true
      }
      if (showHarnessPermissionsOption) {
        if (harnessPermissionsEnabled) {
          agentConfig.hitlMode = 'interactive'
          agentConfig.harnessPermissions = 'interactive'
        } else {
          agentConfig.hitlMode = 'off'
          agentConfig.harnessPermissions = 'bypass'
        }
      }
      if (Object.keys(agentConfig).length > 0) {
        req.agentConfig = agentConfig
      }
      const session = await api.sessions.create(req)
      api.settings.update({ lastAgentType: selectedId }).catch(() => {})
      reset()
      onCreated(session)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create session.')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="customize-panel flex flex-1 min-h-0 flex-col overflow-hidden">
      <div className="conversation-header justify-between">
        <div className="flex items-center gap-2 min-w-0">
          <Plus className="size-4 shrink-0 text-muted-foreground" />
          <h1 className="text-sm font-semibold truncate">New Session</h1>
        </div>
        <Button variant="ghost" size="sm" onClick={handleClose} aria-label="Close new session panel">
          <X className="size-4" />
        </Button>
      </div>

      <form onSubmit={handleSubmit} className="flex flex-1 flex-col min-h-0">
        <div className="flex flex-1 flex-col min-h-0 overflow-hidden p-6">
          <div className="customize-tab-content mx-auto flex w-full min-h-0 flex-1 flex-col gap-5">

            {(builtins.length > 0 || userDefined.length > 0) && (
            <div className="flex min-h-0 flex-1 flex-col gap-2">
              {builtins.length > 0 && <Label className="shrink-0">Built-in Agents</Label>}
              <div className="flex min-h-0 flex-1 flex-col gap-1.5">
                {builtins.length > 0 && (
                  <div className={[
                    'shrink-0 rounded-lg border px-3 py-2.5 transition-colors',
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

                {userDefined.length > 0 && (
                  <div className="flex min-h-0 flex-1 flex-col gap-2">
                    <Label className="shrink-0">Custom Agents</Label>
                    {customSourceOptions.length > 0 && (
                      <div className="flex flex-wrap items-center gap-x-2 gap-y-1.5 shrink-0">
                        <Label className="shrink-0 text-muted-foreground">Source</Label>
                        <div className="flex flex-wrap gap-1.5" role="group" aria-label="Filter by agent source">
                          <button
                            type="button"
                            aria-pressed={allSourcesActive}
                            onClick={toggleAllSourceFilters}
                            className={[
                              'rounded-md px-2.5 py-1 text-xs font-medium transition-colors border',
                              allSourcesActive
                                ? 'border-primary bg-primary text-primary-foreground'
                                : 'border-border bg-background text-muted-foreground hover:bg-muted',
                            ].join(' ')}
                          >
                            All
                          </button>
                          {customSourceOptions.map((option) => {
                            const active = selectedSourceKeys.has(option.key)
                            return (
                              <button
                                key={option.key}
                                type="button"
                                aria-pressed={active}
                                onClick={() => toggleSourceFilter(option.key)}
                                className={[
                                  'rounded-md px-2.5 py-1 text-xs font-medium transition-colors border',
                                  active
                                    ? 'border-primary bg-primary text-primary-foreground'
                                    : 'border-border bg-background text-muted-foreground hover:bg-muted',
                                ].join(' ')}
                              >
                                {option.label}
                              </button>
                            )
                          })}
                        </div>
                      </div>
                    )}
                    <div className="relative shrink-0">
                      <Search className="pointer-events-none absolute left-2.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
                      <Input
                        value={customSearch}
                        onChange={(e) => setCustomSearch(e.target.value)}
                        placeholder="Search by name or description…"
                        className="pl-8 pr-8"
                        aria-label="Search custom agents"
                      />
                      {customSearch && (
                        <Button
                          type="button"
                          variant="ghost"
                          size="icon-sm"
                          className="absolute right-1 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                          onClick={() => setCustomSearch('')}
                          aria-label="Clear search"
                        >
                          <X className="size-3.5" />
                        </Button>
                      )}
                    </div>
                    <div ref={customAgentsScrollRef} className="min-h-0 flex-1 overflow-y-auto">
                      <div className="flex flex-col gap-1.5 pr-1">
                        {filteredCustom.length === 0 ? (
                          <p className="text-sm text-muted-foreground py-2">
                            {searchQuery || selectedSourceKeys.size > 0
                              ? 'No agents match your filters.'
                              : 'No custom agents available.'}
                          </p>
                        ) : (
                          filteredCustom.map((a) => (
                            <AgentCard
                              key={a.id}
                              agent={a}
                              selected={selectedId === a.id}
                              onSelect={() => setSelectedId(a.id)}
                            />
                          ))
                        )}
                      </div>
                    </div>
                  </div>
                )}
              </div>
            </div>
            )}

            <div className="shrink-0 space-y-1.5">
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

            {selected?.workingDirInput && (
              <div className="shrink-0 space-y-1.5">
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

            {showUserScopeOption && (
              <div className="shrink-0 flex items-start gap-2">
                <input
                  id="userScopeHarnessConfig"
                  type="checkbox"
                  checked={userScopeHarnessConfig}
                  onChange={(e) => setUserScopeHarnessConfig(e.target.checked)}
                  className="mt-1 size-4 shrink-0 rounded border border-input"
                />
                <div className="space-y-1">
                  <Label htmlFor="userScopeHarnessConfig" className="cursor-pointer">
                    User-scope harness config
                  </Label>
                  <p className="text-xs text-muted-foreground leading-snug">
                    Also load your harness user and project settings from the working directory.
                    ADL MCP servers are still merged in when supported.
                  </p>
                </div>
              </div>
            )}

            {showHarnessPermissionsOption && (
              <div className="shrink-0 flex items-start gap-2">
                <input
                  id="harnessPermissionsEnabled"
                  type="checkbox"
                  checked={harnessPermissionsEnabled}
                  onChange={(e) => setHarnessPermissionsEnabled(e.target.checked)}
                  className="mt-1 size-4 shrink-0 rounded border border-input"
                />
                <div className="space-y-1">
                  <Label htmlFor="harnessPermissionsEnabled" className="cursor-pointer">
                    Tool approvals
                  </Label>
                  <p className="text-xs text-muted-foreground leading-snug">
                    Human in the loop: ask questions via prompt cards and require approval before
                    sensitive tools (bash, file writes, etc.).
                  </p>
                </div>
              </div>
            )}

            {error && <p className="shrink-0 text-sm text-destructive">{error}</p>}
          </div>
        </div>

        <div className="shrink-0 border-t px-6 py-4">
          <div className="customize-tab-content mx-auto flex w-full items-center justify-between gap-4">
            <div className="min-w-0">
              {selected ? (
                <p className="text-sm truncate">
                  <span className="text-muted-foreground">Agent:</span>{' '}
                  <span className="font-medium">{selected.label}</span>
                </p>
              ) : (
                <p className="text-sm text-muted-foreground">Select an agent</p>
              )}
            </div>
            <div className="flex shrink-0 gap-2">
              <Button type="button" variant="outline" onClick={handleClose}>
                Cancel
              </Button>
              <Button type="submit" disabled={loading || !selectedId}>
                {loading ? 'Creating…' : 'Create Session'}
              </Button>
            </div>
          </div>
        </div>
      </form>
    </div>
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
      data-agent-id={agent.id}
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
