// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import type { ReactNode } from 'react'
import { cn } from '@/lib/utils'

interface Props {
  label: string
  /** Distinct accessible name so rows do not collide with agent picker / CTA buttons. */
  ariaLabel: string
  onClick: () => void
  icon: ReactNode
  className?: string
}

export function RecentsListRow({ label, ariaLabel, onClick, icon, className }: Props) {
  return (
    <button
      type="button"
      className={cn('recents-section__row', className)}
      aria-label={ariaLabel}
      onClick={onClick}
    >
      {icon}
      <span className="recents-section__row-title" aria-hidden="true">{label}</span>
    </button>
  )
}
