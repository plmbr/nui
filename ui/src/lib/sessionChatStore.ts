// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { HttpAgent } from '@ag-ui/client'
import { EventType } from '@ag-ui/core'
import { v4 as uuidv4 } from 'uuid'
import { api, type AgentRunEvent } from '@/api'
import {
  apiMessagesToSessionMessages,
  appendTextPart,
  applyAssistantError,
  assistantTextContent,
  type SessionChatMessage,
  type ToolCallPart,
  updateToolPart,
} from '@/lib/chatMessageUtils'
import { deriveSessionProgress, encodeSessionProgress, type SessionProgress } from '@/lib/sessionProgress'

type StopFn = () => void | Promise<void>

interface SessionEntry {
  messages: SessionChatMessage[]
  isRunning: boolean
  isLoaded: boolean
  isLoading: boolean
  assistantMsgId: string | null
  pendingTools: Record<string, string>
  agent: HttpAgent
  subscription: { unsubscribe: () => void } | null
  listeners: Set<() => void>
  snapshot: SessionChatSnapshot
}

function syncSnapshot(entry: SessionEntry) {
  entry.snapshot = {
    messages: entry.messages,
    isRunning: entry.isRunning,
    isLoading: !entry.isLoaded || entry.isLoading,
  }
}

const entries = new Map<string, SessionEntry>()
const globalListeners = new Set<() => void>()
const progressListeners = new Set<() => void>()
let runningSnapshot = ''
let progressSnapshot = ''

function getOrCreateEntry(sessionId: string): SessionEntry {
  let entry = entries.get(sessionId)
  if (!entry) {
    entry = {
      messages: [],
      isRunning: false,
      isLoaded: false,
      isLoading: false,
      assistantMsgId: null,
      pendingTools: {},
      agent: new HttpAgent({
        url: `/api/sessions/${sessionId}/ag-ui`,
        threadId: sessionId,
      }),
      subscription: null,
      listeners: new Set(),
      snapshot: { messages: [], isRunning: false, isLoading: true },
    }
    syncSnapshot(entry)
    entries.set(sessionId, entry)
  }
  return entry
}

function emitRunning() {
  const ids = [...entries.entries()]
    .filter(([, e]) => e.isRunning)
    .map(([id]) => id)
    .sort()
  runningSnapshot = ids.join(',')
  for (const listener of globalListeners) {
    listener()
  }
}

function emitProgress() {
  const parts: string[] = []
  for (const [sessionId, entry] of entries) {
    if (!entry.isRunning) continue
    const progress = deriveSessionProgress(entry.messages, entry.assistantMsgId)
    if (progress) {
      parts.push(`${sessionId}=${encodeSessionProgress(progress)}`)
    }
  }
  progressSnapshot = parts.join('|')
  for (const listener of progressListeners) {
    listener()
  }
}

function emitSession(sessionId: string) {
  const entry = entries.get(sessionId)
  if (!entry) return
  syncSnapshot(entry)
  for (const listener of entry.listeners) {
    listener()
  }
  emitRunning()
  emitProgress()
}

function setEntry(
  sessionId: string,
  update: (entry: SessionEntry) => Partial<SessionEntry> | void,
) {
  const entry = getOrCreateEntry(sessionId)
  const patch = update(entry)
  if (patch) {
    Object.assign(entry, patch)
  }
  emitSession(sessionId)
}

function finishRun(sessionId: string) {
  const entry = entries.get(sessionId)
  if (!entry) return
  entry.subscription?.unsubscribe()
  entry.subscription = null
  entry.assistantMsgId = null
  entry.isRunning = false
  emitSession(sessionId)
}

function hasAssistantContent(messages: SessionChatMessage[]): boolean {
  return messages.some(
    (m) => m.role === 'assistant' && (assistantTextContent(m).trim() !== '' || (m.parts?.length ?? 0) > 0),
  )
}

async function fetchMessagesFromServer(sessionId: string): Promise<SessionChatMessage[]> {
  const stored = await api.messages.list(sessionId)
  if (stored.length > 0) {
    return apiMessagesToSessionMessages(stored)
  }
  const history = await api.history.get(sessionId)
  return apiMessagesToSessionMessages(history)
}

async function reloadMessages(sessionId: string) {
  const entry = getOrCreateEntry(sessionId)
  const previous = entry.messages
  try {
    let next = await fetchMessagesFromServer(sessionId)
    if (!hasAssistantContent(next) && hasAssistantContent(previous)) {
      await new Promise((resolve) => setTimeout(resolve, 200))
      next = await fetchMessagesFromServer(sessionId)
    }
    if (!hasAssistantContent(next) && hasAssistantContent(previous)) {
      entry.messages = previous
      return
    }
    entry.messages = next
  } catch {
    entry.messages = []
  }
}

function applyAgentEvent(
  sessionId: string,
  assistantMsgId: string,
  ev: AgentRunEvent,
) {
  setEntry(sessionId, (entry) => {
    const pendingTools = { ...entry.pendingTools }
    const messages = entry.messages.map((m) => {
      if (m.id !== assistantMsgId) return m

      switch (ev.type) {
        case 'text':
          if (!ev.content) return m
          {
            const parts = appendTextPart(m.parts ?? [], ev.content)
            return { ...m, parts, content: assistantTextContent({ ...m, parts }) }
          }
        case 'tool_call_start': {
          const partId = uuidv4()
          if (ev.toolCallId) pendingTools[ev.toolCallId] = partId
          const toolPart: ToolCallPart = {
            type: 'tool',
            id: partId,
            toolCallId: ev.toolCallId,
            toolName: ev.toolName,
            toolArgs: {},
          }
          return { ...m, parts: [...(m.parts ?? []), toolPart] }
        }
        case 'tool_call_args': {
          if (!ev.toolCallId || !ev.toolArgs) return m
          const partId = pendingTools[ev.toolCallId]
          if (!partId || !m.parts) return m
          try {
            const args = JSON.parse(ev.toolArgs) as Record<string, unknown>
            const parts = m.parts.map((part) =>
              part.type === 'tool' && part.id === partId
                ? { ...part, toolArgs: { ...part.toolArgs, ...args } }
                : part,
            )
            return { ...m, parts }
          } catch {
            return m
          }
        }
        case 'tool_call_result': {
          if (!ev.toolCallId) return m
          const partId = pendingTools[ev.toolCallId]
          if (!partId || !m.parts) return m
          let result: unknown = ev.content
          try {
            result = JSON.parse(ev.content ?? '')
          } catch {
            /* keep string */
          }
          return { ...m, parts: updateToolPart(m.parts, partId, { toolResult: result }) }
        }
        case 'image': {
          if (!ev.imageData) return m
          return {
            ...m,
            images: [
              ...(m.images ?? []),
              {
                id: uuidv4(),
                mediaType: ev.imageMediaType ?? 'image/png',
                data: ev.imageData,
              },
            ],
          }
        }
        case 'error':
          return applyAssistantError(m, ev.error ?? '')
        default:
          return m
      }
    })
    return { messages, pendingTools }
  })
}

function attachRunEvents(sessionId: string, runId: string) {
  const entry = getOrCreateEntry(sessionId)
  entry.subscription?.unsubscribe()

  const unsub = api.runs.subscribeEvents(sessionId, runId, {
    onEvent: (event) => {
      const current = entries.get(sessionId)
      if (!current?.assistantMsgId) return
      applyAgentEvent(sessionId, current.assistantMsgId, event)
    },
    onFinished: () => {
      void (async () => {
        finishRun(sessionId)
        await reloadMessages(sessionId)
        emitSession(sessionId)
      })()
    },
    onError: (err) => {
      if (err.name === 'AbortError') return
      finishRun(sessionId)
    },
  })

  entry.subscription = { unsubscribe: unsub }
}

async function reconnectActiveRun(sessionId: string) {
  const entry = getOrCreateEntry(sessionId)
  if (entry.isRunning && entry.subscription) return

  let runs: Awaited<ReturnType<typeof api.runs.list>>
  try {
    runs = await api.runs.list(sessionId)
  } catch {
    return
  }

  const active = [...runs].reverse().find((r) => r.status === 'running')
  if (!active) return

  if (entry.messages.length === 0) {
    await reloadMessages(sessionId)
  }

  let assistantMsgId = entry.assistantMsgId
  const last = entry.messages[entry.messages.length - 1]
  if (!assistantMsgId) {
    if (last?.role === 'assistant') {
      assistantMsgId = last.id
    } else {
      assistantMsgId = uuidv4()
      const assistantMsg: SessionChatMessage = {
        id: assistantMsgId,
        role: 'assistant',
        content: '',
        parts: [],
      }
      entry.messages = [...entry.messages, assistantMsg]
    }
  }

  entry.assistantMsgId = assistantMsgId
  entry.isRunning = true
  entry.pendingTools = {}
  entry.messages = entry.messages.map((m) =>
    m.id === assistantMsgId
      ? { ...m, content: '', parts: [], images: undefined, error: false }
      : m,
  )
  registerSessionRun(sessionId, () => stopRun(sessionId))
  syncSnapshot(entry)
  emitSession(sessionId)
  attachRunEvents(sessionId, active.runId)
}

/** Re-attach to in-flight runs after page refresh (all sessions, for sidebar + chat). */
export async function probeActiveRuns(sessionIds: string[]) {
  await Promise.all(sessionIds.map((sessionId) => reconnectActiveRun(sessionId)))
}

export interface SessionChatSnapshot {
  messages: SessionChatMessage[]
  isRunning: boolean
  isLoading: boolean
}

export function getSessionChatSnapshot(sessionId: string): SessionChatSnapshot {
  const entry = getOrCreateEntry(sessionId)
  return entry.snapshot
}

export function subscribeSessionChat(sessionId: string, listener: () => void): () => void {
  const entry = getOrCreateEntry(sessionId)
  entry.listeners.add(listener)
  return () => entry.listeners.delete(listener)
}

export function getRunningSessionsSnapshot(): string {
  return runningSnapshot
}

export function subscribeSessionRuns(listener: () => void): () => void {
  globalListeners.add(listener)
  return () => globalListeners.delete(listener)
}

export function isSessionRunning(sessionId: string): boolean {
  return entries.get(sessionId)?.isRunning ?? false
}

export function getRunningProgressSnapshot(): string {
  return progressSnapshot
}

export function subscribeRunningProgress(listener: () => void): () => void {
  progressListeners.add(listener)
  return () => progressListeners.delete(listener)
}

export function getSessionProgressLabel(sessionId: string): string | null {
  const entry = entries.get(sessionId)
  if (!entry?.isRunning) return null
  return deriveSessionProgress(entry.messages, entry.assistantMsgId)?.label ?? null
}

export function getSessionProgress(sessionId: string): SessionProgress | null {
  const entry = entries.get(sessionId)
  if (!entry?.isRunning) return null
  return deriveSessionProgress(entry.messages, entry.assistantMsgId)
}

export async function ensureSessionChatLoaded(sessionId: string): Promise<void> {
  const entry = getOrCreateEntry(sessionId)
  if (entry.isLoaded || entry.isLoading) return

  entry.isLoading = true
  emitSession(sessionId)

  if (!entry.isRunning && entry.messages.length === 0) {
    await reloadMessages(sessionId)
  }
  entry.isLoaded = true
  entry.isLoading = false
  emitSession(sessionId)

  if (!entry.isRunning) {
    await reconnectActiveRun(sessionId)
  }
}

export async function stopRun(sessionId: string): Promise<void> {
  const entry = entries.get(sessionId)
  if (!entry?.isRunning && !entry?.assistantMsgId) return

  const cancelledAssistantId = entry?.assistantMsgId ?? null
  entry?.subscription?.unsubscribe()
  if (entry) {
    entry.subscription = null
    entry.assistantMsgId = null
    entry.isRunning = false
  }
  emitSession(sessionId)

  try {
    await api.sessions.stop(sessionId)
  } catch {
    /* best-effort */
  }

  if (cancelledAssistantId && entry) {
    entry.messages = entry.messages.map((m) => {
      if (m.id !== cancelledAssistantId) return m
      if (assistantTextContent(m).trim()) {
        return { ...m, error: true }
      }
      return applyAssistantError(m, 'Stopped.')
    })
    emitSession(sessionId)
  }
}

export function registerSessionRun(sessionId: string, _stop: StopFn) {
  emitRunning()
}

export async function stopSessionRun(sessionId: string) {
  await stopRun(sessionId)
}

export function sendMessage(sessionId: string, text: string) {
  const entry = getOrCreateEntry(sessionId)
  if (entry.isRunning || !text.trim()) return

  const userMsg: SessionChatMessage = { id: uuidv4(), role: 'user', content: text }
  const assistantMsgId = uuidv4()
  const assistantMsg: SessionChatMessage = {
    id: assistantMsgId,
    role: 'assistant',
    content: '',
    parts: [],
  }

  entry.messages = [...entry.messages, userMsg, assistantMsg]
  entry.isRunning = true
  entry.assistantMsgId = assistantMsgId
  entry.pendingTools = {}
  registerSessionRun(sessionId, () => stopRun(sessionId))
  emitSession(sessionId)

  const history = [
    ...entry.messages.slice(0, -2).map((m) => ({
      role: m.role as 'user' | 'assistant',
      content: m.role === 'assistant' ? assistantTextContent(m) : m.content,
      id: m.id,
    })),
    { role: 'user' as const, content: text, id: userMsg.id },
  ]

  const obs = entry.agent.run({
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
      const current = entries.get(sessionId)
      if (!current) return

      if (event.type === EventType.TEXT_MESSAGE_CHUNK) {
        const chunk = event as { delta?: string }
        if (chunk.delta) {
          setEntry(sessionId, (e) => ({
            messages: e.messages.map((m) => {
              if (m.id !== assistantMsgId) return m
              const parts = appendTextPart(m.parts ?? [], chunk.delta!)
              return { ...m, parts, content: assistantTextContent({ ...m, parts }) }
            }),
          }))
        }
      } else if (event.type === EventType.TOOL_CALL_START) {
        const e = event as { toolCallId?: string; toolCallName?: string }
        const partId = uuidv4()
        const pendingTools = { ...current.pendingTools }
        if (e.toolCallId) pendingTools[e.toolCallId] = partId
        const toolPart: ToolCallPart = {
          type: 'tool',
          id: partId,
          toolCallId: e.toolCallId,
          toolName: e.toolCallName,
          toolArgs: {},
        }
        setEntry(sessionId, (ent) => ({
          pendingTools,
          messages: ent.messages.map((m) =>
            m.id === assistantMsgId ? { ...m, parts: [...(m.parts ?? []), toolPart] } : m,
          ),
        }))
      } else if (event.type === EventType.TOOL_CALL_ARGS) {
        const e = event as { toolCallId?: string; delta?: string }
        if (!e.toolCallId) return
        const partId = current.pendingTools[e.toolCallId]
        if (!partId || !e.delta) return
        try {
          const args = JSON.parse(e.delta) as Record<string, unknown>
          setEntry(sessionId, (ent) => ({
            messages: ent.messages.map((m) => {
              if (m.id !== assistantMsgId || !m.parts) return m
              const parts = m.parts.map((part) =>
                part.type === 'tool' && part.id === partId
                  ? { ...part, toolArgs: { ...part.toolArgs, ...args } }
                  : part,
              )
              return { ...m, parts }
            }),
          }))
        } catch {
          /* ignore partial JSON */
        }
      } else if (event.type === EventType.TOOL_CALL_RESULT) {
        const e = event as { toolCallId?: string; content?: string }
        if (!e.toolCallId) return
        const partId = current.pendingTools[e.toolCallId]
        if (!partId) return
        let result: unknown = e.content
        try {
          result = JSON.parse(e.content ?? '')
        } catch {
          /* keep as string */
        }
        setEntry(sessionId, (ent) => ({
          messages: ent.messages.map((m) => {
            if (m.id !== assistantMsgId || !m.parts) return m
            return { ...m, parts: updateToolPart(m.parts, partId, { toolResult: result }) }
          }),
        }))
      } else if (event.type === EventType.CUSTOM) {
        const e = event as { name?: string; value?: Record<string, unknown> }
        if (!e.value) return
        if (e.name === 'image') {
          const { mediaType, data } = e.value as { mediaType?: string; data?: string }
          if (!data) return
          setEntry(sessionId, (ent) => ({
            messages: ent.messages.map((m) =>
              m.id === assistantMsgId
                ? {
                    ...m,
                    images: [
                      ...(m.images ?? []),
                      { id: uuidv4(), mediaType: mediaType ?? 'image/png', data },
                    ],
                  }
                : m,
            ),
          }))
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
        const partId = current.pendingTools[toolCallId]
        if (!partId) return
        setEntry(sessionId, (ent) => ({
          messages: ent.messages.map((m) => {
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
        }))
      } else if (event.type === EventType.RUN_FINISHED) {
        finishRun(sessionId)
        void reloadMessages(sessionId).then(() => emitSession(sessionId))
      } else if (event.type === EventType.RUN_ERROR) {
        const errEvent = event as { message?: string }
        const errText = errEvent.message?.trim()
        const isCancelled = errText?.toLowerCase() === 'cancelled'
        setEntry(sessionId, (ent) => ({
          messages: ent.messages.map((m) => {
            if (m.id !== assistantMsgId) return m
            if (isCancelled) {
              if (assistantTextContent(m).trim()) {
                return { ...m, error: true }
              }
              return applyAssistantError(m, 'Stopped.')
            }
            return applyAssistantError(m, errText ?? '')
          }),
        }))
        finishRun(sessionId)
        void reloadMessages(sessionId).then(() => emitSession(sessionId))
      }
    },
    error(err) {
      setEntry(sessionId, (ent) => ({
        messages: ent.messages.map((m) =>
          m.id === assistantMsgId
            ? applyAssistantError(
                m,
                `Connection error: ${err instanceof Error ? err.message : String(err)}`,
              )
            : m,
        ),
      }))
      finishRun(sessionId)
    },
  })

  entry.subscription = sub
}

export function clearSessionChat(sessionId: string) {
  const entry = entries.get(sessionId)
  if (!entry) return
  entry.subscription?.unsubscribe()
  entries.delete(sessionId)
  emitRunning()
  emitProgress()
}
