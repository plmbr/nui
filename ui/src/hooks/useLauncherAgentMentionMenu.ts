// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  detectLauncherMentionTrigger,
  formatLauncherAgentMentionToken,
  listLauncherMentionItems,
} from '@/lib/launcherAgentMentions'
import type { AgentType, MentionBreadcrumb, MentionItem } from '@/types'

interface UseLauncherAgentMentionMenuOptions {
  agentTypes: AgentType[]
  input: string
  setInput: (value: string) => void
  inputRef: React.RefObject<HTMLTextAreaElement | null>
  disabled?: boolean
}

export function useLauncherAgentMentionMenu({
  agentTypes,
  input,
  setInput,
  inputRef,
  disabled,
}: UseLauncherAgentMentionMenuOptions) {
  const [activeIndex, setActiveIndex] = useState(0)
  const [dismissed, setDismissed] = useState(false)

  const trigger = useMemo(() => {
    if (disabled) return null
    const cursor = inputRef.current?.selectionStart ?? input.length
    return detectLauncherMentionTrigger(input, cursor)
  }, [disabled, input, inputRef])

  useEffect(() => {
    // Re-arm suggestions when the @-mention starts fresh (or is cleared).
    setDismissed(false)
  }, [trigger?.triggerStart])

  const open = trigger !== null && !dismissed

  const { items, breadcrumb } = useMemo(
    () => listLauncherMentionItems(agentTypes, trigger?.query ?? ''),
    [agentTypes, trigger?.query],
  )

  useEffect(() => {
    if (!open) return
    setActiveIndex(0)
  }, [open, trigger?.query])

  const applySelection = useCallback(
    (item: MentionItem) => {
      const el = inputRef.current
      if (!trigger || !el || item.hasChildren) return

      const cursor = el.selectionStart ?? input.length
      const before = input.slice(0, trigger.triggerStart)
      const after = input.slice(cursor)
      const insertion = `${formatLauncherAgentMentionToken(item.value, item.label)} `
      const next = `${before}${insertion}${after}`
      setInput(next)
      setDismissed(false)
      setActiveIndex(0)
      requestAnimationFrame(() => {
        const pos = before.length + insertion.length
        el.focus()
        el.setSelectionRange(pos, pos)
      })
    },
    [input, inputRef, setInput, trigger],
  )

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
      if (!open) return false

      if (items.length === 0) {
        if (e.key === 'Escape') {
          e.preventDefault()
          setDismissed(true)
          return true
        }
        return false
      }

      switch (e.key) {
        case 'ArrowDown':
          e.preventDefault()
          setActiveIndex((index) => (index + 1) % items.length)
          return true
        case 'ArrowUp':
          e.preventDefault()
          setActiveIndex((index) => (index - 1 + items.length) % items.length)
          return true
        case 'Enter':
        case 'Tab':
          e.preventDefault()
          applySelection(items[activeIndex]!)
          return true
        case 'Escape':
          e.preventDefault()
          setDismissed(true)
          return true
        default:
          // Space stays in the input so multi-word agent names keep filtering.
          return false
      }
    },
    [activeIndex, applySelection, items, open],
  )

  return {
    open,
    items,
    breadcrumb: breadcrumb as MentionBreadcrumb[],
    activeIndex,
    loading: false,
    applySelection,
    handleKeyDown,
    setActiveIndex,
  }
}
