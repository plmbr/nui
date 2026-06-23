// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { Plus, Trash2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import {
  type AgentFormModel,
  type AgentFormOptions,
  type FormMCPServer,
  type FormSkill,
  type KeyValue,
  slugFromName,
} from '@/lib/adlAgentForm'

interface Props {
  form: AgentFormModel
  options: AgentFormOptions
  hasWorkflowSteps?: boolean
  onChange: (form: AgentFormModel) => void
}

function groupBy<T extends { group: string }>(items: T[]): Map<string, T[]> {
  const map = new Map<string, T[]>()
  for (const item of items) {
    const list = map.get(item.group) ?? []
    list.push(item)
    map.set(item.group, list)
  }
  return map
}

function selectedHarness(form: AgentFormModel, options: AgentFormOptions) {
  return options.harnesses.find((h) => h.id === form.harnessOptionId)
}

function isCliHarness(harnessType: string | undefined): boolean {
  return harnessType != null && ['claude-code', 'pi', 'codex', 'opencode'].includes(harnessType)
}

function KeyValueList({
  label,
  description,
  entries,
  onChange,
}: {
  label: string
  description?: string
  entries: KeyValue[]
  onChange: (entries: KeyValue[]) => void
}) {
  return (
    <div className="space-y-2">
      <div>
        <Label>{label}</Label>
        {description && <p className="text-xs text-muted-foreground mt-0.5">{description}</p>}
      </div>
      {entries.map((entry, index) => (
        <div key={index} className="grid grid-cols-[minmax(7rem,10rem)_minmax(0,1fr)_auto] gap-2 items-center">
          <Input
            placeholder="KEY"
            value={entry.key}
            onChange={(e) => {
              const next = [...entries]
              next[index] = { ...entry, key: e.target.value }
              onChange(next)
            }}
            className="font-mono text-xs"
          />
          <Input
            placeholder="value"
            value={entry.value}
            onChange={(e) => {
              const next = [...entries]
              next[index] = { ...entry, value: e.target.value }
              onChange(next)
            }}
            className="font-mono text-xs"
          />
          <Button
            type="button"
            variant="ghost"
            size="sm"
            className="shrink-0"
            onClick={() => onChange(entries.filter((_, i) => i !== index))}
          >
            <Trash2 className="size-3.5" />
          </Button>
        </div>
      ))}
      <Button
        type="button"
        variant="outline"
        size="sm"
        onClick={() => onChange([...entries, { key: '', value: '' }])}
      >
        <Plus className="size-3.5" />
        Add variable
      </Button>
    </div>
  )
}

function SelectGrouped<T extends { id: string; label: string; group: string }>({
  value,
  onValueChange,
  items,
  placeholder,
}: {
  value: string
  onValueChange: (value: string) => void
  items: T[]
  placeholder: string
}) {
  const groups = groupBy(items)
  return (
    <Select value={value || null} onValueChange={(v) => v && onValueChange(v)}>
      <SelectTrigger className="w-full">
        <SelectValue placeholder={placeholder} />
      </SelectTrigger>
      <SelectContent>
        {[...groups.entries()].map(([group, groupItems]) => (
          <SelectGroup key={group}>
            <SelectLabel>{group}</SelectLabel>
            {groupItems.map((item) => (
              <SelectItem key={item.id} value={item.id}>
                {item.label}
              </SelectItem>
            ))}
          </SelectGroup>
        ))}
      </SelectContent>
    </Select>
  )
}

export function AgentForm({ form, options, hasWorkflowSteps, onChange }: Props) {
  const harness = selectedHarness(form, options)
  const harnessType = harness?.harnessType

  const patch = (partial: Partial<AgentFormModel>) => onChange({ ...form, ...partial })

  const addSkill = (optionId: string) => {
    const opt = options.skills.find((s) => s.id === optionId)
    if (!opt) return
    if (form.skills.some((s) => s.optionId === optionId)) return
    const next: FormSkill[] = [...form.skills, { optionId, name: opt.name }]
    patch({ skills: next })
  }

  const addMCP = (optionId: string) => {
    const opt = options.mcpServers.find((s) => s.id === optionId)
    if (!opt) return
    if (form.mcpServers.some((s) => s.optionId === optionId)) return
    const next: FormMCPServer[] = [...form.mcpServers, { optionId, name: opt.name }]
    patch({ mcpServers: next })
  }

  return (
    <div className="agent-form space-y-6 max-w-2xl">
      {hasWorkflowSteps && (
        <div className="rounded-lg border border-amber-500/40 bg-amber-500/10 px-3 py-2 text-xs text-amber-800 dark:text-amber-300">
          This agent has workflow steps defined in YAML. The form edits top-level fields only; steps
          are preserved when you save.
        </div>
      )}

      <section className="space-y-3">
        <h3 className="text-sm font-semibold">Identity</h3>
        <div className="grid gap-3 sm:grid-cols-2">
          <div className="space-y-1.5">
            <Label htmlFor="agent-id">ID</Label>
            <Input
              id="agent-id"
              value={form.id}
              onChange={(e) => patch({ id: e.target.value })}
              placeholder="my-agent"
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="agent-name">Name</Label>
            <Input
              id="agent-name"
              value={form.name}
              onChange={(e) => {
                const name = e.target.value
                patch({
                  name,
                  id: form.id === slugFromName(form.name) || !form.id ? slugFromName(name) : form.id,
                })
              }}
              placeholder="My Agent"
            />
          </div>
        </div>
        <div className="space-y-1.5">
          <Label htmlFor="agent-description">Description</Label>
          <Input
            id="agent-description"
            value={form.description}
            onChange={(e) => patch({ description: e.target.value })}
            placeholder="What this agent does"
          />
        </div>
      </section>

      <section className="space-y-3">
        <h3 className="text-sm font-semibold">Harness</h3>
        <div className="grid gap-3 sm:grid-cols-2">
          <div className="space-y-1.5">
            <Label>Harness type</Label>
            <SelectGrouped
              value={form.harnessOptionId}
              onValueChange={(harnessOptionId) => patch({ harnessOptionId })}
              items={options.harnesses}
              placeholder="Select harness"
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="harness-model">Model</Label>
            <Input
              id="harness-model"
              value={form.harnessModel}
              onChange={(e) => patch({ harnessModel: e.target.value })}
              placeholder={isCliHarness(harnessType) ? 'claude-sonnet-4-6' : '—'}
              disabled={!isCliHarness(harnessType)}
            />
          </div>
        </div>

        {harnessType === 'docker' && (
          <div className="grid gap-3 sm:grid-cols-2">
            <div className="space-y-1.5">
              <Label htmlFor="docker-image">Container image</Label>
              <Input
                id="docker-image"
                value={form.dockerImage}
                onChange={(e) => patch({ dockerImage: e.target.value })}
                placeholder="loop-echo-agent"
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="container-port">Container port</Label>
              <Input
                id="container-port"
                value={form.containerPort}
                onChange={(e) => patch({ containerPort: e.target.value })}
                placeholder="9090"
              />
            </div>
          </div>
        )}

        {harnessType === 'remote' && (
          <div className="grid gap-3 sm:grid-cols-2">
            <div className="space-y-1.5">
              <Label htmlFor="remote-host">Host</Label>
              <Input
                id="remote-host"
                value={form.remoteHost}
                onChange={(e) => patch({ remoteHost: e.target.value })}
                placeholder="127.0.0.1"
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="remote-port">Port</Label>
              <Input
                id="remote-port"
                value={form.remotePort}
                onChange={(e) => patch({ remotePort: e.target.value })}
                placeholder="9090"
              />
            </div>
          </div>
        )}
      </section>

      <section className="space-y-3">
        <h3 className="text-sm font-semibold">Prompt</h3>
        <div className="space-y-1.5">
          <Label htmlFor="system-prompt">System prompt</Label>
          <Textarea
            id="system-prompt"
            value={form.systemPrompt}
            onChange={(e) => patch({ systemPrompt: e.target.value })}
            rows={4}
            placeholder="Instructions for the agent…"
          />
        </div>
        <div className="grid gap-3 sm:grid-cols-2">
          <div className="space-y-1.5">
            <Label>Prompt mode</Label>
            <Select
              value={form.promptMode}
              onValueChange={(v) => patch({ promptMode: (v ?? 'user') as 'user' | 'auto' })}
            >
              <SelectTrigger className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="user">User (interactive chat)</SelectItem>
                <SelectItem value="auto">Auto (run default prompt on start)</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
        <div className="space-y-1.5">
          <Label htmlFor="default-prompt">Default prompt</Label>
          <p className="text-xs text-muted-foreground">
            Used when prompt mode is auto — sent when a session starts.
          </p>
          <Textarea
            id="default-prompt"
            value={form.defaultPrompt}
            onChange={(e) => patch({ defaultPrompt: e.target.value })}
            rows={3}
            placeholder="Review the README and suggest improvements"
            disabled={form.promptMode !== 'auto'}
          />
        </div>
      </section>

      <section className="space-y-3">
        <h3 className="text-sm font-semibold">Session</h3>
        <div className="space-y-1.5">
          <Label>Working directory</Label>
          <p className="text-xs text-muted-foreground">
            When enabled, users choose a project directory when creating a session.
            Otherwise Loop uses an isolated workspace that is removed when the session is deleted.
          </p>
          <Select
            value={form.workingDirInput ? 'true' : 'false'}
            onValueChange={(v) => patch({ workingDirInput: v === 'true' })}
          >
            <SelectTrigger className="w-full max-w-md">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="false">Isolated workspace (default)</SelectItem>
              <SelectItem value="true">User picks directory at session create</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </section>

      <section className="space-y-3">
        <h3 className="text-sm font-semibold">Skills</h3>
        {form.skills.length > 0 && (
          <ul className="space-y-2">
            {form.skills.map((skill, index) => {
              const opt = options.skills.find((s) => s.id === skill.optionId)
              return (
                <li key={`${skill.optionId}-${index}`} className="flex items-center gap-2 rounded-md border px-3 py-2">
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium truncate">{opt?.label ?? skill.name}</p>
                    <p className="text-xs text-muted-foreground truncate">{opt?.ref ?? skill.optionId}</p>
                  </div>
                  <Input
                    value={skill.name}
                    onChange={(e) => {
                      const next = [...form.skills]
                      next[index] = { ...skill, name: e.target.value }
                      patch({ skills: next })
                    }}
                    className="w-36 text-xs"
                    placeholder="ADL name"
                  />
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    onClick={() => patch({ skills: form.skills.filter((_, i) => i !== index) })}
                  >
                    <Trash2 className="size-3.5" />
                  </Button>
                </li>
              )
            })}
          </ul>
        )}
        {options.skills.length > 0 ? (
          <Select key={`add-skill-${form.skills.length}`} onValueChange={(v) => v && addSkill(v)}>
            <SelectTrigger className="w-full">
              <SelectValue placeholder="Add skill…" />
            </SelectTrigger>
            <SelectContent>
              {[...groupBy(options.skills).entries()].map(([group, items]) => (
                <SelectGroup key={group}>
                  <SelectLabel>{group}</SelectLabel>
                  {items.map((item) => (
                    <SelectItem key={item.id} value={item.id} disabled={form.skills.some((s) => s.optionId === item.id)}>
                      {item.label}
                    </SelectItem>
                  ))}
                </SelectGroup>
              ))}
            </SelectContent>
          </Select>
        ) : (
          <p className="text-xs text-muted-foreground">No skills available. Install skills or enable extensions.</p>
        )}
      </section>

      <section className="space-y-3">
        <h3 className="text-sm font-semibold">MCP servers</h3>
        {form.mcpServers.length > 0 && (
          <ul className="space-y-2">
            {form.mcpServers.map((server, index) => {
              const opt = options.mcpServers.find((s) => s.id === server.optionId)
              return (
                <li key={`${server.optionId}-${index}`} className="flex items-center gap-2 rounded-md border px-3 py-2">
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium truncate">{opt?.label ?? server.name}</p>
                    <p className="text-xs text-muted-foreground truncate">
                      {opt?.ref ?? opt?.server?.url ?? opt?.server?.command ?? server.optionId}
                    </p>
                  </div>
                  <Input
                    value={server.name}
                    onChange={(e) => {
                      const next = [...form.mcpServers]
                      next[index] = { ...server, name: e.target.value }
                      patch({ mcpServers: next })
                    }}
                    className="w-36 text-xs"
                    placeholder="ADL name"
                  />
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    onClick={() => patch({ mcpServers: form.mcpServers.filter((_, i) => i !== index) })}
                  >
                    <Trash2 className="size-3.5" />
                  </Button>
                </li>
              )
            })}
          </ul>
        )}
        {options.mcpServers.length > 0 ? (
          <Select key={`add-mcp-${form.mcpServers.length}`} onValueChange={(v) => v && addMCP(v)}>
            <SelectTrigger className="w-full">
              <SelectValue placeholder="Add MCP server…" />
            </SelectTrigger>
            <SelectContent>
              {[...groupBy(options.mcpServers).entries()].map(([group, items]) => (
                <SelectGroup key={group}>
                  <SelectLabel>{group}</SelectLabel>
                  {items.map((item) => (
                    <SelectItem key={item.id} value={item.id} disabled={form.mcpServers.some((s) => s.optionId === item.id)}>
                      {item.label}
                    </SelectItem>
                  ))}
                </SelectGroup>
              ))}
            </SelectContent>
          </Select>
        ) : (
          <p className="text-xs text-muted-foreground">
            No MCP servers available. Add servers under Customize → MCP servers or enable extensions.
          </p>
        )}
      </section>

      <section className="space-y-3">
        <h3 className="text-sm font-semibold">Environment</h3>
        <KeyValueList
          label="Env vars"
          description="Applied to the agent process."
          entries={form.env}
          onChange={(env) => patch({ env })}
        />
      </section>
    </div>
  )
}
