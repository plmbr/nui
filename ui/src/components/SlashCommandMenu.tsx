// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { Sparkles } from 'lucide-react'
import type { SlashCommandItem } from '@/hooks/useSlashCommandMenu'

interface Props {
  open: boolean
  items: SlashCommandItem[]
  activeIndex: number
  onSelect: (item: SlashCommandItem) => void
  onHover: (index: number) => void
}

export function SlashCommandMenu({
  open,
  items,
  activeIndex,
  onSelect,
  onHover,
}: Props) {
  if (!open) return null

  return (
    <div
      id="slash-command-menu"
      className="mention-menu"
      role="listbox"
      aria-label="Slash commands"
    >
      {items.length === 0 ? (
        <div className="mention-menu__empty">No matches</div>
      ) : (
        items.map((item, index) => (
          <div
            key={`${item.command}-${index}`}
            id={`slash-command-option-${index}`}
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
            <Sparkles className="mention-menu__icon" aria-hidden />
            <span className="mention-menu__label">/{item.label}</span>
          </div>
        ))
      )}
    </div>
  )
}
