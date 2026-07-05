// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import type { ChatImage, ChatMessage, ChatMessagePart } from '@/types'
import { extractVisualization, normalizeVisualizationParts } from '@/lib/visualization'

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
  visualizationHtml?: string
  visualizationTitle?: string
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
}

export function assistantTextContent(msg: SessionChatMessage): string {
  if (msg.parts?.length) {
    return msg.parts
      .filter((part): part is TextPart => part.type === 'text')
      .map((part) => part.content)
      .join('')
  }
  return msg.content
}

export function appendTextPart(parts: AssistantPart[], delta: string): AssistantPart[] {
  const next = [...parts]
  const last = next[next.length - 1]
  if (last?.type === 'text') {
    next[next.length - 1] = { ...last, content: last.content + delta }
  } else {
    next.push({ type: 'text', content: delta })
  }
  return next
}

export function updateToolPart(
  parts: AssistantPart[],
  partId: string,
  update: Partial<ToolCallPart>,
): AssistantPart[] {
  return parts.map((part) =>
    part.type === 'tool' && part.id === partId ? { ...part, ...update } : part,
  )
}

export function applyAssistantError(msg: SessionChatMessage, text: string): SessionChatMessage {
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

function mapPart(part: ChatMessagePart): AssistantPart {
  if (part.type === 'text') {
    return { type: 'text', content: part.content ?? '' }
  }
  const mapped: ToolCallPart = {
    type: 'tool',
    id: part.id ?? '',
    toolCallId: part.toolCallId,
    toolName: part.toolName,
    toolArgs: part.toolArgs,
    toolResult: part.toolResult,
    mcpAppResourceUri: part.mcpAppResourceUri,
    mcpAppServerName: part.mcpAppServerName,
    mcpAppToolInput: part.mcpAppToolInput,
    visualizationHtml: part.visualizationHtml,
    visualizationTitle: part.visualizationTitle,
  }
  const viz = extractVisualization(mapped)
  if (viz && !mapped.visualizationHtml) {
    mapped.visualizationHtml = viz.html
    mapped.visualizationTitle = viz.title
  }
  return mapped
}

export function dedupeChatMessages(messages: SessionChatMessage[]): SessionChatMessage[] {
  const result: SessionChatMessage[] = []
  for (const msg of messages) {
    const prev = result[result.length - 1]
    if (prev && prev.role === msg.role) {
      const prevText =
        prev.role === 'assistant' ? assistantTextContent(prev).trim() : prev.content.trim()
      const msgText =
        msg.role === 'assistant' ? assistantTextContent(msg).trim() : msg.content.trim()
      if (prevText !== '' && prevText === msgText) continue
    }
    result.push(msg)
  }
  return result
}

export function apiMessagesToSessionMessages(history: ChatMessage[]): SessionChatMessage[] {
  return history
    .filter((m) => m.role === 'user' || m.role === 'assistant')
    .map((m) => {
      const parts = m.parts?.map(mapPart)
      return {
        id: m.id,
        role: m.role,
        content: m.content,
        parts: parts ? normalizeVisualizationParts(parts) : undefined,
        images: m.images,
        error: m.error,
      }
    })
}

export function imageSrc(image: ChatImage): string {
  if (image.data.startsWith('http://') || image.data.startsWith('https://')) {
    return image.data
  }
  return `data:${image.mediaType};base64,${image.data}`
}
