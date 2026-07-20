// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useEffect, useMemo, useRef, useState } from 'react'
import { X } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { filterTagSuggestions } from '@/lib/agentTags'
import { cn } from '@/lib/utils'

interface Props {
  availableTags: string[]
  selectedTags: string[]
  onChange: (tags: string[]) => void
  className?: string
}

export function TagFilterInput({ availableTags, selectedTags, onChange, className }: Props) {
  const [query, setQuery] = useState('')
  const [focused, setFocused] = useState(false)
  const [activeIndex, setActiveIndex] = useState(0)
  const inputRef = useRef<HTMLInputElement>(null)
  const selectedSet = useMemo(() => new Set(selectedTags), [selectedTags])

  const suggestions = useMemo(
    () => filterTagSuggestions(availableTags, selectedSet, query),
    [availableTags, selectedSet, query],
  )

  const listOpen = focused

  useEffect(() => {
    setActiveIndex(0)
  }, [query, suggestions.length])

  function addTag(tag: string) {
    const trimmed = tag.trim()
    if (!trimmed || selectedSet.has(trimmed)) return
    onChange([...selectedTags, trimmed])
    setQuery('')
    setActiveIndex(0)
    setFocused(false)
    inputRef.current?.blur()
  }

  function removeTag(tag: string) {
    onChange(selectedTags.filter((item) => item !== tag))
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === 'Backspace' && query === '' && selectedTags.length > 0) {
      e.preventDefault()
      removeTag(selectedTags[selectedTags.length - 1])
      return
    }
    if (!listOpen) {
      if (e.key === 'ArrowDown' && suggestions.length > 0) {
        e.preventDefault()
        setFocused(true)
      }
      return
    }
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      setActiveIndex((current) => (current + 1) % suggestions.length)
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      setActiveIndex((current) => (current - 1 + suggestions.length) % suggestions.length)
    } else if (e.key === 'Enter' || e.key === 'Tab') {
      if (suggestions.length === 0) return
      e.preventDefault()
      addTag(suggestions[activeIndex])
    } else if (e.key === 'Escape') {
      e.preventDefault()
      setFocused(false)
    }
  }

  return (
    <div className={cn('grid shrink-0 grid-cols-[auto_minmax(0,1fr)] items-center gap-x-2 gap-y-1.5', className)}>
      <Label htmlFor="agent-tag-filter" className="shrink-0 text-muted-foreground">
        Tags
      </Label>
      <div className="relative min-w-0">
        <Input
          ref={inputRef}
          id="agent-tag-filter"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          onFocus={() => setFocused(true)}
          onBlur={() => {
            window.setTimeout(() => setFocused(false), 120)
          }}
          onKeyDown={handleKeyDown}
          placeholder="Filter by tag…"
          role="combobox"
          aria-autocomplete="list"
          aria-expanded={listOpen}
          aria-controls="agent-tag-suggestions"
          aria-activedescendant={listOpen ? `agent-tag-option-${activeIndex}` : undefined}
        />
        {listOpen && (
          <div
            id="agent-tag-suggestions"
            role="listbox"
            aria-label="Tag suggestions"
            className="absolute z-20 mt-1 max-h-44 w-full overflow-y-auto rounded-lg border bg-popover p-1 text-popover-foreground shadow-md"
          >
            {suggestions.length === 0 ? (
              <div className="px-2 py-1.5 text-sm text-muted-foreground">
                {availableTags.length === 0
                  ? 'No tags yet — add tags in Customize or agent YAML.'
                  : 'No matching tags.'}
              </div>
            ) : (
              suggestions.map((tag, index) => (
                <div
                  id={`agent-tag-option-${index}`}
                  key={tag}
                  role="option"
                  aria-selected={index === activeIndex}
                  className={cn(
                    'flex cursor-default items-center rounded-md px-2 py-1.5 text-sm outline-none',
                    index === activeIndex && 'bg-accent text-accent-foreground',
                  )}
                  onMouseDown={(e) => {
                    e.preventDefault()
                    addTag(tag)
                  }}
                  onMouseEnter={() => setActiveIndex(index)}
                >
                  {tag}
                </div>
              ))
            )}
          </div>
        )}
      </div>
      {selectedTags.length > 0 && (
        <>
          <div aria-hidden="true" />
          <div className="flex flex-wrap gap-1.5">
            {selectedTags.map((tag) => (
              <span
                key={tag}
                className="inline-flex items-center gap-1 rounded-md border border-border bg-muted px-2 py-0.5 text-xs font-medium"
              >
                {tag}
                <button
                  type="button"
                  className="rounded-sm text-muted-foreground hover:text-foreground"
                  onClick={() => removeTag(tag)}
                  aria-label={`Remove tag ${tag}`}
                >
                  <X className="size-3" />
                </button>
              </span>
            ))}
          </div>
        </>
      )}
    </div>
  )
}
