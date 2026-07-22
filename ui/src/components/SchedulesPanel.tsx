// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useCallback, useEffect, useMemo, useState } from 'react'
import { CalendarClock, Pencil, Play, Plus, Trash2, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { cn } from '@/lib/utils'
import { formatRelativeTime } from '@/lib/formatRelativeTime'
import { ConfirmDeleteDialog } from '@/components/ConfirmDeleteDialog'
import { SearchableSelect } from '@/components/SearchableSelect'
import { api } from '@/api'
import type { AgentType, Schedule } from '@/types'

type ScheduleKind = 'interval' | 'cron' | 'once'
type FormMode = 'create' | 'edit'

const INTERVAL_PRESETS = ['5m', '15m', '1h', '1d']

interface Props {
  agentTypes: AgentType[]
  onClose: () => void
}

function pad2(n: number): string {
  return String(n).padStart(2, '0')
}

function toDatetimeLocalValue(d: Date): string {
  return `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())}T${pad2(d.getHours())}:${pad2(d.getMinutes())}`
}

function defaultRunAtLocal(): string {
  return toDatetimeLocalValue(new Date(Date.now() + 60 * 60 * 1000))
}

function datetimeLocalToISO(local: string): string {
  return new Date(local).toISOString()
}

function isoToDatetimeLocal(iso: string): string {
  if (!iso) return ''
  return toDatetimeLocalValue(new Date(iso))
}

function emptyForm(autoAgentId: string) {
  return {
    name: '',
    agentType: autoAgentId,
    kind: 'interval' as ScheduleKind,
    interval: '1h',
    cron: '0 9 * * *',
    runAt: defaultRunAtLocal(),
    prompt: '',
    workingDir: '',
  }
}

function formFromSchedule(s: Schedule) {
  const kind: ScheduleKind = s.runAt ? 'once' : s.interval ? 'interval' : 'cron'
  return {
    name: s.name,
    agentType: s.agentType,
    kind,
    interval: s.interval || '1h',
    cron: s.cron || '0 9 * * *',
    runAt: s.runAt ? isoToDatetimeLocal(s.runAt) : defaultRunAtLocal(),
    prompt: s.prompt || '',
    workingDir: s.workingDir || '',
  }
}

function scheduleWhen(s: Schedule): string {
  if (s.interval) return s.interval
  if (s.cron) return s.cron
  if (s.runAt) return `once at ${new Date(s.runAt).toLocaleString()}`
  return '—'
}

interface ScheduleFormFieldsProps {
  name: string
  agentType: string
  kind: ScheduleKind
  interval: string
  cron: string
  runAt: string
  prompt: string
  workingDir: string
  autoAgents: AgentType[]
  onNameChange: (value: string) => void
  onAgentTypeChange: (value: string) => void
  onKindChange: (value: ScheduleKind) => void
  onIntervalChange: (value: string) => void
  onCronChange: (value: string) => void
  onRunAtChange: (value: string) => void
  onPromptChange: (value: string) => void
  onWorkingDirChange: (value: string) => void
}

function ScheduleFormFields({
  name,
  agentType,
  kind,
  interval,
  cron,
  runAt,
  prompt,
  workingDir,
  autoAgents,
  onNameChange,
  onAgentTypeChange,
  onKindChange,
  onIntervalChange,
  onCronChange,
  onRunAtChange,
  onPromptChange,
  onWorkingDirChange,
}: ScheduleFormFieldsProps) {
  return (
    <>
      <div className="grid gap-2">
        <Label htmlFor="schedule-name">Name</Label>
        <Input
          id="schedule-name"
          value={name}
          onChange={(e) => onNameChange(e.target.value)}
          placeholder="Daily report"
        />
      </div>
      <div className="grid gap-2">
        <Label>Agent</Label>
        <SearchableSelect
          value={agentType}
          onValueChange={onAgentTypeChange}
          items={autoAgents.map((agent) => ({
            id: agent.id,
            label: agent.label,
            description: agent.description,
          }))}
          placeholder="Select agent"
          searchPlaceholder="Search agents…"
        />
      </div>
      <div className="grid gap-2">
        <Label>Schedule type</Label>
        <div className="inline-flex rounded-lg border p-0.5">
          <button
            type="button"
            className={cn('rounded-md px-3 py-1 text-xs', kind === 'interval' && 'bg-muted')}
            onClick={() => onKindChange('interval')}
          >
            Interval
          </button>
          <button
            type="button"
            className={cn('rounded-md px-3 py-1 text-xs', kind === 'cron' && 'bg-muted')}
            onClick={() => onKindChange('cron')}
          >
            Cron
          </button>
          <button
            type="button"
            className={cn('rounded-md px-3 py-1 text-xs', kind === 'once' && 'bg-muted')}
            onClick={() => onKindChange('once')}
          >
            Once
          </button>
        </div>
      </div>
      {kind === 'interval' ? (
        <div className="grid gap-2">
          <Label htmlFor="schedule-interval">Interval</Label>
          <div className="flex flex-wrap gap-2">
            {INTERVAL_PRESETS.map((preset) => (
              <Button
                key={preset}
                type="button"
                size="sm"
                variant={interval === preset ? 'secondary' : 'outline'}
                onClick={() => onIntervalChange(preset)}
              >
                {preset}
              </Button>
            ))}
          </div>
          <Input
            id="schedule-interval"
            value={interval}
            onChange={(e) => onIntervalChange(e.target.value)}
            placeholder="5m, 1h, 1d"
          />
        </div>
      ) : kind === 'cron' ? (
        <div className="grid gap-2">
          <Label htmlFor="schedule-cron">Cron (5-field)</Label>
          <Input
            id="schedule-cron"
            value={cron}
            onChange={(e) => onCronChange(e.target.value)}
            placeholder="0 9 * * MON-FRI"
          />
        </div>
      ) : (
        <div className="grid gap-2">
          <Label htmlFor="schedule-run-at">Run at</Label>
          <Input
            id="schedule-run-at"
            type="datetime-local"
            value={runAt}
            onChange={(e) => onRunAtChange(e.target.value)}
          />
          <p className="text-xs text-muted-foreground">
            Runs once at the chosen time, then disables automatically.
          </p>
        </div>
      )}
      <div className="grid gap-2">
        <Label htmlFor="schedule-prompt">Prompt override (optional)</Label>
        <Textarea
          id="schedule-prompt"
          value={prompt}
          onChange={(e) => onPromptChange(e.target.value)}
          rows={2}
        />
      </div>
      <div className="grid gap-2">
        <Label htmlFor="schedule-wd">Working directory (optional)</Label>
        <Input
          id="schedule-wd"
          value={workingDir}
          onChange={(e) => onWorkingDirChange(e.target.value)}
        />
      </div>
    </>
  )
}

export function SchedulesPanel({ agentTypes, onClose }: Props) {
  const autoAgents = useMemo(
    () => agentTypes.filter((a) => a.promptMode === 'auto' && a.available),
    [agentTypes],
  )
  const defaultAgentId = autoAgents[0]?.id ?? ''

  const [schedules, setSchedules] = useState<Schedule[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [formMode, setFormMode] = useState<FormMode | null>(null)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)

  const [name, setName] = useState('')
  const [agentType, setAgentType] = useState('')
  const [kind, setKind] = useState<ScheduleKind>('interval')
  const [interval, setInterval] = useState('1h')
  const [cron, setCron] = useState('0 9 * * *')
  const [runAt, setRunAt] = useState(defaultRunAtLocal())
  const [prompt, setPrompt] = useState('')
  const [workingDir, setWorkingDir] = useState('')

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const list = await api.schedules.list()
      setSchedules(list.sort((a, b) => a.name.localeCompare(b.name)))
      setError(null)
    } catch (err) {
      setSchedules([])
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  function resetForm() {
    const defaults = emptyForm(defaultAgentId)
    setName(defaults.name)
    setAgentType(defaults.agentType)
    setKind(defaults.kind)
    setInterval(defaults.interval)
    setCron(defaults.cron)
    setRunAt(defaults.runAt)
    setPrompt(defaults.prompt)
    setWorkingDir(defaults.workingDir)
  }

  function openCreate() {
    resetForm()
    setEditingId(null)
    setFormMode('create')
  }

  function openEdit(schedule: Schedule) {
    const values = formFromSchedule(schedule)
    setEditingId(schedule.id)
    setName(values.name)
    setAgentType(values.agentType)
    setKind(values.kind)
    setInterval(values.interval)
    setCron(values.cron)
    setRunAt(values.runAt)
    setPrompt(values.prompt)
    setWorkingDir(values.workingDir)
    setFormMode('edit')
  }

  function closeForm() {
    setFormMode(null)
    setEditingId(null)
    resetForm()
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)
    setSaving(true)
    const payload = {
      name: name.trim() || agentType,
      agentType,
      prompt: prompt.trim(),
      workingDir: workingDir.trim(),
      interval: kind === 'interval' ? interval.trim() : undefined,
      cron: kind === 'cron' ? cron.trim() : undefined,
      runAt: kind === 'once' ? datetimeLocalToISO(runAt) : undefined,
    }
    try {
      if (formMode === 'edit' && editingId) {
        await api.schedules.patch(editingId, payload)
      } else {
        await api.schedules.create(payload)
      }
      closeForm()
      await load()
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setSaving(false)
    }
  }

  async function toggleEnabled(schedule: Schedule) {
    await api.schedules.patch(schedule.id, { enabled: !schedule.enabled })
    await load()
  }

  async function runNow(id: string) {
    await api.schedules.runNow(id)
    await load()
  }

  async function confirmDelete() {
    if (!deleteTarget) return
    await api.schedules.delete(deleteTarget)
    if (deleteTarget === editingId) {
      closeForm()
    }
    setDeleteTarget(null)
    await load()
  }

  const formOpen = formMode != null && autoAgents.length > 0

  return (
    <div className="customize-panel flex flex-1 flex-col overflow-hidden">
      <div className="conversation-header justify-between">
        <div className="flex items-center gap-2 min-w-0">
          <CalendarClock className="size-4 shrink-0 text-muted-foreground" />
          <h1 className="text-sm font-semibold truncate">Schedules</h1>
        </div>
        <Button variant="ghost" size="sm" onClick={onClose} aria-label="Close schedules panel">
          <X className="size-4" />
        </Button>
      </div>

      <div className="flex-1 overflow-y-auto p-4 md:p-6">
        <div className="flex flex-col gap-6 max-w-3xl mx-auto">
          <div className="flex items-center justify-between gap-4">
            <div>
              <p className="text-sm text-muted-foreground">
                Recurring or one-time runs for auto-prompt agents. Each run creates a new session.
              </p>
            </div>
            <Button
              type="button"
              size="sm"
              onClick={openCreate}
              disabled={autoAgents.length === 0 || formMode === 'create'}
            >
              <Plus className="size-3.5" />
              New schedule
            </Button>
          </div>

          {autoAgents.length === 0 && (
            <p className="text-sm text-muted-foreground">
              No schedulable agents. Create an ADL agent with <code className="text-xs">promptMode: auto</code> first.
            </p>
          )}

          {error && <p className="text-sm text-destructive">{error}</p>}

          {formOpen && (
            <form onSubmit={(e) => void handleSubmit(e)} className="rounded-lg border p-4 space-y-4">
              <h3 className="text-sm font-medium">
                {formMode === 'edit' ? 'Edit schedule' : 'New schedule'}
              </h3>
              <ScheduleFormFields
                name={name}
                agentType={agentType}
                kind={kind}
                interval={interval}
                cron={cron}
                runAt={runAt}
                prompt={prompt}
                workingDir={workingDir}
                autoAgents={autoAgents}
                onNameChange={setName}
                onAgentTypeChange={setAgentType}
                onKindChange={setKind}
                onIntervalChange={setInterval}
                onCronChange={setCron}
                onRunAtChange={setRunAt}
                onPromptChange={setPrompt}
                onWorkingDirChange={setWorkingDir}
              />
              <div className="flex gap-2">
                <Button type="submit" disabled={saving}>
                  {formMode === 'edit' ? 'Save changes' : 'Create schedule'}
                </Button>
                <Button type="button" variant="outline" onClick={closeForm} disabled={saving}>
                  Cancel
                </Button>
              </div>
            </form>
          )}

          {loading ? (
            <p className="text-sm text-muted-foreground">Loading schedules…</p>
          ) : schedules.length === 0 ? (
            <p className="text-sm text-muted-foreground">No schedules yet.</p>
          ) : (
            <ul className="rounded-lg border divide-y">
              {schedules.map((s) => (
                <li
                  key={s.id}
                  className={cn(
                    'flex flex-col gap-2 p-4 sm:flex-row sm:items-center sm:justify-between',
                    editingId === s.id && 'bg-muted/40',
                  )}
                >
                  <div className="min-w-0">
                    <div className="flex items-center gap-2">
                      <CalendarClock className="size-4 shrink-0 text-muted-foreground" />
                      <span className="font-medium truncate">{s.name}</span>
                      {!s.enabled && (
                        <span className="text-xs text-muted-foreground">
                          {s.runAt && s.lastRunAt ? '(completed)' : '(disabled)'}
                        </span>
                      )}
                    </div>
                    <p className="text-xs text-muted-foreground mt-1">
                      {s.agentType} · {scheduleWhen(s)}
                      {s.nextRunAt && s.enabled && (
                        <> · next {formatRelativeTime(s.nextRunAt)}</>
                      )}
                      {s.lastRunAt && <> · last {formatRelativeTime(s.lastRunAt)}</>}
                    </p>
                  </div>
                  <div className="flex shrink-0 flex-wrap gap-2">
                    <Button type="button" size="sm" variant="outline" onClick={() => openEdit(s)}>
                      <Pencil className="size-3.5" />
                      Edit
                    </Button>
                    {s.enabled && (
                      <Button type="button" size="sm" variant="outline" onClick={() => void runNow(s.id)}>
                        <Play className="size-3.5" />
                        Run now
                      </Button>
                    )}
                    <Button type="button" size="sm" variant="outline" onClick={() => void toggleEnabled(s)}>
                      {s.enabled ? 'Disable' : 'Enable'}
                    </Button>
                    <Button type="button" size="sm" variant="ghost" onClick={() => setDeleteTarget(s.id)}>
                      <Trash2 className="size-3.5" />
                    </Button>
                  </div>
                </li>
              ))}
            </ul>
          )}

          <ConfirmDeleteDialog
            open={deleteTarget != null}
            onOpenChange={(open) => { if (!open) setDeleteTarget(null) }}
            title="Delete schedule?"
            description="This removes the schedule. Existing sessions created by it are kept."
            onConfirm={confirmDelete}
          />
        </div>
      </div>
    </div>
  )
}
