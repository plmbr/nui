// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { harnessLabel } from '@/lib/agentDisplay'
import { isNuiAgent, showToolApprovalsOption } from '@/lib/agentTypes'
import { sessionDisplayName } from '@/lib/sessionDisplay'
import type {
  AgentType,
  CreateSessionRequest,
  RecentAgentEntry,
  Session,
} from '@/types'

export const RECENTS_PREVIEW_LIMIT = 8
export const RECENTS_STORE_LIMIT = 20

export interface ResolvedRecentSession {
  id: string
  session: Session
  label: string
  secondary: string
  agentType: AgentType | undefined
}

export interface ResolvedRecentAgent {
  entry: RecentAgentEntry
  agentType: AgentType
  label: string
  secondary: string
}

export interface RecentAgentFormState {
  selectedId: string
  workingDir: string
  harnessOverride: string
  userScopeHarnessConfig: boolean
  harnessPermissionsEnabled: boolean
}

export function touchRecentSessionIds(current: string[] | undefined, sessionId: string): string[] {
  const id = sessionId.trim()
  if (!id) return current ?? []
  const filtered = (current ?? []).filter((item) => item !== id)
  const next = [id, ...filtered]
  return next.slice(0, RECENTS_STORE_LIMIT)
}

export function resolveRecentSessions(
  ids: string[] | undefined,
  sessions: Session[],
  agentTypes: AgentType[],
): ResolvedRecentSession[] {
  const byId = new Map((sessions ?? []).map((session) => [session.id, session]))
  const agentsById = new Map(agentTypes.map((agent) => [agent.id, agent]))
  const resolved: ResolvedRecentSession[] = []
  for (const id of ids ?? []) {
    const session = byId.get(id)
    if (!session) continue
    const agentType = agentsById.get(session.agentType)
    resolved.push({
      id,
      session,
      label: sessionDisplayName(session),
      secondary: shortenPath(session.workingDir) || (agentType ? harnessLabel(agentType.harness) : session.agentType),
      agentType,
    })
  }
  return resolved
}

export function resolveRecentAgents(
  entries: RecentAgentEntry[] | undefined,
  agentTypes: AgentType[],
): ResolvedRecentAgent[] {
  const byId = new Map(agentTypes.map((agent) => [agent.id, agent]))
  const resolved: ResolvedRecentAgent[] = []
  for (const entry of entries ?? []) {
    const agentType = byId.get(entry.agentType)
    if (!agentType || !agentType.available || isNuiAgent(agentType)) continue
    resolved.push({
      entry,
      agentType,
      label: agentType.label,
      secondary: entry.workingDir
        ? shortenPath(entry.workingDir)
        : harnessLabel(agentType.harness),
    })
  }
  return resolved
}

export function buildCreateRequestFromRecent(entry: RecentAgentEntry): CreateSessionRequest {
  const req: CreateSessionRequest = {
    agentType: entry.agentType,
  }
  if (entry.workingDir?.trim()) {
    req.workingDir = entry.workingDir.trim()
  }
  if (entry.agentConfig && Object.keys(entry.agentConfig).length > 0) {
    req.agentConfig = { ...entry.agentConfig }
  }
  return req
}

export function applyRecentAgentToForm(
  entry: RecentAgentEntry,
  agentTypes: AgentType[],
): RecentAgentFormState | null {
  const agent = agentTypes.find((item) => item.id === entry.agentType)
  if (!agent || !agent.available) return null

  const config = entry.agentConfig ?? {}
  const allowed = agent.allowedHarnesses ?? []
  const configuredHarness = typeof config.harnessType === 'string' ? config.harnessType : ''
  const harnessOverride = allowed.length > 1
    ? (configuredHarness || agent.harness)
    : ''

  const effectiveHarness = (harnessOverride || agent.harness) as AgentType['harness']
  const effectiveAgent: AgentType = { ...agent, harness: effectiveHarness }
  const showPermissions = showToolApprovalsOption(effectiveAgent)

  let harnessPermissionsEnabled = true
  if (config.harnessPermissions === 'bypass' || config.hitlMode === 'off') {
    harnessPermissionsEnabled = false
  } else if (config.harnessPermissions === 'interactive' || config.hitlMode === 'interactive') {
    harnessPermissionsEnabled = true
  }

  return {
    selectedId: entry.agentType,
    workingDir: entry.workingDir ?? '',
    harnessOverride,
    userScopeHarnessConfig: Boolean(config.userScopeHarnessConfig),
    harnessPermissionsEnabled: showPermissions ? harnessPermissionsEnabled : true,
  }
}

function shortenPath(path: string): string {
  const trimmed = path.trim()
  if (!trimmed) return ''
  const home = trimmed.replace(/^\/Users\/[^/]+/, '~')
  if (home.length <= 48) return home
  const parts = home.split('/')
  if (parts.length <= 3) return home
  return `…/${parts.slice(-2).join('/')}`
}

export function removeRecentSessionId(ids: string[] | undefined, sessionId: string): string[] {
  return (ids ?? []).filter((id) => id !== sessionId)
}

export function removeRecentAgent(entries: RecentAgentEntry[] | undefined, agentType: string): RecentAgentEntry[] {
  return (entries ?? []).filter((entry) => entry.agentType !== agentType)
}
