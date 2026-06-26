// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

export const CUSTOMIZE_PATH = '/customize'
export const NEW_SESSION_PATH = '/sessions/new'

export function sessionPath(id: string): string {
  return `/sessions/${id}`
}

export function sessionIdFromPath(pathname = window.location.pathname): string | null {
  const match = pathname.match(/^\/sessions\/([^/]+)\/?$/)
  if (!match || match[1] === 'new') return null
  return match[1]
}

export function isCustomizePath(pathname = window.location.pathname): boolean {
  return pathname === CUSTOMIZE_PATH || pathname === `${CUSTOMIZE_PATH}/`
}

export function isNewSessionPath(pathname = window.location.pathname): boolean {
  return pathname === NEW_SESSION_PATH || pathname === NEW_SESSION_PATH + '/'
}

export function agentFromNewSessionSearch(search = window.location.search): string | null {
  const agent = new URLSearchParams(search).get('agent')?.trim()
  return agent || null
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

export function navigateToNewSession(agent?: string, replace = false): void {
  const query = agent?.trim() ? `?agent=${encodeURIComponent(agent.trim())}` : ''
  setPath(`${NEW_SESSION_PATH}${query}`, replace)
}

export function navigateToHome(replace = false): void {
  setPath('/', replace)
}
