// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useCallback, useEffect, useState, type ReactNode, type RefObject } from 'react'
import { createPortal } from 'react-dom'
import { copyTextToClipboard } from '@/lib/clipboard'

type MenuState = {
  x: number
  y: number
  hasSelection: boolean
}

function selectedText(): string {
  return window.getSelection()?.toString() ?? ''
}

function isEditableTarget(target: EventTarget | null): boolean {
  if (!(target instanceof Element)) return false
  if (target.closest('textarea, input, [contenteditable="true"]')) return true
  return false
}

/** Desktop-only Select All / Copy menu for selectable chat (and similar) regions. */
export function TextContextMenu({
  containerRef,
  children,
}: {
  containerRef: RefObject<HTMLElement | null>
  children?: ReactNode
}) {
  const desktop = typeof window !== 'undefined' && !!window.__NUI_DESKTOP__
  const [menu, setMenu] = useState<MenuState | null>(null)

  const close = useCallback(() => setMenu(null), [])

  useEffect(() => {
    if (!desktop) return
    const el = containerRef.current
    if (!el) return

    const onContextMenu = (e: MouseEvent) => {
      if (isEditableTarget(e.target)) return
      e.preventDefault()
      e.stopPropagation()
      setMenu({
        x: e.clientX,
        y: e.clientY,
        hasSelection: selectedText().trim().length > 0,
      })
    }

    el.addEventListener('contextmenu', onContextMenu)
    return () => el.removeEventListener('contextmenu', onContextMenu)
  }, [containerRef, desktop])

  useEffect(() => {
    if (!menu) return
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') close()
    }
    const onPointer = (e: MouseEvent) => {
      const t = e.target
      if (t instanceof Element && t.closest('[data-nui-text-context-menu]')) return
      close()
    }
    window.addEventListener('keydown', onKey)
    window.addEventListener('mousedown', onPointer, true)
    window.addEventListener('scroll', close, true)
    return () => {
      window.removeEventListener('keydown', onKey)
      window.removeEventListener('mousedown', onPointer, true)
      window.removeEventListener('scroll', close, true)
    }
  }, [menu, close])

  const onCopy = async () => {
    const text = selectedText()
    if (text) await copyTextToClipboard(text)
    close()
  }

  const onSelectAll = () => {
    const el = containerRef.current
    if (!el) {
      close()
      return
    }
    const selection = window.getSelection()
    if (!selection) {
      close()
      return
    }
    const range = document.createRange()
    range.selectNodeContents(el)
    selection.removeAllRanges()
    selection.addRange(range)
    close()
  }

  return (
    <>
      {children}
      {menu &&
        createPortal(
          <div
            data-nui-text-context-menu
            role="menu"
            className="nui-text-context-menu"
            style={{ left: menu.x, top: menu.y }}
          >
            <button
              type="button"
              role="menuitem"
              className="nui-text-context-menu__item"
              onClick={onSelectAll}
            >
              Select All
            </button>
            <button
              type="button"
              role="menuitem"
              className="nui-text-context-menu__item"
              disabled={!menu.hasSelection}
              onClick={() => void onCopy()}
            >
              Copy
            </button>
          </div>,
          document.body,
        )}
    </>
  )
}
