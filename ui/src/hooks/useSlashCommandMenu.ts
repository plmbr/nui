// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useCallback, useEffect, useMemo, useRef, useState } from 'react'

export interface SlashCommandItem {
  command: string
  label: string
}

export interface SlashCommandTrigger {
  triggerStart: number
  query: string
}

export function detectSlashCommandTrigger(value: string, cursor: number): SlashCommandTrigger | null {
  const before = value.slice(0, cursor)
  const slash = before.lastIndexOf('/')
  if (slash < 0) return null
  if (slash > 0 && !/\s/.test(before[slash - 1] ?? '')) return null
  const query = before.slice(slash + 1)
  if (/[\s\n]/.test(query)) return null
  return { triggerStart: slash, query }
}

function filterCommands(commands: SlashCommandItem[], query: string): SlashCommandItem[] {
  if (!query) return commands
  const lower = query.toLowerCase()
  return commands.filter((item) => item.command.toLowerCase().startsWith(lower))
}

function toCommandItems(names: string[]): SlashCommandItem[] {
  return [...names]
    .sort((a, b) => a.localeCompare(b))
    .map((name) => ({ command: name, label: name }))
}

interface UseSlashCommandMenuOptions {
  commands: string[]
  input: string
  setInput: (value: string) => void
  inputRef: React.RefObject<HTMLTextAreaElement | null>
  disabled?: boolean
}

export function useSlashCommandMenu({
  commands,
  input,
  setInput,
  inputRef,
  disabled,
}: UseSlashCommandMenuOptions) {
  const allCommands = useMemo(() => toCommandItems(commands), [commands])
  const [open, setOpen] = useState(false)
  const [items, setItems] = useState<SlashCommandItem[]>([])
  const [activeIndex, setActiveIndex] = useState(0)
  const triggerRef = useRef<SlashCommandTrigger | null>(null)

  const close = useCallback(() => {
    setOpen(false)
    setActiveIndex(0)
    triggerRef.current = null
  }, [])

  const syncTrigger = useCallback(() => {
    const el = inputRef.current
    if (!el || disabled || allCommands.length === 0) {
      close()
      return
    }
    const trigger = detectSlashCommandTrigger(input, el.selectionStart ?? input.length)
    if (!trigger) {
      close()
      return
    }
    triggerRef.current = trigger
    setOpen(true)
  }, [allCommands.length, close, disabled, input, inputRef])

  useEffect(() => {
    syncTrigger()
  }, [input, syncTrigger])

  useEffect(() => {
    if (!open || disabled) return
    const query = triggerRef.current?.query ?? ''
    setItems(filterCommands(allCommands, query))
    setActiveIndex(0)
  }, [allCommands, disabled, input, open])

  const applySelection = useCallback(
    (item: SlashCommandItem) => {
      const trigger = triggerRef.current
      const el = inputRef.current
      if (!trigger || !el) return

      const cursor = el.selectionStart ?? input.length
      const before = input.slice(0, trigger.triggerStart)
      const after = input.slice(cursor)
      const insertion = `/${item.command} `
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

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
      if (!open) return false

      if (items.length === 0) {
        if (e.key === 'Escape') {
          e.preventDefault()
          close()
          return true
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
        default:
          return false
      }
    },
    [activeIndex, applySelection, close, items, open],
  )

  return {
    open,
    items,
    activeIndex,
    close,
    applySelection,
    handleKeyDown,
    setActiveIndex,
  }
}
