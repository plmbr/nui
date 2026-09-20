// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react'
import { Filter, Folder, X } from 'lucide-react'
import { HarnessIcon } from '@/components/HarnessIcon'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { SearchInput } from '@/components/SearchInput'
import { cn } from '@/lib/utils'
import { api } from '@/api'
import { harnessLabel } from '@/lib/agentDisplay'
import { fuzzyMatchScore } from '@/lib/fuzzyMatch'
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
  showToolApprovalsOption,
  isNuiAgent,
  isApiBuiltinAgent,
  isCliBuiltinAgent,
  harnessSupportsPermissions,
} from '@/lib/agentTypes'
import { BUILTIN_AGENTS_LABEL, INSTALLED_AGENTS_LABEL } from '@/lib/sessionGroups'
import { TagFilterInput } from '@/components/TagFilterInput'
import type { AgentType, CreateSessionRequest, ExtensionInfo, Session } from '@/types'

const BUILTIN_AGENT_SOURCE = 'builtin'

interface Props {
  agentTypes: AgentType[]
  initialAgentTypeId?: string | null
  initialWorkingDir?: string | null
  onClose: () => void
  onCreated: (session: Session) => void
}

function agentSearchHaystack(agent: AgentType): string {
  return [
    agent.label,
    agent.id,
    agent.description ?? '',
    agent.harness,
    agent.sandbox ?? '',
    ...(agent.tags ?? []),
  ].join(' ')
}

function agentSearchScore(agent: AgentType, query: string): number {
  const labelScore = fuzzyMatchScore(query, agent.label)
  const idScore = fuzzyMatchScore(query, agent.id)
  const fullScore = fuzzyMatchScore(query, agentSearchHaystack(agent))
  return Math.max(labelScore * 1.25, idScore * 1.1, fullScore)
}

function rankAgentsBySearch(agents: AgentType[], query: string): AgentType[] {
  if (!query) return agents
  return agents
    .map((agent) => ({ agent, score: agentSearchScore(agent, query) }))
    .filter(({ score }) => score > 0)
    .sort((a, b) => b.score - a.score || a.agent.label.localeCompare(b.agent.label))
    .map(({ agent }) => agent)
}

export function NewSessionPanel({
  agentTypes,
  initialAgentTypeId,
  initialWorkingDir,
  onClose,
  onCreated,
}: Props) {
  const [workingDir, setWorkingDir] = useState(initialWorkingDir ?? '')
  const [selectedId, setSelectedId] = useState('')
  const [search, setSearch] = useState('')
  const [selectedSourceKeys, setSelectedSourceKeys] = useState<Set<string>>(() => new Set())
  const [selectedTags, setSelectedTags] = useState<string[]>([])
  const [filtersOpen, setFiltersOpen] = useState(false)
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
  const agentsScrollRef = useRef<HTMLDivElement>(null)

  const orderedBuiltins = useMemo(
    () => orderedBuiltinAgentsForPicker(agentTypes),
    [agentTypes],
  )
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
    // Match ADL harness.permissions default (bypass for antigravity/claude-code/codex).
    if (showToolApprovalsOption(agent)) {
      setHarnessPermissionsEnabled(agent.harnessPermissions !== 'bypass')
    }
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
  }, [agentTypes, initialAgentTypeId, userDefined])

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
    setSearch('')
    setSelectedSourceKeys(new Set())
    setSelectedTags([])
    setFiltersOpen(false)
    setError('')
    setUserScopeHarnessConfig(false)
    setHarnessPermissionsEnabled(true)
    setHarnessOverride('')
    setDirectorySuggestions([])
    setDirectoryInputFocused(false)
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
    () => [BUILTIN_AGENT_SOURCE, ...customSourceOptions.map((option) => option.key)],
    [customSourceOptions],
  )
  const allSourcesActive = selectedSourceKeys.size === 0
    || (allSourceKeys.length > 0 && allSourceKeys.every((key) => selectedSourceKeys.has(key)))
  const searchQuery = search.trim().toLowerCase()
  const sourceFilteredCustom = useMemo(
    () => filterCustomAgentsBySources(userDefined, selectedSourceKeys),
    [userDefined, selectedSourceKeys],
  )
  const builtinSourceActive =
    selectedSourceKeys.size === 0 || selectedSourceKeys.has(BUILTIN_AGENT_SOURCE)
  const tagCandidates = useMemo(
    () => [...(builtinSourceActive ? orderedBuiltins : []), ...sourceFilteredCustom],
    [builtinSourceActive, orderedBuiltins, sourceFilteredCustom],
  )
  const availableTags = useMemo(
    () => collectAgentTags(tagCandidates).filter((tag) => tag !== 'builtin' && tag !== 'nui'),
    [tagCandidates],
  )
  const selectedTagSet = useMemo(() => new Set(selectedTags), [selectedTags])
  const filteredBuiltins = useMemo(() => {
    const byTags = filterAgentsByTags(orderedBuiltins, selectedTagSet)
    return rankAgentsBySearch(byTags, searchQuery)
  }, [orderedBuiltins, searchQuery, selectedTagSet])
  const filteredCustom = useMemo(() => {
    const byTags = filterAgentsByTags(sourceFilteredCustom, selectedTagSet)
    return rankAgentsBySearch(byTags, searchQuery)
  }, [sourceFilteredCustom, searchQuery, selectedTagSet])
  const firstSearchResult = searchQuery
    ? (builtinSourceActive ? filteredBuiltins[0] : undefined) ?? filteredCustom[0]
    : undefined

  useEffect(() => {
    if (!firstSearchResult) return
    setSelectedId(firstSearchResult.id)
    if (showToolApprovalsOption(firstSearchResult)) {
      setHarnessPermissionsEnabled(firstSearchResult.harnessPermissions !== 'bypass')
    }
  }, [firstSearchResult])

  useEffect(() => {
    const validKeys = new Set([BUILTIN_AGENT_SOURCE, ...customSourceOptions.map((option) => option.key)])
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
  const showBuiltins = builtinSourceActive && orderedBuiltins.length > 0
  const showInstalled = userDefined.length > 0
  const visibleResultCount =
    (showBuiltins ? filteredBuiltins.length : 0) + (showInstalled ? filteredCustom.length : 0)
  const activeFilterCount = selectedSourceKeys.size + selectedTags.length
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
    const container = agentsScrollRef.current
    if (!container) return
    const el = container.querySelector(`[data-agent-id="${CSS.escape(selectedId)}"]`)
    el?.scrollIntoView({ block: 'nearest' })
  }, [selectedId, filteredCustom])

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
      api.state.update({ lastAgentType: selectedId }).catch(() => {})
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
      <form
        onSubmit={handleSubmit}
        onKeyDown={(event) => {
          if (event.key !== 'Enter' || event.nativeEvent.isComposing) return
          const target = event.target
          if (!(target instanceof HTMLElement) || !target.closest('[data-agent-id]')) return
          event.preventDefault()
          event.currentTarget.requestSubmit()
        }}
        className="flex flex-1 flex-col min-h-0"
      >
        <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
          <div className="flex min-h-0 flex-1 flex-col overflow-hidden p-4 md:p-6">
            <div className="customize-tab-content mx-auto flex w-full min-h-0 flex-1 flex-col gap-5">

            {(orderedBuiltins.length > 0 || userDefined.length > 0) && (
              <div className="flex min-h-0 flex-1 flex-col gap-3" aria-label="Agent picker">
                <div className="flex shrink-0 flex-col gap-3">
                  <div className="flex items-center gap-2">
                    <div className="min-w-48 flex-1">
                      <SearchInput
                        value={search}
                        onChange={setSearch}
                        placeholder="Search agents…"
                        aria-label="Search agents"
                        autoFocus
                        onKeyDown={(event) => {
                          if (event.key !== 'Enter' || event.nativeEvent.isComposing) return
                          event.preventDefault()
                          if (searchQuery && !firstSearchResult) return
                          event.currentTarget.form?.requestSubmit()
                        }}
                      />
                    </div>
                    <button
                      type="button"
                      aria-label="Filters"
                      aria-expanded={filtersOpen}
                      aria-controls="agent-filters"
                      onClick={() => setFiltersOpen((open) => !open)}
                      className={cn(
                        'relative inline-flex size-9 shrink-0 items-center justify-center rounded-lg border transition-colors',
                        filtersOpen || activeFilterCount > 0
                          ? 'border-primary bg-primary/5 text-foreground'
                          : 'border-input bg-background text-muted-foreground hover:bg-muted',
                      )}
                    >
                      <Filter className="size-4" />
                      {activeFilterCount > 0 && (
                        <span className="absolute ml-7 -mt-7 min-w-4 rounded-full bg-primary px-1 text-center text-[9px] leading-4 text-primary-foreground">
                          {activeFilterCount}
                        </span>
                      )}
                    </button>
                  </div>
                  {filtersOpen && (
                    <div id="agent-filters" className="flex w-full flex-col gap-4 rounded-xl border bg-muted/20 p-4">
                      <div className="flex flex-col gap-4">
                        {customSourceOptions.length > 0 && (
                          <div className="flex flex-col gap-2 sm:flex-row sm:items-start">
                            <Label className="w-16 shrink-0 pt-1 text-xs text-muted-foreground">Source</Label>
                            <div className="flex flex-1 flex-wrap gap-1.5" role="group" aria-label="Filter by agent source">
                              <FilterChip active={allSourcesActive} onClick={toggleAllSourceFilters}>All</FilterChip>
                              <FilterChip
                                active={selectedSourceKeys.has(BUILTIN_AGENT_SOURCE)}
                                onClick={() => toggleSourceFilter(BUILTIN_AGENT_SOURCE)}
                              >
                                Built-in
                              </FilterChip>
                              {customSourceOptions.map((option) => (
                                <FilterChip key={option.key} active={selectedSourceKeys.has(option.key)} onClick={() => toggleSourceFilter(option.key)}>
                                  {option.label}
                                </FilterChip>
                              ))}
                            </div>
                          </div>
                        )}
                        <div className="flex flex-col gap-2 sm:flex-row sm:items-start">
                          <Label className="w-16 shrink-0 pt-2 text-xs text-muted-foreground">Tags</Label>
                          <div className="min-w-0 flex-1">
                            <TagFilterInput
                              availableTags={availableTags}
                              selectedTags={selectedTags}
                              onChange={setSelectedTags}
                              showLabel={false}
                            />
                          </div>
                        </div>
                      </div>
                    </div>
                  )}
                  {activeFilterCount > 0 && (
                    <div className="flex flex-wrap gap-1.5" aria-label="Active filters">
                      {[...selectedSourceKeys].map((key) => (
                        <ActiveFilterChip
                          key={key}
                          label={key === BUILTIN_AGENT_SOURCE
                            ? 'Built-in'
                            : customSourceOptions.find((option) => option.key === key)?.label ?? key}
                          onRemove={() => toggleSourceFilter(key)}
                        />
                      ))}
                      {selectedTags.map((tag) => (
                        <ActiveFilterChip
                          key={tag}
                          label={tag}
                          onRemove={() => setSelectedTags((current) => current.filter((value) => value !== tag))}
                        />
                      ))}
                      <button
                        type="button"
                        className="ml-1 self-center text-xs text-muted-foreground hover:text-foreground"
                        onClick={() => {
                          setSelectedSourceKeys(new Set())
                          setSelectedTags([])
                        }}
                      >
                        Clear all
                      </button>
                    </div>
                  )}
                </div>

                <div ref={agentsScrollRef} className="min-h-0 flex-1 overflow-y-auto pr-1" aria-label="Agent results">
                  {visibleResultCount === 0 ? (
                    <div className="flex min-h-40 flex-col items-center justify-center gap-3 text-center">
                      <p className="text-sm text-muted-foreground">No agents match your search and filters.</p>
                      {(searchQuery || activeFilterCount > 0) && (
                        <Button
                          type="button"
                          variant="outline"
                          size="sm"
                          onClick={() => {
                            setSearch('')
                            setSelectedSourceKeys(new Set())
                            setSelectedTags([])
                          }}
                        >
                          Clear filters
                        </Button>
                      )}
                    </div>
                  ) : (
                    <div className="space-y-5">
                      {showBuiltins && filteredBuiltins.length > 0 && (
                        <AgentSection
                          label={BUILTIN_AGENTS_LABEL}
                          agents={filteredBuiltins}
                          variant="builtin"
                          selectedId={selectedId}
                          onSelect={selectAgent}
                        />
                      )}
                      {showInstalled && filteredCustom.length > 0 && (
                        <AgentSection
                          label={INSTALLED_AGENTS_LABEL}
                          agents={filteredCustom}
                          variant="installed"
                          selectedId={selectedId}
                          onSelect={selectAgent}
                        />
                      )}
                    </div>
                  )}
                </div>
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

interface AgentSectionProps {
  label: string
  agents: AgentType[]
  variant: 'builtin' | 'installed'
  selectedId: string
  onSelect: (id: string) => void
}

function AgentSection({ label, agents, variant, selectedId, onSelect }: AgentSectionProps) {
  return (
    <section aria-label={label} className="space-y-2">
      <div className="flex items-baseline gap-2">
        <h2 className="text-sm font-semibold">{label}</h2>
        <span className="text-xs text-muted-foreground">{agents.length}</span>
      </div>
      <div className={variant === 'builtin'
        ? 'grid grid-cols-2 gap-2 sm:grid-cols-4'
        : 'flex flex-col gap-1.5'}
      >
        {agents.map((agent) => (
          variant === 'builtin' ? (
            <BuiltinAgentCard
              key={agent.id}
              agent={agent}
              selected={selectedId === agent.id}
              onSelect={() => onSelect(agent.id)}
            />
          ) : (
            <InstalledAgentRow
              key={agent.id}
              agent={agent}
              selected={selectedId === agent.id}
              onSelect={() => onSelect(agent.id)}
            />
          )
        ))}
      </div>
    </section>
  )
}

interface FilterChipProps {
  active: boolean
  onClick: () => void
  children: React.ReactNode
}

function FilterChip({ active, onClick, children }: FilterChipProps) {
  return (
    <button
      type="button"
      aria-pressed={active}
      onClick={onClick}
      className={cn(
        'rounded-md border px-2.5 py-1 text-xs font-medium transition-colors',
        active
          ? 'border-primary bg-primary text-primary-foreground'
          : 'border-border bg-background text-muted-foreground hover:bg-muted',
      )}
    >
      {children}
    </button>
  )
}

function ActiveFilterChip({ label, onRemove }: { label: string; onRemove: () => void }) {
  return (
    <span className="inline-flex items-center gap-1 rounded-full bg-muted px-2 py-1 text-xs text-muted-foreground">
      {label}
      <button type="button" onClick={onRemove} aria-label={`Remove filter ${label}`} className="hover:text-foreground">
        <X className="size-3" />
      </button>
    </span>
  )
}

interface AgentCardProps {
  agent: AgentType
  selected: boolean
  onSelect: () => void
}

function BuiltinAgentCard({ agent, selected, onSelect }: AgentCardProps) {
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
      data-agent-id={agent.id}
      onClick={onSelect}
      aria-pressed={selected}
      aria-label={agent.label}
      className={cn(
        'relative flex flex-col items-center gap-1 rounded-lg border px-2 py-2 text-center transition-colors',
        selected
          ? 'border-primary bg-primary/5 text-foreground'
          : 'border-border bg-background hover:bg-muted/60',
      )}
    >
      <HarnessIcon harness={agent.harness} provider={agent.provider} agentId={agent.id} size="lg" />
      <span className={cn(
        'text-sm leading-tight',
        selected ? 'font-medium text-foreground' : 'text-muted-foreground',
      )} aria-hidden="true">{agent.label}</span>
      {kindLabel && (
        <span aria-hidden="true" className="absolute right-2 top-2 rounded-full border border-border/60 bg-muted/50 px-1.5 py-px text-[10px] font-medium leading-tight text-muted-foreground">
          {kindLabel}
        </span>
      )}
    </button>
  )
}

function InstalledAgentRow({ agent, selected, onSelect }: AgentCardProps) {
  const visibleTags = (agent.tags ?? []).filter((tag) => tag !== 'builtin')
  return (
    <button
      type="button"
      data-agent-id={agent.id}
      onClick={onSelect}
      aria-pressed={selected}
      aria-label={agent.label}
      className={cn(
        'flex items-start gap-3 rounded-lg border px-3 py-2.5 text-left transition-colors',
        selected
          ? 'border-primary bg-primary/5 text-foreground'
          : 'border-border bg-background hover:bg-muted/60',
      )}
    >
      <HarnessIcon harness={agent.harness} provider={agent.provider} agentId={agent.id} size="lg" className="mt-0.5 shrink-0" />
      <span className="flex min-w-0 flex-1 flex-col gap-1.5">
        <span className="flex items-start gap-2 min-w-0">
          <span className="min-w-0 flex-1">
            <span className="block text-sm font-medium leading-tight">{agent.label}</span>
            {agent.description && (
              <span className="mt-0.5 block text-xs leading-snug text-muted-foreground">
                {agent.description}
              </span>
            )}
          </span>
          <span className="max-w-[40%] shrink-0 truncate rounded bg-muted px-1.5 py-0.5 font-mono text-[10px] text-muted-foreground" title={harnessLabel(agent.harness, agent.sandbox)}>
            {harnessLabel(agent.harness, agent.sandbox)}
          </span>
        </span>
        {visibleTags.length > 0 && (
          <span className="flex w-full flex-wrap gap-1">
            {visibleTags.slice(0, 3).map((tag) => (
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
