// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { unwrapToolResultJSON } from '@/lib/openSession'

export type UIAction =
  | { type: 'navigate'; target: 'customize' | 'new_session' | 'launch' | 'schedules' }
  | { type: 'set_theme'; theme: 'dark' | 'light' }
  | { type: 'refresh_ui' }

export interface UIActionEvent {
  actions: UIAction[]
  toolCallId?: string
  sourceSessionId?: string
}

export interface UIActionHandlers {
  navigateCustomize: () => void
  navigateNewSession: () => void
  navigateLaunch: () => void
  navigateSchedules: () => void
  setTheme: (theme: 'dark' | 'light') => void
  refreshUI?: () => void
}

export function isControlUIToolName(toolName?: string): boolean {
  if (!toolName) return false
  const bare = toolName.split('__').pop()?.trim().toLowerCase() ?? ''
  return bare === 'control_ui' || bare === 'set_extension_enabled'
}

export function parseUIActions(raw: unknown): UIAction[] {
  if (!Array.isArray(raw)) return []
  const out: UIAction[] = []
  for (const item of raw) {
    if (!item || typeof item !== 'object' || Array.isArray(item)) continue
    const obj = item as Record<string, unknown>
    const type = typeof obj.type === 'string' ? obj.type.trim() : ''
    if (type === 'navigate') {
      const target = typeof obj.target === 'string' ? obj.target.trim() : ''
      if (
        target === 'customize' ||
        target === 'new_session' ||
        target === 'launch' ||
        target === 'schedules'
      ) {
        out.push({ type: 'navigate', target })
      }
      continue
    }
    if (type === 'set_theme') {
      const theme = typeof obj.theme === 'string' ? obj.theme.trim() : ''
      if (theme === 'dark' || theme === 'light') {
        out.push({ type: 'set_theme', theme })
      }
      continue
    }
    if (type === 'refresh_ui') {
      out.push({ type: 'refresh_ui' })
    }
  }
  return out
}

export function parseUIActionCustomValue(
  value: Record<string, unknown> | undefined,
  sourceSessionId?: string,
): UIActionEvent | null {
  if (!value) return null
  const actions = parseUIActions(value.actions)
  if (actions.length === 0) return null
  const toolCallId = typeof value.toolCallId === 'string' ? value.toolCallId : undefined
  return { actions, toolCallId, sourceSessionId }
}

export function parseControlUIToolResult(
  content: string | undefined,
  sourceSessionId?: string,
  toolCallId?: string,
): UIActionEvent | null {
  if (!content?.trim()) return null
  const raw = unwrapToolResultJSON(content)
  if (!raw || raw.toLowerCase().startsWith('error:')) return null
  try {
    const parsed: unknown = JSON.parse(raw)
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) return null
    const obj = parsed as Record<string, unknown>
    const actions = parseUIActions(obj.actions)
    if (actions.length === 0) return null
    return { actions, toolCallId, sourceSessionId }
  } catch {
    return null
  }
}

export function applyUIActions(actions: UIAction[], handlers: UIActionHandlers): void {
  for (const action of actions) {
    switch (action.type) {
      case 'navigate':
        switch (action.target) {
          case 'customize':
            handlers.navigateCustomize()
            break
          case 'new_session':
            handlers.navigateNewSession()
            break
          case 'launch':
            handlers.navigateLaunch()
            break
          case 'schedules':
            handlers.navigateSchedules()
            break
        }
        break
      case 'set_theme':
        handlers.setTheme(action.theme)
        break
      case 'refresh_ui':
        handlers.refreshUI?.()
        break
    }
  }
}
