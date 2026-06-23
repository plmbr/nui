// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { parse, stringify } from 'yaml'
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
  preserved: Record<string, unknown>
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
  return { optionId: match?.id ?? `custom:inline:${name}`, name }
}

export function parseAgentYaml(
  content: string,
  options: AgentFormOptions,
): ParsedAgentDoc {
  const doc = parse(content) as Record<string, unknown> | null
  if (!doc || typeof doc !== 'object') {
    return { form: defaultAgentForm(), preserved: {}, hasWorkflowSteps: false }
  }

  const harness = doc.harness as Record<string, unknown> | undefined
  const aiAssets = doc.aiAssets as Record<string, unknown> | undefined
  const skillsRaw = (aiAssets?.skills as Record<string, unknown>[] | undefined) ?? []
  const mcpRaw = (aiAssets?.mcpServers as Record<string, unknown>[] | undefined) ?? []
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

  const preserved: Record<string, unknown> = {}
  if (doc.kind) preserved.kind = doc.kind
  if (doc.version) preserved.version = doc.version
  if (steps) preserved.steps = steps
  if (doc.constraints) preserved.constraints = doc.constraints
  if (doc.schedule) preserved.schedule = doc.schedule
  if (doc.skill) preserved.skill = doc.skill
  if (harness) {
    const extras: Record<string, unknown> = {}
    if (harness.sandbox != null) extras.sandbox = harness.sandbox
    if (harness.env != null) extras.env = harness.env
    if (Object.keys(extras).length > 0) preserved.harnessExtras = extras
  }
  if (doc.promptMode === 'auto' && !form.defaultPrompt.trim()) {
    preserved.promptMode = 'auto'
  }

  return { form, preserved, hasWorkflowSteps }
}

function resolveHarnessOption(optionId: string, options: HarnessOption[]): HarnessOption | undefined {
  return options.find((o) => o.id === optionId)
}

function buildHarnessBlock(form: AgentFormModel, options: HarnessOption[]): Record<string, unknown> {
  const selected = resolveHarnessOption(form.harnessOptionId, options)
  const harnessType = selected?.harnessType ?? 'claude-code'
  const block: Record<string, unknown> = { type: harnessType }

  if (harnessType.startsWith('ext:')) {
    return block
  }

  if (harnessType === 'docker') {
    if (form.dockerImage.trim()) block.image = form.dockerImage.trim()
    if (form.containerPort.trim()) block.containerPort = Number(form.containerPort)
    return block
  }

  if (harnessType === 'remote') {
    if (form.remoteHost.trim()) block.host = form.remoteHost.trim()
    if (form.remotePort.trim()) block.port = Number(form.remotePort)
    return block
  }

  if (form.harnessModel.trim()) block.model = form.harnessModel.trim()
  return block
}

function buildSkillsBlock(form: AgentFormModel, options: SkillOption[]): Record<string, unknown>[] {
  return form.skills.map((s) => {
    const opt = options.find((o) => o.id === s.optionId)
    const name = s.name.trim() || opt?.name || 'skill'
    if (opt?.ref) {
      return { name, ref: opt.ref }
    }
    if (s.optionId.startsWith('custom:ref:')) {
      return { name, ref: s.optionId.slice('custom:ref:'.length) }
    }
    if (s.optionId.startsWith('custom:path:')) {
      return { name, path: s.optionId.slice('custom:path:'.length) }
    }
    return { name, ref: name }
  })
}

function buildMCPServersBlock(form: AgentFormModel, options: MCPOption[]): Record<string, unknown>[] {
  return form.mcpServers.map((s) => {
    const opt = options.find((o) => o.id === s.optionId)
    const name = s.name.trim() || opt?.name || 'mcp-server'
    if (opt?.ref) {
      return { name, ref: opt.ref }
    }
    if (opt?.server) {
      const server = { ...opt.server, name }
      return Object.fromEntries(Object.entries(server).filter(([, v]) => v != null && v !== ''))
    }
    if (s.optionId.startsWith('custom:ref:')) {
      return { name, ref: s.optionId.slice('custom:ref:'.length) }
    }
    return { name }
  })
}

export function formToAgentYaml(
  form: AgentFormModel,
  options: AgentFormOptions,
  preserved: Record<string, unknown> = {},
): string {
  const doc: Record<string, unknown> = {
    adl: '1.0',
    id: form.id.trim() || 'my-agent',
    name: form.name.trim() || form.id.trim() || 'My Agent',
    harness: buildHarnessBlock(form, options.harnesses),
  }

  const harnessExtras = preserved.harnessExtras as Record<string, unknown> | undefined
  if (harnessExtras) {
    doc.harness = { ...(doc.harness as Record<string, unknown>), ...harnessExtras }
  }

  if (form.description.trim()) doc.description = form.description.trim()
  doc.version = typeof preserved.version === 'string' && preserved.version.trim()
    ? preserved.version.trim()
    : '0.1.0'
  if (form.systemPrompt.trim()) doc.systemPrompt = form.systemPrompt.trim()
  if (form.defaultPrompt.trim()) {
    doc.promptMode = 'auto'
    doc.defaultPrompt = form.defaultPrompt.trim()
  } else if (preserved.promptMode === 'auto') {
    doc.promptMode = 'auto'
  }

  const env = entriesToMap(form.env)
  if (env) doc.env = env

  const skills = buildSkillsBlock(form, options.skills)
  const mcpServers = buildMCPServersBlock(form, options.mcpServers)
  if (skills.length > 0 || mcpServers.length > 0) {
    doc.aiAssets = {
      ...(skills.length > 0 ? { skills } : {}),
      ...(mcpServers.length > 0 ? { mcpServers } : {}),
    }
  }

  for (const [key, value] of Object.entries(preserved)) {
    if (key === 'harnessExtras' || key === 'promptMode' || key === 'version') continue
    if (value !== undefined) doc[key] = value
  }

  return stringify(doc, { lineWidth: 0 })
}

export function slugFromName(name: string): string {
  return name
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '') || 'my-agent'
}
