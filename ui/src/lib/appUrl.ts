// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

export const LAUNCH_PATH = '/launch'
export const SCHEDULES_PATH = '/schedules'
export const CUSTOMIZE_PATH = '/customize'
export const NEW_SESSION_PATH = '/sessions/new'
export const CREATE_SESSION_PATH = '/sessions/create'

export function sessionPath(id: string): string {
  return `/sessions/${id}`
}

export function sessionIdFromPath(pathname = window.location.pathname): string | null {
  const match = pathname.match(/^\/sessions\/([^/]+)\/?$/)
  if (!match || match[1] === 'new' || match[1] === 'create') return null
  return match[1]
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
