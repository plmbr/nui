// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

export const LAUNCH_PATH = '/launch'
export const SCHEDULES_PATH = '/schedules'
export const CUSTOMIZE_PATH = '/customize'
export const NEW_SESSION_PATH = '/sessions/new'
export const CREATE_SESSION_PATH = '/sessions/create'

const SESSION_PATH_RE = /^\/sessions\/([^/]+)\/?$/
const SESSION_ID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i
const SESSION_LIST_RESERVED_SEGMENTS = new Set(['new', 'create'])

function sessionPathSegment(pathname = window.location.pathname): string | null {
  const match = pathname.match(SESSION_PATH_RE)
  if (!match) return null
  return decodeURIComponent(match[1])
}

function isSessionIdSegment(segment: string): boolean {
  return SESSION_ID_RE.test(segment)
}

export function sessionPath(id: string): string {
  return `/sessions/${id}`
}

export function sessionListPath(agentGroupId: string): string {
  return `/sessions/${encodeURIComponent(agentGroupId)}`
}

export function sessionIdFromPath(pathname = window.location.pathname): string | null {
  const segment = sessionPathSegment(pathname)
  if (!segment || SESSION_LIST_RESERVED_SEGMENTS.has(segment) || !isSessionIdSegment(segment)) {
    return null
  }
  return segment
}

export function agentGroupIdFromPath(pathname = window.location.pathname): string | null {
  const segment = sessionPathSegment(pathname)
  if (!segment || SESSION_LIST_RESERVED_SEGMENTS.has(segment) || isSessionIdSegment(segment)) {
    return null
  }
  return segment
}

export function isLaunchPath(pathname = window.location.pathname): boolean {
  return pathname === LAUNCH_PATH || pathname === `${LAUNCH_PATH}/`
}

export function isCustomizePath(pathname = window.location.pathname): boolean {
  return pathname === CUSTOMIZE_PATH || pathname === `${CUSTOMIZE_PATH}/`
}

export function isSchedulesPath(pathname = window.location.pathname): boolean {
  return pathname === SCHEDULES_PATH || pathname === `${SCHEDULES_PATH}/`
}

export function isNewSessionPath(pathname = window.location.pathname): boolean {
  return pathname === NEW_SESSION_PATH || pathname === NEW_SESSION_PATH + '/'
}

export function isCreateSessionPath(pathname = window.location.pathname): boolean {
  return pathname === CREATE_SESSION_PATH || pathname === CREATE_SESSION_PATH + '/'
}

export function agentFromNewSessionSearch(search = window.location.search): string | null {
  const agent = new URLSearchParams(search).get('agent')?.trim()
  return agent || null
}

export function cwdFromNewSessionSearch(search = window.location.search): string | null {
  const cwd = new URLSearchParams(search).get('cwd')?.trim()
  return cwd || null
}

function setPath(path: string, replace: boolean): void {
  if (window.location.pathname + window.location.search === path) return
  if (replace) {
    window.history.replaceState(null, '', path)
  } else {
    window.history.pushState(null, '', path)
  }
}

export function navigateToSession(id: string, replace = false): void {
  setPath(sessionPath(id), replace)
}

export function navigateToSessionList(agentGroupId: string, replace = false): void {
  setPath(sessionListPath(agentGroupId), replace)
}

export function navigateToCustomize(replace = false): void {
  setPath(CUSTOMIZE_PATH, replace)
}

export function navigateToSchedules(replace = false): void {
  setPath(SCHEDULES_PATH, replace)
}

export function navigateToNewSession(opts?: { agent?: string; cwd?: string }, replace = false): void {
  const params = new URLSearchParams()
  if (opts?.agent?.trim()) params.set('agent', opts.agent.trim())
  if (opts?.cwd?.trim()) params.set('cwd', opts.cwd.trim())
  const query = params.toString()
  setPath(`${NEW_SESSION_PATH}${query ? `?${query}` : ''}`, replace)
}

export function navigateToLaunch(replace = false): void {
  setPath(LAUNCH_PATH, replace)
}

export function navigateToHome(replace = false): void {
  navigateToLaunch(replace)
}
