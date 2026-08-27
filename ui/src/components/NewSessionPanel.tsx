// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react'
import { ArrowLeft, ChevronRight, Filter, Folder, Search, X } from 'lucide-react'
import { HarnessIcon } from '@/components/HarnessIcon'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { cn } from '@/lib/utils'
import { api } from '@/api'
import { harnessLabel } from '@/lib/agentDisplay'
import {
  buildCustomAgentSourceOptions,
  filterCustomAgentsBySources,
  sortCustomAgentsByName,
} from '@/lib/agentSources'
import {
  collectAgentTags,
  filterAgentsByTags,
} from '@/lib/agentTags'
import {
  orderedBuiltinAgentsForPicker,
  pickNewSessionAgentTypeId,
  selectableAgentTypes,
  defaultUserScopeHarnessConfig,
  showUserScopeOption,
  isNuiAgent,
  isApiBuiltinAgent,
  isCliBuiltinAgent,
  harnessSupportsPermissions,
} from '@/lib/agentTypes'
import { BUILTIN_AGENTS_LABEL, INSTALLED_AGENTS_LABEL } from '@/lib/sessionGroups'
import { TagFilterInput } from '@/components/TagFilterInput'
import { RecentsSection } from '@/components/RecentsSection'
import type { AgentType, CreateSessionRequest, ExtensionInfo, RecentAgentEntry, Session } from '@/types'

interface Props {
  agentTypes: AgentType[]
  sessions: Session[]
  recentSessionIds?: string[]
  recentAgents?: RecentAgentEntry[]
  recentsOpen: boolean
  onRecentsOpenChange: (open: boolean) => void
  initialAgentTypeId?: string | null
  initialWorkingDir?: string | null
  onClose: () => void
  onCreated: (session: Session) => void
  onCreateFromRecentAgent: (entry: RecentAgentEntry) => Promise<void>
  onOpenRecentSession: (sessionId: string) => void
  onRecentsChange: (patch: { recentSessionIds?: string[]; recentAgents?: RecentAgentEntry[] }) => void
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

const BUILTIN_TAG_CHIP_ORDER = ['nui', 'api', 'cli'] as const

function builtinTagChipLabel(tag: string): string {
  if (tag === 'api') return 'API'
  if (tag === 'cli') return 'CLI'
  return tag
}

function orderedBuiltinTagChips(tags: string[]): string[] {
  const set = new Set(tags.filter((tag) => tag !== 'builtin'))
  const ordered: string[] = []
  for (const tag of BUILTIN_TAG_CHIP_ORDER) {
    if (set.has(tag)) {
      ordered.push(tag)
      set.delete(tag)
    }
  }
  for (const tag of [...set].sort((a, b) => a.localeCompare(b, undefined, { sensitivity: 'base' }))) {
    ordered.push(tag)
  }
  return ordered
}

type AgentPane = 'builtin' | 'installed'

export function NewSessionPanel({
  agentTypes,
  sessions,
  recentSessionIds,
  recentAgents,
  recentsOpen,
  onRecentsOpenChange,
  initialAgentTypeId,
  initialWorkingDir,
  onClose,
  onCreated,
  onCreateFromRecentAgent,
  onOpenRecentSession,
  onRecentsChange,
}: Props) {
  const [workingDir, setWorkingDir] = useState(initialWorkingDir ?? '')
  const [selectedId, setSelectedId] = useState('')
  const [customSearch, setCustomSearch] = useState('')
  const [agentPane, setAgentPane] = useState<AgentPane>('builtin')
  const [builtinTag, setBuiltinTag] = useState<string | null>(null)
  const [selectedSourceKeys, setSelectedSourceKeys] = useState<Set<string>>(() => new Set())
  const [selectedTags, setSelectedTags] = useState<string[]>([])
  const [extensions, setExtensions] = useState<ExtensionInfo[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [userScopeHarnessConfig, setUserScopeHarnessConfig] = useState(false)
  const [harnessPermissionsEnabled, setHarnessPermissionsEnabled] = useState(true)
  const [harnessOverride, setHarnessOverride] = useState('')
  const [directorySuggestions, setDirectorySuggestions] = useState<string[]>([])
  const [directoryInputFocused, setDirectoryInputFocused] = useState(false)
  const [activeDirectoryIndex, setActiveDirectoryIndex] = useState(0)
  const suppressDirectoryLookupForValue = useRef<string | null>(null)
  const customAgentsScrollRef = useRef<HTMLDivElement>(null)
  const initialPaneSynced = useRef(false)

  const builtins = useMemo(
    () => agentTypes.filter((a) => a.isBuiltin && !isNuiAgent(a)),
    [agentTypes],
  )
  const orderedBuiltins = useMemo(
    () => orderedBuiltinAgentsForPicker(agentTypes),
    [agentTypes],
  )
  const builtinTagChips = useMemo(
    () => orderedBuiltinTagChips(collectAgentTags(orderedBuiltins)),
    [orderedBuiltins],
  )
  const filteredBuiltins = useMemo(() => {
    if (!builtinTag) return orderedBuiltins
    return filterAgentsByTags(orderedBuiltins, new Set([builtinTag]))
  }, [orderedBuiltins, builtinTag])
  const userDefined = useMemo(
    () => sortCustomAgentsByName(
      selectableAgentTypes(agentTypes).filter((a) => !a.isBuiltin),
    ),
    [agentTypes],
  )

  function selectAgent(id: string) {
    const agent = agentTypes.find((a) => a.id === id)
    if (!agent?.available) return
    setSelectedId(id)
    if (userDefined.some((a) => a.id === id)) setAgentPane('installed')
    else if (builtins.some((a) => a.id === id) || isNuiAgent(id)) setAgentPane('builtin')
  }

  useEffect(() => {
    if (agentTypes.length === 0) return
    const selectable = selectableAgentTypes(agentTypes)
    if (selectable.length === 0) return

    if (initialAgentTypeId && selectable.some((t) => t.id === initialAgentTypeId)) {
      selectAgent(initialAgentTypeId)
      return
    }

    setSelectedId((current) => {
      if (current && selectable.some((t) => t.id === current)) return current
      // Plain /sessions/new always defaults to nui (built-in tab).
      return pickNewSessionAgentTypeId(agentTypes)
    })
  }, [agentTypes, initialAgentTypeId, builtins, userDefined])

  useEffect(() => {
    if (initialPaneSynced.current || !selectedId) return
    initialPaneSynced.current = true
    if (userDefined.some((a) => a.id === selectedId)) setAgentPane('installed')
    else setAgentPane('builtin')
  }, [selectedId, userDefined])

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
    setWorkingDir('')
    setCustomSearch('')
    setSelectedSourceKeys(new Set())
    setSelectedTags([])
    setBuiltinTag(null)
    setAgentPane('builtin')
    setError('')
    setUserScopeHarnessConfig(false)
    setHarnessPermissionsEnabled(true)
    setHarnessOverride('')
    setDirectorySuggestions([])
    setDirectoryInputFocused(false)
    initialPaneSynced.current = false
  }

  function handleClose() {
    reset()
    onClose()
  }

  const selected = agentTypes.find((a) => a.id === selectedId)
  const allowedHarnesses = selected?.allowedHarnesses ?? []
  const showHarnessPicker = !isNuiAgent(selected) && allowedHarnesses.length > 1
  const effectiveHarness = (harnessOverride || selected?.harness || '') as AgentType['harness']
  const selectedForOptions = selected
    ? { ...selected, harness: effectiveHarness || selected.harness }
    : undefined
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
  const sourceFilteredCustom = useMemo(
    () => filterCustomAgentsBySources(userDefined, selectedSourceKeys),
    [userDefined, selectedSourceKeys],
  )
  const availableTags = useMemo(() => collectAgentTags(sourceFilteredCustom), [sourceFilteredCustom])
  const selectedTagSet = useMemo(() => new Set(selectedTags), [selectedTags])
  const filteredCustom = useMemo(() => {
    const byTags = filterAgentsByTags(sourceFilteredCustom, selectedTagSet)
    if (!searchQuery) return byTags
    return byTags.filter((a) => agentMatchesSearch(a, searchQuery))
  }, [searchQuery, sourceFilteredCustom, selectedTagSet])

  useEffect(() => {
    const validKeys = new Set(customSourceOptions.map((option) => option.key))
    setSelectedSourceKeys((current) => {
      const next = new Set([...current].filter((key) => validKeys.has(key)))
      return next.size === current.size ? current : next
    })
  }, [customSourceOptions])

  useEffect(() => {
    const allowed = new Set(availableTags)
    setSelectedTags((current) => {
      const next = current.filter((tag) => allowed.has(tag))
      return next.length === current.length ? current : next
    })
  }, [availableTags])

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
  const showBuiltinPane = orderedBuiltins.length > 0
  const showInstalledBrowse = userDefined.length > 0
  const showingInstalled = agentPane === 'installed' && showInstalledBrowse
  const directoryListOpen = directoryInputFocused && directorySuggestions.length > 0
  const showUserScope = showUserScopeOption(selectedForOptions)
  const showHarnessPermissionsOption = Boolean(
    selectedForOptions
      && !isNuiAgent(selectedForOptions)
      && harnessSupportsPermissions(effectiveHarness || selectedForOptions.harness)
      && selectedForOptions.toolApprovalPolicy !== 'all',
  )

  useEffect(() => {
    const agent = agentTypes.find((a) => a.id === selectedId)
    const allowed = agent?.allowedHarnesses ?? []
    const nextHarness = allowed.length > 1 ? (agent?.harness ?? '') : ''
    setHarnessOverride(nextHarness)
    const forOptions = agent
      ? { ...agent, harness: (nextHarness || agent.harness) as AgentType['harness'] }
      : undefined
    setUserScopeHarnessConfig(defaultUserScopeHarnessConfig(forOptions))
  }, [selectedId, agentTypes])

  useEffect(() => {
    if (!selectedForOptions) return
    setUserScopeHarnessConfig(defaultUserScopeHarnessConfig(selectedForOptions))
  }, [harnessOverride]) // eslint-disable-line react-hooks/exhaustive-deps -- sync checkbox when override harness changes

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

  async function handleRecentAgentClick(entry: RecentAgentEntry) {
    if (loading) return
    setLoading(true)
    setError('')
    try {
      await onCreateFromRecentAgent(entry)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create session.')
    } finally {
      setLoading(false)
    }
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!selectedId) {
      setError('Select an agent type.')
      return
    }
    if (!selected?.available) {
      setError('Selected agent is not available on this system.')
      return
    }
    setLoading(true)
    setError('')
    try {
      const req: CreateSessionRequest = {
        agentType: selectedId,
      }
      if (selected?.workingDirInput) {
        req.workingDir = workingDir.trim()
      }
      const agentConfig: NonNullable<CreateSessionRequest['agentConfig']> = {}
      if (showHarnessPicker && harnessOverride && harnessOverride !== selected.harness) {
        agentConfig.harnessType = harnessOverride
      }
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
      <h1 className="sr-only">New Session</h1>
      <form onSubmit={handleSubmit} className="flex flex-1 flex-col min-h-0">
        <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
          <div className="flex min-h-0 flex-1 flex-col overflow-hidden p-4 md:p-6">
            <div className="customize-tab-content mx-auto flex w-full min-h-0 flex-1 flex-col gap-5">

            {(showBuiltinPane || showInstalledBrowse) && (
            <div className="flex min-h-0 flex-1 flex-col gap-3">
              {!showingInstalled && showBuiltinPane && (
                <div className="flex min-h-0 flex-1 flex-col gap-3" aria-label="Built-in agents">
                  {(builtinTagChips.length > 0 || showInstalledBrowse) && (
                    <div className="flex items-center gap-3 shrink-0">
                      {builtinTagChips.length > 0 && (
                        <div className="flex min-w-0 flex-1 flex-wrap items-center gap-x-2 gap-y-1.5">
                          <Filter className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />
                          <div className="flex flex-wrap gap-1.5" role="group" aria-label="Filter built-in agents by tag">
                            <button
                              type="button"
                              aria-pressed={builtinTag === null}
                              onClick={() => setBuiltinTag(null)}
                              className={cn(
                                'rounded-md px-2.5 py-1 text-xs font-medium transition-colors border',
                                builtinTag === null
                                  ? 'border-primary bg-primary text-primary-foreground'
                                  : 'border-border bg-background text-muted-foreground hover:bg-muted',
                              )}
                            >
                              All
                            </button>
                            {builtinTagChips.map((tag) => {
                              const active = builtinTag === tag
                              return (
                                <button
                                  key={tag}
                                  type="button"
                                  aria-pressed={active}
                                  onClick={() => setBuiltinTag(active ? null : tag)}
                                  className={cn(
                                    'rounded-md px-2.5 py-1 text-xs font-medium transition-colors border',
                                    active
                                      ? 'border-primary bg-primary text-primary-foreground'
                                      : 'border-border bg-background text-muted-foreground hover:bg-muted',
                                  )}
                                >
                                  {builtinTagChipLabel(tag)}
                                </button>
                              )
                            })}
                          </div>
                        </div>
                      )}
                      {showInstalledBrowse && (
                        <Button
                          type="button"
                          variant="ghost"
                          size="sm"
                          className="ml-auto shrink-0 gap-1 text-muted-foreground hover:text-foreground"
                          onClick={() => setAgentPane('installed')}
                        >
                          <span>
                            {INSTALLED_AGENTS_LABEL}
                            <span className="font-normal"> ({userDefined.length})</span>
                          </span>
                          <ChevronRight className="size-4" />
                        </Button>
                      )}
                    </div>
                  )}
                  <div className="min-h-0 flex-1 overflow-y-auto">
                    {filteredBuiltins.length === 0 ? (
                      <p className="text-sm text-muted-foreground py-2">No agents match this filter.</p>
                    ) : (
                      <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
                        {filteredBuiltins.map((agent) => (
                          <BuiltinAgentCard
                            key={agent.id}
                            agent={agent}
                            selected={selectedId === agent.id}
                            onSelect={() => selectAgent(agent.id)}
                          />
                        ))}
                      </div>
                    )}
                  </div>
                </div>
              )}

              {showingInstalled && (
                <div
                  aria-label={INSTALLED_AGENTS_LABEL}
                  className="flex min-h-0 flex-1 flex-col gap-2"
                >
                  <div className="flex items-center gap-1 shrink-0">
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      className="gap-1.5 px-2"
                      onClick={() => setAgentPane('builtin')}
                    >
                      <ArrowLeft className="size-4" />
                      {BUILTIN_AGENTS_LABEL}
                    </Button>
                    <h2 className="text-sm font-medium truncate">
                      {INSTALLED_AGENTS_LABEL}
                      <span className="text-muted-foreground font-normal"> ({userDefined.length})</span>
                    </h2>
                  </div>
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
                  <TagFilterInput
                    availableTags={availableTags}
                    selectedTags={selectedTags}
                    onChange={setSelectedTags}
                  />
                  <div ref={customAgentsScrollRef} className="min-h-0 flex-1 overflow-y-auto">
                    <div className="flex flex-col gap-1.5 pr-1">
                      {filteredCustom.length === 0 ? (
                        <p className="text-sm text-muted-foreground py-2">
                          {searchQuery || selectedSourceKeys.size > 0 || selectedTags.length > 0
                            ? 'No agents match your filters.'
                            : 'No custom agents available.'}
                        </p>
                      ) : (
                        filteredCustom.map((a) => (
                          <AgentCard
                            key={a.id}
                            agent={a}
                            selected={selectedId === a.id}
                            onSelect={() => selectAgent(a.id)}
                          />
                        ))
                      )}
                    </div>
                  </div>
                </div>
              )}

              {!showBuiltinPane && showInstalledBrowse && !showingInstalled && (
                <div className="flex min-h-0 flex-1 flex-col gap-2">
                  <Button
                    type="button"
                    variant="outline"
                    className="w-full justify-between shrink-0"
                    onClick={() => setAgentPane('installed')}
                  >
                    <span>
                      {INSTALLED_AGENTS_LABEL}
                      <span className="text-muted-foreground font-normal"> ({userDefined.length})</span>
                    </span>
                    <ChevronRight className="size-4 text-muted-foreground" />
                  </Button>
                </div>
              )}
            </div>
            )}

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

            {showHarnessPicker && (
              <div className="shrink-0 space-y-2">
                <Label htmlFor="harnessOverride">Harness</Label>
                <select
                  id="harnessOverride"
                  value={harnessOverride || selected?.harness || ''}
                  onChange={(e) => setHarnessOverride(e.target.value)}
                  className="flex h-9 w-full max-w-md rounded-lg border border-input bg-transparent px-3 py-1 text-sm outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50"
                >
                  {allowedHarnesses.map((h) => (
                    <option key={h} value={h}>
                      {harnessLabel(h)}
                      {h === selected?.harness ? ' (default)' : ''}
                    </option>
                  ))}
                </select>
                <p className="text-xs text-muted-foreground leading-snug">
                  Choose which CLI runtime to use for this session. Default is {selected?.harness}.
                </p>
              </div>
            )}

            {showUserScope && (
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

          <div className="shrink-0 border-t px-4 py-3 md:px-6">
            <div className="customize-tab-content mx-auto w-full">
              <RecentsSection
                sessions={sessions}
                agentTypes={agentTypes}
                recentSessionIds={recentSessionIds}
                recentAgents={recentAgents}
                open={recentsOpen}
                onOpenChange={onRecentsOpenChange}
                onRecentAgentClick={(entry) => void handleRecentAgentClick(entry)}
                onRecentSessionClick={onOpenRecentSession}
                onRecentsChange={onRecentsChange}
              />
            </div>
          </div>
        </div>

        <div className="shrink-0 border-t px-4 py-4 md:px-6">
          <div className="customize-tab-content mx-auto flex w-full flex-col items-stretch gap-3 sm:flex-row sm:items-center sm:justify-between sm:gap-4">
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
              <Button type="submit" disabled={loading || !selectedId || !selected?.available}>
                {loading ? 'Creating…' : 'Create Session'}
              </Button>
            </div>
          </div>
        </div>
      </form>
    </div>
  )
}

interface BuiltinAgentCardProps {
  agent: AgentType
  selected: boolean
  onSelect: () => void
}

function BuiltinAgentCard({ agent, selected, onSelect }: BuiltinAgentCardProps) {
  const kindLabel = isNuiAgent(agent)
    ? null
    : isApiBuiltinAgent(agent)
      ? 'API'
      : isCliBuiltinAgent(agent)
        ? 'CLI'
        : null
  return (
    <button
      type="button"
      onClick={onSelect}
      aria-pressed={selected}
      aria-label={agent.label}
      className={cn(
        'flex flex-col items-center gap-1.5 rounded-lg border px-2 py-3 text-center transition-colors',
        selected
          ? 'border-primary bg-primary/5 text-foreground'
          : 'border-border bg-background hover:bg-muted/60',
      )}
    >
      <HarnessIcon harness={agent.harness} provider={agent.provider} agentId={agent.id} size="xl" />
      <span className={cn(
        'text-xs leading-tight',
        selected ? 'font-medium text-foreground' : 'text-muted-foreground',
      )} aria-hidden="true">{agent.label}</span>
      {kindLabel && (
        <span aria-hidden="true" className="rounded-full border border-border/60 bg-muted/50 px-1.5 py-px text-[10px] font-medium leading-tight text-muted-foreground">
          {kindLabel}
        </span>
      )}
    </button>
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
      <HarnessIcon harness={agent.harness} provider={agent.provider} agentId={agent.id} size="lg" className="shrink-0 mt-0.5" />
      <span className="flex min-w-0 flex-1 flex-col gap-1.5">
        <span className="flex items-start gap-2 min-w-0">
          <span className="min-w-0 flex-1">
            <span className="block text-sm font-medium leading-tight">{agent.label}</span>
            {agent.description && (
              <span className="block text-xs text-muted-foreground mt-0.5 leading-snug">
                {agent.description}
              </span>
            )}
          </span>
          <span className="shrink-0 max-w-[40%] truncate rounded px-1.5 py-0.5 text-[10px] font-mono text-muted-foreground bg-muted" title={harnessLabel(agent.harness, agent.sandbox)}>
            {harnessLabel(agent.harness, agent.sandbox)}
          </span>
        </span>
        {agent.tags && agent.tags.length > 0 && (
          <span className="flex w-full flex-wrap gap-1">
            {agent.tags.map((tag) => (
              <span
                key={tag}
                className="rounded px-1.5 py-0.5 text-[10px] font-medium text-muted-foreground bg-muted"
              >
                {tag}
              </span>
            ))}
          </span>
        )}
      </span>
    </button>
  )
}
