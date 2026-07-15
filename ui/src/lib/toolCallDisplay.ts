// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import type { AssistantPart, ToolCallPart } from '@/lib/chatMessageUtils'
import {
  extractVisualization,
  shouldHideToolBubble,
  shouldRenderVisualization,
} from '@/lib/visualization'

export interface ToolNameParts {
  integration?: string
  bare: string
  mcpServer?: string
}

export type AssistantRenderSegment =
  | { type: 'text'; key: string; content: string }
  | { type: 'visualization'; key: string; part: ToolCallPart; html: string; title?: string }
  | { type: 'tool-group'; key: string; parts: ToolCallPart[] }

const FRIENDLY_TOOL_NAMES: Record<string, string> = {
  websearch: 'Search the web',
  webfetch: 'Fetch page',
  toolsearch: 'Search tools',
  semanticsearch: 'Search codebase',
  grep: 'Search files',
  glob: 'Find files',
  glob_file_search: 'Find files',
  read: 'Read file',
  write: 'Write file',
  edit: 'Edit file',
  strreplace: 'Edit file',
  delete: 'Delete file',
  list: 'List directory',
  ls: 'List directory',
  bash: 'Run command',
  shell: 'Run command',
  run_terminal_cmd: 'Run command',
  task: 'Run task',
}

function humanizeIdentifier(value: string): string {
  const spaced = value
    .replace(/([a-z0-9])([A-Z])/g, '$1 $2')
    .replace(/[_-]+/g, ' ')
    .trim()
  if (!spaced) return value
  return spaced.charAt(0).toUpperCase() + spaced.slice(1)
}

function formatIntegrationName(raw: string): string {
  const cleaned = raw
    .replace(/^user[-_]/, '')
    .replace(/^plugin[-_]/, '')
    .replace(/[-_]+/g, ' ')
    .trim()
  if (!cleaned) return raw
  return cleaned.replace(/\b\w/g, (char) => char.toUpperCase())
}

export function isMcpToolName(toolName: string | undefined): boolean {
  if (!toolName) return false
  if (toolName.startsWith('mcp__')) return true
  const slash = toolName.indexOf('/')
  return slash > 0 && !toolName.includes(' ')
}

export function parseToolName(toolName: string | undefined): ToolNameParts {
  if (!toolName) return { bare: 'tool' }

  if (toolName.startsWith('mcp__')) {
    const body = toolName.slice('mcp__'.length)
    const splitIdx = body.indexOf('__')
    if (splitIdx >= 0) {
      const server = body.slice(0, splitIdx)
      const bare = body.slice(splitIdx + 2)
      return {
        integration: formatIntegrationName(server),
        bare,
        mcpServer: server,
      }
    }
  }

  if (toolName.includes(':')) {
    const colon = toolName.indexOf(':')
    return {
      integration: formatIntegrationName(toolName.slice(0, colon)),
      bare: toolName.slice(colon + 1),
    }
  }

  if (toolName.includes('/')) {
    const slash = toolName.indexOf('/')
    const server = toolName.slice(0, slash)
    const bare = toolName.slice(slash + 1)
    if (server && bare) {
      return {
        integration: formatIntegrationName(server),
        bare,
        mcpServer: server,
      }
    }
  }

  if (toolName.includes('__')) {
    const parts = toolName.split('__')
    if (parts.length >= 2) {
      return {
        integration: formatIntegrationName(parts[0]),
        bare: parts[parts.length - 1],
      }
    }
  }

  return { bare: toolName.split(':').pop() ?? toolName }
}

export function formatMcpServerLabel(toolName: string | undefined): string | undefined {
  const parsed = parseToolName(toolName)
  if (!isMcpToolName(toolName) || !parsed.integration) return undefined
  return parsed.integration
}

export function formatToolDisplayName(toolName: string | undefined): string {
  const { bare } = parseToolName(toolName)
  const normalized = bare.split('__').pop()?.toLowerCase() ?? bare.toLowerCase()
  return FRIENDLY_TOOL_NAMES[normalized] ?? humanizeIdentifier(bare.split('__').pop() ?? bare)
}

export function buildToolGroupSummary(parts: ToolCallPart[]): string {
  if (parts.length === 0) return 'Tools'

  const integrationLabels = new Set<string>()
  const displayNames = new Set<string>()
  for (const part of parts) {
    const parsed = parseToolName(part.toolName)
    if (parsed.integration) {
      const label = isMcpToolName(part.toolName)
        ? `${parsed.integration} MCP`
        : `${parsed.integration} integration`
      integrationLabels.add(label)
    }
    displayNames.add(formatToolDisplayName(part.toolName))
  }

  if (integrationLabels.size === 1) {
    const integration = [...integrationLabels][0]
    if (parts.length === 1) {
      return `Used ${integration}`
    }
    return `Used ${integration} · ${parts.length} tools`
  }

  if (displayNames.size === 1 && parts.length > 1) {
    return `${[...displayNames][0]} · ${parts.length}×`
  }

  if (displayNames.size <= 3) {
    return [...displayNames].join(', ')
  }

  return `${parts.length} tools`
}

export function isToolGroupRunning(parts: ToolCallPart[]): boolean {
  return parts.some((part) => part.toolResult === undefined)
}

export function segmentAssistantParts(parts: AssistantPart[]): AssistantRenderSegment[] {
  const segments: AssistantRenderSegment[] = []
  let toolBuffer: ToolCallPart[] = []

  const flushTools = () => {
    if (toolBuffer.length === 0) return
    segments.push({
      type: 'tool-group',
      key: `tools-${toolBuffer[0].id}`,
      parts: toolBuffer,
    })
    toolBuffer = []
  }

  for (let index = 0; index < parts.length; index += 1) {
    const part = parts[index]
    if (part.type === 'text') {
      flushTools()
      segments.push({
        type: 'text',
        key: `text-${index}`,
        content: part.content,
      })
      continue
    }

    if (shouldHideToolBubble(part)) {
      if (shouldRenderVisualization(part, parts, index)) {
        flushTools()
        const viz = extractVisualization(part)
        if (viz) {
          segments.push({
            type: 'visualization',
            key: part.id,
            part,
            html: viz.html,
            title: viz.title,
          })
        }
      }
      continue
    }

    toolBuffer.push(part)
  }

  flushTools()
  return segments
}
