// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { cn } from '@/lib/utils'
import { useEffect, useRef } from 'react'
import { ChevronLeft, File, Folder, Puzzle, Bot } from 'lucide-react'
import type { MentionBreadcrumb, MentionItem } from '@/types'

interface Props {
  open: boolean
  items: MentionItem[]
  breadcrumb: MentionBreadcrumb[]
  activeIndex: number
  loading: boolean
  parent: string
  onSelect: (item: MentionItem) => void
  onBack: () => void
  onHover: (index: number) => void
  header?: string
  className?: string
}

function itemIcon(item: MentionItem) {
  if (item.icon === 'folder') return Folder
  if (item.icon === 'extension') return Puzzle
  if (item.icon === 'agent') return Bot
  return File
}

export function MentionMenu({
  open,
  items,
  breadcrumb,
  activeIndex,
  loading,
  parent,
  onSelect,
  onBack,
  onHover,
  header,
  className,
}: Props) {
  const menuRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    const active = menuRef.current?.querySelector<HTMLElement>('.mention-menu__item--active')
    active?.scrollIntoView({ block: 'nearest' })
  }, [activeIndex, open])

  if (!open) return null

  const showBack = parent !== ''

  return (
    <div
      ref={menuRef}
      id="mention-menu"
      className={cn('mention-menu', className)}
      role="listbox"
      aria-label="Mention suggestions"
      aria-busy={loading}
    >
      {showBack && (
        <button type="button" className="mention-menu__back" onMouseDown={(e) => {
          e.preventDefault()
          onBack()
        }}
        >
          <ChevronLeft className="size-4" aria-hidden />
          Back
        </button>
      )}
      {header ? (
        <div className="mention-menu__header" aria-hidden>
          {header}
        </div>
      ) : breadcrumb.length > 1 ? (
        <div className="mention-menu__crumb" aria-hidden>
          {breadcrumb.slice(1).map((c) => c.label).join(' / ')}
        </div>
      ) : null}
      {loading && items.length === 0 ? (
        <div className="mention-menu__empty">Loading…</div>
      ) : items.length === 0 ? (
        <div className="mention-menu__empty">No matches</div>
      ) : (
        items.map((item, index) => {
          const Icon = itemIcon(item)
          const active = index === activeIndex
          return (
            <div
              key={`${item.value}-${index}`}
              id={`mention-option-${index}`}
              role="option"
              aria-selected={active}
              className={cn(
                'mention-menu__item',
                active && 'mention-menu__item--active',
              )}
              onMouseDown={(e) => {
                e.preventDefault()
                onSelect(item)
              }}
              onMouseEnter={() => onHover(index)}
            >
              <Icon className="mention-menu__icon" aria-hidden />
              <span className="mention-menu__label">{item.label}</span>
              {item.hasChildren && <span className="mention-menu__hint">›</span>}
            </div>
          )
        })
      )}
    </div>
  )
}
