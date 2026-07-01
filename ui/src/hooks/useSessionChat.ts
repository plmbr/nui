// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { HttpAgent } from '@ag-ui/client'
import { EventType } from '@ag-ui/core'
import { useCallback, useEffect, useRef, useState } from 'react'
import { v4 as uuidv4 } from 'uuid'
import { api } from '@/api'
import {
  registerSessionRun,
  unregisterSessionRun,
} from '@/lib/sessionRunRegistry'

export interface ToolCallPart {
  type: 'tool'
  id: string
  toolCallId?: string
  toolName?: string
  toolArgs?: Record<string, unknown>
  toolResult?: unknown
  mcpAppResourceUri?: string
  mcpAppServerName?: string
  mcpAppToolInput?: Record<string, unknown>
}

export interface TextPart {
  type: 'text'
  content: string
}

export type AssistantPart = TextPart | ToolCallPart

export interface SessionChatMessage {
  id: string
  role: 'user' | 'assistant'
  content: string
  parts?: AssistantPart[]
  error?: boolean
  images?: ChatImage[]
  /** @deprecated tool fields on the message root; use parts instead */
  toolCallId?: string
  toolName?: string
  toolArgs?: Record<string, unknown>
  toolResult?: unknown
  mcpAppResourceUri?: string
  mcpAppServerName?: string
  mcpAppToolInput?: Record<string, unknown>
}

function assistantTextContent(msg: SessionChatMessage): string {
  if (msg.parts?.length) {
    return msg.parts
      .filter((part): part is TextPart => part.type === 'text')
      .map((part) => part.content)
      .join('')
  }
  return msg.content
}

function appendTextPart(parts: AssistantPart[], delta: string): AssistantPart[] {
  const next = [...parts]
  const last = next[next.length - 1]
  if (last?.type === 'text') {
    next[next.length - 1] = { ...last, content: last.content + delta }
  } else {
    next.push({ type: 'text', content: delta })
  }
  return next
}

function updateToolPart(
  parts: AssistantPart[],
  partId: string,
  update: Partial<ToolCallPart>,
): AssistantPart[] {
  return parts.map((part) =>
    part.type === 'tool' && part.id === partId ? { ...part, ...update } : part,
  )
}

function applyAssistantError(msg: SessionChatMessage, text: string): SessionChatMessage {
  const content = text || assistantTextContent(msg) || 'An error occurred.'
  if (!msg.parts?.length) {
    return { ...msg, content, error: true }
  }
  if (assistantTextContent(msg).trim()) {
    return { ...msg, content: assistantTextContent(msg), error: true }
  }
  const parts = appendTextPart(msg.parts, content)
  return { ...msg, parts, content, error: true }
}

export interface ChatImage {
  id: string
  mediaType: string
  data: string
}

function imageSrc(image: ChatImage): string {
  if (image.data.startsWith('http://') || image.data.startsWith('https://')) {
    return image.data
  }
  return `data:${image.mediaType};base64,${image.data}`
}

export { imageSrc }

function historyToMessages(
  history: { id: string; role: string; content: string }[],
): SessionChatMessage[] {
  return history
    .filter((m) => m.role === 'user' || m.role === 'assistant')
    .map((m) => ({
      id: m.id,
      role: m.role as 'user' | 'assistant',
      content: m.content,
    }))
}

export function useSessionChat(sessionId: string) {
  const [messages, setMessages] = useState<SessionChatMessage[]>([])
  const [isRunning, setIsRunning] = useState(false)
  const [isLoading, setIsLoading] = useState(true)
  const agentRef = useRef(
    new HttpAgent({
      url: `/api/sessions/${sessionId}/ag-ui`,
      threadId: sessionId,
    }),
  )
  const pendingToolsRef = useRef<Record<string, string>>({})
  const subscriptionRef = useRef<{ unsubscribe: () => void } | null>(null)
  const assistantMsgIdRef = useRef<string | null>(null)

  const finishRun = useCallback(() => {
    subscriptionRef.current?.unsubscribe()
    subscriptionRef.current = null
    assistantMsgIdRef.current = null
    unregisterSessionRun(sessionId)
    setIsRunning(false)
  }, [sessionId])

  const stopRun = useCallback(async () => {
    if (!subscriptionRef.current && !assistantMsgIdRef.current) return

    subscriptionRef.current?.unsubscribe()
    subscriptionRef.current = null

    const cancelledAssistantId = assistantMsgIdRef.current
    assistantMsgIdRef.current = null
    unregisterSessionRun(sessionId)
    setIsRunning(false)

    try {
      await api.sessions.stop(sessionId)
    } catch {
      /* best-effort */
    }

    if (cancelledAssistantId) {
      setMessages((prev) =>
        prev.map((m) => {
          if (m.id !== cancelledAssistantId) return m
          if (assistantTextContent(m).trim()) {
            return { ...m, error: true }
          }
          return applyAssistantError(m, 'Stopped.')
        }),
      )
    }
  }, [sessionId])

  useEffect(() => {
    return () => {
      subscriptionRef.current?.unsubscribe()
      unregisterSessionRun(sessionId)
    }
  }, [sessionId])

  useEffect(() => {
    agentRef.current = new HttpAgent({
      url: `/api/sessions/${sessionId}/ag-ui`,
      threadId: sessionId,
    })
    pendingToolsRef.current = {}
    setIsRunning(false)
    setIsLoading(true)
    api.messages
      .list(sessionId)
      .then((stored) => {
        if (stored.length > 0) {
          setMessages(historyToMessages(stored))
          return
        }
        return api.history.get(sessionId).then((history) => setMessages(historyToMessages(history)))
      })
      .catch(() => setMessages([]))
      .finally(() => setIsLoading(false))
  }, [sessionId])

  const sendMessage = useCallback(
    (text: string) => {
      if (isRunning || !text.trim()) return

      const userMsg: SessionChatMessage = { id: uuidv4(), role: 'user', content: text }
      const assistantMsgId = uuidv4()
      const assistantMsg: SessionChatMessage = {
        id: assistantMsgId,
        role: 'assistant',
        content: '',
        parts: [],
      }

      setMessages((prev) => [...prev, userMsg, assistantMsg])
      setIsRunning(true)
      assistantMsgIdRef.current = assistantMsgId
      pendingToolsRef.current = {}
      registerSessionRun(sessionId, stopRun)

      const history = [
        ...messages.map((m) => ({
          role: m.role as 'user' | 'assistant',
          content: m.role === 'assistant' ? assistantTextContent(m) : m.content,
          id: m.id,
        })),
        { role: 'user' as const, content: text, id: userMsg.id },
      ]

      const obs = agentRef.current.run({
        threadId: sessionId,
        runId: uuidv4(),
        messages: history,
        tools: [],
        state: {},
        context: [],
        forwardedProps: {},
      })

      const sub = obs.subscribe({
        next(event) {
          if (event.type === EventType.TEXT_MESSAGE_CHUNK) {
            const chunk = event as { delta?: string }
            if (chunk.delta) {
              setMessages((prev) =>
                prev.map((m) => {
                  if (m.id !== assistantMsgId) return m
                  const parts = appendTextPart(m.parts ?? [], chunk.delta!)
                  return { ...m, parts, content: assistantTextContent({ ...m, parts }) }
                }),
              )
            }
          } else if (event.type === EventType.TOOL_CALL_START) {
            const e = event as { toolCallId?: string; toolCallName?: string }
            const partId = uuidv4()
            if (e.toolCallId) {
              pendingToolsRef.current[e.toolCallId] = partId
            }
            const toolPart: ToolCallPart = {
              type: 'tool',
              id: partId,
              toolCallId: e.toolCallId,
              toolName: e.toolCallName,
              toolArgs: {},
            }
            setMessages((prev) =>
              prev.map((m) =>
                m.id === assistantMsgId
                  ? { ...m, parts: [...(m.parts ?? []), toolPart] }
                  : m,
              ),
            )
          } else if (event.type === EventType.TOOL_CALL_ARGS) {
            const e = event as { toolCallId?: string; delta?: string }
            if (!e.toolCallId) return
            const partId = pendingToolsRef.current[e.toolCallId]
            if (!partId || !e.delta) return
            try {
              const args = JSON.parse(e.delta) as Record<string, unknown>
              setMessages((prev) =>
                prev.map((m) => {
                  if (m.id !== assistantMsgId || !m.parts) return m
                  const parts = m.parts.map((part) =>
                    part.type === 'tool' && part.id === partId
                      ? { ...part, toolArgs: { ...part.toolArgs, ...args } }
                      : part,
                  )
                  return { ...m, parts }
                }),
              )
            } catch {
              /* ignore partial JSON */
            }
          } else if (event.type === EventType.TOOL_CALL_RESULT) {
            const e = event as { toolCallId?: string; content?: string }
            if (!e.toolCallId) return
            const partId = pendingToolsRef.current[e.toolCallId]
            if (!partId) return
            let result: unknown = e.content
            try {
              result = JSON.parse(e.content ?? '')
            } catch {
              /* keep as string */
            }
            setMessages((prev) =>
              prev.map((m) => {
                if (m.id !== assistantMsgId || !m.parts) return m
                return { ...m, parts: updateToolPart(m.parts, partId, { toolResult: result }) }
              }),
            )
          } else if (event.type === EventType.CUSTOM) {
            const e = event as { name?: string; value?: Record<string, unknown> }
            if (!e.value) return
            if (e.name === 'image') {
              const { mediaType, data } = e.value as { mediaType?: string; data?: string }
              if (!data) return
              setMessages((prev) =>
                prev.map((m) =>
                  m.id === assistantMsgId
                    ? {
                        ...m,
                        images: [
                          ...(m.images ?? []),
                          {
                            id: uuidv4(),
                            mediaType: mediaType ?? 'image/png',
                            data,
                          },
                        ],
                      }
                    : m,
                ),
              )
              return
            }
            if (e.name !== 'mcp_app') return
            const { toolCallId, serverName, resourceUri, toolInput } = e.value as {
              toolCallId?: string
              serverName?: string
              resourceUri?: string
              toolInput?: Record<string, unknown>
            }
            if (!toolCallId || !serverName || !resourceUri) return
            const partId = pendingToolsRef.current[toolCallId]
            if (!partId) return
            setMessages((prev) =>
              prev.map((m) => {
                if (m.id !== assistantMsgId || !m.parts) return m
                return {
                  ...m,
                  parts: updateToolPart(m.parts, partId, {
                    mcpAppResourceUri: resourceUri,
                    mcpAppServerName: serverName,
                    mcpAppToolInput: toolInput,
                  }),
                }
              }),
            )
          } else if (event.type === EventType.RUN_FINISHED) {
            finishRun()
          } else if (event.type === EventType.RUN_ERROR) {
            const errEvent = event as { message?: string }
            const errText = errEvent.message?.trim()
            const isCancelled = errText?.toLowerCase() === 'cancelled'
            setMessages((prev) =>
              prev.map((m) => {
                if (m.id !== assistantMsgId) return m
                if (isCancelled) {
                  if (assistantTextContent(m).trim()) {
                    return { ...m, error: true }
                  }
                  return applyAssistantError(m, 'Stopped.')
                }
                return applyAssistantError(m, errText ?? '')
              }),
            )
            finishRun()
          }
        },
        error(err) {
          setMessages((prev) =>
            prev.map((m) =>
              m.id === assistantMsgId
                ? applyAssistantError(
                    m,
                    `Connection error: ${err instanceof Error ? err.message : String(err)}`,
                  )
                : m,
            ),
          )
          finishRun()
        },
      })
      subscriptionRef.current = sub
    },
    [messages, isRunning, sessionId, stopRun, finishRun],
  )

  return { messages, sendMessage, stopRun, isRunning, isLoading }
}
