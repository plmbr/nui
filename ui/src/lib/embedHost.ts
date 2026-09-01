// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

export type EmbedHost = 'vscode'

const EMBED_STORAGE_KEY = 'nui.embedHost'

export function embedHostFromSearch(search = window.location.search): EmbedHost | null {
  const host = new URLSearchParams(search).get('embed')?.trim()
  if (host === 'vscode') {
    try {
      sessionStorage.setItem(EMBED_STORAGE_KEY, host)
    } catch {
      // sessionStorage may be unavailable
    }
    return 'vscode'
  }
  return null
}

export function isEmbedHost(search = window.location.search): boolean {
  if (embedHostFromSearch(search) !== null) return true
  try {
    return sessionStorage.getItem(EMBED_STORAGE_KEY) === 'vscode'
  } catch {
    return false
  }
}

export function embedHostQueryValue(): string | undefined {
  return isEmbedHost() ? 'vscode' : undefined
}
