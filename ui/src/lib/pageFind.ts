// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

export const PAGE_FIND_HIGHLIGHT = 'nui-page-find'
export const PAGE_FIND_ACTIVE_HIGHLIGHT = 'nui-page-find-active'

export type PageFindMatch = {
  range: Range
  element: HTMLElement
}

const EXCLUDED_SELECTOR = [
  '[data-page-find-ignore]',
  '[aria-hidden="true"]',
  '[hidden]',
  'script',
  'style',
  'noscript',
  'textarea',
  'select',
  'option',
  '[contenteditable]:not([contenteditable="false"])',
].join(',')

function isSearchableTextNode(node: Text, root: HTMLElement): boolean {
  const element = node.parentElement
  if (!element || !node.data.trim() || !root.contains(element)) return false
  if (element.closest(EXCLUDED_SELECTOR)) return false

  for (let current: HTMLElement | null = element; current; current = current.parentElement) {
    const style = window.getComputedStyle(current)
    if (style.display === 'none' || style.visibility === 'hidden' || style.visibility === 'collapse') {
      return false
    }
    if (current === root) break
  }
  return true
}

export function findPageMatches(root: HTMLElement, query: string): PageFindMatch[] {
  const needle = query.toLocaleLowerCase()
  if (!needle) return []

  const matches: PageFindMatch[] = []
  const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT)
  let current: Node | null

  while ((current = walker.nextNode())) {
    const textNode = current as Text
    if (!isSearchableTextNode(textNode, root)) continue

    const haystack = textNode.data.toLocaleLowerCase()
    let from = 0
    while (from <= haystack.length - needle.length) {
      const index = haystack.indexOf(needle, from)
      if (index === -1) break
      const range = document.createRange()
      range.setStart(textNode, index)
      range.setEnd(textNode, index + needle.length)
      matches.push({ range, element: textNode.parentElement! })
      from = index + Math.max(needle.length, 1)
    }
  }

  return matches
}

function highlightRegistry(): HighlightRegistry | undefined {
  return typeof CSS !== 'undefined' ? CSS.highlights : undefined
}

let pageHighlight: Highlight | undefined
let activeHighlight: Highlight | undefined

export function renderPageFindHighlights(matches: PageFindMatch[], activeIndex: number): void {
  const registry = highlightRegistry()
  if (!registry || typeof Highlight === 'undefined') return

  pageHighlight ??= new Highlight()
  activeHighlight ??= new Highlight()
  pageHighlight.clear()
  activeHighlight.clear()
  for (const match of matches) pageHighlight.add(match.range)
  const active = matches[activeIndex]
  if (active) activeHighlight.add(active.range)
  registry.set(PAGE_FIND_HIGHLIGHT, pageHighlight)
  registry.set(PAGE_FIND_ACTIVE_HIGHLIGHT, activeHighlight)
}

export function clearPageFindHighlights(): void {
  const registry = highlightRegistry()
  pageHighlight?.clear()
  activeHighlight?.clear()
  registry?.delete(PAGE_FIND_HIGHLIGHT)
  registry?.delete(PAGE_FIND_ACTIVE_HIGHLIGHT)
}

export function scrollToPageFindMatch(match: PageFindMatch | undefined): void {
  match?.element.scrollIntoView({ block: 'center', inline: 'nearest', behavior: 'smooth' })
}
