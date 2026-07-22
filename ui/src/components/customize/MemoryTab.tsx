// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useCallback, useEffect, useMemo, useState } from 'react'
import { Check, Copy, Trash2 } from 'lucide-react'
import { ConfirmDeleteDialog } from '@/components/ConfirmDeleteDialog'
import { Button } from '@/components/ui/button'
import { SearchInput } from '@/components/SearchInput'
import { api } from '@/api'
import { selectableAgentTypes } from '@/lib/agentTypes'
import { filterBySearchQuery } from '@/lib/searchFilter'
import type { AgentType, MemoryAgentEntry, MemoryMode, MemorySummary } from '@/types'

type DeleteTarget = { scope: 'user' } | { scope: 'agent'; agentId: string }

const MEMORY_MODE_OPTIONS: { value: MemoryMode; label: string }[] = [
  { value: 'manual', label: 'Manual' },
  { value: 'auto', label: 'Auto' },
  { value: 'disabled', label: 'Disabled' },
]

function memoryPath(scope: 'user' | 'agent', agentId?: string): string {
  if (scope === 'user') return '~/.nui/memory/user.md'
  return `~/.nui/memory/agents/${agentId}.md`
}

function resolveUserMode(summary: MemorySummary | null, fallback?: MemoryMode): MemoryMode {
  return summary?.user.mode ?? fallback ?? 'manual'
}

function resolveAgentMode(
  agentId: string,
  summary: MemorySummary | null,
  settingsModes?: Record<string, MemoryMode>,
): MemoryMode {
  const fromSummary = summary?.agents?.find((a) => a.agentId === agentId)?.mode
  if (fromSummary) return fromSummary
  return settingsModes?.[agentId] ?? 'manual'
}

function MemoryModeSelect({
  value,
  disabled,
  onChange,
}: {
  value: MemoryMode
  disabled?: boolean
  onChange: (mode: MemoryMode) => void
}) {
  return (
    <select
      className="h-8 rounded-md border border-input bg-background px-2 text-sm"
      value={value}
      disabled={disabled}
      onChange={(e) => onChange(e.target.value as MemoryMode)}
      aria-label="Memory mode"
    >
      {MEMORY_MODE_OPTIONS.map((opt) => (
        <option key={opt.value} value={opt.value}>
          {opt.label}
        </option>
      ))}
    </select>
  )
}

export function MemoryTab() {
  const [summary, setSummary] = useState<MemorySummary | null>(null)
  const [agentTypes, setAgentTypes] = useState<AgentType[]>([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState<string | null>(null)
  const [memoryUserMode, setMemoryUserMode] = useState<MemoryMode>('manual')
  const [memoryAgentsMode, setMemoryAgentsMode] = useState<Record<string, MemoryMode>>({})
  const [userContent, setUserContent] = useState('')
  const [selectedAgentId, setSelectedAgentId] = useState<string | null>(null)
  const [agentContent, setAgentContent] = useState('')
  const [deleteTarget, setDeleteTarget] = useState<DeleteTarget | null>(null)
  const [copiedPath, setCopiedPath] = useState<string | null>(null)
  const [agentSearchQuery, setAgentSearchQuery] = useState('')

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const [memorySummary, types, settings, userMemory] = await Promise.all([
        api.memory.list(),
        api.agentTypes.list(),
        api.settings.get().catch(() => ({ theme: 'light' as const })),
        api.memory.getUser().catch(() => ({ content: '' })),
      ])
      setSummary(memorySummary)
      setAgentTypes(types)
      setMemoryUserMode(resolveUserMode(memorySummary, settings.memoryUserMode))
      setMemoryAgentsMode(settings.memoryAgentsMode ?? {})
      setUserContent(userMemory.content ?? '')
    } catch {
      setSummary(null)
      setAgentTypes([])
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  useEffect(() => {
    if (!selectedAgentId) {
      setAgentContent('')
      return
    }
    void api.memory.getAgent(selectedAgentId)
      .then((res) => setAgentContent(res.content ?? ''))
      .catch(() => setAgentContent(''))
  }, [selectedAgentId])

  const setUserMode = async (mode: MemoryMode) => {
    setSaving('user-mode')
    setMemoryUserMode(mode)
    try {
      await api.settings.update({ memoryUserMode: mode })
      await load()
    } finally {
      setSaving(null)
    }
  }

  const setAgentMode = async (agentId: string, mode: MemoryMode) => {
    setSaving(`mode-${agentId}`)
    setMemoryAgentsMode((prev) => ({ ...prev, [agentId]: mode }))
    try {
      const settings = await api.settings.get()
      const map = { ...(settings.memoryAgentsMode ?? {}), [agentId]: mode }
      await api.settings.update({ memoryAgentsMode: map })
      await load()
    } finally {
      setSaving(null)
    }
  }

  const saveUserContent = async () => {
    setSaving('user-save')
    try {
      await api.memory.saveUser(userContent)
      await load()
    } finally {
      setSaving(null)
    }
  }

  const saveAgentContent = async () => {
    if (!selectedAgentId) return
    setSaving(`save-${selectedAgentId}`)
    try {
      await api.memory.saveAgent(selectedAgentId, agentContent)
      await load()
    } finally {
      setSaving(null)
    }
  }

  const copyPath = async (path: string) => {
    try {
      await navigator.clipboard.writeText(path)
      setCopiedPath(path)
      window.setTimeout(() => setCopiedPath(null), 2000)
    } catch {
      // Clipboard may be denied.
    }
  }

  const confirmDelete = async () => {
    if (!deleteTarget) return
    if (deleteTarget.scope === 'user') {
      setSaving('delete-user')
      try {
        await api.memory.deleteUser()
        setUserContent('')
        await load()
      } finally {
        setSaving(null)
        setDeleteTarget(null)
      }
      return
    }
    const { agentId } = deleteTarget
    setSaving(`delete-${agentId}`)
    try {
      await api.memory.deleteAgent(agentId)
      if (selectedAgentId === agentId) {
        setSelectedAgentId(null)
        setAgentContent('')
      }
      await load()
    } finally {
      setSaving(null)
      setDeleteTarget(null)
    }
  }

  if (loading) {
    return <p className="text-sm text-muted-foreground">Loading memory…</p>
  }

  const selectableAgents = selectableAgentTypes(agentTypes)
  const filteredAgents = useMemo(
    () =>
      filterBySearchQuery(selectableAgents, agentSearchQuery, (agent) =>
        [agent.label, agent.id, agent.description].filter(Boolean).join(' '),
      ),
    [selectableAgents, agentSearchQuery],
  )
  const agentEntries: MemoryAgentEntry[] = summary?.agents ?? []
  const agentMemoryById = new Map(agentEntries.map((entry) => [entry.agentId, entry]))
  const userPath = memoryPath('user')
  const deleting = deleteTarget !== null && saving?.startsWith('delete')

  return (
    <div className="customize-tab-content space-y-6">
      <p className="text-sm text-muted-foreground">
        Persistent markdown memory in <code className="text-xs">~/.nui/memory/</code>.
        <strong className="font-medium"> Disabled</strong> turns off read and write.
        <strong className="font-medium"> Manual</strong> injects memory; saves on request.
        <strong className="font-medium"> Auto</strong> injects memory and lets the agent save proactively.
      </p>

      <div className="space-y-3 rounded-lg border p-4">
        <div className="flex items-start justify-between gap-4">
          <div className="min-w-0">
            <p className="text-sm font-medium">User memory</p>
            <p className="text-xs text-muted-foreground mt-1">
              <code className="text-xs">{userPath}</code>
              {summary?.user.size ? ` · ${summary.user.size} bytes` : ''}
            </p>
          </div>
          <div className="flex shrink-0 items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => void copyPath(userPath)}
              aria-label={copiedPath === userPath ? 'Copied path' : 'Copy path'}
            >
              {copiedPath === userPath ? <Check className="size-3.5" /> : <Copy className="size-3.5" />}
            </Button>
            <Button
              variant="ghost"
              size="sm"
              disabled={saving === 'delete-user'}
              onClick={() => setDeleteTarget({ scope: 'user' })}
              aria-label="Delete user memory"
            >
              <Trash2 className="size-3.5" />
            </Button>
            <MemoryModeSelect
              value={memoryUserMode}
              disabled={saving === 'user-mode'}
              onChange={(mode) => void setUserMode(mode)}
            />
          </div>
        </div>
        <textarea
          className="min-h-40 w-full rounded-lg border bg-background p-3 text-sm font-mono"
          value={userContent}
          onChange={(e) => setUserContent(e.target.value)}
          placeholder="Cross-agent preferences and durable facts…"
        />
        <div className="flex justify-end">
          <Button size="sm" disabled={saving === 'user-save'} onClick={() => void saveUserContent()}>
            Save user memory
          </Button>
        </div>
      </div>

      <div className="space-y-3">
        <p className="text-sm font-medium">Per-agent memory</p>
        <SearchInput
          value={agentSearchQuery}
          onChange={setAgentSearchQuery}
          placeholder="Search agents…"
          aria-label="Search agents with memory"
        />
        {filteredAgents.length === 0 ? (
          <p className="text-sm text-muted-foreground">No agents match your search.</p>
        ) : (
        <ul className="divide-y rounded-lg border">
          {filteredAgents.map((agent) => {
            const entry = agentMemoryById.get(agent.id)
            const path = memoryPath('agent', agent.id)
            const selected = selectedAgentId === agent.id
            const mode = resolveAgentMode(agent.id, summary, memoryAgentsMode)
            return (
              <li key={agent.id}>
                <div
                  className={`flex items-start justify-between gap-4 p-4 cursor-pointer ${selected ? 'bg-muted/40' : ''}`}
                  onClick={() => setSelectedAgentId(agent.id)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter' || e.key === ' ') {
                      e.preventDefault()
                      setSelectedAgentId(agent.id)
                    }
                  }}
                  role="button"
                  tabIndex={0}
                >
                  <div className="min-w-0">
                    <p className="font-medium text-sm">{agent.label}</p>
                    <p className="text-xs text-muted-foreground mt-1">
                      <code className="text-xs">{path}</code>
                      {entry?.size ? ` · ${entry.size} bytes` : ''}
                    </p>
                  </div>
                  <div className="flex shrink-0 items-center gap-2" onClick={(e) => e.stopPropagation()}>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => void copyPath(path)}
                      aria-label={copiedPath === path ? 'Copied path' : 'Copy path'}
                    >
                      {copiedPath === path ? <Check className="size-3.5" /> : <Copy className="size-3.5" />}
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      disabled={saving === `delete-${agent.id}`}
                      onClick={() => setDeleteTarget({ scope: 'agent', agentId: agent.id })}
                      aria-label="Delete agent memory"
                    >
                      <Trash2 className="size-3.5" />
                    </Button>
                    <MemoryModeSelect
                      value={mode}
                      disabled={saving === `mode-${agent.id}`}
                      onChange={(next) => void setAgentMode(agent.id, next)}
                    />
                  </div>
                </div>
              </li>
            )
          })}
        </ul>
        )}
      </div>

      {selectedAgentId && (
        <div className="space-y-3 rounded-lg border p-4">
          <p className="text-sm font-medium">
            Edit agent memory: {selectableAgents.find((a) => a.id === selectedAgentId)?.label ?? selectedAgentId}
          </p>
          <textarea
            className="min-h-40 w-full rounded-lg border bg-background p-3 text-sm font-mono"
            value={agentContent}
            onChange={(e) => setAgentContent(e.target.value)}
            placeholder="Agent-specific learned context…"
          />
          <div className="flex justify-end">
            <Button
              size="sm"
              disabled={saving === `save-${selectedAgentId}`}
              onClick={() => void saveAgentContent()}
            >
              Save agent memory
            </Button>
          </div>
        </div>
      )}

      <ConfirmDeleteDialog
        open={deleteTarget !== null}
        onOpenChange={(open) => { if (!open) setDeleteTarget(null) }}
        title={deleteTarget?.scope === 'user' ? 'Delete user memory?' : 'Delete agent memory?'}
        description={
          deleteTarget?.scope === 'user'
            ? <>This will permanently delete <code>{userPath}</code>.</>
            : deleteTarget
              ? <>This will permanently delete <code>{memoryPath('agent', deleteTarget.agentId)}</code>.</>
              : null
        }
        confirming={deleting}
        onConfirm={confirmDelete}
      />
    </div>
  )
}
