// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react'
import { ChevronDown, ChevronUp, Search, X } from 'lucide-react'
import { Input } from '@/components/ui/input'
import {
  clearPageFindHighlights,
  findPageMatches,
  renderPageFindHighlights,
  scrollToPageFindMatch,
  type PageFindMatch,
} from '@/lib/pageFind'

const SHOW_FIND_EVENT = 'nui:show-page-find'
const FIND_NEXT_EVENT = 'nui:find-next-page'
const FIND_PREVIOUS_EVENT = 'nui:find-previous-page'

export function PageFindBar() {
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState('')
  const [matches, setMatches] = useState<PageFindMatch[]>([])
  const [activeIndex, setActiveIndex] = useState(0)
  const [focusToken, setFocusToken] = useState(0)
  const inputRef = useRef<HTMLInputElement>(null)
  const stateRef = useRef({ open, matches, activeIndex })
  stateRef.current = { open, matches, activeIndex }

  const show = useCallback(() => {
    setOpen(true)
    setFocusToken((token) => token + 1)
  }, [])

  const close = useCallback(() => {
    setOpen(false)
    clearPageFindHighlights()
  }, [])

  const navigate = useCallback((direction: 1 | -1) => {
    const current = stateRef.current
    if (!current.open) {
      show()
      return
    }
    if (current.matches.length === 0) return
    const next = (current.activeIndex + direction + current.matches.length) % current.matches.length
    setActiveIndex(next)
    scrollToPageFindMatch(current.matches[next])
  }, [show])

  useLayoutEffect(() => {
    if (!open) return
    inputRef.current?.focus({ preventScroll: true })
    inputRef.current?.select()
  }, [open, focusToken])

  useEffect(() => {
    if (!open) return
    const nextMatches = findPageMatches(document.body, query)
    setMatches(nextMatches)
    setActiveIndex(0)
    renderPageFindHighlights(nextMatches, 0)
    if (query) scrollToPageFindMatch(nextMatches[0])
  }, [open, query])

  useEffect(() => {
    if (!open) return
    renderPageFindHighlights(matches, activeIndex)
  }, [open, matches, activeIndex])

  useEffect(() => {
    if (!open) return
    let frame = 0
    const observer = new MutationObserver((records) => {
      const hasPageMutation = records.some((record) => {
        const target = record.target instanceof Element ? record.target : record.target.parentElement
        return !target?.closest('[data-page-find-ignore]')
      })
      if (!hasPageMutation) return
      cancelAnimationFrame(frame)
      frame = requestAnimationFrame(() => {
        const nextMatches = findPageMatches(document.body, query)
        setMatches(nextMatches)
        setActiveIndex((current) => Math.min(current, Math.max(nextMatches.length - 1, 0)))
      })
    })
    observer.observe(document.body, { childList: true, subtree: true, characterData: true, attributes: true })
    return () => {
      cancelAnimationFrame(frame)
      observer.disconnect()
    }
  }, [open, query])

  useEffect(() => {
    if (!window.__NUI_DESKTOP__) return

    const offShow = window.runtime?.EventsOn?.(SHOW_FIND_EVENT, show)
    const offNext = window.runtime?.EventsOn?.(FIND_NEXT_EVENT, () => navigate(1))
    const offPrevious = window.runtime?.EventsOn?.(FIND_PREVIOUS_EVENT, () => navigate(-1))
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.isComposing || event.altKey || !(event.metaKey || event.ctrlKey)) return
      const key = event.key.toLowerCase()
      if (key === 'f') {
        event.preventDefault()
        show()
      } else if (key === 'g') {
        event.preventDefault()
        navigate(event.shiftKey ? -1 : 1)
      }
    }
    window.addEventListener('keydown', onKeyDown)
    return () => {
      offShow?.()
      offNext?.()
      offPrevious?.()
      window.removeEventListener('keydown', onKeyDown)
      clearPageFindHighlights()
    }
  }, [navigate, show])

  if (!window.__NUI_DESKTOP__ || !open) return null

  return (
    <>
      {/* Kept inline because the CSS optimizer does not yet recognize the standards-based ::highlight pseudo-element. */}
      <style>{`
        ::highlight(nui-page-find) {
          color: inherit;
          background-color: color-mix(in srgb, #facc15 55%, transparent);
        }
        ::highlight(nui-page-find-active) {
          color: inherit;
          background-color: #fb923c;
          text-decoration: underline;
          text-decoration-color: currentColor;
        }
      `}</style>
      <div className="page-find-bar" data-page-find-ignore role="search">
        <Search className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />
        <Input
          ref={inputRef}
          className="page-find-bar__input"
          type="text"
          role="searchbox"
          value={query}
          aria-label="Find in page"
          placeholder="Find"
          onChange={(event) => setQuery(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === 'Escape') {
              event.preventDefault()
              close()
            } else if (event.key === 'Enter') {
              event.preventDefault()
              navigate(event.shiftKey ? -1 : 1)
            }
          }}
        />
        <span className="page-find-bar__count" aria-live="polite">
          {query ? `${matches.length ? activeIndex + 1 : 0} of ${matches.length}` : '0 of 0'}
        </span>
        <button type="button" onClick={() => navigate(-1)} disabled={!matches.length} aria-label="Previous match">
          <ChevronUp className="size-4" />
        </button>
        <button type="button" onClick={() => navigate(1)} disabled={!matches.length} aria-label="Next match">
          <ChevronDown className="size-4" />
        </button>
        <button type="button" onClick={close} aria-label="Close find">
          <X className="size-4" />
        </button>
      </div>
    </>
  )
}
