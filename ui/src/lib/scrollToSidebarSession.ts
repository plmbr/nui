// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

export function scrollToSidebarSession(sessionId: string): boolean {
  const el = document.querySelector(`[data-sidebar-session-id="${sessionId}"]`)
  if (!el) return false
  el.scrollIntoView({ block: 'nearest', behavior: 'smooth' })
  return true
}
