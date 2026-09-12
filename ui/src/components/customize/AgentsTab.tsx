// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useCallback, useEffect, useMemo, useState } from 'react'
import { FileCode2, ChevronLeft, Copy, FlaskConical, FormInput, MoreHorizontal, Pencil, Plus, Rocket, Trash2 } from 'lucide-react'
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
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { cn } from '@/lib/utils'
import { AgentForm } from '@/components/customize/AgentForm'
import {
  allocateAgentCopyIdentity,
  defaultAgentForm,
  formToAgentYaml,
  mergeFormIntoAgentYaml,
  parseAgentYaml,
  syncYamlFromForm,
  type AgentFormModel,
} from '@/lib/adlAgentForm'
import { useAgentFormOptions } from '@/lib/useAgentFormOptions'
import { useIsMobile } from '@/hooks/use-mobile'
import { ConfirmDeleteDialog } from '@/components/ConfirmDeleteDialog'
import { SearchInput } from '@/components/SearchInput'
import { api } from '@/api'
import { filterBySearchQuery } from '@/lib/searchFilter'
import type { AgentDeployerInfo, AgentDeployResult, AgentEvalSummary, AgentFileInfo } from '@/types'
import { EvalResultRow } from '@/components/customize/EvalResultRow'

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
  const isMobile = useIsMobile()
  const { options, loading: optionsLoading } = useAgentFormOptions()
  const [agents, setAgents] = useState<AgentFileInfo[]>([])
  const [selectedFile, setSelectedFile] = useState<string | null>(null)
  const [content, setContent] = useState('')
  const [form, setForm] = useState<AgentFormModel>(defaultAgentForm())
  const [hasWorkflowSteps, setHasWorkflowSteps] = useState(false)
  const [editMode, setEditMode] = useState<EditMode>('form')
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [creating, setCreating] = useState(false)
  const [newFilename, setNewFilename] = useState('my-agent.yaml')
  const [renaming, setRenaming] = useState(false)
  const [renameValue, setRenameValue] = useState('')
  const [renamingSaving, setRenamingSaving] = useState(false)
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
  const [runningEvalCase, setRunningEvalCase] = useState<string | null>(null)
  const [evalSummary, setEvalSummary] = useState<AgentEvalSummary | null>(null)
  const [evalRunError, setEvalRunError] = useState<string | null>(null)
  const [evalCasesToRun, setEvalCasesToRun] = useState<string[] | undefined>(undefined)
  const [agentSearchQuery, setAgentSearchQuery] = useState('')

  const filteredAgents = useMemo(
    () =>
      filterBySearchQuery(agents, agentSearchQuery, (agent) =>
        [agent.name, agent.description, agent.file].filter(Boolean).join(' '),
      ),
    [agents, agentSearchQuery],
  )

  const syncFormFromContent = useCallback(
    (yaml: string) => {
      const parsed = parseAgentYaml(yaml, options)
      setForm(parsed.form)
      setHasWorkflowSteps(parsed.hasWorkflowSteps)
      if (parsed.parseError) {
        setError('YAML could not be parsed; form shows last known values.')
      }
      return parsed
    },
    [options],
  )

  useEffect(() => {
    if (optionsLoading || (!selectedFile && !creating)) return
    if (!content) return
    // Re-resolve catalog option ids once options finish loading.
    // Always parse from YAML content — never syncYamlFromForm(default/stale form)
    // first, which would wipe orchestration and other fields not yet in form state.
    syncFormFromContent(content)
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
    setRenaming(false)
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

  const startRename = () => {
    if (!selectedFile || creating) return
    setRenameValue(selectedFile)
    setRenaming(true)
    setError(null)
  }

  const cancelRename = () => {
    setRenaming(false)
    setRenameValue('')
  }

  const commitRename = async () => {
    if (!selectedFile || creating || renamingSaving) return
    const next = renameValue.trim()
    if (!next || next === selectedFile) {
      cancelRename()
      return
    }
    setRenamingSaving(true)
    setError(null)
    try {
      const info = await api.agents.rename(selectedFile, next)
      setSelectedFile(info.file)
      setRenaming(false)
      setRenameValue('')
      await load()
      onChanged?.()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to rename agent file')
    } finally {
      setRenamingSaving(false)
    }
  }

  const handleModeChange = (mode: EditMode) => {
    if (mode === editMode) return
    if (mode === 'yaml' && editMode === 'form') {
      const merged = syncYamlFromForm(content, form, options)
      setContent(merged)
      const parsed = parseAgentYaml(merged, options)
      setHasWorkflowSteps(parsed.hasWorkflowSteps)
    } else if (mode === 'form' && editMode === 'yaml') {
      setError(null)
      syncFormFromContent(content)
    }
    setEditMode(mode)
  }

  const saveAgentYaml = async (yaml: string): Promise<void> => {
    if (!selectedFile) return
    await api.agents.save(selectedFile, yaml)
    setContent(yaml)
    syncFormFromContent(yaml)
    await load()
    onChanged?.()
  }

  const save = async () => {
    if (!selectedFile) return
    setSaving(true)
    setError(null)
    try {
      const yaml = editMode === 'form'
        ? mergeFormIntoAgentYaml(content, form, options)
        : content
      await saveAgentYaml(yaml)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to save')
    } finally {
      setSaving(false)
    }
  }

  const ensureSaved = async (): Promise<boolean> => {
    if (!selectedFile) return false
    setError(null)
    try {
      const yaml = editMode === 'form'
        ? mergeFormIntoAgentYaml(content, form, options)
        : content
      if (yaml !== content) {
        setSaving(true)
        await saveAgentYaml(yaml)
        setSaving(false)
      }
      return true
    } catch (e) {
      setSaving(false)
      setError(e instanceof Error ? e.message : 'Failed to save before running evals')
      return false
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

  const duplicate = async (file: string) => {
    setSaving(true)
    setError(null)
    try {
      let yaml: string
      let sourceId: string
      let sourceName: string
      if (selectedFile === file && !creating) {
        yaml = editMode === 'form'
          ? mergeFormIntoAgentYaml(content, form, options)
          : content
        const parsed = parseAgentYaml(yaml, options)
        sourceId = parsed.form.id || form.id
        sourceName = parsed.form.name || form.name
      } else {
        const res = await api.agents.get(file)
        yaml = res.content
        const parsed = parseAgentYaml(yaml, options)
        sourceId = parsed.form.id
        sourceName = parsed.form.name
      }
      const copy = allocateAgentCopyIdentity({
        file,
        id: sourceId,
        name: sourceName,
        existing: agents,
      })
      const parsed = parseAgentYaml(yaml, options)
      const copyYaml = mergeFormIntoAgentYaml(
        yaml,
        { ...parsed.form, id: copy.id, name: copy.name },
        options,
      )
      const info = await api.agents.create(copy.file, copyYaml)
      await load()
      onChanged?.()
      await openAgent(info.file)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to duplicate agent')
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
    setRenaming(false)
    setEditMode('form')
    setForm(defaultAgentForm())
    setHasWorkflowSteps(false)
    setContent(NEW_AGENT_TEMPLATE)
    setNewFilename('my-agent.yaml')
  }

  const closeMobileEditor = () => {
    setSelectedFile(null)
    setCreating(false)
    setRenaming(false)
  }

  const agentIdForDeploy = form.id.trim() || agents.find((a) => a.file === selectedFile)?.id || ''

  const enabledEvals = form.evals.filter((e) => !e.disabled && e.name.trim())

  const openEval = (cases?: string[]) => {
    setEvalSummary(null)
    setEvalRunError(null)
    setEvalWorkingDir('')
    setEvalCasesToRun(cases)
    setEvalOpen(true)
  }

  const runEvalsForAgent = async (cases?: string[]) => {
    if (!agentIdForDeploy) return
    const caseFilter = cases ?? evalCasesToRun
    const evalCount = caseFilter?.length ?? enabledEvals.length
    if (evalCount === 0) return

    setRunningEvals(true)
    setEvalRunError(null)
    if (caseFilter?.length === 1) {
      setRunningEvalCase(caseFilter[0])
    }
    try {
      if (!(await ensureSaved())) return
      const summary = await api.agents.runEvals(agentIdForDeploy, {
        workingDir: evalWorkingDir.trim() || undefined,
        cases: caseFilter,
      })
      setEvalSummary(summary)
    } catch (e) {
      setEvalRunError(e instanceof Error ? e.message : 'Eval run failed')
    } finally {
      setRunningEvals(false)
      setRunningEvalCase(null)
    }
  }

  const handleRunEvalCase = (name: string) => {
    openEval([name])
    void runEvalsForAgent([name])
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
  const mobileShowEditor = isMobile && editing

  return (
    <div className="customize-tab-content customize-tab-content--split flex h-full min-h-0 max-w-none flex-col gap-4 overflow-hidden">
      <p className="text-sm text-muted-foreground shrink-0">
        Agent definitions in <code className="text-xs">~/.nui/agents/</code> (ADL YAML).
      </p>

      <div className="flex min-h-0 flex-1 gap-4 overflow-hidden">
        {(!isMobile || !mobileShowEditor) && (
        <div className="flex w-full min-h-0 shrink-0 flex-col gap-2 overflow-hidden md:w-56">
          <Button variant="outline" size="sm" className="justify-start shrink-0" onClick={startCreate}>
            <Plus className="size-3.5" />
            New agent
          </Button>
          <SearchInput
            value={agentSearchQuery}
            onChange={setAgentSearchQuery}
            placeholder="Search by name or description…"
            aria-label="Search agent definitions"
            className="shrink-0"
            autoFocus
          />
          <ul className="min-h-0 flex-1 overflow-y-auto overscroll-contain rounded-lg border divide-y">
            {filteredAgents.length === 0 ? (
              <li className="px-3 py-4 text-xs text-muted-foreground">
                {agentSearchQuery.trim() ? 'No agents match your search.' : 'No agents yet.'}
              </li>
            ) : (
              filteredAgents.map((agent) => (
              <li key={agent.file} className="group/agent-item relative flex items-stretch">
                <button
                  type="button"
                  className="min-w-0 flex-1 text-left px-3 py-2 pr-8 text-sm hover:bg-muted/60 data-active:bg-muted"
                  data-active={selectedFile === agent.file || undefined}
                  onClick={() => void openAgent(agent.file)}
                >
                  <span className="font-medium block truncate">{agent.name}</span>
                  <span className="text-xs text-muted-foreground block truncate">{agent.file}</span>
                </button>
                <DropdownMenu>
                  <DropdownMenuTrigger
                    render={
                      <button
                        type="button"
                        className="absolute right-1 top-1/2 -translate-y-1/2 inline-flex size-6 items-center justify-center rounded-md text-muted-foreground hover:bg-muted hover:text-foreground md:opacity-0 group-hover/agent-item:opacity-100 group-focus-within/agent-item:opacity-100 aria-expanded:opacity-100"
                        aria-label={`Options for ${agent.name}`}
                        onClick={(e) => e.stopPropagation()}
                      />
                    }
                  >
                    <MoreHorizontal className="size-3.5" />
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end" className="w-40">
                    <DropdownMenuItem
                      disabled={saving}
                      onClick={() => void duplicate(agent.file)}
                    >
                      <Copy className="size-3.5 text-muted-foreground" />
                      Duplicate
                    </DropdownMenuItem>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem
                      className="text-destructive data-highlighted:bg-destructive/10 data-highlighted:text-destructive"
                      onClick={() => setDeleteTarget(agent.file)}
                    >
                      <Trash2 className="size-3.5" />
                      Delete
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </li>
              ))
            )}
          </ul>
        </div>
        )}

        {(!isMobile || mobileShowEditor) && (
        <div className="flex min-h-0 min-w-0 flex-1 flex-col gap-3 overflow-hidden">
          {editing ? (
            <>
              <div className="flex items-center justify-between gap-2 shrink-0">
                <div className="flex min-w-0 items-center gap-2">
                  {isMobile && (
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      className="shrink-0"
                      onClick={closeMobileEditor}
                      aria-label="Back to agent list"
                    >
                      <ChevronLeft className="size-4" />
                    </Button>
                  )}
                  <div className="min-w-0 flex-1">
                    {creating ? (
                      <p className="text-sm font-medium">New agent</p>
                    ) : renaming ? (
                      <Input
                        value={renameValue}
                        onChange={(e) => setRenameValue(e.target.value)}
                        onBlur={() => void commitRename()}
                        onKeyDown={(e) => {
                          if (e.key === 'Enter') {
                            e.preventDefault()
                            e.currentTarget.blur()
                          } else if (e.key === 'Escape') {
                            e.preventDefault()
                            cancelRename()
                          }
                        }}
                        disabled={renamingSaving}
                        autoFocus
                        aria-label="Rename agent file"
                        className="h-8 max-w-md font-mono text-sm"
                      />
                    ) : (
                      <button
                        type="button"
                        className="group/rename inline-flex max-w-full items-center gap-1.5 rounded-md px-1 -mx-1 py-0.5 text-left text-sm font-medium hover:bg-muted/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                        onClick={startRename}
                        title="Rename file"
                        aria-label={`Rename ${selectedFile}`}
                      >
                        <span className="truncate font-mono">{selectedFile}</span>
                        <Pencil className="size-3 shrink-0 text-muted-foreground opacity-0 transition-opacity group-hover/rename:opacity-100 group-focus-visible/rename:opacity-100" />
                      </button>
                    )}
                  </div>
                </div>
                <div className="flex items-center gap-2 shrink-0">
                  <ModeToggle mode={editMode} onChange={handleModeChange} />
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

              <div className="min-h-0 flex-1 overflow-y-auto overscroll-contain">
                {editMode === 'form' ? (
                  <AgentForm
                    form={form}
                    options={options}
                    hasWorkflowSteps={hasWorkflowSteps}
                    editingAgentId={form.id}
                    onChange={handleFormChange}
                    onRunEvalCase={
                      !creating && selectedFile && agentIdForDeploy
                        ? handleRunEvalCase
                        : undefined
                    }
                    runningEvalCase={runningEvalCase}
                  />
                ) : (
                  <Textarea
                    className="min-h-[420px] font-mono text-xs w-full"
                    value={content}
                    onChange={(e) => setContent(e.target.value)}
                  />
                )}
              </div>

              <div className="flex flex-wrap gap-2 shrink-0">
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
                      <Button variant="outline" size="sm" onClick={() => openEval()}>
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
            !isMobile && (
            <div className="flex flex-1 items-start text-sm text-muted-foreground">
              Select an agent to edit, or create a new one.
            </div>
            )
          )}
        </div>
        )}
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
              {evalCasesToRun?.length === 1 ? (
                <>
                  Run eval <strong>{evalCasesToRun[0]}</strong> for{' '}
                  <strong>{agentIdForDeploy}</strong>.
                </>
              ) : (
                <>
                  Run {enabledEvals.length} eval case{enabledEvals.length === 1 ? '' : 's'} for{' '}
                  <strong>{agentIdForDeploy}</strong>. Unsaved changes are saved automatically
                  before running.
                </>
              )}
            </DialogDescription>
          </DialogHeader>
          {evalRunError && (
            <p className="text-sm text-destructive">{evalRunError}</p>
          )}
          {evalSummary ? (
            <div className="space-y-2 max-h-80 overflow-y-auto text-sm">
              {evalSummary.results.map((res) => (
                <EvalResultRow key={res.name} res={res} />
              ))}
              <p className="text-xs text-muted-foreground pt-1">
                {evalSummary.passed} passed, {evalSummary.failed} failed
                {evalSummary.errors > 0 ? `, ${evalSummary.errors} errors` : ''}
                {evalSummary.skipped > 0 ? `, ${evalSummary.skipped} skipped` : ''}
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
                disabled={runningEvals}
              />
              <p className="text-xs text-muted-foreground">
                Per-eval working dir in Advanced overrides this default.
              </p>
              <ul className="text-xs text-muted-foreground list-disc pl-4 space-y-0.5">
                {(evalCasesToRun ?? enabledEvals.map((ev) => ev.name)).map((name) => (
                  <li key={name}>{name}</li>
                ))}
              </ul>
            </div>
          )}
          <DialogFooter>
            {evalSummary ? (
              <Button onClick={() => setEvalOpen(false)}>Close</Button>
            ) : (
              <>
                <Button variant="outline" onClick={() => setEvalOpen(false)} disabled={runningEvals}>
                  Cancel
                </Button>
                <Button
                  onClick={() => void runEvalsForAgent()}
                  disabled={runningEvals || enabledEvals.length === 0}
                >
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
