// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { v4 as uuidv4 } from 'uuid'
import {
  appendTextPart,
  type AssistantPart,
  type SessionChatMessage,
  type ToolCallPart,
  updateToolPart,
} from '@/lib/chatMessageUtils'
import { mergeToolCallArgsDelta } from '@/lib/chatMessageUtils'

export function isHarnessSubagentTool(toolName: string | undefined): boolean {
  const raw = toolName?.split(':').pop() ?? toolName ?? ''
  const bare = raw.split('__').pop()?.toLowerCase() ?? ''
  return bare === 'task' || bare === 'agent'
}

function findParentToolPart(
  msg: SessionChatMessage,
  parentToolCallId: string,
): ToolCallPart | undefined {
  const part = msg.parts?.find(
    (p): p is ToolCallPart => p.type === 'tool' && p.toolCallId === parentToolCallId,
  )
  return part
}

function updateParentToolPart(
  msg: SessionChatMessage,
  parentToolCallId: string,
  updater: (part: ToolCallPart) => ToolCallPart,
): SessionChatMessage {
  if (!msg.parts) return msg
  return {
    ...msg,
    parts: msg.parts.map((part) => {
      if (part.type !== 'tool' || part.toolCallId !== parentToolCallId) return part
      return updater(part)
    }),
  }
}

function updateSubagentToolPart(
  trace: AssistantPart[],
  partId: string,
  update: Partial<ToolCallPart>,
): AssistantPart[] {
  return trace.map((part) =>
    part.type === 'tool' && part.id === partId ? { ...part, ...update } : part,
  )
}

function updateSubagentToolArgsInTrace(
  trace: AssistantPart[],
  partId: string,
  delta: string,
  buffers: Record<string, string>,
  toolCallId: string,
): AssistantPart[] {
  return trace.map((part) => {
    if (part.type !== 'tool' || part.id !== partId) return part
    const merged = mergeToolCallArgsDelta(part.toolArgs, buffers[toolCallId] ?? '', delta)
    buffers[toolCallId] = merged.buffer
    return { ...part, toolArgs: merged.toolArgs }
  })
}

function finalizeSubagentToolArgs(
  part: ToolCallPart,
  argBuffer: string,
): ToolCallPart {
  if (!argBuffer.trim()) return part
  try {
    const parsed = JSON.parse(argBuffer) as Record<string, unknown>
    return { ...part, toolArgs: { ...(part.toolArgs ?? {}), ...parsed } }
  } catch {
    return part
  }
}

export function applySubagentTextChunk(
  msg: SessionChatMessage,
  parentToolCallId: string,
  delta: string,
): SessionChatMessage {
  return updateParentToolPart(msg, parentToolCallId, (part) => ({
    ...part,
    subagentTrace: appendTextPart(part.subagentTrace ?? [], delta),
  }))
}

export function applySubagentToolStart(
  msg: SessionChatMessage,
  parentToolCallId: string,
  toolCallId: string | undefined,
  toolName: string | undefined,
  pendingSubagentTools: Record<string, string>,
): SessionChatMessage {
  const partId = uuidv4()
  if (toolCallId) pendingSubagentTools[toolCallId] = partId
  const toolPart: ToolCallPart = {
    type: 'tool',
    id: partId,
    toolCallId,
    toolName,
    toolArgs: {},
  }
  return updateParentToolPart(msg, parentToolCallId, (part) => ({
    ...part,
    subagentTrace: [...(part.subagentTrace ?? []), toolPart],
  }))
}

export function applySubagentToolArgs(
  msg: SessionChatMessage,
  parentToolCallId: string,
  toolCallId: string,
  delta: string,
  pendingSubagentTools: Record<string, string>,
  buffers: Record<string, string>,
): SessionChatMessage {
  const partId = pendingSubagentTools[toolCallId]
  if (!partId) return msg
  return updateParentToolPart(msg, parentToolCallId, (part) => ({
    ...part,
    subagentTrace: updateSubagentToolArgsInTrace(
      part.subagentTrace ?? [],
      partId,
      delta,
      buffers,
      toolCallId,
    ),
  }))
}

export function applySubagentToolEnd(
  msg: SessionChatMessage,
  parentToolCallId: string,
  toolCallId: string,
  pendingSubagentTools: Record<string, string>,
  buffers: Record<string, string>,
): SessionChatMessage {
  const partId = pendingSubagentTools[toolCallId]
  const argBuffer = buffers[toolCallId] ?? ''
  if (!partId || !argBuffer.trim()) return msg
  return updateParentToolPart(msg, parentToolCallId, (part) => ({
    ...part,
    subagentTrace: (part.subagentTrace ?? []).map((tracePart) =>
      tracePart.type === 'tool' && tracePart.id === partId
        ? finalizeSubagentToolArgs(tracePart, argBuffer)
        : tracePart,
    ),
  }))
}

export function applySubagentToolResult(
  msg: SessionChatMessage,
  parentToolCallId: string,
  toolCallId: string,
  result: unknown,
  pendingSubagentTools: Record<string, string>,
): SessionChatMessage {
  const partId = pendingSubagentTools[toolCallId]
  if (!partId) return msg
  return updateParentToolPart(msg, parentToolCallId, (part) => ({
    ...part,
    subagentTrace: updateSubagentToolPart(part.subagentTrace ?? [], partId, { toolResult: result }),
  }))
}

export function parentToolCallIdFromEvent(event: unknown): string | undefined {
  const e = event as { parentToolCallId?: string }
  return e.parentToolCallId?.trim() || undefined
}

export function hasParentToolPart(msg: SessionChatMessage, parentToolCallId: string): boolean {
  return findParentToolPart(msg, parentToolCallId) !== undefined
}

export function latestSubagentProgressLabel(part: ToolCallPart): string | undefined {
  const trace = part.subagentTrace ?? []
  if (trace.length === 0) return undefined
  const last = trace[trace.length - 1]
  if (last.type === 'text') {
    const snippet = last.content.trim().split('\n').pop()?.trim()
    if (snippet) return snippet.length > 80 ? `${snippet.slice(0, 79)}…` : snippet
    return 'Working…'
  }
  if (last.type === 'tool') {
    const name = last.toolName?.split('__').pop() ?? last.toolName ?? 'tool'
    if (last.toolResult === undefined) return `${name}…`
    return undefined
  }
  return undefined
}
