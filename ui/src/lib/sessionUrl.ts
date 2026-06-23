// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

export function sessionPath(id: string): string {
  return `/sessions/${id}`
}

export function sessionIdFromPath(pathname = window.location.pathname): string | null {
  const match = pathname.match(/^\/sessions\/([^/]+)\/?$/)
  return match?.[1] ?? null
}

export function navigateToSession(id: string, replace = false): void {
  const path = sessionPath(id)
  if (window.location.pathname === path) return
  if (replace) {
    window.history.replaceState(null, '', path)
  } else {
    window.history.pushState(null, '', path)
  }
}
