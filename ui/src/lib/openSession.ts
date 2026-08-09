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
