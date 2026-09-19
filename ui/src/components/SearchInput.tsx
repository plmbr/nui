// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useLayoutEffect, useRef } from 'react'
import { Search, X } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { cn } from '@/lib/utils'

interface Props {
  value: string
  onChange: (value: string) => void
  placeholder?: string
  id?: string
  className?: string
  inputClassName?: string
  'aria-label'?: string
  onKeyDown?: React.KeyboardEventHandler<HTMLInputElement>
  /** Focus the input when the control mounts or when focusKey changes. */
  autoFocus?: boolean
  /** Remount/activation token; changing it re-focuses when autoFocus is set. */
  focusKey?: string | number
}

export function SearchInput({
  value,
  onChange,
  placeholder = 'Search…',
  id,
  className,
  inputClassName,
  'aria-label': ariaLabel = 'Search',
  onKeyDown,
  autoFocus = false,
  focusKey,
}: Props) {
  const inputRef = useRef<HTMLInputElement>(null)

  useLayoutEffect(() => {
    if (!autoFocus) return

    let cancelled = false
    const focus = () => {
      if (!cancelled) {
        inputRef.current?.focus({ preventScroll: true })
      }
    }

    const frame = requestAnimationFrame(() => {
      requestAnimationFrame(focus)
    })

    return () => {
      cancelled = true
      cancelAnimationFrame(frame)
    }
  }, [autoFocus, focusKey])

  return (
    <div className={cn('relative', className)}>
      <Search
        className="pointer-events-none absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2 text-muted-foreground"
        aria-hidden
      />
      <Input
        ref={inputRef}
        id={id}
        type="text"
        role="searchbox"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        onKeyDown={onKeyDown}
        placeholder={placeholder}
        aria-label={ariaLabel}
        className={cn('h-8 pl-8 pr-8', inputClassName)}
      />
      {value && (
        <button
          type="button"
          className="absolute top-1/2 right-1.5 inline-flex size-6 -translate-y-1/2 items-center justify-center rounded-md text-muted-foreground hover:bg-muted hover:text-foreground"
          onClick={() => onChange('')}
          aria-label="Clear search"
        >
          <X className="size-3.5" />
        </button>
      )}
    </div>
  )
}
