// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { describe, expect, it } from 'vitest'
import {
  buildToolGroupSummary,
  formatMcpServerLabel,
  formatToolDisplayName,
  isMcpToolName,
  parseToolName,
  segmentAssistantParts,
} from '@/lib/toolCallDisplay'

describe('toolCallDisplay', () => {
  it('parses integration prefixes from tool names', () => {
    expect(parseToolName('user-acme-tasks:create_item')).toEqual({
      integration: 'Acme Tasks',
      bare: 'create_item',
    })
    expect(parseToolName('server__WebSearch')).toEqual({
      integration: 'Server',
      bare: 'WebSearch',
      mcpServer: 'server',
    })
  })

  it('parses MCP server names from mcp__ and server/tool formats', () => {
    expect(parseToolName('mcp__github__list_pull_requests')).toEqual({
      integration: 'Github',
      bare: 'list_pull_requests',
      mcpServer: 'github',
    })
    expect(parseToolName('github/list_pull_requests')).toEqual({
      integration: 'Github',
      bare: 'list_pull_requests',
      mcpServer: 'github',
    })
    expect(isMcpToolName('mcp__github__list_pull_requests')).toBe(true)
    expect(isMcpToolName('github/list_pull_requests')).toBe(true)
    expect(isMcpToolName('Read')).toBe(false)
    expect(formatMcpServerLabel('mcp__github__list_pull_requests')).toBe('Github')
    expect(isMcpToolName('alpha-mcp__ping')).toBe(true)
    expect(parseToolName('alpha-mcp__ping')).toEqual({
      integration: 'Alpha Mcp',
      bare: 'ping',
      mcpServer: 'alpha-mcp',
    })
    expect(formatMcpServerLabel('alpha-mcp__ping')).toBe('Alpha Mcp')
  })

  it('formats friendly tool names', () => {
    expect(formatToolDisplayName('WebSearch')).toBe('Search the web')
    expect(formatToolDisplayName('create_item')).toBe('Create item')
  })

  it('builds group summaries', () => {
    const parts = [
      { type: 'tool' as const, id: '1', toolName: 'user-acme-tasks:create_item' },
      { type: 'tool' as const, id: '2', toolName: 'user-acme-tasks:complete_item' },
    ]
    expect(buildToolGroupSummary(parts)).toBe('Used Acme Tasks integration · 2 tools')
  })

  it('builds MCP group summaries with server names', () => {
    const parts = [
      { type: 'tool' as const, id: '1', toolName: 'mcp__github__list_pull_requests' },
      { type: 'tool' as const, id: '2', toolName: 'mcp__github__get_pull_request' },
    ]
    expect(buildToolGroupSummary(parts)).toBe('Used Github MCP · 2 tools')
    expect(buildToolGroupSummary([parts[0]])).toBe('Used Github MCP')
  })

  it('segments text and consecutive tool calls', () => {
    const segments = segmentAssistantParts([
      { type: 'tool', id: 't1', toolName: 'WebSearch' },
      { type: 'tool', id: 't2', toolName: 'WebFetch' },
      { type: 'text', content: 'Done searching.' },
      { type: 'tool', id: 't3', toolName: 'Read' },
    ])

    expect(segments).toHaveLength(3)
    expect(segments[0]).toMatchObject({ type: 'tool-group', parts: [{ id: 't1' }, { id: 't2' }] })
    expect(segments[1]).toMatchObject({ type: 'text', content: 'Done searching.' })
    expect(segments[2]).toMatchObject({ type: 'tool-group', parts: [{ id: 't3' }] })
  })

  it('routes visualization tools to visualization segments', () => {
    const html =
      '<svg xmlns="http://www.w3.org/2000/svg" width="120" height="120"><rect width="120" height="120"/></svg>'
    const segments = segmentAssistantParts([
      { type: 'tool', id: 't1', toolName: 'WebSearch' },
      {
        type: 'tool',
        id: 't2',
        toolName: 'show_visualization',
        toolArgs: { html },
      },
    ])

    expect(segments).toHaveLength(2)
    expect(segments[0]).toMatchObject({ type: 'tool-group' })
    expect(segments[1]).toMatchObject({ type: 'visualization' })
    if (segments[1].type === 'visualization') {
      expect(segments[1].html).toContain('<svg')
    }
  })
})
