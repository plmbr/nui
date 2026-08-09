// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

/** AG-UI CUSTOM open_session payload — jump the UI to a launched session. */
export interface OpenSessionEvent {
  sessionId: string
  prompt?: string
  agentType?: string
  toolCallId?: string
  sourceSessionId: string
}

export function parseOpenSessionCustomValue(
  value: Record<string, unknown> | undefined,
  sourceSessionId: string,
): OpenSessionEvent | null {
  if (!value) return null
  const sessionId = typeof value.sessionId === 'string' ? value.sessionId.trim() : ''
  if (!sessionId) return null
  const prompt = typeof value.prompt === 'string' ? value.prompt : undefined
  const agentType = typeof value.agentType === 'string' ? value.agentType : undefined
  const toolCallId = typeof value.toolCallId === 'string' ? value.toolCallId : undefined
  return {
    sessionId,
    prompt: prompt?.trim() ? prompt : undefined,
    agentType: agentType?.trim() ? agentType : undefined,
    toolCallId,
    sourceSessionId,
  }
}

export function isLaunchSessionToolName(toolName?: string): boolean {
  if (!toolName) return false
  const bare = toolName.split('__').pop()?.trim().toLowerCase() ?? ''
  return bare === 'launch_session'
}

/** Normalize Claude/MCP content-block wrappers to a JSON object string. */
export function unwrapToolResultJSON(content: string): string {
  const trimmed = content.trim()
  if (!trimmed) return ''
  if (trimmed.startsWith('{')) return trimmed

  try {
    const parsed: unknown = JSON.parse(trimmed)
    if (!Array.isArray(parsed)) return trimmed
    const parts: string[] = []
    for (const block of parsed) {
      if (!block || typeof block !== 'object') continue
      const text = (block as { text?: unknown }).text
      if (typeof text === 'string' && text.trim()) parts.push(text.trim())
    }
    if (parts.length === 0) return trimmed
    return parts.join('\n').trim()
  } catch {
    return trimmed
  }
}

/**
 * Parse a launch_session TOOL_CALL_RESULT payload into an OpenSessionEvent.
 * Handles both bare JSON and Claude Code content-block arrays.
 */
export function parseLaunchSessionToolResult(
  content: string | undefined,
  sourceSessionId: string,
  toolCallId?: string,
): OpenSessionEvent | null {
  if (!content?.trim()) return null
  const raw = unwrapToolResultJSON(content)
  if (!raw || raw.toLowerCase().startsWith('error:')) return null

  try {
    const parsed: unknown = JSON.parse(raw)
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) return null
    const obj = parsed as Record<string, unknown>
    const session = obj.session
    if (!session || typeof session !== 'object' || Array.isArray(session)) return null
    const sessionObj = session as Record<string, unknown>
    const sessionId = typeof sessionObj.id === 'string' ? sessionObj.id.trim() : ''
    if (!sessionId) return null
    const prompt = typeof obj.prompt === 'string' ? obj.prompt : undefined
    const agentType = typeof sessionObj.agentType === 'string' ? sessionObj.agentType : undefined
    return {
      sessionId,
      prompt: prompt?.trim() ? prompt : undefined,
      agentType: agentType?.trim() ? agentType : undefined,
      toolCallId,
      sourceSessionId,
    }
  } catch {
    return null
  }
}
