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
    expect(prepared).toContain('<canvas id="nui-chart"')
    expect(prepared).toContain('January')
    expect(prepared).toContain('[12,19,3]')
    expect(prepared).toContain('http://localhost:8080/vendor/chart.min.js')
    expect(prepared).toContain('plugins:{legend:{display:false}}')
  })

  it('renders Chart.js v4 dashboard HTML after CDN rewrite to bundled library', async () => {
    const { JSDOM } = await import('jsdom')
    const { readFileSync } = await import('node:fs')
    const { fileURLToPath } = await import('node:url')
    const { dirname, join } = await import('node:path')

    const chartJs = readFileSync(
      join(dirname(fileURLToPath(import.meta.url)), '../../public/vendor/chart.min.js'),
      'utf8',
    )
    const raw =
      '<script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.4/dist/chart.umd.min.js"></script>' +
      '<canvas id="costChart"></canvas>' +
      `<script>
Chart.defaults.color = '#999';
Chart.defaults.borderColor = '#ccc';
Chart.defaults.font.size = 11;
new Chart(document.getElementById('costChart'), {
  type: 'bar',
  data: { labels: ['07-02'], datasets: [{ data: [9.72], backgroundColor: '#4c6ef5' }] },
  options: { plugins: { legend: { display: false } }, scales: { y: { beginAtZero: true } } }
});
</script>`

    const prepared = prepareVisualizationHtml(raw, '/vendor/chart.min.js')
    expect(prepared).not.toContain('cdn.jsdelivr.net')
    expect(prepared).toContain('/vendor/chart.min.js')
    expect(prepared).toContain('Chart.defaults.font.size')
    expect(prepared).toContain('plugins:')

    // Regression: v2 threw here (Chart.defaults.font is undefined), killing all charts.
    const dom = new JSDOM('<!DOCTYPE html><html><body></body></html>', {
      runScripts: 'dangerously',
    })
    const script = dom.window.document.createElement('script')
    script.textContent = chartJs
    dom.window.document.body.appendChild(script)
    await new Promise((resolve) => setTimeout(resolve, 50))

    expect(() => {
      dom.window.eval(`
        Chart.defaults.color = '#999';
        Chart.defaults.borderColor = '#ccc';
        Chart.defaults.font.size = 11;
      `)
    }).not.toThrow()
    expect(dom.window.Chart.defaults.font.size).toBe(11)
  })

  it('injects chart error handler once and before inline chart scripts', () => {
    const raw =
      '<script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.4/dist/chart.umd.min.js"></script>' +
      '<canvas id="c"></canvas>' +
      "<script>new Chart(document.getElementById('c'),{type:'bar'});</script>"
    const once = prepareVisualizationHtml(raw, '/vendor/chart.min.js')
    const twice = prepareVisualizationHtml(once, '/vendor/chart.min.js')
    const handlerCount = (twice.match(/Chart failed:/g) ?? []).length
    expect(handlerCount).toBe(1)
    expect(twice.indexOf("Chart failed: '")).toBeLessThan(twice.search(/new Chart/i))
  })
})
