// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useCallback, useEffect, useRef, useState } from 'react'
import { api } from '@/api'
import type { MentionBreadcrumb, MentionItem } from '@/types'

export interface MentionTrigger {
  triggerStart: number
  query: string
}

export function detectMentionTrigger(value: string, cursor: number): MentionTrigger | null {
  const before = value.slice(0, cursor)
  const at = before.lastIndexOf('@')
  if (at < 0) return null
  if (at > 0 && !/\s/.test(before[at - 1] ?? '')) return null
  const query = before.slice(at + 1)
  if (/[\s\n]/.test(query)) return null
  return { triggerStart: at, query }
}

interface UseMentionMenuOptions {
  sessionId: string
  input: string
  setInput: (value: string) => void
  inputRef: React.RefObject<HTMLTextAreaElement | null>
  disabled?: boolean
}

export function useMentionMenu({
  sessionId,
  input,
  setInput,
  inputRef,
  disabled,
}: UseMentionMenuOptions) {
  const [open, setOpen] = useState(false)
  const [parent, setParent] = useState('')
  const [items, setItems] = useState<MentionItem[]>([])
  const [breadcrumb, setBreadcrumb] = useState<MentionBreadcrumb[]>([])
  const [activeIndex, setActiveIndex] = useState(0)
  const [loading, setLoading] = useState(false)
  const triggerRef = useRef<MentionTrigger | null>(null)
  const abortRef = useRef<AbortController | null>(null)

  const close = useCallback(() => {
    setOpen(false)
    setParent('')
    setItems([])
    setBreadcrumb([])
    setActiveIndex(0)
    triggerRef.current = null
    abortRef.current?.abort()
  }, [])

  const syncTrigger = useCallback(() => {
    const el = inputRef.current
    if (!el || disabled) {
      close()
      return
    }
    const trigger = detectMentionTrigger(input, el.selectionStart ?? input.length)
    if (!trigger) {
      close()
      return
    }
    triggerRef.current = trigger
    setOpen(true)
  }, [close, disabled, input, inputRef])

  useEffect(() => {
    syncTrigger()
  }, [input, syncTrigger])

  useEffect(() => {
    if (!open || disabled) return
    const query = triggerRef.current?.query ?? ''
    abortRef.current?.abort()
    const controller = new AbortController()
    abortRef.current = controller
    const timer = window.setTimeout(() => {
      setLoading(true)
      api.mentions
        .list(sessionId, { parent, query }, controller.signal)
        .then(({ items: nextItems, breadcrumb: nextCrumb }) => {
          setItems(nextItems)
          setBreadcrumb(nextCrumb)
          setActiveIndex(0)
        })
        .catch((err: unknown) => {
          if (controller.signal.aborted) return
          console.error('mention list failed', err)
          setItems([])
        })
        .finally(() => {
          if (!controller.signal.aborted) setLoading(false)
        })
    }, 150)
    return () => {
      window.clearTimeout(timer)
      controller.abort()
    }
  }, [disabled, open, parent, sessionId, input])

  const applySelection = useCallback(
    (item: MentionItem) => {
      const trigger = triggerRef.current
      const el = inputRef.current
      if (!trigger || !el) return

      if (item.hasChildren) {
        setParent(item.value)
        setActiveIndex(0)
        return
      }

      const cursor = el.selectionStart ?? input.length
      const before = input.slice(0, trigger.triggerStart)
      const after = input.slice(cursor)
      const insertion = `@${item.value} `
      const next = `${before}${insertion}${after}`
      setInput(next)
      close()
      requestAnimationFrame(() => {
        const pos = before.length + insertion.length
        el.focus()
        el.setSelectionRange(pos, pos)
      })
    },
    [close, input, inputRef, setInput],
  )

  const goBack = useCallback(() => {
    if (breadcrumb.length <= 1) {
      setParent('')
      return
    }
    const prev = breadcrumb[breadcrumb.length - 2]
    setParent(prev?.parent ?? '')
    setActiveIndex(0)
  }, [breadcrumb])

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
      if (!open || items.length === 0) {
        if (open && e.key === 'Escape') {
          e.preventDefault()
          close()
        }
        return false
      }

      switch (e.key) {
        case 'ArrowDown':
          e.preventDefault()
          setActiveIndex((i) => (i + 1) % items.length)
          return true
        case 'ArrowUp':
          e.preventDefault()
          setActiveIndex((i) => (i - 1 + items.length) % items.length)
          return true
        case 'Enter':
        case 'Tab':
          e.preventDefault()
          applySelection(items[activeIndex]!)
          return true
        case 'Escape':
          e.preventDefault()
          close()
          return true
        case 'Backspace':
          if ((triggerRef.current?.query ?? '') === '' && parent !== '') {
            e.preventDefault()
            goBack()
            return true
          }
          return false
        case 'ArrowLeft':
          if ((triggerRef.current?.query ?? '') === '' && parent !== '') {
            e.preventDefault()
            goBack()
            return true
          }
          return false
        default:
          return false
      }
    },
    [activeIndex, applySelection, close, goBack, items, open, parent],
  )

  return {
    open,
    items,
    breadcrumb,
    activeIndex,
    loading,
    parent,
    close,
    applySelection,
    goBack,
    handleKeyDown,
    setActiveIndex,
  }
}
