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

export interface AgentFormModel {
  id: string
  name: string
  description: string
  systemPrompt: string
  defaultPrompt: string
  harnessOptionId: string
  harnessModel: string
  dockerImage: string
  containerPort: string
  remoteHost: string
  remotePort: string
  env: KeyValue[]
  skills: FormSkill[]
  mcpServers: FormMCPServer[]
}

export interface AgentFormOptions {
  harnesses: HarnessOption[]
  skills: SkillOption[]
  mcpServers: MCPOption[]
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
}

const BUILTIN_HARNESSES: HarnessOption[] = [
  { id: 'builtin:claude-code', label: 'Claude Code', group: 'Built-in', harnessType: 'claude-code', sandbox: 'none' },
  { id: 'builtin:pi', label: 'Pi', group: 'Built-in', harnessType: 'pi', sandbox: 'none' },
  { id: 'builtin:codex', label: 'Codex', group: 'Built-in', harnessType: 'codex', sandbox: 'none' },
  { id: 'builtin:opencode', label: 'OpenCode', group: 'Built-in', harnessType: 'opencode', sandbox: 'none' },
  { id: 'builtin:docker', label: 'Docker (HTTP/SSE container)', group: 'Built-in', harnessType: 'docker' },
  { id: 'builtin:remote', label: 'Remote (HTTP/SSE server)', group: 'Built-in', harnessType: 'remote' },
]

export function defaultAgentForm(): AgentFormModel {
  return {
    id: 'my-agent',
    name: 'My Agent',
    description: '',
    systemPrompt: '',
    defaultPrompt: '',
    harnessOptionId: 'builtin:claude-code',
    harnessModel: '',
    dockerImage: '',
    containerPort: '',
    remoteHost: '127.0.0.1',
    remotePort: '9090',
    env: [],
    skills: [],
    mcpServers: [],
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
  if (type === 'docker' || type === 'remote') {
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

/** Read supported form fields from YAML without modifying the source text. */
export function parseAgentYaml(content: string, options: AgentFormOptions): ParsedAgentDoc {
  const doc = parse(content) as Record<string, unknown> | null
  if (!doc || typeof doc !== 'object') {
    return { form: defaultAgentForm(), hasWorkflowSteps: false }
  }

  const harness = doc.harness as Record<string, unknown> | undefined
  const { skills: skillsRaw, mcpServers: mcpRaw } = aiAssetsLists(doc)
  const steps = doc.steps as unknown[] | undefined
  const hasWorkflowSteps = Array.isArray(steps) && steps.length > 0

  const form: AgentFormModel = {
    id: String(doc.id ?? doc.name ?? 'my-agent'),
    name: String(doc.name ?? doc.id ?? 'My Agent'),
    description: String(doc.description ?? ''),
    systemPrompt: String(doc.systemPrompt ?? ''),
    defaultPrompt: String(doc.defaultPrompt ?? ''),
    harnessOptionId: harnessOptionIdFromDoc(harness, options.harnesses),
    harnessModel: String(harness?.model ?? ''),
    dockerImage: String(harness?.image ?? ''),
    containerPort: harness?.containerPort != null ? String(harness.containerPort) : '',
    remoteHost: String(harness?.host ?? '127.0.0.1'),
    remotePort: harness?.port != null ? String(harness.port) : '9090',
    env: mapToEntries(doc.env as Record<string, string> | undefined),
    skills: skillsRaw.map((s) => findSkillOption(s, options.skills)).filter(Boolean) as FormSkill[],
    mcpServers: mcpRaw.map((s) => findMCPOption(s, options.mcpServers)).filter(Boolean) as FormMCPServer[],
  }

  return { form, hasWorkflowSteps }
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
    for (const key of ['model', 'image', 'containerPort', 'host', 'port']) harness.delete(key)
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
  for (const key of ['image', 'containerPort', 'host', 'port']) harness.delete(key)
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
  if (opt?.ref) return { name, ref: opt.ref }
  if (opt?.server) {
    return Object.fromEntries(
      Object.entries({ ...opt.server, name }).filter(([, v]) => v != null && v !== ''),
    )
  }
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
  setMapKey(root, 'systemPrompt', form.systemPrompt.trim() || undefined)

  if (form.defaultPrompt.trim()) {
    root.set('promptMode', 'auto')
    root.set('defaultPrompt', form.defaultPrompt.trim())
  } else {
    root.delete('defaultPrompt')
    if (root.get('promptMode') === 'auto') root.delete('promptMode')
  }

  const env = entriesToMap(form.env)
  if (env) root.set('env', env)
  else root.delete('env')

  mergeHarness(root, form, options.harnesses)
  mergeAiAssets(root, form, options, original)

  if (!root.has('version')) root.set('version', '0.1.0')

  return doc.toString()
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
