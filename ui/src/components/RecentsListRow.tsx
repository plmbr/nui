// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import type { ReactNode } from 'react'
import { cn } from '@/lib/utils'

interface Props {
  label: string
  onClick: () => void
  icon: ReactNode
  className?: string
}

export function RecentsListRow({ label, onClick, icon, className }: Props) {
  return (
    <button
      type="button"
      className={cn('recents-section__row', className)}
      onClick={onClick}
    >
      {icon}
      <span className="recents-section__row-title">{label}</span>
    </button>
  )
}
