// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { ChevronLeft, File, Folder, Puzzle } from 'lucide-react'
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
}

function itemIcon(item: MentionItem) {
  if (item.icon === 'folder') return Folder
  if (item.icon === 'extension') return Puzzle
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
}: Props) {
  if (!open) return null

  const showBack = parent !== ''

  return (
    <div
      id="mention-menu"
      className="mention-menu"
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
      {breadcrumb.length > 1 && (
        <div className="mention-menu__crumb" aria-hidden>
          {breadcrumb.slice(1).map((c) => c.label).join(' / ')}
        </div>
      )}
      {loading && items.length === 0 ? (
        <div className="mention-menu__empty">Loading…</div>
      ) : items.length === 0 ? (
        <div className="mention-menu__empty">No matches</div>
      ) : (
        items.map((item, index) => {
          const Icon = itemIcon(item)
          return (
            <div
              key={`${item.value}-${index}`}
              id={`mention-option-${index}`}
              role="option"
              aria-selected={index === activeIndex}
              className={[
                'mention-menu__item',
                index === activeIndex ? 'mention-menu__item--active' : '',
              ].join(' ')}
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
