// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  clearPageFindHighlights,
  findPageMatches,
  PAGE_FIND_ACTIVE_HIGHLIGHT,
  PAGE_FIND_HIGHLIGHT,
  renderPageFindHighlights,
} from '@/lib/pageFind'

describe('findPageMatches', () => {
  afterEach(() => {
    document.body.replaceChildren()
    vi.unstubAllGlobals()
  })

  it('finds case-insensitive matches in page order', () => {
    document.body.innerHTML = '<main><p>Alpha beta ALPHA</p><button>alpha action</button></main>'
    const root = document.querySelector('main')!

    const matches = findPageMatches(root, 'alpha')

    expect(matches).toHaveLength(3)
    expect(matches.map(({ range }) => range.toString())).toEqual(['Alpha', 'ALPHA', 'alpha'])
  })

  it('ignores hidden, editable, and find UI text', () => {
    document.body.innerHTML = `
      <main>
        <p>visible result</p>
        <div hidden>hidden result</div>
        <div style="display: none">display result</div>
        <div contenteditable="true">editable result</div>
        <div data-page-find-ignore>find bar result</div>
      </main>
    `

    const matches = findPageMatches(document.querySelector('main')!, 'result')

    expect(matches).toHaveLength(1)
    expect(matches[0].range.toString()).toBe('result')
  })

  it('returns no matches for an empty query', () => {
    document.body.innerHTML = '<main>some text</main>'
    expect(findPageMatches(document.querySelector('main')!, '')).toEqual([])
  })

  it('renders and clears normal and active CSS highlights', () => {
    document.body.innerHTML = '<main>result result</main>'
    const matches = findPageMatches(document.querySelector('main')!, 'result')
    const registry = new Map<string, unknown>()
    vi.stubGlobal('CSS', { highlights: registry })
    vi.stubGlobal('Highlight', class extends Set<Range> {
      constructor(...ranges: Range[]) {
        super(ranges)
      }
    })

    renderPageFindHighlights(matches, 1)

    expect(registry.has(PAGE_FIND_HIGHLIGHT)).toBe(true)
    expect(registry.has(PAGE_FIND_ACTIVE_HIGHLIGHT)).toBe(true)
    expect((registry.get(PAGE_FIND_HIGHLIGHT) as Set<Range>).size).toBe(2)

    renderPageFindHighlights(matches.slice(0, 1), 0)
    expect((registry.get(PAGE_FIND_HIGHLIGHT) as Set<Range>).size).toBe(1)
    expect((registry.get(PAGE_FIND_ACTIVE_HIGHLIGHT) as Set<Range>).size).toBe(1)

    clearPageFindHighlights()
    expect(registry.size).toBe(0)
  })
})
