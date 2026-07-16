// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import type { SessionChatMessage, ToolCallPart } from '@/lib/chatMessageUtils'
import { formatToolCallSummary } from '@/lib/toolCallSummary'
import { isHarnessSubagentTool, latestSubagentProgressLabel } from '@/lib/subagentTrace'

export type SessionProgressKind = 'thinking' | 'generating' | 'tool'

export interface SessionProgress {
  kind: SessionProgressKind
  label: string
}

const PROGRESS_SEP = '\x1f'

export function encodeSessionProgress(progress: SessionProgress): string {
  return encodeURIComponent(`${progress.kind}${PROGRESS_SEP}${progress.label}`)
}

export function decodeSessionProgress(encoded: string): SessionProgress | null {
  const raw = decodeURIComponent(encoded)
  const sep = raw.indexOf(PROGRESS_SEP)
  if (sep <= 0) return null
  const kind = raw.slice(0, sep) as SessionProgressKind
  if (kind !== 'thinking' && kind !== 'generating' && kind !== 'tool') return null
  return { kind, label: raw.slice(sep + 1) }
}

function assistantTextContent(msg: SessionChatMessage): string {
  if (msg.parts?.length) {
    return msg.parts
      .filter((part) => part.type === 'text')
      .map((part) => part.content)
      .join('')
  }
  return msg.content
}

function normalizeToolName(toolName: string | undefined): string {
  const raw = toolName?.split(':').pop() ?? toolName ?? ''
  return raw.split('__').pop() ?? 'tool'
}

export function deriveSessionProgress(
  messages: SessionChatMessage[],
  assistantMsgId: string | null,
): SessionProgress | null {
  if (!assistantMsgId) return { kind: 'thinking', label: 'Thinking…' }

  const msg = messages.find((m) => m.id === assistantMsgId)
  if (!msg) return { kind: 'thinking', label: 'Thinking…' }

  const parts = msg.parts ?? []
  if (parts.length > 0) {
    const last = parts[parts.length - 1]
    if (last.type === 'tool') {
      const tool = last as ToolCallPart
      if (tool.toolResult === undefined) {
        const name = normalizeToolName(tool.toolName)
        const summary = formatToolCallSummary(tool.toolName, tool.toolArgs)
        if (isHarnessSubagentTool(tool.toolName)) {
          const subLabel = latestSubagentProgressLabel(tool)
          if (subLabel) return { kind: 'tool', label: `Subagent — ${subLabel}` }
        }
        if (summary) return { kind: 'tool', label: `${name} — ${summary}` }
        return { kind: 'tool', label: name }
      }
    }
  }

  if (assistantTextContent(msg).trim()) {
    return { kind: 'generating', label: 'Generating…' }
  }

  return { kind: 'thinking', label: 'Thinking…' }
}
