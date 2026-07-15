// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { describe, expect, it } from 'vitest'
import { prepareVisualizationHtml, scriptsBalanced } from '@/lib/prepareVisualizationHtml'

describe('prepareVisualizationHtml', () => {
  it('closes unclosed script tags and wraps fragments', () => {
    const raw =
      '<script src="https://cdn.jsdelivr.net/npm/chart.js@2.9.4/dist/Chart.min.js"></script>' +
      '<canvas id="myChart" width="400" height="200"></canvas>' +
      "<script>var ctx = document.getElementById('myChart').getContext('2d');new Chart(ctx, {type: 'bar'});"
    expect(scriptsBalanced(raw)).toBe(false)
    const prepared = prepareVisualizationHtml(raw, '/vendor/chart.min.js')
    expect(scriptsBalanced(prepared)).toBe(true)
    expect(prepared).toContain('<!DOCTYPE html>')
    expect(prepared).toContain('/vendor/chart.min.js')
    expect(prepared).not.toContain('cdn.jsdelivr.net')
  })

  it('injects Chart.js when the model omits the library script', () => {
    const raw =
      `<canvas id="chart" width="400" height="200"></canvas><script>new Chart(document.getElementById('chart').getContext('2d'),{type:'bar',data:{labels:['A'],datasets:[{data:[1]}]}});</script>`
    const prepared = prepareVisualizationHtml(raw, 'http://localhost:8080/vendor/chart.min.js')
    expect(prepared).toContain('http://localhost:8080/vendor/chart.min.js')
  })

  it('rebuilds broken Ollama chart HTML that uses a div instead of canvas', () => {
    const raw =
      '<script src="https://cdn.jsdelivr.net/npm/chart.js@2.9.4/dist/Chart.min.js"></script>' +
      '<div id="line-chart"></div>' +
      `<script>var ctx = document.getElementById("line-chart").getContext('2d');new Chart(ctx, {type: 'line', data: {labels: ['January', 'February', 'March'], datasets: [{'label': 'Series 1', 'data': [12, 19, 3], 'backgroundColor': 'rgba(255,99,132,1)', 'borderColor': 'rgba(255,99,132,1),0,2)}]}})`
    const prepared = prepareVisualizationHtml(raw, 'http://localhost:8080/vendor/chart.min.js')
    expect(prepared).toContain('<canvas id="loop-chart"')
    expect(prepared).toContain('January')
    expect(prepared).toContain('[12,19,3]')
    expect(prepared).toContain('http://localhost:8080/vendor/chart.min.js')
  })
})
