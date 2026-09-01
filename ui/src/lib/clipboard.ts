// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { isEmbedHost } from '@/lib/embedHost'

/** Copy text to the clipboard, with fallbacks for restricted contexts (e.g. VS Code webview iframes). */
export async function copyTextToClipboard(text: string): Promise<boolean> {
  if (!text) return false

  try {
    await navigator.clipboard.writeText(text)
    return true
  } catch {
    // fall through
  }

  try {
    const textarea = document.createElement('textarea')
    textarea.value = text
    textarea.style.position = 'fixed'
    textarea.style.opacity = '0'
    document.body.appendChild(textarea)
    textarea.select()
    const ok = document.execCommand('copy')
    document.body.removeChild(textarea)
    if (ok) return true
  } catch {
    // fall through
  }

  if (isEmbedHost() && window.parent !== window) {
    window.parent.postMessage({ type: 'nuiClipboardWrite', text }, '*')
    return true
  }

  return false
}
