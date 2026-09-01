// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { isEmbedHost } from '@/lib/embedHost'

const STYLE_ID = 'nui-external-theme'
const APPLIED_VARS: string[] = []

export type ExternalThemeMode = 'light' | 'dark'

interface ExternalThemeMessage {
  type: 'nuiTheme'
  mode?: ExternalThemeMode
  vars?: Record<string, string>
}

export function isExternalThemeActive(): boolean {
  return document.documentElement.dataset.externalTheme === 'vscode'
}

/** Ask the embed host webview shell to (re)send VS Code theme tokens. */
export function requestEmbedHostTheme(): void {
  if (!isEmbedHost() || window.parent === window) return
  window.parent.postMessage({ type: 'nuiThemeRequest' }, '*')
}

/** Listen for theme CSS variables from an embed host (e.g. VS Code extension webview). */
export function initExternalThemeListener(): () => void {
  if (!isEmbedHost()) return () => {}

  const handler = (event: MessageEvent) => {
    const msg = event.data as ExternalThemeMessage | null
    if (!msg || typeof msg !== 'object' || msg.type !== 'nuiTheme') return
    applyExternalTheme(msg.mode, msg.vars ?? {})
  }

  window.addEventListener('message', handler)

  requestEmbedHostTheme()
  const retryTimers = [100, 300, 800, 2000].map((delay) =>
    window.setTimeout(requestEmbedHostTheme, delay),
  )

  return () => {
    window.removeEventListener('message', handler)
    for (const timer of retryTimers) window.clearTimeout(timer)
  }
}

function applyExternalTheme(mode: ExternalThemeMode | undefined, vars: Record<string, string>): void {
  document.documentElement.dataset.externalTheme = 'vscode'

  if (mode === 'dark' || mode === 'light') {
    document.documentElement.classList.toggle('dark', mode === 'dark')
    document.documentElement.style.colorScheme = mode
  }

  const entries = Object.entries(vars).filter(([, value]) => value)
  if (entries.length === 0) return

  // Inline custom properties beat .dark { --token } rules in index.css.
  for (const [key, value] of entries) {
    document.documentElement.style.setProperty(key, value)
    if (!APPLIED_VARS.includes(key)) APPLIED_VARS.push(key)
  }

  const font = vars['--font-sans']
  if (font) {
    document.body.style.fontFamily = font
  }

  applyEmbedFontSize(vars['--embed-font-size'])

  // Keep a stylesheet copy for any late-mounted shadow roots / debugging.
  let style = document.getElementById(STYLE_ID) as HTMLStyleElement | null
  if (!style) {
    style = document.createElement('style')
    style.id = STYLE_ID
    document.head.appendChild(style)
  }
  style.textContent = `html[data-external-theme="vscode"] {\n${entries.map(([key, value]) => `  ${key}: ${value};`).join('\n')}\n}`
}

function applyEmbedFontSize(fontSize: string | undefined): void {
  if (!fontSize) return
  const px = Number.parseFloat(fontSize)
  const adjustedPx = Number.isNaN(px) ? 14 : Math.max(px, 14)
  const adjusted = `${adjustedPx}px`
  document.documentElement.style.fontSize = adjusted
  document.documentElement.style.setProperty('--text-ui', adjusted)
  document.documentElement.style.setProperty('--text-body', adjusted)
  document.documentElement.style.setProperty('--text-code', `${adjustedPx - 1}px`)
  document.documentElement.style.setProperty('--text-meta', `${Math.max(adjustedPx - 2, 11)}px`)
}
