// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { describe, expect, it } from 'vitest'
import {
  buildToolGroupSummary,
  formatToolDisplayName,
  parseToolName,
  segmentAssistantParts,
} from '@/lib/toolCallDisplay'

describe('toolCallDisplay', () => {
  it('parses integration prefixes from tool names', () => {
    expect(parseToolName('user-data-analytics:run_sql_query')).toEqual({
      integration: 'Data Analytics',
      bare: 'run_sql_query',
    })
    expect(parseToolName('server__WebSearch')).toEqual({
      integration: 'Server',
      bare: 'WebSearch',
    })
  })

  it('formats friendly tool names', () => {
    expect(formatToolDisplayName('WebSearch')).toBe('Search the web')
    expect(formatToolDisplayName('run_sql_query')).toBe('Run sql query')
  })

  it('builds group summaries', () => {
    const parts = [
      { type: 'tool' as const, id: '1', toolName: 'user-data-analytics:run_sql_query' },
      { type: 'tool' as const, id: '2', toolName: 'user-data-analytics:get_results' },
    ]
    expect(buildToolGroupSummary(parts)).toBe('Used Data Analytics integration · 2 tools')
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
