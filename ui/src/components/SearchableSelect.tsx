// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useEffect, useId, useLayoutEffect, useMemo, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { Check, ChevronDown, Search } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { filterSearchableItems, normalizeSearchQuery, type SearchableListItem } from '@/lib/searchFilter'
import { cn } from '@/lib/utils'

export type { SearchableListItem }

interface Props {
  value?: string | null
  onValueChange: (value: string) => void
  items: SearchableListItem[]
  placeholder?: string
  searchPlaceholder?: string
  emptyMessage?: string
  className?: string
  triggerClassName?: string
  id?: string
  disabled?: boolean
  /** Show search when item count exceeds this threshold (default 0 = always). */
  searchThreshold?: number
  /** Clear selection display after pick (for add-item pickers). */
  resetOnSelect?: boolean
}

function groupItems(items: SearchableListItem[]): Map<string, SearchableListItem[]> {
  const groups = new Map<string, SearchableListItem[]>()
  for (const item of items) {
    const group = item.group ?? ''
    const list = groups.get(group) ?? []
    list.push(item)
    groups.set(group, list)
  }
  return groups
}

export function SearchableSelect({
  value,
  onValueChange,
  items,
  placeholder = 'Select…',
  searchPlaceholder = 'Search…',
  emptyMessage = 'No matches.',
  className,
  triggerClassName,
  id,
  disabled = false,
  searchThreshold = 0,
  resetOnSelect = false,
}: Props) {
  const listId = useId()
  const triggerRef = useRef<HTMLButtonElement>(null)
  const searchRef = useRef<HTMLInputElement>(null)
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState('')
  const [activeIndex, setActiveIndex] = useState(0)
  const [position, setPosition] = useState({ top: 0, left: 0, width: 0 })
  const [instanceKey, setInstanceKey] = useState(0)

  const selectedItem = useMemo(
    () => items.find((item) => item.id === value) ?? null,
    [items, value],
  )

  const filteredItems = useMemo(() => filterSearchableItems(items, query), [items, query])
  const groupedItems = useMemo(() => groupItems(filteredItems), [filteredItems])
  const flatItems = useMemo(() => filteredItems.filter((item) => !item.disabled), [filteredItems])
  const showSearch = searchThreshold === 0 || items.length > searchThreshold

  const updatePosition = () => {
    const trigger = triggerRef.current
    if (!trigger) return
    const rect = trigger.getBoundingClientRect()
    setPosition({
      top: rect.bottom + 4,
      left: rect.left,
      width: rect.width,
    })
  }

  useLayoutEffect(() => {
    if (!open) return
    updatePosition()
    const onLayoutChange = () => updatePosition()
    window.addEventListener('resize', onLayoutChange)
    window.addEventListener('scroll', onLayoutChange, true)
    return () => {
      window.removeEventListener('resize', onLayoutChange)
      window.removeEventListener('scroll', onLayoutChange, true)
    }
  }, [open])

  useEffect(() => {
    if (!open) return
    setQuery('')
    setActiveIndex(0)
    const frame = window.requestAnimationFrame(() => {
      if (showSearch) {
        searchRef.current?.focus()
      }
    })
    return () => window.cancelAnimationFrame(frame)
  }, [open, showSearch, instanceKey])

  useEffect(() => {
    setActiveIndex(0)
  }, [query, filteredItems.length])

  useEffect(() => {
    if (!open) return
    const onPointerDown = (event: MouseEvent) => {
      const target = event.target as Node
      if (triggerRef.current?.contains(target)) return
      const panel = document.getElementById(listId)
      if (panel?.contains(target)) return
      setOpen(false)
    }
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        event.preventDefault()
        setOpen(false)
        triggerRef.current?.focus()
      }
    }
    document.addEventListener('mousedown', onPointerDown)
    document.addEventListener('keydown', onKeyDown)
    return () => {
      document.removeEventListener('mousedown', onPointerDown)
      document.removeEventListener('keydown', onKeyDown)
    }
  }, [open, listId])

  function selectItem(item: SearchableListItem) {
    if (item.disabled) return
    onValueChange(item.id)
    setOpen(false)
    if (resetOnSelect) {
      setInstanceKey((current) => current + 1)
    }
    triggerRef.current?.focus()
  }

  function handleTriggerKeyDown(event: React.KeyboardEvent<HTMLButtonElement>) {
    if (disabled) return
    if (event.key === 'ArrowDown' || event.key === 'Enter' || event.key === ' ') {
      event.preventDefault()
      setOpen(true)
    }
  }

  function handleSearchKeyDown(event: React.KeyboardEvent<HTMLInputElement>) {
    if (event.key === 'ArrowDown') {
      event.preventDefault()
      if (flatItems.length === 0) return
      setActiveIndex((current) => (current + 1) % flatItems.length)
    } else if (event.key === 'ArrowUp') {
      event.preventDefault()
      if (flatItems.length === 0) return
      setActiveIndex((current) => (current - 1 + flatItems.length) % flatItems.length)
    } else if (event.key === 'Enter') {
      event.preventDefault()
      const item = flatItems[activeIndex]
      if (item) selectItem(item)
    } else if (event.key === 'Tab') {
      setOpen(false)
    }
  }

  const activeItemId = flatItems[activeIndex]?.id

  return (
    <div key={instanceKey} className={cn('relative w-full', className)}>
      <button
        ref={triggerRef}
        id={id}
        type="button"
        disabled={disabled}
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-controls={open ? listId : undefined}
        className={cn(
          'flex h-8 w-full items-center justify-between gap-1.5 rounded-lg border border-input bg-transparent py-2 pr-2 pl-2.5 text-sm transition-colors outline-none select-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50 dark:bg-input/30 dark:hover:bg-input/50',
          !selectedItem && 'text-muted-foreground',
          triggerClassName,
        )}
        onClick={() => {
          if (disabled) return
          setOpen((current) => !current)
        }}
        onKeyDown={handleTriggerKeyDown}
      >
        <span className="min-w-0 flex-1 truncate text-left">
          {selectedItem?.label ?? placeholder}
        </span>
        <ChevronDown className="size-4 shrink-0 text-muted-foreground" aria-hidden />
      </button>

      {open &&
        createPortal(
          <div
            id={listId}
            role="listbox"
            aria-label={placeholder}
            className="fixed z-[100] overflow-hidden rounded-lg border bg-popover text-popover-foreground shadow-md ring-1 ring-foreground/10"
            style={{
              top: position.top,
              left: position.left,
              width: Math.max(position.width, 220),
            }}
          >
            {showSearch && (
              <div className="border-b p-2">
                <div className="relative">
                  <Search
                    className="pointer-events-none absolute top-1/2 left-2 size-3.5 -translate-y-1/2 text-muted-foreground"
                    aria-hidden
                  />
                  <Input
                    ref={searchRef}
                    value={query}
                    onChange={(e) => setQuery(e.target.value)}
                    onKeyDown={handleSearchKeyDown}
                    placeholder={searchPlaceholder}
                    aria-label={searchPlaceholder}
                    className="h-8 pl-7"
                  />
                </div>
              </div>
            )}
            <div className="max-h-64 overflow-y-auto p-1">
              {filteredItems.length === 0 ? (
                <div className="px-2 py-2 text-sm text-muted-foreground">
                  {normalizeSearchQuery(query) ? emptyMessage : 'No options available.'}
                </div>
              ) : (
                [...groupedItems.entries()].map(([group, groupItemsList]) => (
                  <div key={group || '__default__'}>
                    {group && (
                      <div className="px-2 py-1 text-xs text-muted-foreground">{group}</div>
                    )}
                    {groupItemsList.map((item) => {
                      const selectableIndex = flatItems.findIndex((entry) => entry.id === item.id)
                      const isActive = item.id === activeItemId
                      const isSelected = item.id === value
                      return (
                        <button
                          key={item.id}
                          type="button"
                          role="option"
                          aria-selected={isSelected}
                          disabled={item.disabled}
                          className={cn(
                            'relative flex w-full items-start gap-2 rounded-md px-2 py-1.5 text-left text-sm outline-none disabled:pointer-events-none disabled:opacity-50',
                            isActive && 'bg-accent text-accent-foreground',
                            !isActive && !item.disabled && 'hover:bg-accent hover:text-accent-foreground',
                          )}
                          onMouseDown={(event) => event.preventDefault()}
                          onMouseEnter={() => {
                            if (selectableIndex >= 0) setActiveIndex(selectableIndex)
                          }}
                          onClick={() => selectItem(item)}
                        >
                          <span className="min-w-0 flex-1">
                            <span className="block truncate">{item.label}</span>
                            {item.description && (
                              <span className="block truncate text-xs text-muted-foreground">
                                {item.description}
                              </span>
                            )}
                          </span>
                          {isSelected && <Check className="mt-0.5 size-4 shrink-0" aria-hidden />}
                        </button>
                      )
                    })}
                  </div>
                ))
              )}
            </div>
          </div>,
          document.body,
        )}
    </div>
  )
}
