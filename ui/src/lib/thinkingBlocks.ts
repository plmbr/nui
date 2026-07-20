// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

const THINKING_OPEN = '<think>'
const THINKING_CLOSE = '</think>'

export type ParsedContentSegment =
  | { type: 'text'; content: string }
  | { type: 'thinking'; content: string; complete: boolean }

export function segmentThinkingContent(text: string): ParsedContentSegment[] {
  const segments: ParsedContentSegment[] = []
  let remaining = text

  while (remaining.length > 0) {
    const openIdx = remaining.indexOf(THINKING_OPEN)
    if (openIdx === -1) {
      if (remaining) segments.push({ type: 'text', content: remaining })
      break
    }

    if (openIdx > 0) {
      segments.push({ type: 'text', content: remaining.slice(0, openIdx) })
    }

    const afterOpen = remaining.slice(openIdx + THINKING_OPEN.length)
    const closeIdx = afterOpen.indexOf(THINKING_CLOSE)

    if (closeIdx === -1) {
      segments.push({ type: 'thinking', content: afterOpen, complete: false })
      break
    }

    segments.push({
      type: 'thinking',
      content: afterOpen.slice(0, closeIdx),
      complete: true,
    })
    remaining = afterOpen.slice(closeIdx + THINKING_CLOSE.length)
  }

  return segments
}

export function stripThinkingBlocks(text: string): string {
  return segmentThinkingContent(text)
    .filter((segment): segment is Extract<ParsedContentSegment, { type: 'text' }> => segment.type === 'text')
    .map((segment) => segment.content)
    .join('')
}

export function hasThinkingContent(text: string): boolean {
  return segmentThinkingContent(text).some((segment) => segment.type === 'thinking')
}

export function hasIncompleteThinkingBlock(text: string): boolean {
  return segmentThinkingContent(text).some(
    (segment) => segment.type === 'thinking' && !segment.complete,
  )
}
