// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import type { AssistantPart, ToolCallPart } from '@/lib/chatMessageUtils'
import { prepareVisualizationHtml, scriptsBalanced } from '@/lib/prepareVisualizationHtml'

const VIZ_TOOL_NAME = 'show_visualization'

export interface VisualizationContent {
  html: string
  title?: string
}

export function isVisualizationTool(toolName?: string): boolean {
  if (!toolName) return false
  return bareToolName(toolName) === VIZ_TOOL_NAME
}

function bareToolName(toolName: string): string {
  return toolName.split('__').pop()?.split(':').pop() ?? toolName
}

function isInlineHTMLTool(toolName?: string): boolean {
  if (!toolName) return false
  switch (bareToolName(toolName)) {
    case 'Write':
    case 'Edit':
    case 'write_file':
    case 'create_file':
      return true
    default:
      return false
  }
}

function looksLikeHTML(content: string): boolean {
  const lower = content.trim().toLowerCase()
  return (
    lower.startsWith('<!doctype') ||
    lower.startsWith('<html') ||
    lower.includes('<canvas') ||
    lower.includes('<svg')
  )
}

export function visualizationHTMLReady(html: string): boolean {
  const trimmed = html.trim()
  if (trimmed.length < 40) return false
  const lower = trimmed.toLowerCase()
  if (lower.includes('<canvas')) {
    if (!lower.includes('</canvas>')) return false
    if (!lower.includes('new chart') && !lower.includes('getcontext(')) return false
    if (lower.includes('<script') && !scriptsBalanced(trimmed)) return false
    return true
  }
  if (lower.includes('<svg')) {
    return lower.includes('</svg>')
  }
  if (lower.includes('<html') || lower.includes('<!doctype')) {
    return lower.includes('</html>')
  }
  return looksLikeHTML(trimmed)
}

function titleFromWriteArgs(args: Record<string, unknown>): string | undefined {
  for (const key of ['file_path', 'filePath', 'path']) {
    const v = args[key]
    if (typeof v !== 'string' || !v.trim()) continue
    let base = v.split(/[/\\]/).pop() ?? v
    const dot = base.lastIndexOf('.')
    if (dot > 0) base = base.slice(0, dot)
    base = base.replace(/[_-]+/g, ' ').trim()
    if (base) return base
  }
  return undefined
}

function normalizeHTML(html: string): string {
  return html.replace(/\s+/g, '').toLowerCase()
}

export function visualizationsMatch(a: string, b: string): boolean {
  const na = normalizeHTML(a)
  const nb = normalizeHTML(b)
  if (!na || !nb) return false
  if (na === nb) return true
  const shorter = na.length <= nb.length ? na : nb
  const longer = na.length <= nb.length ? nb : na
  if (shorter.length < 100) return false
  if (longer.includes(shorter)) return true
  const prefixLen = Math.min(200, shorter.length)
  if (shorter.slice(0, prefixLen) !== longer.slice(0, prefixLen)) return false
  const diff = Math.abs(longer.length - shorter.length)
  return (diff * 100) / longer.length <= 2
}

function parseShowVisualizationArgs(args: Record<string, unknown>): VisualizationContent | null {
  const raw = typeof args.html === 'string' ? args.html.trim() : ''
  if (!raw) return null
  const html = prepareVisualizationHtml(raw)
  if (!visualizationHTMLReady(html)) return null
  const title = typeof args.title === 'string' ? args.title.trim() : undefined
  return { html, title: title || undefined }
}

function parseWriteHTMLArgs(args: Record<string, unknown>): VisualizationContent | null {
  const content = typeof args.content === 'string' ? args.content.trim() : ''
  if (!looksLikeHTML(content)) return null
  return { html: content, title: titleFromWriteArgs(args) }
}

export function extractVisualization(part: {
  toolName?: string
  toolArgs?: Record<string, unknown>
  toolResult?: unknown
  visualizationHtml?: string
  visualizationTitle?: string
}): VisualizationContent | null {
  if (part.visualizationHtml?.trim()) {
    const html = prepareVisualizationHtml(part.visualizationHtml)
    if (!visualizationHTMLReady(html)) return null
    return {
      html,
      title: part.visualizationTitle,
    }
  }
  if (isVisualizationTool(part.toolName)) {
    const fromResult = visualizationFromToolResult(part.toolResult)
    if (fromResult) return fromResult
  }
  const args = part.toolArgs
  if (!args) return null
  if (isVisualizationTool(part.toolName)) {
    return parseShowVisualizationArgs(args)
  }
  if (isInlineHTMLTool(part.toolName)) {
    return parseWriteHTMLArgs(args)
  }
  return null
}

export function shouldHideToolBubble(part: { toolName?: string }): boolean {
  return isVisualizationTool(part.toolName) || isInlineHTMLTool(part.toolName)
}

export function visualizationFromArgs(
  toolName: string | undefined,
  args: Record<string, unknown>,
): VisualizationContent | null {
  if (isVisualizationTool(toolName)) {
    return parseShowVisualizationArgs(args)
  }
  if (isInlineHTMLTool(toolName)) {
    return parseWriteHTMLArgs(args)
  }
  return null
}

export function visualizationFromToolResult(result: unknown): VisualizationContent | null {
  if (!result || typeof result !== 'object' || Array.isArray(result)) return null
  const rec = result as Record<string, unknown>
  if (typeof rec.html !== 'string') return null
  const title = typeof rec.title === 'string' ? rec.title : undefined
  return parseShowVisualizationArgs({ html: rec.html, title })
}

/** Remove embedded chart images models sometimes paste after show_visualization. */
export function stripVisualizationTextLeaks(text: string): string {
  if (!text) return text
  let out = text
  // Markdown image syntax with data URIs.
  out = out.replace(/!\[[^\]]*]\s*\(\s*data:image\/[^)]+\)/gi, '')
  // Parenthesized or bare data URIs (base64 payloads).
  out = out.replace(
    /\(?\s*data:image\/[a-z0-9.+-]+;base64,[a-z0-9+/=\s]{80,}\s*\)?/gi,
    '',
  )
  // Orphan markdown image labels left after stripping the URI.
  out = out.replace(/!\[[^\]]*]\s*(?=\n|$)/g, '')
  return out.replace(/\n{3,}/g, '\n\n').trimEnd()
}

function messageHasVisualization(parts: AssistantPart[]): boolean {
  return parts.some(
    (p) => p.type === 'tool' && isVisualizationTool(p.toolName) && !!extractVisualization(p),
  )
}

/** Enrich and dedupe visualization parts so reloads render each chart once. */
export function normalizeVisualizationParts(parts: AssistantPart[]): AssistantPart[] {
  const enriched = parts.map((part) => {
    if (part.type !== 'tool') return part
    const viz = extractVisualization(part)
    if (!viz) return part
    return {
      ...part,
      visualizationHtml: viz.html,
      visualizationTitle: viz.title,
    }
  })

  const showVizHTML = enriched
    .filter(
      (p): p is ToolCallPart =>
        p.type === 'tool' && isVisualizationTool(p.toolName) && !!p.visualizationHtml,
    )
    .map((p) => p.visualizationHtml!)

  const withoutWriteDupes = enriched.map((part) => {
    if (part.type !== 'tool' || !isInlineHTMLTool(part.toolName) || !part.visualizationHtml) {
      return part
    }
    for (const sv of showVizHTML) {
      if (visualizationsMatch(part.visualizationHtml, sv)) {
        return { ...part, visualizationHtml: undefined, visualizationTitle: undefined }
      }
    }
    return part
  })

  const seenByToolCall = new Map<string, string>()
  const deduped = withoutWriteDupes.map((part) => {
    if (part.type !== 'tool' || !part.visualizationHtml) return part
    if (part.toolCallId) {
      const prev = seenByToolCall.get(part.toolCallId)
      if (prev && visualizationsMatch(part.visualizationHtml, prev)) {
        return { ...part, visualizationHtml: undefined, visualizationTitle: undefined }
      }
      seenByToolCall.set(part.toolCallId, part.visualizationHtml)
      return part
    }
    for (const prev of seenByToolCall.values()) {
      if (visualizationsMatch(part.visualizationHtml, prev)) {
        return { ...part, visualizationHtml: undefined, visualizationTitle: undefined }
      }
    }
    seenByToolCall.set(part.id, part.visualizationHtml)
    return part
  })

  if (!messageHasVisualization(deduped)) {
    return deduped
  }
  return deduped.map((part) => {
    if (part.type !== 'text') return part
    const content = stripVisualizationTextLeaks(part.content)
    return content === part.content ? part : { ...part, content }
  })
}

export function shouldRenderVisualization(
  part: ToolCallPart,
  parts: AssistantPart[],
  index: number,
): boolean {
  const viz = extractVisualization(part)
  if (!viz) return false

  if (isInlineHTMLTool(part.toolName)) {
    for (const p of parts) {
      if (p.type !== 'tool' || p === part) continue
      if (!isVisualizationTool(p.toolName)) continue
      const other = extractVisualization(p)
      if (other && visualizationsMatch(viz.html, other.html)) return false
    }
  }

  // Suppress duplicate segments from the same tool call id (streaming retries), not
  // separate charts that happen to reuse similar Ollama template HTML.
  for (let i = 0; i < index; i++) {
    const p = parts[i]
    if (p.type !== 'tool' || p === part) continue
    if (part.toolCallId && p.toolCallId === part.toolCallId) return false
  }
  return true
}
