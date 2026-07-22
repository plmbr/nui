// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { isMap, parse, parseDocument, YAMLMap } from 'yaml'
import type { MCPServer } from '@/types'

export interface KeyValue {
  key: string
  value: string
}

export interface FormSkill {
  optionId: string
  name: string
}

export interface FormMCPServer {
  optionId: string
  name: string
}

export type ToolApprovalPolicy = 'default' | 'all' | 'allowlist' | 'denylist' | ''
export type HitlMode = 'interactive' | 'auto' | 'off' | ''
export type EvalExpectType = 'contains' | 'exact' | 'regex' | 'llm' | 'none' | ''

export interface FormEvalMessage {
  role: 'user' | 'assistant'
  content: string
}

export interface FormEval {
  name: string
  description: string
  inputMode: 'single' | 'conversation'
  input: string
  messages: FormEvalMessage[]
  expectType: EvalExpectType
  expectValue: string
  expectCriteria: string
  tags: string
  timeout: string
  workingDir: string
  disabled: boolean
}

export interface AgentFormModel {
  id: string
  name: string
  description: string
  tags: string
  systemPrompt: string
  promptMode: 'user' | 'auto'
  defaultPrompt: string
  workingDirInput: boolean
  harnessOptionId: string
  harnessModel: string
  apiProvider: string
  innerHarness: string
  dockerImage: string
  containerPort: string
  remoteHost: string
  remotePort: string
  env: KeyValue[]
  skills: FormSkill[]
  mcpServers: FormMCPServer[]
  toolApprovalPolicy: ToolApprovalPolicy
  toolApprovalTools: string[]
  hitlMode: HitlMode
  evals: FormEval[]
  subAgents: string[]
}

export interface AgentFormOptions {
  harnesses: HarnessOption[]
  skills: SkillOption[]
  mcpServers: MCPOption[]
  agents: AgentOption[]
}

export interface AgentOption {
  id: string
  label: string
  description: string
  group: 'Built-in' | 'Installed' | 'Extension'
}

export interface HarnessOption {
  id: string
  label: string
  group: 'Built-in' | 'Extension'
  harnessType: string
  sandbox?: string
}

export interface SkillOption {
  id: string
  label: string
  group: string
  name: string
  ref: string
}

export interface MCPOption {
  id: string
  label: string
  group: string
  name: string
  ref?: string
  server?: MCPServer
}

export interface ParsedAgentDoc {
  form: AgentFormModel
  hasWorkflowSteps: boolean
  hasSubAgents: boolean
  parseError?: boolean
}

const BUILTIN_HARNESSES: HarnessOption[] = [
  { id: 'builtin:claude-code', label: 'Claude Code', group: 'Built-in', harnessType: 'claude-code', sandbox: 'none' },
  { id: 'builtin:pi', label: 'Pi', group: 'Built-in', harnessType: 'pi', sandbox: 'none' },
  { id: 'builtin:codex', label: 'Codex', group: 'Built-in', harnessType: 'codex', sandbox: 'none' },
  { id: 'builtin:opencode', label: 'OpenCode', group: 'Built-in', harnessType: 'opencode', sandbox: 'none' },
  { id: 'builtin:api', label: 'API (LLM provider)', group: 'Built-in', harnessType: 'api' },
  { id: 'builtin:devcontainer', label: 'Dev container', group: 'Built-in', harnessType: 'devcontainer' },
  { id: 'builtin:docker', label: 'Docker (HTTP/SSE container)', group: 'Built-in', harnessType: 'docker' },
  { id: 'builtin:remote', label: 'Remote (HTTP/SSE server)', group: 'Built-in', harnessType: 'remote' },
]

export function defaultAgentForm(): AgentFormModel {
  return {
    id: 'my-agent',
    name: 'My Agent',
    description: '',
    tags: '',
    systemPrompt: '',
    promptMode: 'user',
    defaultPrompt: '',
    workingDirInput: false,
    harnessOptionId: 'builtin:claude-code',
    harnessModel: '',
    apiProvider: 'anthropic',
    innerHarness: 'claude-code',
    dockerImage: '',
    containerPort: '',
    remoteHost: '127.0.0.1',
    remotePort: '9090',
    env: [],
    skills: [],
    mcpServers: [],
    toolApprovalPolicy: '',
    toolApprovalTools: [],
    hitlMode: '',
    evals: [],
    subAgents: [],
  }
}

export function buildHarnessOptions(agentTypes: Array<{ id: string; label: string; source?: string; harness: string }>): HarnessOption[] {
  const options = [...BUILTIN_HARNESSES]
  const seen = new Set(options.map((o) => o.id))
  for (const t of agentTypes) {
    if (t.source !== 'extension') continue
    if (!t.id.startsWith('ext:')) continue
    const id = `ext:${t.id.slice(4)}`
    if (seen.has(id)) continue
    seen.add(id)
    options.push({
      id,
      label: t.label,
      group: 'Extension',
      harnessType: t.id,
    })
  }
  return options
}

function mapToEntries(obj: Record<string, string> | undefined): KeyValue[] {
  if (!obj) return []
  return Object.entries(obj).map(([key, value]) => ({ key, value: String(value) }))
}

function entriesToMap(entries: KeyValue[]): Record<string, string> | undefined {
  const out: Record<string, string> = {}
  for (const { key, value } of entries) {
    const k = key.trim()
    if (!k) continue
    out[k] = value
  }
  return Object.keys(out).length > 0 ? out : undefined
}

function harnessOptionIdFromDoc(
  harness: Record<string, unknown> | undefined,
  options: HarnessOption[],
): string {
  const type = String(harness?.type ?? 'claude-code')
  if (type.startsWith('ext:')) {
    const id = `ext:${type.slice(4)}`
    return options.find((o) => o.harnessType === type)?.id ?? id
  }
  if (type === 'docker' || type === 'remote' || type === 'api' || type === 'devcontainer') {
    return `builtin:${type}`
  }
  return options.find((o) => o.harnessType === type && o.group === 'Built-in')?.id ?? `builtin:${type}`
}

function findSkillOption(skill: Record<string, unknown>, options: SkillOption[]): FormSkill | null {
  const name = String(skill.name ?? '').trim()
  if (!name) return null
  const ref = skill.ref != null ? String(skill.ref) : ''
  const path = skill.path != null ? String(skill.path) : ''
  const content = skill.content != null ? String(skill.content) : ''
  if (ref) {
    const match = options.find((o) => o.ref === ref)
    return { optionId: match?.id ?? `custom:ref:${ref}`, name: match?.name ?? name }
  }
  if (path) {
    return { optionId: `custom:path:${path}`, name }
  }
  if (content) {
    return { optionId: 'custom:content', name }
  }
  return { optionId: 'custom:unknown', name }
}

function findMCPOption(server: Record<string, unknown>, options: MCPOption[]): FormMCPServer | null {
  const name = String(server.name ?? '').trim()
  if (!name) return null
  const ref = server.ref != null ? String(server.ref) : ''
  if (ref) {
    const match = options.find((o) => o.ref === ref)
    return { optionId: match?.id ?? `custom:ref:${ref}`, name: match?.name ?? name }
  }
  const match = options.find((o) => o.server?.name === name && !o.ref)
  if (match) return { optionId: match.id, name }
  return { optionId: `custom:inline:${name}`, name }
}

function aiAssetsLists(doc: Record<string, unknown>) {
  const aiAssets = doc.aiAssets as Record<string, unknown> | undefined
  return {
    skills: (aiAssets?.skills as Record<string, unknown>[] | undefined) ?? [],
    mcpServers: (aiAssets?.mcpServers as Record<string, unknown>[] | undefined) ?? [],
  }
}

const TOOL_APPROVAL_POLICIES = new Set(['default', 'all', 'allowlist', 'denylist'])
const HITL_MODES = new Set(['interactive', 'auto', 'off'])

function parseToolApprovalPolicy(value: unknown): ToolApprovalPolicy {
  const policy = String(value ?? '').trim()
  return TOOL_APPROVAL_POLICIES.has(policy) ? (policy as ToolApprovalPolicy) : ''
}

function parseHitlMode(value: unknown): HitlMode {
  const mode = String(value ?? '').trim()
  return HITL_MODES.has(mode) ? (mode as HitlMode) : ''
}

function parseStringList(value: unknown): string[] {
  if (!Array.isArray(value)) return []
  return value.map((item) => String(item).trim()).filter(Boolean)
}

const EVAL_EXPECT_TYPES = new Set(['contains', 'exact', 'regex', 'llm', 'none'])

function parseEvalExpectType(value: unknown): EvalExpectType {
  const t = String(value ?? '').trim()
  return EVAL_EXPECT_TYPES.has(t) ? (t as EvalExpectType) : ''
}

function parseFormEval(raw: Record<string, unknown>): FormEval | null {
  const name = String(raw.name ?? '').trim()
  if (!name) return null

  const messagesRaw = (raw.messages as Record<string, unknown>[] | undefined) ?? []
  const messages: FormEvalMessage[] = messagesRaw
    .map((m) => ({
      role: String(m.role ?? 'user').trim() === 'assistant' ? 'assistant' as const : 'user' as const,
      content: String(m.content ?? ''),
    }))
    .filter((m) => m.content.trim() !== '')

  const input = String(raw.input ?? '').trim()
  const inputMode: 'single' | 'conversation' = messages.length > 0 ? 'conversation' : 'single'

  const expect = raw.expect as Record<string, unknown> | undefined
  const tags = parseStringList(raw.tags)

  return {
    name,
    description: String(raw.description ?? ''),
    inputMode,
    input,
    messages,
    expectType: parseEvalExpectType(expect?.type),
    expectValue: String(expect?.value ?? ''),
    expectCriteria: String(expect?.criteria ?? ''),
    tags: tags.join(', '),
    timeout: raw.timeout != null ? String(raw.timeout) : '',
    workingDir: String(raw.workingDir ?? ''),
    disabled: raw.disabled === true,
  }
}

function parseEvals(doc: Record<string, unknown>): FormEval[] {
  const raw = doc.evals as Record<string, unknown>[] | undefined
  if (!Array.isArray(raw)) return []
  return raw.map((e) => parseFormEval(e)).filter(Boolean) as FormEval[]
}

function parseSubAgents(doc: Record<string, unknown>): string[] {
  const raw = doc.subAgents
  if (!Array.isArray(raw)) return []
  return raw.map((item) => String(item).trim()).filter(Boolean)
}

function hasMultiStepPipeline(steps: unknown[] | undefined): boolean {
  if (!Array.isArray(steps) || steps.length === 0) return false
  if (steps.length > 1) return true
  const step = steps[0]
  if (!step || typeof step !== 'object') return false
  return String((step as Record<string, unknown>).type ?? '').trim() === 'hitl'
}

/** Read supported form fields from YAML without modifying the source text. */
export function parseAgentYaml(content: string, options: AgentFormOptions): ParsedAgentDoc {
  let doc: Record<string, unknown> | null = null
  try {
    doc = parse(content) as Record<string, unknown> | null
  } catch {
    return { form: defaultAgentForm(), hasWorkflowSteps: false, hasSubAgents: false, parseError: true }
  }
  if (!doc || typeof doc !== 'object') {
    return { form: defaultAgentForm(), hasWorkflowSteps: false, hasSubAgents: false }
  }

  const harness = doc.harness as Record<string, unknown> | undefined
  const toolApprovals = doc.toolApprovals as Record<string, unknown> | undefined
  const hitl = doc.hitl as Record<string, unknown> | undefined
  const { skills: skillsRaw, mcpServers: mcpRaw } = aiAssetsLists(doc)
  const steps = doc.steps as unknown[] | undefined
  const hasWorkflowSteps = hasMultiStepPipeline(steps)
  const subAgents = parseSubAgents(doc)
  const hasSubAgents = subAgents.length > 0

  const form: AgentFormModel = {
    id: String(doc.id ?? doc.name ?? 'my-agent'),
    name: String(doc.name ?? doc.id ?? 'My Agent'),
    description: String(doc.description ?? ''),
    tags: parseStringList(doc.tags).join(', '),
    systemPrompt: String(doc.systemPrompt ?? ''),
    promptMode: doc.promptMode === 'auto' ? 'auto' : 'user',
    defaultPrompt: String(doc.defaultPrompt ?? ''),
    workingDirInput: doc.workingDirInput === true,
    harnessOptionId: harnessOptionIdFromDoc(harness, options.harnesses),
    harnessModel: String(harness?.model ?? ''),
    apiProvider: String(harness?.provider ?? 'anthropic'),
    innerHarness: String(harness?.innerHarness ?? 'claude-code'),
    dockerImage: String(harness?.image ?? ''),
    containerPort: harness?.containerPort != null ? String(harness.containerPort) : '',
    remoteHost: String(harness?.host ?? '127.0.0.1'),
    remotePort: harness?.port != null ? String(harness.port) : '9090',
    env: mapToEntries(doc.env as Record<string, string> | undefined),
    skills: skillsRaw.map((s) => findSkillOption(s, options.skills)).filter(Boolean) as FormSkill[],
    mcpServers: mcpRaw.map((s) => findMCPOption(s, options.mcpServers)).filter(Boolean) as FormMCPServer[],
    toolApprovalPolicy: parseToolApprovalPolicy(toolApprovals?.policy),
    toolApprovalTools: parseStringList(toolApprovals?.tools),
    hitlMode: parseHitlMode(hitl?.mode),
    evals: parseEvals(doc),
    subAgents,
  }

  return { form, hasWorkflowSteps, hasSubAgents }
}

function resolveHarnessOption(optionId: string, options: HarnessOption[]): HarnessOption | undefined {
  return options.find((o) => o.id === optionId)
}

function setMapKey(map: YAMLMap, key: string, value: unknown) {
  if (value === undefined || value === null || value === '') {
    map.delete(key)
  } else {
    map.set(key, value)
  }
}

function ensureMap(root: YAMLMap, key: string): YAMLMap {
  let child = root.get(key)
  if (!isMap(child)) {
    child = new YAMLMap()
    root.set(key, child)
  }
  return child as YAMLMap
}

function mergeHarness(root: YAMLMap, form: AgentFormModel, options: HarnessOption[]) {
  const selected = resolveHarnessOption(form.harnessOptionId, options)
  const harnessType = selected?.harnessType ?? 'claude-code'
  const harness = ensureMap(root, 'harness')

  harness.set('type', harnessType)

  if (harnessType.startsWith('ext:')) {
    for (const key of ['model', 'provider', 'innerHarness', 'image', 'containerPort', 'host', 'port']) harness.delete(key)
    return
  }

  if (harnessType === 'api') {
    setMapKey(harness, 'provider', form.apiProvider.trim() || 'anthropic')
    setMapKey(harness, 'model', form.harnessModel.trim() || undefined)
    for (const key of ['innerHarness', 'image', 'containerPort', 'host', 'port']) harness.delete(key)
    return
  }

  if (harnessType === 'devcontainer') {
    setMapKey(harness, 'innerHarness', form.innerHarness.trim() || 'claude-code')
    for (const key of ['model', 'provider', 'image', 'containerPort', 'host', 'port']) harness.delete(key)
    return
  }

  if (harnessType === 'docker') {
    setMapKey(harness, 'image', form.dockerImage.trim() || undefined)
    setMapKey(harness, 'containerPort', form.containerPort.trim() ? Number(form.containerPort) : undefined)
    for (const key of ['model', 'host', 'port']) harness.delete(key)
    return
  }

  if (harnessType === 'remote') {
    setMapKey(harness, 'host', form.remoteHost.trim() || undefined)
    setMapKey(harness, 'port', form.remotePort.trim() ? Number(form.remotePort) : undefined)
    for (const key of ['model', 'image', 'containerPort']) harness.delete(key)
    return
  }

  setMapKey(harness, 'model', form.harnessModel.trim() || undefined)
  for (const key of ['provider', 'innerHarness', 'image', 'containerPort', 'host', 'port']) harness.delete(key)
}

function buildSkillEntry(
  skill: FormSkill,
  options: SkillOption[],
  originalSkills: Record<string, unknown>[],
): Record<string, unknown> {
  if (skill.optionId === 'custom:content' || skill.optionId === 'custom:path' || skill.optionId === 'custom:unknown') {
    const orig = originalSkills.find((s) => String(s.name ?? '') === skill.name)
    if (orig) {
      const next = { ...orig }
      if (skill.name.trim()) next.name = skill.name.trim()
      return next
    }
  }

  const opt = options.find((o) => o.id === skill.optionId)
  const name = skill.name.trim() || opt?.name || 'skill'
  if (opt?.ref) return { name, ref: opt.ref }
  if (skill.optionId.startsWith('custom:ref:')) {
    return { name, ref: skill.optionId.slice('custom:ref:'.length) }
  }
  if (skill.optionId.startsWith('custom:path:')) {
    return { name, path: skill.optionId.slice('custom:path:'.length) }
  }
  return { name, ref: name }
}

function buildMCPEntry(
  server: FormMCPServer,
  options: MCPOption[],
  originalServers: Record<string, unknown>[],
): Record<string, unknown> {
  if (server.optionId.startsWith('custom:inline:')) {
    const origName = server.optionId.slice('custom:inline:'.length)
    const orig = originalServers.find((s) => String(s.name ?? '') === origName)
    if (orig) {
      const next = { ...orig }
      if (server.name.trim()) next.name = server.name.trim()
      return next
    }
  }

  const opt = options.find((o) => o.id === server.optionId)
  const name = server.name.trim() || opt?.name || 'mcp-server'
  if (opt?.server) {
    return Object.fromEntries(
      Object.entries({ ...opt.server, name }).filter(([, v]) => v != null && v !== ''),
    )
  }
  if (opt?.ref) return { name, ref: opt.ref }
  if (server.optionId.startsWith('custom:ref:')) {
    return { name, ref: server.optionId.slice('custom:ref:'.length) }
  }
  return { name }
}

function mergeAiAssets(
  root: YAMLMap,
  form: AgentFormModel,
  options: AgentFormOptions,
  original: Record<string, unknown>,
) {
  const { skills: originalSkills, mcpServers: originalMcp } = aiAssetsLists(original)
  const skills = form.skills.map((s) => buildSkillEntry(s, options.skills, originalSkills))
  const mcpServers = form.mcpServers.map((s) => buildMCPEntry(s, options.mcpServers, originalMcp))

  if (skills.length === 0 && mcpServers.length === 0) {
    root.delete('aiAssets')
    return
  }

  const aiAssets = ensureMap(root, 'aiAssets')
  if (skills.length > 0) aiAssets.set('skills', skills)
  else aiAssets.delete('skills')
  if (mcpServers.length > 0) aiAssets.set('mcpServers', mcpServers)
  else aiAssets.delete('mcpServers')
  if (aiAssets.items.length === 0) root.delete('aiAssets')
}

function mergeToolApprovals(root: YAMLMap, form: AgentFormModel) {
  const policy = form.toolApprovalPolicy.trim()
  const tools = form.toolApprovalTools.map((tool) => tool.trim()).filter(Boolean)

  if (!policy) {
    root.delete('toolApprovals')
    return
  }

  const toolApprovals = ensureMap(root, 'toolApprovals')
  toolApprovals.set('policy', policy)
  if (policy === 'allowlist' || policy === 'denylist') {
    if (tools.length > 0) toolApprovals.set('tools', tools)
    else toolApprovals.delete('tools')
  } else {
    toolApprovals.delete('tools')
  }
}

function mergeHitl(root: YAMLMap, form: AgentFormModel) {
  const mode = form.hitlMode.trim()
  if (!mode) {
    const hitl = root.get('hitl')
    if (isMap(hitl)) {
      ;(hitl as YAMLMap).delete('mode')
      if ((hitl as YAMLMap).items.length === 0) root.delete('hitl')
    } else {
      root.delete('hitl')
    }
    return
  }

  const hitl = ensureMap(root, 'hitl')
  hitl.set('mode', mode)
}

function buildEvalEntry(ev: FormEval): Record<string, unknown> | null {
  const name = ev.name.trim()
  if (!name) return null

  const entry: Record<string, unknown> = { name }
  const description = ev.description.trim()
  if (description) entry.description = description

  if (ev.inputMode === 'conversation') {
    const messages = ev.messages
      .filter((m) => m.content.trim())
      .map((m) => ({ role: m.role, content: m.content.trim() }))
    if (messages.length > 0) entry.messages = messages
  } else {
    const input = ev.input.trim()
    if (input) entry.input = input
  }

  const expectType = ev.expectType.trim()
  if (expectType) {
    const expect: Record<string, unknown> = { type: expectType }
    const value = ev.expectValue.trim()
    const criteria = ev.expectCriteria.trim()
    if (value && (expectType === 'contains' || expectType === 'exact' || expectType === 'regex')) {
      expect.value = value
    }
    if (criteria && expectType === 'llm') {
      expect.criteria = criteria
    }
    entry.expect = expect
  }

  const tags = ev.tags
    .split(',')
    .map((t) => t.trim())
    .filter(Boolean)
  if (tags.length > 0) entry.tags = tags

  const timeout = ev.timeout.trim()
  if (timeout) entry.timeout = Number(timeout)

  const workingDir = ev.workingDir.trim()
  if (workingDir) entry.workingDir = workingDir

  if (ev.disabled) entry.disabled = true

  return entry
}

function mergeEvals(root: YAMLMap, form: AgentFormModel) {
  const evals = form.evals.map((e) => buildEvalEntry(e)).filter(Boolean)
  if (evals.length === 0) {
    root.delete('evals')
    return
  }
  root.set('evals', evals)
}

/** Best-effort merge of form fields into YAML, preserving unsupported sections. */
export function syncYamlFromForm(
  content: string,
  form: AgentFormModel,
  options: AgentFormOptions,
): string {
  return mergeFormIntoAgentYaml(content, form, options)
}

/** Apply form edits onto the original YAML document, preserving unsupported sections and formatting where possible. */
export function mergeFormIntoAgentYaml(
  originalContent: string,
  form: AgentFormModel,
  options: AgentFormOptions,
): string {
  const original = parse(originalContent) as Record<string, unknown> | null
  if (!original || typeof original !== 'object') {
    return formToAgentYaml(form, options)
  }

  const doc = parseDocument(originalContent)
  if (!doc.contents || !isMap(doc.contents)) {
    return formToAgentYaml(form, options)
  }

  const root = doc.contents as YAMLMap

  setMapKey(root, 'id', form.id.trim() || undefined)
  setMapKey(root, 'name', form.name.trim() || undefined)
  setMapKey(root, 'description', form.description.trim() || undefined)
  const tags = form.tags
    .split(',')
    .map((tag) => tag.trim())
    .filter(Boolean)
  if (tags.length > 0) root.set('tags', tags)
  else root.delete('tags')
  setMapKey(root, 'systemPrompt', form.systemPrompt.trim() || undefined)

  if (form.promptMode === 'auto') {
    root.set('promptMode', 'auto')
  } else {
    root.delete('promptMode')
  }
  setMapKey(root, 'defaultPrompt', form.defaultPrompt.trim() || undefined)

  if (form.workingDirInput) {
    root.set('workingDirInput', true)
  } else {
    root.delete('workingDirInput')
  }

  const env = entriesToMap(form.env)
  if (env) root.set('env', env)
  else root.delete('env')

  mergeHarness(root, form, options.harnesses)
  mergeAiAssets(root, form, options, original)
  mergeToolApprovals(root, form)
  mergeHitl(root, form)
  mergeEvals(root, form)
  mergeSubAgents(root, form)

  if (!root.has('version')) root.set('version', '0.1.0')

  return doc.toString()
}

function mergeSubAgents(root: YAMLMap, form: AgentFormModel) {
  const ids = form.subAgents.map((id) => id.trim()).filter(Boolean)
  if (ids.length === 0) {
    root.delete('subAgents')
    return
  }
  root.set('subAgents', ids)
}

/** Generate YAML for a brand-new agent (no original file). */
export function formToAgentYaml(form: AgentFormModel, options: AgentFormOptions): string {
  return mergeFormIntoAgentYaml(
    `adl: "1.0"\nid: ${form.id.trim() || 'my-agent'}\nname: ${form.name.trim() || 'My Agent'}\nharness:\n  type: claude-code\n`,
    form,
    options,
  )
}

export function slugFromName(name: string): string {
  return name
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '') || 'my-agent'
}

export function isConversationEval(ev: FormEval): boolean {
  return ev.inputMode === 'conversation'
}

export function usesSimpleGrader(ev: FormEval): boolean {
  const t = ev.expectType.trim()
  return t === '' || t === 'contains'
}

export function defaultFormEval(name = ''): FormEval {
  return {
    name,
    description: '',
    inputMode: 'single',
    input: '',
    messages: [{ role: 'user', content: '' }],
    expectType: 'contains',
    expectValue: '',
    expectCriteria: '',
    tags: '',
    timeout: '',
    workingDir: '',
    disabled: false,
  }
}
