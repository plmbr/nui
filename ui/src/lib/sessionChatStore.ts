// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { HttpAgent } from '@ag-ui/client'
import { EventType } from '@ag-ui/core'
import { v4 as uuidv4 } from 'uuid'
import { api, type AgentRunEvent } from '@/api'
import type { HitlRequest } from '@/types'
import {
  dedupeChatMessages,
  apiMessagesToSessionMessages,
  appendTextPart,
  applyAssistantError,
  assistantTextContent,
  mergeToolCallArgsDelta,
  type SessionChatMessage,
  type ToolCallPart,
  updateToolPart,
} from '@/lib/chatMessageUtils'
import { deriveSessionProgress, encodeSessionProgress, type SessionProgress } from '@/lib/sessionProgress'
import {
  applySubagentTextChunk,
  applySubagentToolArgs,
  applySubagentToolEnd,
  applySubagentToolResult,
  applySubagentToolStart,
  parentToolCallIdFromEvent,
} from '@/lib/subagentTrace'
import { visualizationFromArgs, visualizationFromToolResult, visualizationHTMLReady } from '@/lib/visualization'
import { prepareVisualizationHtml } from '@/lib/prepareVisualizationHtml'

type StopFn = () => void | Promise<void>

interface SessionEntry {
  messages: SessionChatMessage[]
  isRunning: boolean
  isLoaded: boolean
  isLoading: boolean
  assistantMsgId: string | null
  activeRunId: string | null
  pendingTools: Record<string, string>
  pendingToolArgBuffers: Record<string, string>
  pendingSubagentTools: Record<string, Record<string, string>>
  pendingSubagentArgBuffers: Record<string, Record<string, string>>
  pendingHitl: Record<string, HitlRequest>
  agent: HttpAgent
  subscription: { unsubscribe: () => void } | null
  listeners: Set<() => void>
  snapshot: SessionChatSnapshot
}

function clearPendingToolState(entry: SessionEntry) {
  entry.pendingTools = {}
  entry.pendingToolArgBuffers = {}
  entry.pendingSubagentTools = {}
  entry.pendingSubagentArgBuffers = {}
}

function syncSnapshot(entry: SessionEntry) {
  entry.snapshot = {
    messages: entry.messages,
    isRunning: entry.isRunning,
    isLoading: !entry.isLoaded || entry.isLoading,
    pendingHitl: Object.values(entry.pendingHitl),
  }
}

const entries = new Map<string, SessionEntry>()
const aguiFinishedRuns = new Map<string, Set<string>>()
const globalListeners = new Set<() => void>()
const progressListeners = new Set<() => void>()
let runningSnapshot = ''
let progressSnapshot = ''

function createSessionAgent(sessionId: string): HttpAgent {
  return new HttpAgent({
    url: `/api/sessions/${sessionId}/ag-ui`,
    threadId: sessionId,
  })
}

function getOrCreateEntry(sessionId: string): SessionEntry {
  let entry = entries.get(sessionId)
  if (!entry) {
    entry = {
      messages: [],
      isRunning: false,
      isLoaded: false,
      isLoading: false,
      assistantMsgId: null,
      activeRunId: null,
      pendingTools: {},
      pendingToolArgBuffers: {},
      pendingSubagentTools: {},
      pendingSubagentArgBuffers: {},
      pendingHitl: {},
      agent: createSessionAgent(sessionId),
      subscription: null,
      listeners: new Set(),
      snapshot: { messages: [], isRunning: false, isLoading: true, pendingHitl: [] },
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

function chartScriptSrc(): string {
  if (typeof window !== 'undefined') {
    return `${window.location.origin}/vendor/chart.min.js`
  }
  return '/vendor/chart.min.js'
}

function applyToolArgsToPart(
  part: ToolCallPart,
  argBuffer: string,
  delta: string,
): { part: ToolCallPart; buffer: string } {
  const { toolArgs, buffer } = mergeToolCallArgsDelta(part.toolArgs, argBuffer, delta)
  const viz = visualizationFromArgs(part.toolName, toolArgs)
  return {
    buffer,
    part: {
      ...part,
      toolArgs,
      visualizationHtml: viz?.html ?? part.visualizationHtml,
      visualizationTitle: viz?.title ?? part.visualizationTitle,
    },
  }
}

function finalizeToolCallArgs(part: ToolCallPart, argBuffer: string): ToolCallPart {
  if (!argBuffer.trim()) return part
  try {
    const parsed: unknown = JSON.parse(argBuffer)
    if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
      const toolArgs = parsed as Record<string, unknown>
      const viz = visualizationFromArgs(part.toolName, toolArgs)
      return {
        ...part,
        toolArgs,
        visualizationHtml: viz?.html ?? part.visualizationHtml,
        visualizationTitle: viz?.title ?? part.visualizationTitle,
      }
    }
  } catch {
    /* ignore invalid JSON */
  }
  return part
}

function updateToolArgsInMessage(
  msg: SessionChatMessage,
  partId: string,
  delta: string,
  argBuffers: Record<string, string>,
  toolCallId: string,
): SessionChatMessage {
  if (!msg.parts) return msg
  const buffer = argBuffers[toolCallId] ?? ''
  let nextBuffer = buffer
  const parts = msg.parts.map((part) => {
    if (part.type !== 'tool' || part.id !== partId) return part
    const updated = applyToolArgsToPart(part, nextBuffer, delta)
    nextBuffer = updated.buffer
    return updated.part
  })
  argBuffers[toolCallId] = nextBuffer
  return { ...msg, parts }
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

function markAguiRunFinished(sessionId: string, runId: string) {
  let finished = aguiFinishedRuns.get(sessionId)
  if (!finished) {
    finished = new Set()
    aguiFinishedRuns.set(sessionId, finished)
  }
  finished.add(runId)
}

function finishRun(sessionId: string, runId?: string) {
  const entry = entries.get(sessionId)
  if (!entry) return
  if (runId && entry.activeRunId && entry.activeRunId !== runId) return
  if (runId) markAguiRunFinished(sessionId, runId)
  entry.subscription?.unsubscribe()
  entry.subscription = null
  entry.assistantMsgId = null
  entry.activeRunId = null
  clearPendingToolState(entry)
  entry.isRunning = false
  emitSession(sessionId)
}

async function fetchMessagesFromServer(sessionId: string): Promise<SessionChatMessage[]> {
  const stored = await api.messages.list(sessionId)
  if (stored.length > 0) {
    return apiMessagesToSessionMessages(stored)
  }
  const history = await api.history.get(sessionId)
  return apiMessagesToSessionMessages(history)
}

function upsertHitlRequest(sessionId: string, req: HitlRequest) {
  if (!req.requestId) return
  setEntry(sessionId, (entry) => ({
    pendingHitl: { ...entry.pendingHitl, [req.requestId]: req },
  }))
}

async function reloadPendingHitl(sessionId: string) {
  try {
    const pending = await api.hitl.listPending(sessionId)
    if (pending.length === 0) return
    setEntry(sessionId, (entry) => {
      const pendingHitl = { ...entry.pendingHitl }
      for (const req of pending) {
        if (req.requestId) pendingHitl[req.requestId] = req
      }
      return { pendingHitl }
    })
  } catch {
    /* best-effort */
  }
}

function scheduleHitlReload(sessionId: string) {
  void (async () => {
    for (let attempt = 0; attempt < 30; attempt++) {
      await reloadPendingHitl(sessionId)
      const entry = entries.get(sessionId)
      if (entry && Object.keys(entry.pendingHitl).length > 0) return
      await new Promise((resolve) => setTimeout(resolve, 500))
    }
  })()
}

export function dismissHitlRequest(sessionId: string, requestId: string) {
  setEntry(sessionId, (entry) => {
    if (!entry.pendingHitl[requestId]) return
    const pendingHitl = { ...entry.pendingHitl }
    delete pendingHitl[requestId]
    return { pendingHitl }
  })
}

function hitlRequestFromCustomValue(value: Record<string, unknown>): HitlRequest | null {
  const requestId = typeof value.requestId === 'string' ? value.requestId : ''
  if (!requestId) return null
  return {
    requestId,
    sessionId: typeof value.sessionId === 'string' ? value.sessionId : undefined,
    runId: typeof value.runId === 'string' ? value.runId : undefined,
    stepName: typeof value.stepName === 'string' ? value.stepName : undefined,
    kind: typeof value.kind === 'string' ? value.kind : 'question',
    payload: value.payload as HitlRequest['payload'],
    status: typeof value.status === 'string' ? value.status : undefined,
    expiresAt: typeof value.expiresAt === 'string' ? value.expiresAt : undefined,
    createdAt: typeof value.createdAt === 'string' ? value.createdAt : undefined,
  }
}

function handleHitlCustomEvent(sessionId: string, value: Record<string, unknown>) {
  const req = hitlRequestFromCustomValue(value)
  if (req) upsertHitlRequest(sessionId, req)
}

async function handleHitlRunEvent(sessionId: string, requestId: string) {
  if (!requestId.trim()) return
  try {
    const req = await api.hitl.get(requestId.trim())
    upsertHitlRequest(sessionId, req)
  } catch {
    /* best-effort */
  }
}

async function reloadMessages(sessionId: string) {
  const entry = getOrCreateEntry(sessionId)
  if (entry.isRunning) return
  // SSE is authoritative during an active chat; only hydrate from server on first load.
  if (entry.messages.length > 0) return

  try {
    const next = await fetchMessagesFromServer(sessionId)
    const current = entries.get(sessionId)
    if (!current || current.isRunning || current.messages.length > 0) return
    current.messages = dedupeChatMessages(next)
  } catch {
    /* keep empty */
  }
}

function applyAgentEvent(
  sessionId: string,
  assistantMsgId: string,
  ev: AgentRunEvent,
) {
  if (ev.type === 'hitl_request') {
    void handleHitlRunEvent(sessionId, ev.content ?? '')
    return
  }

  setEntry(sessionId, (entry) => {
    const pendingTools = { ...entry.pendingTools }
    const pendingToolArgBuffers = { ...entry.pendingToolArgBuffers }
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
          if (!partId) return m
          return updateToolArgsInMessage(m, partId, ev.toolArgs, pendingToolArgBuffers, ev.toolCallId)
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
          const vizFromResult = visualizationFromToolResult(result)
          return {
            ...m,
            parts: updateToolPart(m.parts, partId, {
              toolResult: result,
              ...(vizFromResult
                ? {
                    visualizationHtml: vizFromResult.html,
                    visualizationTitle: vizFromResult.title,
                  }
                : {}),
            }),
          }
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
    return { messages, pendingTools, pendingToolArgBuffers }
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
      finishRun(sessionId, runId)
    },
    onError: (err) => {
      if (err.name === 'AbortError') return
      finishRun(sessionId, runId)
    },
  })

  entry.subscription = { unsubscribe: unsub }
}

async function reconnectActiveRun(sessionId: string) {
  const entry = getOrCreateEntry(sessionId)
  // Never attach a second event stream while the live AG-UI run owns this session.
  if (entry.isRunning) return

  let runs: Awaited<ReturnType<typeof api.runs.list>>
  try {
    runs = await api.runs.list(sessionId)
  } catch {
    return
  }

  const active = [...runs].reverse().find((r) => r.status === 'running' || r.status === 'awaiting_user')
  if (!active) {
    await reloadPendingHitl(sessionId)
    return
  }

  const finishedViaAgui = aguiFinishedRuns.get(sessionId)
  if (finishedViaAgui?.has(active.runId)) {
    return
  }

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
  entry.activeRunId = active.runId
  entry.isRunning = true
  clearPendingToolState(entry)
  entry.messages = entry.messages.map((m) =>
    m.id === assistantMsgId &&
    m.role === 'assistant' &&
    !assistantTextContent(m).trim() &&
    !(m.parts?.length ?? 0)
      ? { ...m, content: '', parts: [], images: undefined, error: false }
      : m,
  )
  registerSessionRun(sessionId, () => stopRun(sessionId))
  syncSnapshot(entry)
  emitSession(sessionId)
  attachRunEvents(sessionId, active.runId)
  await reloadPendingHitl(sessionId)
}

/** Re-attach to in-flight runs after page refresh (all sessions, for sidebar + chat). */
export async function probeActiveRuns(sessionIds: string[]) {
  await Promise.all(sessionIds.map((sessionId) => reconnectActiveRun(sessionId)))
}

export interface SessionChatSnapshot {
  messages: SessionChatMessage[]
  isRunning: boolean
  isLoading: boolean
  pendingHitl: HitlRequest[]
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

  if (!entry.isRunning && entry.messages.length === 0) {
    await reconnectActiveRun(sessionId)
  }
  await reloadPendingHitl(sessionId)
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
  const trimmed = text.trim()
  if (!trimmed) return

  const entry = getOrCreateEntry(sessionId)
  if (entry.isRunning) return

  entry.isRunning = true
  emitSession(sessionId)
  startSend(sessionId, trimmed)
}

function startSend(sessionId: string, text: string) {
  const entry = getOrCreateEntry(sessionId)

  const userMsg: SessionChatMessage = { id: uuidv4(), role: 'user', content: text }
  const assistantMsgId = uuidv4()
  const runId = uuidv4()
  const assistantMsg: SessionChatMessage = {
    id: assistantMsgId,
    role: 'assistant',
    content: '',
    parts: [],
  }

  entry.messages = [...entry.messages, userMsg, assistantMsg]
  entry.assistantMsgId = assistantMsgId
  entry.activeRunId = runId
  clearPendingToolState(entry)
  entry.subscription?.unsubscribe()
  entry.subscription = null
  entry.agent = createSessionAgent(sessionId)

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
    runId,
    messages: history,
    tools: [],
    state: {},
    context: [],
    forwardedProps: {},
  })

  const sub = obs.subscribe({
    next(event) {
      const current = entries.get(sessionId)
      if (!current || current.activeRunId !== runId) return

      if (event.type === EventType.TEXT_MESSAGE_CHUNK) {
        const chunk = event as { delta?: string; parentToolCallId?: string }
        const parentToolCallId = parentToolCallIdFromEvent(chunk)
        if (chunk.delta) {
          setEntry(sessionId, (e) => ({
            messages: e.messages.map((m) => {
              if (m.id !== assistantMsgId) return m
              if (parentToolCallId) {
                return applySubagentTextChunk(m, parentToolCallId, chunk.delta!)
              }
              const parts = appendTextPart(m.parts ?? [], chunk.delta!)
              return { ...m, parts, content: assistantTextContent({ ...m, parts }) }
            }),
          }))
        }
      } else if (event.type === EventType.TOOL_CALL_START) {
        const e = event as { toolCallId?: string; toolCallName?: string; parentToolCallId?: string }
        const parentToolCallId = parentToolCallIdFromEvent(e)
        if (parentToolCallId) {
          setEntry(sessionId, (ent) => {
            const pendingSubagentTools = {
              ...ent.pendingSubagentTools,
              [parentToolCallId]: { ...(ent.pendingSubagentTools[parentToolCallId] ?? {}) },
            }
            const subPending = pendingSubagentTools[parentToolCallId]
            return {
              pendingSubagentTools,
              messages: ent.messages.map((m) => {
                if (m.id !== assistantMsgId) return m
                return applySubagentToolStart(
                  m,
                  parentToolCallId,
                  e.toolCallId,
                  e.toolCallName,
                  subPending,
                )
              }),
            }
          })
          return
        }
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
        const toolName = (e.toolCallName ?? '').toLowerCase()
        if (toolName.includes('askuserquestion') || toolName.includes('ask_user')) {
          scheduleHitlReload(sessionId)
        }
      } else if (event.type === EventType.TOOL_CALL_ARGS) {
        const e = event as { toolCallId?: string; delta?: string; parentToolCallId?: string }
        const parentToolCallId = parentToolCallIdFromEvent(e)
        if (!e.toolCallId || !e.delta) return
        if (parentToolCallId) {
          setEntry(sessionId, (ent) => {
            const pendingSubagentTools = {
              ...ent.pendingSubagentTools,
              [parentToolCallId]: { ...(ent.pendingSubagentTools[parentToolCallId] ?? {}) },
            }
            const pendingSubagentArgBuffers = {
              ...ent.pendingSubagentArgBuffers,
              [parentToolCallId]: { ...(ent.pendingSubagentArgBuffers[parentToolCallId] ?? {}) },
            }
            const subPending = pendingSubagentTools[parentToolCallId]
            const subBuffers = pendingSubagentArgBuffers[parentToolCallId]
            return {
              pendingSubagentTools,
              pendingSubagentArgBuffers,
              messages: ent.messages.map((m) => {
                if (m.id !== assistantMsgId) return m
                return applySubagentToolArgs(
                  m,
                  parentToolCallId,
                  e.toolCallId!,
                  e.delta!,
                  subPending,
                  subBuffers,
                )
              }),
            }
          })
          return
        }
        setEntry(sessionId, (ent) => {
          const partId = ent.pendingTools[e.toolCallId!]
          if (!partId) return ent
          const pendingToolArgBuffers = { ...ent.pendingToolArgBuffers }
          return {
            pendingToolArgBuffers,
            messages: ent.messages.map((m) =>
              m.id === assistantMsgId
                ? updateToolArgsInMessage(m, partId, e.delta!, pendingToolArgBuffers, e.toolCallId!)
                : m,
            ),
          }
        })
      } else if (event.type === EventType.TOOL_CALL_END) {
        const e = event as { toolCallId?: string; parentToolCallId?: string }
        const parentToolCallId = parentToolCallIdFromEvent(e)
        if (!e.toolCallId) return
        if (parentToolCallId) {
          setEntry(sessionId, (ent) => {
            const pendingSubagentArgBuffers = {
              ...ent.pendingSubagentArgBuffers,
              [parentToolCallId]: { ...(ent.pendingSubagentArgBuffers[parentToolCallId] ?? {}) },
            }
            const subBuffers = pendingSubagentArgBuffers[parentToolCallId]
            return {
              pendingSubagentArgBuffers,
              messages: ent.messages.map((m) => {
                if (m.id !== assistantMsgId) return m
                return applySubagentToolEnd(
                  m,
                  parentToolCallId,
                  e.toolCallId!,
                  ent.pendingSubagentTools[parentToolCallId] ?? {},
                  subBuffers,
                )
              }),
            }
          })
          return
        }
        setEntry(sessionId, (ent) => {
          const partId = ent.pendingTools[e.toolCallId!]
          const argBuffer = ent.pendingToolArgBuffers[e.toolCallId!] ?? ''
          if (!partId) return ent
          if (!argBuffer.trim()) return ent
          return {
            messages: ent.messages.map((m) => {
              if (m.id !== assistantMsgId || !m.parts) return m
              const parts = m.parts.map((part) =>
                part.type === 'tool' && part.id === partId
                  ? finalizeToolCallArgs(part, argBuffer)
                  : part,
              )
              return { ...m, parts }
            }),
          }
        })
      } else if (event.type === EventType.TOOL_CALL_RESULT) {
        const e = event as { toolCallId?: string; content?: string; parentToolCallId?: string }
        const parentToolCallId = parentToolCallIdFromEvent(e)
        if (!e.toolCallId) return
        let result: unknown = e.content
        try {
          result = JSON.parse(e.content ?? '')
        } catch {
          /* keep as string */
        }
        if (parentToolCallId) {
          setEntry(sessionId, (ent) => ({
            messages: ent.messages.map((m) => {
              if (m.id !== assistantMsgId) return m
              return applySubagentToolResult(
                m,
                parentToolCallId,
                e.toolCallId!,
                result,
                ent.pendingSubagentTools[parentToolCallId] ?? {},
              )
            }),
          }))
          return
        }
        const partId = current.pendingTools[e.toolCallId]
        if (!partId) return
        const vizFromResult = visualizationFromToolResult(result)
        setEntry(sessionId, (ent) => ({
          messages: ent.messages.map((m) => {
            if (m.id !== assistantMsgId || !m.parts) return m
            return {
              ...m,
              parts: updateToolPart(m.parts, partId, {
                toolResult: result,
                ...(vizFromResult
                  ? {
                      visualizationHtml: vizFromResult.html,
                      visualizationTitle: vizFromResult.title,
                    }
                  : {}),
              }),
            }
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
        if (e.name === 'hitl_request') {
          handleHitlCustomEvent(sessionId, e.value)
          return
        }
        if (e.name === 'sub_agent_routed') {
          const { label } = e.value as { agentId?: string; label?: string }
          if (!label) return
          setEntry(sessionId, (ent) => ({
            messages: ent.messages.map((m) =>
              m.id === assistantMsgId ? { ...m, routedAgentLabel: label } : m,
            ),
          }))
          return
        }
        if (e.name === 'visualization') {
          const { toolCallId, html, title } = e.value as {
            toolCallId?: string
            html?: string
            title?: string
          }
          if (!toolCallId || !html) return
          const prepared = prepareVisualizationHtml(html, chartScriptSrc())
          if (!visualizationHTMLReady(prepared)) return
          const partId = current.pendingTools[toolCallId]
          if (!partId) return
          setEntry(sessionId, (ent) => ({
            messages: ent.messages.map((m) => {
              if (m.id !== assistantMsgId || !m.parts) return m
              return {
                ...m,
                parts: updateToolPart(m.parts, partId, {
                  visualizationHtml: prepared,
                  visualizationTitle: typeof title === 'string' ? title : undefined,
                }),
              }
            }),
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
        finishRun(sessionId, runId)
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
        finishRun(sessionId, runId)
      }
    },
    error(err) {
      const current = entries.get(sessionId)
      if (current?.activeRunId !== runId) return
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
      finishRun(sessionId, runId)
    },
    complete() {
      const current = entries.get(sessionId)
      if (!current || current.activeRunId !== runId || !current.isRunning) return
      finishRun(sessionId, runId)
    },
  })

  entry.subscription = sub
  registerSessionRun(sessionId, () => stopRun(sessionId))
  emitSession(sessionId)
}

export function clearSessionChat(sessionId: string) {
  aguiFinishedRuns.delete(sessionId)
  const entry = entries.get(sessionId)
  if (!entry) return
  entry.subscription?.unsubscribe()
  entries.delete(sessionId)
  emitRunning()
  emitProgress()
}
