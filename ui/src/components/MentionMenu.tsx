// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { cn } from '@/lib/utils'
import { useEffect, useLayoutEffect, useRef } from 'react'
import { ChevronLeft, File, Folder, Puzzle, Bot } from 'lucide-react'
import type { MentionBreadcrumb, MentionItem } from '@/types'

/** Matches `.mention-menu` max-h-64; keep in sync with index.css. */
const MENTION_MENU_MAX_HEIGHT_PX = 256
/** Matches `.mention-menu` mb-2 gap above the anchor. */
const MENTION_MENU_GAP_PX = 8
const MENTION_MENU_MIN_HEIGHT_PX = 96

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

function headerBottomPx(): number {
  const header = document.querySelector<HTMLElement>('.app-header')
  return header?.getBoundingClientRect().bottom ?? 0
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

  useLayoutEffect(() => {
    if (!open) return
    const menu = menuRef.current
    if (!menu) return

    const fitHeight = () => {
      const anchor = menu.offsetParent
      if (!(anchor instanceof HTMLElement)) return
      // Menu is `bottom-full` above its positioned parent; cap so it stays below app chrome.
      const available =
        anchor.getBoundingClientRect().top - headerBottomPx() - MENTION_MENU_GAP_PX
      const maxHeight = Math.max(
        MENTION_MENU_MIN_HEIGHT_PX,
        Math.min(MENTION_MENU_MAX_HEIGHT_PX, Math.floor(available)),
      )
      menu.style.maxHeight = `${maxHeight}px`
    }

    fitHeight()
    window.addEventListener('resize', fitHeight)
    return () => {
      window.removeEventListener('resize', fitHeight)
      menu.style.maxHeight = ''
    }
  }, [open, items.length])

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
