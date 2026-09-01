// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { isEmbedHost } from '@/lib/embedHost'
import {
  readEmbedHostClipboard,
  requestEmbedHostClipboardWrite,
} from '@/lib/embedHostMessaging'

function isTextField(el: EventTarget | null): el is HTMLTextAreaElement | HTMLInputElement {
  if (!el || !(el instanceof HTMLElement)) return false
  if (el instanceof HTMLTextAreaElement) return true
  if (el instanceof HTMLInputElement) {
    return !['button', 'submit', 'checkbox', 'radio', 'file', 'reset'].includes(el.type)
  }
  return el.isContentEditable
}

function selectedTextFromInput(el: HTMLInputElement | HTMLTextAreaElement): string {
  const start = el.selectionStart ?? 0
  const end = el.selectionEnd ?? 0
  if (start === end) return ''
  return el.value.slice(start, end)
}

/** Last non-empty document selection — kept because VS Code may clear live selection before the copy bridge runs. */
let lastDocumentSelection = ''

function rememberDocumentSelection(): void {
  const text = window.getSelection()?.toString() ?? ''
  if (text) lastDocumentSelection = text
}

function getDocumentSelectionText(): string {
  const live = window.getSelection()?.toString() ?? ''
  if (live) return live
  return lastDocumentSelection
}

export function insertTextAtField(
  el: HTMLInputElement | HTMLTextAreaElement,
  text: string,
): void {
  const start = el.selectionStart ?? el.value.length
  const end = el.selectionEnd ?? el.value.length
  el.value = el.value.slice(0, start) + text + el.value.slice(end)
  const pos = start + text.length
  el.selectionStart = pos
  el.selectionEnd = pos
  el.dispatchEvent(new Event('input', { bubbles: true }))
}

export function pasteTextIntoFocusedField(text: string): boolean {
  const active = document.activeElement
  if (!isTextField(active)) return false
  if (active instanceof HTMLInputElement || active instanceof HTMLTextAreaElement) {
    insertTextAtField(active, text)
    return true
  }
  return false
}

let lastPasteText = ''
let lastPasteAt = 0

export function pasteTextIntoFocusedFieldDeduped(text: string): boolean {
  const now = Date.now()
  if (text === lastPasteText && now - lastPasteAt < 150) return false
  lastPasteText = text
  lastPasteAt = now
  return pasteTextIntoFocusedField(text)
}

export function copyCurrentSelection(): boolean {
  const active = document.activeElement
  if (active instanceof HTMLInputElement || active instanceof HTMLTextAreaElement) {
    const selected = selectedTextFromInput(active)
    if (selected) {
      requestEmbedHostClipboardWrite(selected)
      return true
    }
  }
  const selection = getDocumentSelectionText()
  if (selection) {
    requestEmbedHostClipboardWrite(selection)
    return true
  }
  return false
}

function copyFromKeyboardEvent(event: KeyboardEvent): boolean {
  if (event.target instanceof HTMLInputElement || event.target instanceof HTMLTextAreaElement) {
    const selected = selectedTextFromInput(event.target)
    if (selected) {
      event.preventDefault()
      requestEmbedHostClipboardWrite(selected)
      return true
    }
  }
  const selection = getDocumentSelectionText()
  if (!selection) return false
  event.preventDefault()
  requestEmbedHostClipboardWrite(selection)
  return true
}

/** Wire copy/paste keyboard shortcuts through the embed host (VS Code webview bridge). */
export function initEmbedClipboardHandlers(): () => void {
  if (!isEmbedHost()) return () => {}

  const onSelectionChange = () => rememberDocumentSelection()

  const onCopy = (event: ClipboardEvent) => {
    if (copyCurrentSelection()) {
      event.preventDefault()
    }
  }

  const onCut = (event: ClipboardEvent) => {
    const active = event.target
    if (active instanceof HTMLInputElement || active instanceof HTMLTextAreaElement) {
      const selected = selectedTextFromInput(active)
      if (!selected) return
      event.preventDefault()
      requestEmbedHostClipboardWrite(selected)
      const start = active.selectionStart ?? 0
      const end = active.selectionEnd ?? 0
      active.value = active.value.slice(0, start) + active.value.slice(end)
      active.selectionStart = active.selectionEnd = start
      active.dispatchEvent(new Event('input', { bubbles: true }))
      return
    }
    if (copyCurrentSelection()) {
      event.preventDefault()
    }
  }

  const onPaste = (event: ClipboardEvent) => {
    if (!isTextField(event.target)) return
    event.preventDefault()
    void readEmbedHostClipboard()
      .then((text) => {
        if (!text) return
        pasteTextIntoFocusedFieldDeduped(text)
      })
      .catch(() => {})
  }

  const onKeyDown = (event: KeyboardEvent) => {
    if (!event.metaKey && !event.ctrlKey) return
    const key = event.key.toLowerCase()

    if (key === 'v') {
      if (!isTextField(event.target)) return
      event.preventDefault()
      void readEmbedHostClipboard()
        .then((text) => {
          if (!text) return
          pasteTextIntoFocusedFieldDeduped(text)
        })
        .catch(() => {})
      return
    }

    if (key === 'c') {
      copyFromKeyboardEvent(event)
      return
    }

    if (key === 'x') {
      if (!(event.target instanceof HTMLInputElement || event.target instanceof HTMLTextAreaElement)) {
        return
      }
      const target = event.target
      const selected = selectedTextFromInput(target)
      if (!selected) return
      event.preventDefault()
      requestEmbedHostClipboardWrite(selected)
      const start = target.selectionStart ?? 0
      const end = target.selectionEnd ?? 0
      target.value = target.value.slice(0, start) + target.value.slice(end)
      target.selectionStart = target.selectionEnd = start
      target.dispatchEvent(new Event('input', { bubbles: true }))
    }
  }

  const onMessage = (event: MessageEvent) => {
    const msg = event.data as { type?: string; text?: string; requestId?: number } | null
    if (!msg || typeof msg !== 'object') return
    if (msg.type === 'nuiClipboardPaste' && typeof msg.text === 'string') {
      pasteTextIntoFocusedFieldDeduped(msg.text)
    } else if (msg.type === 'nuiClipboardCopy') {
      copyCurrentSelection()
    }
  }

  document.addEventListener('selectionchange', onSelectionChange)
  document.addEventListener('copy', onCopy, true)
  document.addEventListener('cut', onCut, true)
  document.addEventListener('paste', onPaste, true)
  document.addEventListener('keydown', onKeyDown, true)
  window.addEventListener('message', onMessage)

  return () => {
    document.removeEventListener('selectionchange', onSelectionChange)
    document.removeEventListener('copy', onCopy, true)
    document.removeEventListener('cut', onCut, true)
    document.removeEventListener('paste', onPaste, true)
    document.removeEventListener('keydown', onKeyDown, true)
    window.removeEventListener('message', onMessage)
  }
}
