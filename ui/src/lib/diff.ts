// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

export type DiffLineType = 'add' | 'delete' | 'context' | 'meta'

export interface DiffLine {
  type: DiffLineType
  content: string
  oldLine?: number
  newLine?: number
}

const HUNK_HEADER = /^@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@/

export function parseUnifiedDiff(text: string): DiffLine[] {
  const result: DiffLine[] = []
  let oldLine = 0
  let newLine = 0

  for (const rawLine of text.replace(/\r\n/g, '\n').split('\n')) {
    const hunk = rawLine.match(HUNK_HEADER)
    if (hunk) {
      oldLine = Number.parseInt(hunk[1], 10)
      newLine = Number.parseInt(hunk[2], 10)
      result.push({ type: 'meta', content: rawLine })
      continue
    }

    if (
      rawLine.startsWith('diff ') ||
      rawLine.startsWith('index ') ||
      rawLine.startsWith('--- ') ||
      rawLine.startsWith('+++ ') ||
      rawLine.startsWith('Binary files ')
    ) {
      result.push({ type: 'meta', content: rawLine })
      continue
    }

    if (rawLine.startsWith('+')) {
      result.push({
        type: 'add',
        content: rawLine.slice(1),
        newLine: newLine++,
      })
      continue
    }

    if (rawLine.startsWith('-')) {
      result.push({
        type: 'delete',
        content: rawLine.slice(1),
        oldLine: oldLine++,
      })
      continue
    }

    if (rawLine.startsWith(' ') || rawLine === '') {
      result.push({
        type: 'context',
        content: rawLine.startsWith(' ') ? rawLine.slice(1) : rawLine,
        oldLine: oldLine++,
        newLine: newLine++,
      })
      continue
    }

    result.push({ type: 'meta', content: rawLine })
  }

  return result
}

export function looksLikeDiff(text: string, className?: string): boolean {
  if (className?.includes('language-diff')) return true

  const lines = text.replace(/\r\n/g, '\n').split('\n').filter((line) => line.length > 0)
  if (lines.length === 0) return false

  const diffLike = lines.filter((line) =>
    /^[+\-@ ]/.test(line) ||
    line.startsWith('diff ') ||
    line.startsWith('--- ') ||
    line.startsWith('+++ '),
  ).length

  return diffLike / lines.length >= 0.5
}
