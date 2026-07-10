// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useCallback, useEffect, useState } from 'react'
import { FileCode2, FlaskConical, FormInput, Plus, Rocket, Trash2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { cn } from '@/lib/utils'
import { AgentForm } from '@/components/customize/AgentForm'
import {
  defaultAgentForm,
  formToAgentYaml,
  mergeFormIntoAgentYaml,
  parseAgentYaml,
  syncYamlFromForm,
  type AgentFormModel,
} from '@/lib/adlAgentForm'
import { useAgentFormOptions } from '@/lib/useAgentFormOptions'
import { ConfirmDeleteDialog } from '@/components/ConfirmDeleteDialog'
import { api } from '@/api'
import type { AgentDeployerInfo, AgentDeployResult, AgentEvalSummary, AgentFileInfo } from '@/types'

type EditMode = 'form' | 'yaml'

const NEW_AGENT_TEMPLATE = `adl: "1.0"
id: my-agent
name: My Agent
description: A custom agent
harness:
  type: claude-code
  sandbox: none
`

function ModeToggle({ mode, onChange }: { mode: EditMode; onChange: (mode: EditMode) => void }) {
  return (
    <div className="inline-flex rounded-lg border p-0.5">
      <button
        type="button"
        className={cn(
          'inline-flex items-center gap-1.5 rounded-md px-2.5 py-1 text-xs font-medium transition-colors',
          mode === 'form' ? 'bg-muted text-foreground' : 'text-muted-foreground hover:text-foreground',
        )}
        onClick={() => onChange('form')}
      >
        <FormInput className="size-3.5" />
        Form
      </button>
      <button
        type="button"
        className={cn(
          'inline-flex items-center gap-1.5 rounded-md px-2.5 py-1 text-xs font-medium transition-colors',
          mode === 'yaml' ? 'bg-muted text-foreground' : 'text-muted-foreground hover:text-foreground',
        )}
        onClick={() => onChange('yaml')}
      >
        <FileCode2 className="size-3.5" />
        YAML
      </button>
    </div>
  )
}

interface Props {
  onChanged?: () => void
}

export function AgentsTab({ onChanged }: Props) {
  const { options, loading: optionsLoading } = useAgentFormOptions()
  const [agents, setAgents] = useState<AgentFileInfo[]>([])
  const [selectedFile, setSelectedFile] = useState<string | null>(null)
  const [content, setContent] = useState('')
  const [form, setForm] = useState<AgentFormModel>(defaultAgentForm())
  const [hasWorkflowSteps, setHasWorkflowSteps] = useState(false)
  const [hasSubAgents, setHasSubAgents] = useState(false)
  const [editMode, setEditMode] = useState<EditMode>('form')
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [creating, setCreating] = useState(false)
  const [newFilename, setNewFilename] = useState('my-agent.yaml')
  const [error, setError] = useState<string | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null)
  const [deployers, setDeployers] = useState<AgentDeployerInfo[]>([])
  const [deployOpen, setDeployOpen] = useState(false)
  const [deployerId, setDeployerId] = useState('')
  const [deploying, setDeploying] = useState(false)
  const [deployResult, setDeployResult] = useState<AgentDeployResult | null>(null)
  const [evalOpen, setEvalOpen] = useState(false)
  const [evalWorkingDir, setEvalWorkingDir] = useState('')
  const [runningEvals, setRunningEvals] = useState(false)
  const [evalSummary, setEvalSummary] = useState<AgentEvalSummary | null>(null)

  const syncFormFromContent = useCallback(
    (yaml: string) => {
      const parsed = parseAgentYaml(yaml, options)
      setForm(parsed.form)
      setHasWorkflowSteps(parsed.hasWorkflowSteps)
      setHasSubAgents(parsed.hasSubAgents)
      if (parsed.parseError) {
        setError('YAML could not be parsed; form shows last known values.')
      }
      return parsed
    },
    [options],
  )

  useEffect(() => {
    if (optionsLoading || (!selectedFile && !creating)) return
    if (editMode === 'form') {
      const yaml = syncYamlFromForm(content, form, options)
      const parsed = parseAgentYaml(yaml, options)
      setForm(parsed.form)
      setHasWorkflowSteps(parsed.hasWorkflowSteps)
      setHasSubAgents(parsed.hasSubAgents)
    } else if (content) {
      syncFormFromContent(content)
    }
    // Re-resolve catalog option ids once options finish loading.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [optionsLoading])

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const list = await api.agents.list()
      setAgents(list.sort((a, b) => a.name.localeCompare(b.name)))
    } catch {
      setAgents([])
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  useEffect(() => {
    void api.agents.listDeployers().then(setDeployers).catch(() => setDeployers([]))
  }, [])

  const openAgent = async (file: string) => {
    setError(null)
    try {
      const res = await api.agents.get(file)
      setSelectedFile(file)
      setContent(res.content)
      setCreating(false)
      setEditMode('form')
      syncFormFromContent(res.content)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load agent')
    }
  }

  const handleModeChange = (mode: EditMode) => {
    if (mode === editMode) return
    if (mode === 'yaml' && editMode === 'form') {
      const merged = syncYamlFromForm(content, form, options)
      setContent(merged)
      const parsed = parseAgentYaml(merged, options)
      setHasWorkflowSteps(parsed.hasWorkflowSteps)
      setHasSubAgents(parsed.hasSubAgents)
    } else if (mode === 'form' && editMode === 'yaml') {
      setError(null)
      syncFormFromContent(content)
    }
    setEditMode(mode)
  }

  const save = async () => {
    if (!selectedFile) return
    setSaving(true)
    setError(null)
    try {
      const yaml = editMode === 'form'
        ? mergeFormIntoAgentYaml(content, form, options)
        : content
      await api.agents.save(selectedFile, yaml)
      setContent(yaml)
      syncFormFromContent(yaml)
      await load()
      onChanged?.()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to save')
    } finally {
      setSaving(false)
    }
  }

  const create = async () => {
    setSaving(true)
    setError(null)
    try {
      const yaml = editMode === 'form' ? formToAgentYaml(form, options) : content
      const info = await api.agents.create(newFilename, yaml)
      await load()
      onChanged?.()
      setCreating(false)
      await openAgent(info.file)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to create agent')
    } finally {
      setSaving(false)
    }
  }

  const remove = async (file: string) => {
    setError(null)
    try {
      await api.agents.remove(file)
      if (selectedFile === file) {
        setSelectedFile(null)
        setContent('')
        setForm(defaultAgentForm())
      }
      await load()
      onChanged?.()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to delete')
    }
  }

  const startCreate = () => {
    setCreating(true)
    setSelectedFile(null)
    setEditMode('form')
    setForm(defaultAgentForm())
    setHasWorkflowSteps(false)
    setHasSubAgents(false)
    setContent(NEW_AGENT_TEMPLATE)
    setNewFilename('my-agent.yaml')
  }

  const agentIdForDeploy = form.id.trim() || agents.find((a) => a.file === selectedFile)?.id || ''

  const enabledEvals = form.evals.filter((e) => !e.disabled && e.name.trim())

  const openEval = () => {
    setEvalSummary(null)
    setEvalWorkingDir('')
    setEvalOpen(true)
  }

  const runEvals = async () => {
    if (!agentIdForDeploy || enabledEvals.length === 0) return
    setRunningEvals(true)
    setError(null)
    try {
      const summary = await api.agents.runEvals(agentIdForDeploy, {
        workingDir: evalWorkingDir.trim() || undefined,
      })
      setEvalSummary(summary)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Eval run failed')
      setEvalOpen(false)
    } finally {
      setRunningEvals(false)
    }
  }

  const openDeploy = () => {
    setDeployResult(null)
    setDeployerId(deployers[0]?.id ?? '')
    setDeployOpen(true)
  }

  const runDeploy = async () => {
    if (!agentIdForDeploy || !deployerId) return
    setDeploying(true)
    setError(null)
    try {
      const result = await api.agents.deploy(agentIdForDeploy, deployerId)
      setDeployResult(result)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Deploy failed')
      setDeployOpen(false)
    } finally {
      setDeploying(false)
    }
  }

  const handleFormChange = (nextForm: AgentFormModel) => {
    setForm(nextForm)
  }

  if (loading || optionsLoading) {
    return <p className="text-sm text-muted-foreground">Loading agents…</p>
  }

  const editing = creating || selectedFile != null

  return (
    <div className="customize-tab-content flex flex-col gap-4 min-h-0 max-w-none">
      <p className="text-sm text-muted-foreground shrink-0">
        Agent definitions in <code className="text-xs">~/.loop/agents/</code> (ADL YAML).
      </p>

      <div className="flex flex-1 min-h-0 gap-4">
        <div className="w-56 shrink-0 flex flex-col gap-2">
          <Button variant="outline" size="sm" className="justify-start" onClick={startCreate}>
            <Plus className="size-3.5" />
            New agent
          </Button>
          <ul className="flex-1 overflow-y-auto rounded-lg border divide-y">
            {agents.map((agent) => (
              <li key={agent.file}>
                <button
                  type="button"
                  className="w-full text-left px-3 py-2 text-sm hover:bg-muted/60 data-active:bg-muted"
                  data-active={selectedFile === agent.file || undefined}
                  onClick={() => void openAgent(agent.file)}
                >
                  <span className="font-medium block truncate">{agent.name}</span>
                  <span className="text-xs text-muted-foreground block truncate">{agent.file}</span>
                </button>
              </li>
            ))}
          </ul>
        </div>

        <div className="flex-1 flex flex-col min-w-0 min-h-0 gap-3">
          {editing ? (
            <>
              <div className="flex items-center justify-between gap-2 shrink-0">
                <div className="min-w-0">
                  {creating ? (
                    <p className="text-sm font-medium">New agent</p>
                  ) : (
                    <p className="text-sm font-medium truncate">{selectedFile}</p>
                  )}
                </div>
                <div className="flex items-center gap-2 shrink-0">
                  <ModeToggle mode={editMode} onChange={handleModeChange} />
                  {!creating && selectedFile && (
                    <Button variant="ghost" size="sm" onClick={() => setDeleteTarget(selectedFile)}>
                      <Trash2 className="size-3.5" />
                    </Button>
                  )}
                </div>
              </div>

              {creating && (
                <div className="space-y-1.5 shrink-0 max-w-md">
                  <Label>
                    Filename <span className="text-destructive">*</span>
                  </Label>
                  <Input value={newFilename} onChange={(e) => setNewFilename(e.target.value)} />
                </div>
              )}

              <div className="flex-1 overflow-y-auto min-h-0">
                {editMode === 'form' ? (
                  <AgentForm
                    form={form}
                    options={options}
                    hasWorkflowSteps={hasWorkflowSteps}
                    hasSubAgents={hasSubAgents}
                    editingAgentId={form.id}
                    onChange={handleFormChange}
                  />
                ) : (
                  <Textarea
                    className="min-h-[420px] font-mono text-xs w-full"
                    value={content}
                    onChange={(e) => setContent(e.target.value)}
                  />
                )}
              </div>

              <div className="flex gap-2 shrink-0">
                {creating ? (
                  <>
                    <Button size="sm" onClick={() => void create()} disabled={saving}>
                      {saving ? 'Creating…' : 'Create agent'}
                    </Button>
                    <Button variant="outline" size="sm" onClick={() => setCreating(false)}>
                      Cancel
                    </Button>
                  </>
                ) : (
                  <>
                    <Button size="sm" onClick={() => void save()} disabled={saving}>
                      {saving ? 'Saving…' : 'Save changes'}
                    </Button>
                    {enabledEvals.length > 0 && agentIdForDeploy && (
                      <Button variant="outline" size="sm" onClick={openEval}>
                        <FlaskConical className="size-3.5" />
                        Run evals
                      </Button>
                    )}
                    {deployers.length > 0 && agentIdForDeploy && (
                      <Button variant="outline" size="sm" onClick={openDeploy}>
                        <Rocket className="size-3.5" />
                        Deploy
                      </Button>
                    )}
                  </>
                )}
              </div>
            </>
          ) : (
            <div className="flex flex-1 items-center justify-center text-sm text-muted-foreground">
              Select an agent to edit, or create a new one.
            </div>
          )}
        </div>
      </div>

      {error && <p className="text-sm text-destructive shrink-0">{error}</p>}

      <ConfirmDeleteDialog
        open={deleteTarget != null}
        onOpenChange={(open) => { if (!open) setDeleteTarget(null) }}
        title="Delete agent?"
        description={
          <>
            This will permanently delete the agent file <strong>{deleteTarget}</strong>. This action
            cannot be undone.
          </>
        }
        onConfirm={async () => {
          if (deleteTarget) await remove(deleteTarget)
        }}
      />

      <Dialog open={deployOpen} onOpenChange={setDeployOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Deploy agent</DialogTitle>
            <DialogDescription>
              Deploy <strong>{agentIdForDeploy}</strong> using an extension agent deployer.
              Registry and platform settings are configured inside the deployer extension.
            </DialogDescription>
          </DialogHeader>
          {deployResult ? (
            <div className="space-y-2 text-sm">
              {deployResult.message && <p>{deployResult.message}</p>}
              {deployResult.deploymentId && (
                <p className="text-muted-foreground">Deployment: {deployResult.deploymentId}</p>
              )}
              {deployResult.endpoint?.url && (
                <p className="text-muted-foreground">Endpoint: {deployResult.endpoint.url}</p>
              )}
              {deployResult.endpoint?.host && deployResult.endpoint.port ? (
                <p className="text-muted-foreground">
                  Endpoint: {deployResult.endpoint.host}:{deployResult.endpoint.port}
                </p>
              ) : null}
            </div>
          ) : (
            <div className="space-y-2">
              <Label htmlFor="deployer-select">Deployer</Label>
              <select
                id="deployer-select"
                className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm"
                value={deployerId}
                onChange={(e) => setDeployerId(e.target.value)}
              >
                {deployers.map((d) => (
                  <option key={d.id} value={d.id}>
                    {d.id}{d.description ? ` — ${d.description}` : ''}
                  </option>
                ))}
              </select>
            </div>
          )}
          <DialogFooter>
            {deployResult ? (
              <Button onClick={() => setDeployOpen(false)}>Close</Button>
            ) : (
              <>
                <Button variant="outline" onClick={() => setDeployOpen(false)}>Cancel</Button>
                <Button onClick={() => void runDeploy()} disabled={deploying || !deployerId}>
                  {deploying ? 'Deploying…' : 'Deploy'}
                </Button>
              </>
            )}
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={evalOpen} onOpenChange={setEvalOpen}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>Run evals</DialogTitle>
            <DialogDescription>
              Run {enabledEvals.length} eval case{enabledEvals.length === 1 ? '' : 's'} for{' '}
              <strong>{agentIdForDeploy}</strong>. Save the agent first so the latest eval
              definitions are used.
            </DialogDescription>
          </DialogHeader>
          {evalSummary ? (
            <div className="space-y-2 max-h-64 overflow-y-auto text-sm">
              {evalSummary.results.map((res) => (
                <div
                  key={res.name}
                  className={cn(
                    'rounded-md border px-3 py-2',
                    res.status === 'pass' && 'border-green-500/40 bg-green-500/5',
                    res.status === 'fail' && 'border-destructive/40 bg-destructive/5',
                    (res.status === 'error' || res.status === 'skip') && 'border-amber-500/40 bg-amber-500/5',
                  )}
                >
                  <div className="flex items-center justify-between gap-2">
                    <span className="font-medium">{res.name}</span>
                    <span className="text-xs uppercase text-muted-foreground">{res.status}</span>
                  </div>
                  {(res.message || res.error) && (
                    <p className="text-xs text-muted-foreground mt-1">{res.error || res.message}</p>
                  )}
                  <p className="text-xs text-muted-foreground mt-0.5">{res.duration}</p>
                </div>
              ))}
              <p className="text-xs text-muted-foreground pt-1">
                {evalSummary.passed} passed, {evalSummary.failed} failed
                {evalSummary.errors > 0 ? `, ${evalSummary.errors} errors` : ''}
              </p>
            </div>
          ) : (
            <div className="space-y-2">
              <Label htmlFor="eval-working-dir">Working directory (optional)</Label>
              <Input
                id="eval-working-dir"
                value={evalWorkingDir}
                onChange={(e) => setEvalWorkingDir(e.target.value)}
                placeholder="Defaults to server process working directory"
              />
              <ul className="text-xs text-muted-foreground list-disc pl-4 space-y-0.5">
                {enabledEvals.map((ev) => (
                  <li key={ev.name}>{ev.name}</li>
                ))}
              </ul>
            </div>
          )}
          <DialogFooter>
            {evalSummary ? (
              <Button onClick={() => setEvalOpen(false)}>Close</Button>
            ) : (
              <>
                <Button variant="outline" onClick={() => setEvalOpen(false)}>Cancel</Button>
                <Button onClick={() => void runEvals()} disabled={runningEvals}>
                  {runningEvals ? 'Running evals…' : 'Run evals'}
                </Button>
              </>
            )}
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
