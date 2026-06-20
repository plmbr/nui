// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { HttpAgent } from '@ag-ui/client'
import { EventType } from '@ag-ui/core'
import { useCallback, useEffect, useRef, useState } from 'react'
import { v4 as uuidv4 } from 'uuid'
import { api } from '@/api'

export interface SessionChatMessage {
  id: string
  role: 'user' | 'assistant' | 'tool'
  content: string
  error?: boolean
  images?: ChatImage[]
  toolCallId?: string
  toolName?: string
  toolArgs?: Record<string, unknown>
  toolResult?: unknown
  mcpAppResourceUri?: string
  mcpAppServerName?: string
  mcpAppToolInput?: Record<string, unknown>
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
  const agentRef = useRef(
    new HttpAgent({
      url: `/api/sessions/${sessionId}/ag-ui`,
      threadId: sessionId,
    }),
  )
  const pendingToolsRef = useRef<Record<string, string>>({})

  useEffect(() => {
    agentRef.current = new HttpAgent({
      url: `/api/sessions/${sessionId}/ag-ui`,
      threadId: sessionId,
    })
    pendingToolsRef.current = {}
    setIsRunning(false)
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
      }

      setMessages((prev) => [...prev, userMsg, assistantMsg])
      setIsRunning(true)
      pendingToolsRef.current = {}

      const history = [
        ...messages.map((m) => ({
          role: m.role === 'tool' ? ('assistant' as const) : (m.role as 'user' | 'assistant'),
          content: m.content,
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

      obs.subscribe({
        next(event) {
          if (event.type === EventType.TEXT_MESSAGE_CHUNK) {
            const chunk = event as { delta?: string }
            if (chunk.delta) {
              setMessages((prev) =>
                prev.map((m) =>
                  m.id === assistantMsgId ? { ...m, content: m.content + chunk.delta } : m,
                ),
              )
            }
          } else if (event.type === EventType.TOOL_CALL_START) {
            const e = event as { toolCallId?: string; toolCallName?: string }
            const toolMsgId = uuidv4()
            if (e.toolCallId) {
              pendingToolsRef.current[e.toolCallId] = toolMsgId
            }
            const toolMsg: SessionChatMessage = {
              id: toolMsgId,
              role: 'tool',
              content: '',
              toolCallId: e.toolCallId,
              toolName: e.toolCallName,
              toolArgs: {},
            }
            setMessages((prev) => {
              const idx = prev.findIndex((m) => m.id === assistantMsgId)
              if (idx === -1) return [...prev, toolMsg]
              return [...prev.slice(0, idx), toolMsg, ...prev.slice(idx)]
            })
          } else if (event.type === EventType.TOOL_CALL_ARGS) {
            const e = event as { toolCallId?: string; delta?: string }
            if (!e.toolCallId) return
            const msgId = pendingToolsRef.current[e.toolCallId]
            if (!msgId || !e.delta) return
            try {
              const args = JSON.parse(e.delta) as Record<string, unknown>
              setMessages((prev) =>
                prev.map((m) =>
                  m.id === msgId ? { ...m, toolArgs: { ...m.toolArgs, ...args } } : m,
                ),
              )
            } catch {
              /* ignore partial JSON */
            }
          } else if (event.type === EventType.TOOL_CALL_RESULT) {
            const e = event as { toolCallId?: string; content?: string }
            if (!e.toolCallId) return
            const msgId = pendingToolsRef.current[e.toolCallId]
            if (!msgId) return
            let result: unknown = e.content
            try {
              result = JSON.parse(e.content ?? '')
            } catch {
              /* keep as string */
            }
            setMessages((prev) =>
              prev.map((m) => (m.id === msgId ? { ...m, toolResult: result } : m)),
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
            const msgId = pendingToolsRef.current[toolCallId]
            if (!msgId) return
            setMessages((prev) =>
              prev.map((m) =>
                m.id === msgId
                  ? {
                      ...m,
                      mcpAppResourceUri: resourceUri,
                      mcpAppServerName: serverName,
                      mcpAppToolInput: toolInput,
                    }
                  : m,
              ),
            )
          } else if (event.type === EventType.RUN_FINISHED) {
            setIsRunning(false)
          } else if (event.type === EventType.RUN_ERROR) {
            const errEvent = event as { message?: string }
            setMessages((prev) =>
              prev.map((m) =>
                m.id === assistantMsgId
                  ? { ...m, content: errEvent.message ?? 'An error occurred.', error: true }
                  : m,
              ),
            )
            setIsRunning(false)
          }
        },
        error(err) {
          setMessages((prev) =>
            prev.map((m) =>
              m.id === assistantMsgId
                ? {
                    ...m,
                    content: `Connection error: ${err instanceof Error ? err.message : String(err)}`,
                    error: true,
                  }
                : m,
            ),
          )
          setIsRunning(false)
        },
      })
    },
    [messages, isRunning, sessionId],
  )

  return { messages, sendMessage, isRunning }
}
