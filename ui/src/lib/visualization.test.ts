// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { describe, expect, it } from 'vitest'
import {
  isVisualizationTool,
  stripVisualizationTextLeaks,
  visualizationsMatch,
  visualizationFromArgs,
  visualizationFromToolResult,
  visualizationHTMLReady,
  normalizeVisualizationParts,
  shouldRenderVisualization,
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

  it('parses show_visualization args when HTML is complete', () => {
    const html =
      '<svg xmlns="http://www.w3.org/2000/svg" width="120" height="120"><rect width="120" height="120"/></svg>'
    expect(visualizationHTMLReady(html)).toBe(true)
    expect(
      visualizationFromArgs('mcp__loop-viz__show_visualization', {
        html,
        title: 'Chart',
      }),
    ).toMatchObject({ title: 'Chart' })
    expect(
      visualizationFromArgs('mcp__loop-viz__show_visualization', {
        html,
        title: 'Chart',
      })?.html,
    ).toContain(html)
  })

  it('ignores partial visualization HTML during streaming', () => {
    const partial = '<canvas id="chart"></canvas>'
    expect(visualizationHTMLReady(partial)).toBe(false)
    expect(
      visualizationFromArgs('show_visualization', { html: partial, title: '' }),
    ).toBeNull()
  })

  it('rejects empty HTML shell documents', () => {
    const emptyShell =
      '<!DOCTYPE html><html><head><meta charset="utf-8"></head><body></body></html>'
    expect(visualizationHTMLReady(emptyShell)).toBe(false)
  })

  it('drops invalid visualization tool parts on normalize', () => {
    const parts = normalizeVisualizationParts([
      { type: 'text', content: 'Hello' },
      {
        type: 'tool',
        id: 't1',
        toolName: 'show_visualization',
        toolArgs: { html: '<!DOCTYPE html><html><body></body></html>' },
      },
    ])
    expect(parts.filter((p) => p.type === 'tool')).toHaveLength(0)
    expect(parts).toHaveLength(1)
  })

  it('extracts visualization html from structured tool results', () => {
    const viz = visualizationFromToolResult({
      html: '<canvas id="c" width="120" height="120"></canvas><script>new Chart(document.getElementById("c"))</script>',
      title: 'Chart',
    })
    expect(viz?.html).toContain('/vendor/chart.min.js')
    expect(viz?.title).toBe('Chart')
  })

  it('repairs Ollama chart HTML with an unclosed script tag', () => {
    const raw =
      '<script src="https://cdn.jsdelivr.net/npm/chart.js@2.9.4/dist/Chart.min.js"></script>' +
      '<canvas id="myChart" width="400" height="200"></canvas>' +
      "<script>var ctx = document.getElementById('myChart').getContext('2d');new Chart(ctx, {type: 'bar'});"
    const viz = visualizationFromArgs('show_visualization', { html: raw, title: 'Sales' })
    expect(viz?.html).toContain('/vendor/chart.min.js')
    expect(viz?.html).toContain('</script>')
  })

  it('dedupes visualization parts on normalize', () => {
    const html =
      '<canvas id="c" width="120" height="120"></canvas><script>new Chart(document.getElementById("c"))</script>'
    const parts: ToolCallPart[] = [
      {
        type: 'tool',
        id: 'p1',
        toolCallId: 'call_0',
        toolName: 'mcp__loop-viz__show_visualization',
        visualizationHtml: html,
      },
      {
        type: 'tool',
        id: 'p2',
        toolCallId: 'call_0',
        toolName: 'mcp__loop-viz__show_visualization',
        visualizationHtml: html,
      },
    ]
    const normalized = normalizeVisualizationParts(parts)
    const withViz = normalized.filter((p) => p.type === 'tool' && p.visualizationHtml)
    expect(withViz).toHaveLength(1)
  })

  it('renders similar template charts from separate tool calls', () => {
    const html =
      '<canvas id="chart" width="400" height="200"></canvas><script>new Chart(document.getElementById("chart").getContext("2d"),{type:"bar",data:{labels:["A"],datasets:[{data:[1]}]}});</script>'
    const parts: ToolCallPart[] = [
      {
        type: 'tool',
        id: 'p1',
        toolCallId: 'call_0',
        toolName: 'show_visualization',
        visualizationHtml: html,
      },
      {
        type: 'tool',
        id: 'p2',
        toolCallId: 'call_1',
        toolName: 'show_visualization',
        visualizationHtml: html,
      },
    ]
    const normalized = normalizeVisualizationParts(parts)
    const withViz = normalized.filter((p) => p.type === 'tool' && p.visualizationHtml)
    expect(withViz).toHaveLength(2)
    expect(shouldRenderVisualization(parts[0], parts, 0)).toBe(true)
    expect(shouldRenderVisualization(parts[1], parts, 1)).toBe(true)
  })

  it('strips base64 image leaks from text when a visualization is present', () => {
    const b64 = 'A'.repeat(120)
    const chartHtml =
      '<canvas id="c" width="120" height="120"></canvas><script>new Chart(document.getElementById("c"))</script>'
    const parts = normalizeVisualizationParts([
      {
        type: 'tool',
        id: 'viz',
        toolName: 'show_visualization',
        visualizationHtml: chartHtml,
      },
      {
        type: 'text',
        content: `Here is the chart.\n\n![Example Line Chart]\n(data:image/png;base64,${b64})`,
      },
    ])
    const text = parts.find((p) => p.type === 'text')
    expect(text?.type).toBe('text')
    if (text?.type === 'text') {
      expect(text.content).toBe('Here is the chart.')
      expect(text.content).not.toContain('base64')
    }
  })

  it('stripVisualizationTextLeaks removes data URIs', () => {
    const leaked = 'See chart\n(data:image/png;base64,' + 'x'.repeat(100) + ')'
    expect(stripVisualizationTextLeaks(leaked)).toBe('See chart')
  })
})
