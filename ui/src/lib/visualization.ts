// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import type { AssistantPart, ToolCallPart } from '@/lib/chatMessageUtils'

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
  const html = typeof args.html === 'string' ? args.html.trim() : ''
  if (!html) return null
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
  visualizationHtml?: string
  visualizationTitle?: string
}): VisualizationContent | null {
  if (part.visualizationHtml?.trim()) {
    return {
      html: part.visualizationHtml,
      title: part.visualizationTitle,
    }
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

  const seen: string[] = []
  return withoutWriteDupes.map((part) => {
    if (part.type !== 'tool' || !part.visualizationHtml) return part
    for (const prev of seen) {
      if (visualizationsMatch(part.visualizationHtml, prev)) {
        return { ...part, visualizationHtml: undefined, visualizationTitle: undefined }
      }
    }
    seen.push(part.visualizationHtml)
    return part
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

  for (let i = 0; i < index; i++) {
    const p = parts[i]
    if (p.type !== 'tool') continue
    const other = extractVisualization(p)
    if (other && visualizationsMatch(viz.html, other.html)) return false
  }
  return true
}
