// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

const CHART_JS_SCRIPT_RE =
  /<script\s+[^>]*src=["']https?:\/\/[^"']*chart(\.min)?\.js[^"']*["'][^>]*>\s*<\/script>/gi

const CHART_CANVAS_ID_RE =
  /getElementById\(\s*['"]([^'"]+)['"]\s*\)\s*\.\s*getContext\(\s*['"]2d['"]\s*\)/g

const CHART_LABELS_RE = /['"]?labels['"]?:\s*\[([^\]]+)\]/
const CHART_DATA_RE = /['"]?data['"]?:\s*\[([^\]]+)\]/
const CHART_BROKEN_COLOR_RE = /rgba\([^)]+\),\s*\d/
const DIV_ID_TAG_RE = /<div\s+[^>]*\bid\s*=\s*["']([^"']+)["']/gi

function scriptOpenCount(html: string): number {
  return (html.toLowerCase().match(/<script/g) ?? []).length
}

function scriptCloseCount(html: string): number {
  return (html.toLowerCase().match(/<\/script>/g) ?? []).length
}

export function scriptsBalanced(html: string): boolean {
  const opens = scriptOpenCount(html)
  const closes = scriptCloseCount(html)
  return opens === 0 || opens === closes
}

function repairOllamaEscapedHTML(html: string): string {
  let out = html
  for (let pass = 0; pass < 8; pass++) {
    const prev = out
    out = out
      .replaceAll('\\\\"', '"')
      .replaceAll('\\"', '"')
      .replaceAll('\\[', '[')
      .replaceAll('\\]', ']')
      .replaceAll('\\n', '\n')
      .replaceAll('\\t', '\t')
      .replaceAll(')_.getContext', ').getContext')
      .replaceAll(')_,', '),')
      .replaceAll('")_,', '"),')
      .replaceAll('getContext("2d")_', 'getContext("2d")')
      .replaceAll("getContext('2d')_", "getContext('2d')")
    if (out === prev) break
  }
  return out
}

function looksOllamaCorrupted(html: string): boolean {
  const lower = html.toLowerCase()
  return (
    html.includes('\\\\"') ||
    html.includes('\\"') ||
    html.includes('\\]') ||
    html.includes('\\[') ||
    html.includes(')_.') ||
    lower.includes('id=\\"') ||
    lower.includes('id=\\\\"')
  )
}

function replaceDivWithCanvas(html: string, id: string): string {
  const canvas = `<canvas id="${id}" width="400" height="200"></canvas>`
  const patterns = [
    new RegExp(`<div\\s+[^>]*\\bid\\s*=\\s*"${id}"[^>]*>\\s*</div>`, 'i'),
    new RegExp(`<div\\s+[^>]*\\bid\\s*=\\s*'${id}'[^>]*>\\s*</div>`, 'i'),
  ]
  let out = html
  for (const pattern of patterns) {
    out = out.replace(pattern, canvas)
  }
  return out
}

function ensureChartCanvas(html: string): string {
  if (!/new\s+chart/i.test(html) || /<canvas/i.test(html)) return html
  let out = html
  for (const match of html.matchAll(new RegExp(CHART_CANVAS_ID_RE.source, 'g'))) {
    const id = match[1]
    if (id) out = replaceDivWithCanvas(out, id)
  }
  return out
}

function extractStringList(re: RegExp, html: string): string[] {
  const m = html.match(re)
  if (!m?.[1]) return []
  return m[1]
    .split(',')
    .map((part) => part.trim().replace(/^['"]|['"]$/g, ''))
    .filter(Boolean)
}

function extractNumberList(re: RegExp, html: string): number[] {
  const m = html.match(re)
  if (!m?.[1]) return []
  const out: number[] = []
  for (const part of m[1].split(',')) {
    const n = Number(part.trim())
    if (!Number.isFinite(n)) return []
    out.push(n)
  }
  return out
}

function shouldRebuildChartHTML(html: string): boolean {
  if (!/new\s+chart/i.test(html)) return false
  if (looksOllamaCorrupted(html)) return true
  if (CHART_BROKEN_COLOR_RE.test(html)) return true
  if (/getcontext\(['"]2d['"]\)/i.test(html)) {
    for (const match of html.matchAll(new RegExp(DIV_ID_TAG_RE.source, 'gi'))) {
      const id = match[1]
      if (
        id &&
        (html.includes(`getElementById("${id}")`) || html.includes(`getElementById('${id}')`))
      ) {
        return true
      }
    }
  }
  return false
}

function rebuildChartHTML(html: string, chartScriptSrc: string): string {
  const labels = extractStringList(CHART_LABELS_RE, html)
  const data = extractNumberList(CHART_DATA_RE, html)
  if (!data.length) return ''
  const chartLabels =
    labels.length > 0 ? labels : data.map((_, i) => `Item ${i + 1}`)
  return `<!DOCTYPE html><html><head><meta charset="utf-8"></head><body><canvas id="loop-chart" width="400" height="200"></canvas><script src="${chartScriptSrc}"></script><script>new Chart(document.getElementById('loop-chart').getContext('2d'),{type:'bar',data:{labels:${JSON.stringify(chartLabels)},datasets:[{label:'Series 1',data:${JSON.stringify(data)},backgroundColor:'rgba(54,162,235,0.5)'}]},options:{responsive:false,legend:{display:false}}});</script></body></html>`
}

function ensureChartJSLibrary(html: string, chartScriptSrc: string): string {
  const lower = html.toLowerCase()
  if (!/new\s+chart/i.test(html)) return html
  if (lower.includes('chart.min.js') || lower.includes('chart.js')) return html
  const tag = `<script src="${chartScriptSrc}"></script>`
  const canvasIdx = lower.indexOf('<canvas')
  if (canvasIdx >= 0) return html.slice(0, canvasIdx) + tag + html.slice(canvasIdx)
  const scriptIdx = lower.indexOf('<script')
  if (scriptIdx >= 0) return html.slice(0, scriptIdx) + tag + html.slice(scriptIdx)
  const bodyIdx = lower.indexOf('<body')
  if (bodyIdx >= 0) {
    const close = lower.indexOf('>', bodyIdx)
    if (close >= 0) {
      const insertAt = close + 1
      return html.slice(0, insertAt) + tag + html.slice(insertAt)
    }
  }
  return tag + html
}

function injectChartErrorHandler(html: string): string {
  const handler =
    "<script>window.addEventListener('error',function(e){var p=document.createElement('pre');p.style.cssText='color:#b91c1c;padding:12px;font:13px/1.4 monospace';p.textContent='Chart failed: '+e.message;document.body.appendChild(p);});</script>"
  const lower = html.toLowerCase()
  const idx = lower.lastIndexOf('</body>')
  if (idx >= 0) return html.slice(0, idx) + handler + html.slice(idx)
  return html + handler
}

/** Repair model HTML before rendering inside the visualization iframe. */
export function prepareVisualizationHtml(
  html: string,
  chartScriptSrc = '/vendor/chart.min.js',
): string {
  let out = html.trim()
  if (!out) return out

  out = repairOllamaEscapedHTML(out)

  while (scriptCloseCount(out) < scriptOpenCount(out)) {
    out += '</script>'
  }
  if (shouldRebuildChartHTML(out)) {
    const rebuilt = rebuildChartHTML(out, chartScriptSrc)
    if (rebuilt) out = rebuilt
  } else {
    out = ensureChartCanvas(out)
  }
  out = ensureChartJSLibrary(out, chartScriptSrc)

  const lower = out.toLowerCase()
  if (!lower.includes('<html') && !lower.includes('<!doctype')) {
    out = `<!DOCTYPE html><html><head><meta charset="utf-8"></head><body>${out}</body></html>`
  }

  out = out.replace(
    CHART_JS_SCRIPT_RE,
    `<script src="${chartScriptSrc}"></script>`,
  )
  return injectChartErrorHandler(out)
}
