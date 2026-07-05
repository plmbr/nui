// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { describe, expect, it } from 'vitest'
import {
  isVisualizationTool,
  visualizationsMatch,
  visualizationFromArgs,
  normalizeVisualizationParts,
} from '@/lib/visualization'
import type { ToolCallPart } from '@/lib/chatMessageUtils'

describe('visualization', () => {
  it('detects visualization tool names', () => {
    expect(isVisualizationTool('mcp__loop-viz__show_visualization')).toBe(true)
    expect(isVisualizationTool('Read')).toBe(false)
  })

  it('matches similar HTML content', () => {
    const a = '<canvas width="100"></canvas>'
    const b = '<canvas width="100"></canvas>'
    expect(visualizationsMatch(a, b)).toBe(true)
    expect(visualizationsMatch('<p>x</p>', '<p>y</p>')).toBe(false)
  })

  it('parses show_visualization args', () => {
    expect(
      visualizationFromArgs('mcp__loop-viz__show_visualization', {
        html: '<svg></svg>',
        title: 'Chart',
      }),
    ).toEqual({ html: '<svg></svg>', title: 'Chart' })
  })

  it('dedupes visualization parts on normalize', () => {
    const html = '<canvas>'.repeat(50)
    const parts: ToolCallPart[] = [
      {
        type: 'tool',
        id: 'p1',
        toolName: 'mcp__loop-viz__show_visualization',
        visualizationHtml: html,
      },
      {
        type: 'tool',
        id: 'p2',
        toolName: 'mcp__loop-viz__show_visualization',
        visualizationHtml: html,
      },
    ]
    const normalized = normalizeVisualizationParts(parts)
    const withViz = normalized.filter((p) => p.type === 'tool' && p.visualizationHtml)
    expect(withViz).toHaveLength(1)
  })
})
