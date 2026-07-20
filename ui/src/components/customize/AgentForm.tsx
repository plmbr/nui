// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import type { ReactNode } from 'react'
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
  selectItemData,
} from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import {
  type AgentFormModel,
  type AgentFormOptions,
  type FormEval,
  type FormMCPServer,
  type FormSkill,
  type HitlMode,
  type KeyValue,
  type ToolApprovalPolicy,
  defaultFormEval,
  slugFromName,
} from '@/lib/adlAgentForm'

interface Props {
  form: AgentFormModel
  options: AgentFormOptions
  hasWorkflowSteps?: boolean
  hasSubAgents?: boolean
  editingAgentId?: string
  onChange: (form: AgentFormModel) => void
}

function RequiredMark() {
  return <span className="text-destructive" aria-hidden="true">*</span>
}

function FieldLabel({
  htmlFor,
  required,
  children,
}: {
  htmlFor?: string
  required?: boolean
  children: ReactNode
}) {
  return (
    <Label htmlFor={htmlFor}>
      {children}
      {required && <> <RequiredMark /></>}
    </Label>
  )
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
        <div key={index} className="grid grid-cols-1 gap-2 items-center sm:grid-cols-[minmax(7rem,10rem)_minmax(0,1fr)_auto]">
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
    <Select
      value={value || null}
      onValueChange={(v) => v && onValueChange(v)}
      items={selectItemData(items)}
    >
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

export function AgentForm({ form, options, hasWorkflowSteps, hasSubAgents, editingAgentId, onChange }: Props) {
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

  const updateEval = (index: number, partial: Partial<FormEval>) => {
    const next = [...form.evals]
    next[index] = { ...next[index], ...partial }
    patch({ evals: next })
  }

  const addEval = () => {
    const base = defaultFormEval(`eval-${form.evals.length + 1}`)
    patch({ evals: [...form.evals, base] })
  }

  const addSubAgent = (agentId: string) => {
    if (!agentId || form.subAgents.includes(agentId)) return
    patch({ subAgents: [...form.subAgents, agentId] })
  }

  const selectableAgents = options.agents.filter(
    (a) => a.id !== editingAgentId && !form.subAgents.includes(a.id),
  )

  return (
    <div className="agent-form space-y-6 max-w-2xl">
      {hasWorkflowSteps && (
        <div className="rounded-lg border border-amber-500/40 bg-amber-500/10 px-3 py-2 text-xs text-amber-800 dark:text-amber-300">
          This agent has workflow steps defined in YAML. The form edits top-level fields only; steps
          are preserved when you save.
        </div>
      )}
      {hasSubAgents && hasWorkflowSteps && (
        <div className="rounded-lg border border-destructive/40 bg-destructive/10 px-3 py-2 text-xs text-destructive">
          This agent defines both sub-agents and workflow steps. Only one orchestration mode is
          supported — remove steps or sub-agents in YAML.
        </div>
      )}

      <section className="space-y-3">
        <h3 className="text-sm font-semibold">Identity</h3>
        <p className="text-xs text-muted-foreground">At least one of ID or Name is required.</p>
        <div className="grid gap-3 sm:grid-cols-2">
          <div className="space-y-1.5">
            <FieldLabel htmlFor="agent-id" required>ID</FieldLabel>
            <Input
              id="agent-id"
              value={form.id}
              onChange={(e) => patch({ id: e.target.value })}
              placeholder="my-agent"
            />
          </div>
          <div className="space-y-1.5">
            <FieldLabel htmlFor="agent-name" required>Name</FieldLabel>
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
        <div className="space-y-1.5">
          <Label htmlFor="agent-tags">Tags</Label>
          <Input
            id="agent-tags"
            value={form.tags}
            onChange={(e) => patch({ tags: e.target.value })}
            placeholder="coding, research"
          />
          <p className="text-xs text-muted-foreground">
            Comma-separated labels for filtering in the new-session UI.
          </p>
        </div>
      </section>

      <section className="space-y-3">
        <h3 className="text-sm font-semibold">Harness</h3>
        <div className="grid gap-3 sm:grid-cols-2">
          <div className="space-y-1.5">
            <FieldLabel required>Harness type</FieldLabel>
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
              <FieldLabel htmlFor="docker-image" required>Container image</FieldLabel>
              <Input
                id="docker-image"
                value={form.dockerImage}
                onChange={(e) => patch({ dockerImage: e.target.value })}
                placeholder="nui-echo-agent"
              />
            </div>
            <div className="space-y-1.5">
              <FieldLabel htmlFor="container-port" required>Container port</FieldLabel>
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
              <FieldLabel htmlFor="remote-host" required>Host</FieldLabel>
              <Input
                id="remote-host"
                value={form.remoteHost}
                onChange={(e) => patch({ remoteHost: e.target.value })}
                placeholder="127.0.0.1"
              />
            </div>
            <div className="space-y-1.5">
              <FieldLabel htmlFor="remote-port" required>Port</FieldLabel>
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
              items={[
                { value: 'user', label: 'User (interactive chat)' },
                { value: 'auto', label: 'Auto (run default prompt on start)' },
              ]}
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
            Pre-filled in the chat input on session start; auto-sent when prompt mode is auto.
          </p>
          <Textarea
            id="default-prompt"
            value={form.defaultPrompt}
            onChange={(e) => patch({ defaultPrompt: e.target.value })}
            rows={3}
            placeholder="Review the README and suggest improvements"
          />
        </div>
      </section>

      <section className="space-y-3">
        <h3 className="text-sm font-semibold">Session</h3>
        <div className="space-y-1.5">
          <Label>Working directory</Label>
          <p className="text-xs text-muted-foreground">
            When enabled, users choose a project directory when creating a session.
            Otherwise nui uses an isolated workspace that is removed when the session is deleted.
          </p>
          <Select
            value={form.workingDirInput ? 'true' : 'false'}
            onValueChange={(v) => patch({ workingDirInput: v === 'true' })}
            items={[
              { value: 'false', label: 'Isolated workspace (default)' },
              { value: 'true', label: 'User picks directory at session create' },
            ]}
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
        <h3 className="text-sm font-semibold">Safety &amp; approvals</h3>
        <div className="grid gap-3 sm:grid-cols-2">
          <div className="space-y-1.5">
            <Label>Tool approval policy</Label>
            <p className="text-xs text-muted-foreground">
              Controls which harness tools require nui approval cards before running.
            </p>
            <Select
              value={form.toolApprovalPolicy || 'unset'}
              onValueChange={(v) => {
                const policy = (v === 'unset' ? '' : v) as ToolApprovalPolicy
                patch({
                  toolApprovalPolicy: policy,
                  toolApprovalTools:
                    policy === 'allowlist' || policy === 'denylist' ? form.toolApprovalTools : [],
                })
              }}
              items={[
                { value: 'unset', label: 'Unset (prompt for every tool)' },
                { value: 'default', label: 'Default (prompt for every tool)' },
                { value: 'all', label: 'All (auto-approve every tool)' },
                { value: 'allowlist', label: 'Allowlist (auto-approve listed tools only)' },
                { value: 'denylist', label: 'Denylist (prompt only for listed tools)' },
              ]}
            >
              <SelectTrigger className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="unset">Unset (prompt for every tool)</SelectItem>
                <SelectItem value="default">Default (prompt for every tool)</SelectItem>
                <SelectItem value="all">All (auto-approve every tool)</SelectItem>
                <SelectItem value="allowlist">Allowlist (auto-approve listed tools only)</SelectItem>
                <SelectItem value="denylist">Denylist (prompt only for listed tools)</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-1.5">
            <Label>HITL mode</Label>
            <p className="text-xs text-muted-foreground">
              Runtime human-in-the-loop prompts via nui UI and MCP tools.
            </p>
            <Select
              value={form.hitlMode || 'unset'}
              onValueChange={(v) => patch({ hitlMode: (v === 'unset' ? '' : v) as HitlMode })}
              items={[
                { value: 'unset', label: 'Unset (interactive for chat, auto for schedules)' },
                { value: 'interactive', label: 'Interactive (show HITL cards)' },
                { value: 'auto', label: 'Auto (for scheduled agents)' },
                { value: 'off', label: 'Off (disable runtime HITL)' },
              ]}
            >
              <SelectTrigger className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="unset">Unset (interactive for chat, auto for schedules)</SelectItem>
                <SelectItem value="interactive">Interactive (show HITL cards)</SelectItem>
                <SelectItem value="auto">Auto (for scheduled agents)</SelectItem>
                <SelectItem value="off">Off (disable runtime HITL)</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
        {(form.toolApprovalPolicy === 'allowlist' || form.toolApprovalPolicy === 'denylist') && (
          <div className="space-y-1.5">
            <FieldLabel htmlFor="tool-approval-tools" required>Tool list</FieldLabel>
            <p className="text-xs text-muted-foreground">
              One tool name per line (e.g. Bash, Write, Read, mcp__nui-hitl__*).
            </p>
            <Textarea
              id="tool-approval-tools"
              value={form.toolApprovalTools.join('\n')}
              onChange={(e) => {
                const toolApprovalTools = e.target.value
                  .split('\n')
                  .map((line) => line.trim())
                  .filter(Boolean)
                patch({ toolApprovalTools })
              }}
              rows={4}
              placeholder={'Bash\nWrite\nEdit'}
              className="font-mono text-xs"
            />
          </div>
        )}
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
          <Select
            key={`add-skill-${form.skills.length}`}
            onValueChange={(v) => v && addSkill(v)}
            items={selectItemData(options.skills)}
          >
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
          <Select
            key={`add-mcp-${form.mcpServers.length}`}
            onValueChange={(v) => v && addMCP(v)}
            items={selectItemData(options.mcpServers)}
          >
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
        <h3 className="text-sm font-semibold">Sub-agents</h3>
        <p className="text-xs text-muted-foreground">
          Orchestrator agents route each user message to the best matching sub-agent. Agent names and
          descriptions come from the registry.
        </p>
        {form.subAgents.length > 0 && (
          <ul className="space-y-2">
            {form.subAgents.map((agentId) => {
              const opt = options.agents.find((a) => a.id === agentId)
              return (
                <li key={agentId} className="flex items-center gap-2 rounded-md border px-3 py-2">
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium truncate">{opt?.label ?? agentId}</p>
                    <p className="text-xs text-muted-foreground truncate">
                      {opt?.description || agentId}
                    </p>
                  </div>
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    onClick={() => patch({ subAgents: form.subAgents.filter((id) => id !== agentId) })}
                  >
                    <Trash2 className="size-3.5" />
                  </Button>
                </li>
              )
            })}
          </ul>
        )}
        {selectableAgents.length > 0 ? (
          <Select
            key={`add-sub-agent-${form.subAgents.length}`}
            onValueChange={(v) => v && addSubAgent(v)}
            items={selectItemData(selectableAgents.map((a) => ({ id: a.id, label: a.label, group: a.group })))}
          >
            <SelectTrigger className="w-full">
              <SelectValue placeholder="Add sub-agent…" />
            </SelectTrigger>
            <SelectContent>
              {[...groupBy(selectableAgents).entries()].map(([group, items]) => (
                <SelectGroup key={group}>
                  <SelectLabel>{group}</SelectLabel>
                  {items.map((item) => (
                    <SelectItem key={item.id} value={item.id}>
                      <span className="block">{item.label}</span>
                      {item.description ? (
                        <span className="block text-xs text-muted-foreground truncate max-w-md">
                          {item.description}
                        </span>
                      ) : null}
                    </SelectItem>
                  ))}
                </SelectGroup>
              ))}
            </SelectContent>
          </Select>
        ) : (
          <p className="text-xs text-muted-foreground">No other agents available to add.</p>
        )}
      </section>

      <section className="space-y-3">
        <h3 className="text-sm font-semibold">Evals</h3>
        <p className="text-xs text-muted-foreground">
          Test cases for verifying agent behavior. Run with{' '}
          <code className="font-mono">nui agent eval run -a …</code>.
        </p>
        {form.evals.length > 0 && (
          <ul className="space-y-3">
            {form.evals.map((ev, index) => (
              <li key={`eval-${index}`} className="rounded-md border p-3 space-y-3">
                <div className="flex items-start gap-2">
                  <div className="flex-1 grid gap-3 sm:grid-cols-2">
                    <div className="space-y-1.5 sm:col-span-2">
                      <FieldLabel required>Name</FieldLabel>
                      <Input
                        value={ev.name}
                        onChange={(e) => updateEval(index, { name: e.target.value })}
                        placeholder="polite-greeting"
                      />
                    </div>
                    <div className="space-y-1.5 sm:col-span-2">
                      <Label>Description</Label>
                      <Input
                        value={ev.description}
                        onChange={(e) => updateEval(index, { description: e.target.value })}
                        placeholder="What this eval verifies"
                      />
                    </div>
                    <div className="space-y-1.5">
                      <Label>Input mode</Label>
                      <Select
                        value={ev.inputMode}
                        onValueChange={(v) =>
                          updateEval(index, {
                            inputMode: (v ?? 'single') as 'single' | 'conversation',
                          })
                        }
                        items={[
                          { value: 'single', label: 'Single prompt' },
                          { value: 'conversation', label: 'Conversation' },
                        ]}
                      >
                        <SelectTrigger className="w-full">
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="single">Single prompt</SelectItem>
                          <SelectItem value="conversation">Conversation</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                    <div className="space-y-1.5">
                      <Label>Grader</Label>
                      <Select
                        value={ev.expectType || 'none'}
                        onValueChange={(v) =>
                          updateEval(index, {
                            expectType: (v ?? 'none') as FormEval['expectType'],
                          })
                        }
                        items={[
                          { value: 'contains', label: 'Contains' },
                          { value: 'exact', label: 'Exact match' },
                          { value: 'regex', label: 'Regex' },
                          { value: 'llm', label: 'LLM judge' },
                          { value: 'none', label: 'Manual (none)' },
                        ]}
                      >
                        <SelectTrigger className="w-full">
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="contains">Contains</SelectItem>
                          <SelectItem value="exact">Exact match</SelectItem>
                          <SelectItem value="regex">Regex</SelectItem>
                          <SelectItem value="llm">LLM judge</SelectItem>
                          <SelectItem value="none">Manual (none)</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                  </div>
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    className="shrink-0"
                    onClick={() => patch({ evals: form.evals.filter((_, i) => i !== index) })}
                  >
                    <Trash2 className="size-3.5" />
                  </Button>
                </div>

                {ev.inputMode === 'single' ? (
                  <div className="space-y-1.5">
                    <FieldLabel required>Prompt</FieldLabel>
                    <Textarea
                      value={ev.input}
                      onChange={(e) => updateEval(index, { input: e.target.value })}
                      rows={3}
                      placeholder="User message sent to the agent"
                    />
                  </div>
                ) : (
                  <div className="space-y-2">
                    <Label>Messages</Label>
                    {ev.messages.map((msg, msgIndex) => (
                      <div
                        key={msgIndex}
                        className="grid grid-cols-1 gap-2 items-center sm:grid-cols-[minmax(6rem,8rem)_minmax(0,1fr)_auto]"
                      >
                        <Select
                          value={msg.role}
                          onValueChange={(v) => {
                            const messages = [...ev.messages]
                            messages[msgIndex] = {
                              ...msg,
                              role: (v ?? 'user') as 'user' | 'assistant',
                            }
                            updateEval(index, { messages })
                          }}
                          items={[
                            { value: 'user', label: 'User' },
                            { value: 'assistant', label: 'Assistant' },
                          ]}
                        >
                          <SelectTrigger className="w-full">
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectItem value="user">User</SelectItem>
                            <SelectItem value="assistant">Assistant</SelectItem>
                          </SelectContent>
                        </Select>
                        <Input
                          value={msg.content}
                          onChange={(e) => {
                            const messages = [...ev.messages]
                            messages[msgIndex] = { ...msg, content: e.target.value }
                            updateEval(index, { messages })
                          }}
                          placeholder="Message content"
                        />
                        <Button
                          type="button"
                          variant="ghost"
                          size="sm"
                          onClick={() =>
                            updateEval(index, {
                              messages: ev.messages.filter((_, i) => i !== msgIndex),
                            })
                          }
                        >
                          <Trash2 className="size-3.5" />
                        </Button>
                      </div>
                    ))}
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      onClick={() =>
                        updateEval(index, {
                          messages: [...ev.messages, { role: 'user', content: '' }],
                        })
                      }
                    >
                      <Plus className="size-3.5" />
                      Add message
                    </Button>
                  </div>
                )}

                {(ev.expectType === 'contains' ||
                  ev.expectType === 'exact' ||
                  ev.expectType === 'regex') && (
                  <div className="space-y-1.5">
                    <FieldLabel required>Expected value</FieldLabel>
                    <Input
                      value={ev.expectValue}
                      onChange={(e) => updateEval(index, { expectValue: e.target.value })}
                      placeholder={
                        ev.expectType === 'regex' ? 'pattern' : 'expected substring or text'
                      }
                    />
                  </div>
                )}

                {ev.expectType === 'llm' && (
                  <div className="space-y-1.5">
                    <FieldLabel required>Criteria</FieldLabel>
                    <Textarea
                      value={ev.expectCriteria}
                      onChange={(e) => updateEval(index, { expectCriteria: e.target.value })}
                      rows={2}
                      placeholder="Rubric for the LLM judge"
                    />
                  </div>
                )}

                <div className="grid gap-3 sm:grid-cols-2">
                  <div className="space-y-1.5">
                    <Label>Tags</Label>
                    <Input
                      value={ev.tags}
                      onChange={(e) => updateEval(index, { tags: e.target.value })}
                      placeholder="smoke, regression"
                    />
                  </div>
                  <div className="space-y-1.5">
                    <Label>Timeout (seconds)</Label>
                    <Input
                      value={ev.timeout}
                      onChange={(e) => updateEval(index, { timeout: e.target.value })}
                      placeholder="120"
                    />
                  </div>
                  <div className="space-y-1.5 sm:col-span-2">
                    <Label>Working dir override</Label>
                    <Input
                      value={ev.workingDir}
                      onChange={(e) => updateEval(index, { workingDir: e.target.value })}
                      placeholder="Optional path for this eval case"
                    />
                  </div>
                </div>

                <label className="flex items-center gap-2 text-sm">
                  <input
                    type="checkbox"
                    checked={!ev.disabled}
                    onChange={(e) => updateEval(index, { disabled: !e.target.checked })}
                    className="rounded border"
                  />
                  Enabled
                </label>
              </li>
            ))}
          </ul>
        )}
        <Button type="button" variant="outline" size="sm" onClick={addEval}>
          <Plus className="size-3.5" />
          Add eval
        </Button>
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
