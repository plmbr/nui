// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useCallback, useRef } from 'react'
import { clampSidebarWidth } from '@/lib/sidebarWidth'

interface Props {
  width: number
  onWidthChange: (width: number) => void
  onWidthCommit: (width: number) => void
}

export function SidebarResizeHandle({ width, onWidthChange, onWidthCommit }: Props) {
  const dragRef = useRef<{ startX: number; startWidth: number } | null>(null)
  const latestWidthRef = useRef(width)
  latestWidthRef.current = width

  const handleMouseDown = useCallback((event: React.MouseEvent) => {
    event.preventDefault()
    dragRef.current = { startX: event.clientX, startWidth: width }

    const handleMouseMove = (moveEvent: MouseEvent) => {
      if (!dragRef.current) return
      const delta = moveEvent.clientX - dragRef.current.startX
      const next = clampSidebarWidth(dragRef.current.startWidth + delta)
      latestWidthRef.current = next
      onWidthChange(next)
    }

    const handleMouseUp = () => {
      document.removeEventListener('mousemove', handleMouseMove)
      document.removeEventListener('mouseup', handleMouseUp)
      document.body.style.cursor = ''
      document.body.style.userSelect = ''
      document.body.classList.remove('sidebar-resizing')
      dragRef.current = null
      onWidthCommit(latestWidthRef.current)
    }

    document.body.style.cursor = 'col-resize'
    document.body.style.userSelect = 'none'
    document.body.classList.add('sidebar-resizing')
    document.addEventListener('mousemove', handleMouseMove)
    document.addEventListener('mouseup', handleMouseUp)
  }, [onWidthChange, onWidthCommit, width])

  return (
    <div
      role="separator"
      aria-orientation="vertical"
      aria-label="Resize sidebar"
      className="sidebar-resize-handle"
      onMouseDown={handleMouseDown}
    />
  )
}
