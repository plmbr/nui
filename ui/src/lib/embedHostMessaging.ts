// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { isEmbedHost } from '@/lib/embedHost'

const SESSION_TITLE_MESSAGE = 'nuiSessionTitle'
const CLIPBOARD_READ_TIMEOUT_MS = 10_000

function postToEmbedHost(message: Record<string, unknown>): void {
  if (!isEmbedHost() || window.parent === window) return
  window.parent.postMessage(message, '*')
}

const pendingClipboardReads = new Map<
  number,
  { resolve: (text: string) => void; reject: (err: Error) => void }
>()
let nextClipboardReadId = 0

/** Notify an embed host (e.g. VS Code extension tab bar) when a session title changes. */
export function notifyEmbedHostSessionTitle(sessionId: string, title: string): void {
  const trimmed = title.trim()
  if (!sessionId || !trimmed) return
  postToEmbedHost({ type: SESSION_TITLE_MESSAGE, sessionId, title: trimmed })
}

/** Ask the embed host to write text to the system clipboard. */
export function requestEmbedHostClipboardWrite(text: string): void {
  if (!text) return
  postToEmbedHost({ type: 'nuiClipboardWrite', text })
}

/** Read text from the embed host clipboard (VS Code extension bridge). */
export function readEmbedHostClipboard(): Promise<string> {
  return new Promise((resolve, reject) => {
    const requestId = ++nextClipboardReadId
    pendingClipboardReads.set(requestId, { resolve, reject })
    postToEmbedHost({ type: 'nuiClipboardRead', requestId })
    window.setTimeout(() => {
      const pending = pendingClipboardReads.get(requestId)
      if (!pending) return
      pendingClipboardReads.delete(requestId)
      pending.reject(new Error('clipboard read timed out'))
    }, CLIPBOARD_READ_TIMEOUT_MS)
  })
}

export function resolveEmbedHostClipboardRead(requestId: number, text: string): void {
  const pending = pendingClipboardReads.get(requestId)
  if (!pending) return
  pendingClipboardReads.delete(requestId)
  pending.resolve(text)
}

/** Listen for clipboard read responses routed from the embed host. */
export function initEmbedHostClipboardResultListener(): () => void {
  if (!isEmbedHost()) return () => {}

  const handler = (event: MessageEvent) => {
    const msg = event.data as { type?: string; requestId?: number; text?: string } | null
    if (!msg || typeof msg !== 'object') return
    if (msg.type === 'nuiClipboardReadResult' && typeof msg.requestId === 'number') {
      resolveEmbedHostClipboardRead(msg.requestId, msg.text ?? '')
    }
  }

  window.addEventListener('message', handler)
  return () => window.removeEventListener('message', handler)
}

/** Ask the embed host to open a URL in the system browser. */
export function requestEmbedHostOpenExternal(url: string): void {
  if (!url) return
  postToEmbedHost({ type: 'nuiOpenExternal', url })
}
