// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

export interface PlumeriaBloomLayout {
  id: string
  left: number
  top: number
  size: number
  duration: number
  delay: number
  direction: 'cw' | 'ccw'
  initialRotation: number
}

interface GenerateOptions {
  /** Keep the central form area clearer by skipping interior grid cells. */
  avoidCenter?: boolean
}

function shuffle<T>(items: T[]): T[] {
  const next = [...items]
  for (let i = next.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1))
    ;[next[i], next[j]] = [next[j], next[i]]
  }
  return next
}

function isCenterCell(col: number, row: number, cols: number, rows: number): boolean {
  const centerColStart = Math.floor(cols / 2) - 1
  const centerColEnd = Math.ceil(cols / 2)
  const centerRowStart = Math.floor(rows / 2) - 1
  const centerRowEnd = Math.ceil(rows / 2)
  return col >= centerColStart && col < centerColEnd && row >= centerRowStart && row < centerRowEnd
}

/** Place blooms on a shuffled grid with jitter so they feel random but stay spread out. */
export function generateEvenRandomBlooms(count: number, options: GenerateOptions = {}): PlumeriaBloomLayout[] {
  const baseCols = Math.max(3, Math.ceil(Math.sqrt(count * (1.2 + Math.random() * 0.4))))
  const cols = baseCols + (Math.random() > 0.6 ? 1 : 0)
  const rows = Math.max(3, Math.ceil(count / cols) + (Math.random() > 0.7 ? 1 : 0))
  const gridOffsetX = (Math.random() - 0.5) * (100 / cols) * 0.35
  const gridOffsetY = (Math.random() - 0.5) * (100 / rows) * 0.35

  let cells: { col: number; row: number }[] = []
  for (let row = 0; row < rows; row++) {
    for (let col = 0; col < cols; col++) {
      if (options.avoidCenter && isCenterCell(col, row, cols, rows)) continue
      cells.push({ col, row })
    }
  }

  if (cells.length < count) {
    cells = []
    for (let row = 0; row < rows; row++) {
      for (let col = 0; col < cols; col++) {
        cells.push({ col, row })
      }
    }
  }

  const cellW = 100 / cols
  const cellH = 100 / rows

  return shuffle(cells)
    .slice(0, count)
    .map((cell, index) => {
      const left = cell.col * cellW + cellW * (0.06 + Math.random() * 0.88) + gridOffsetX
      const top = cell.row * cellH + cellH * (0.06 + Math.random() * 0.88) + gridOffsetY
      return {
        id: `bloom-${index}-${Math.random().toString(36).slice(2, 7)}`,
        left: Math.min(96, Math.max(4, left)),
        top: Math.min(96, Math.max(4, top)),
        size: Math.round(40 + Math.random() * 44),
        duration: Math.round(22 + Math.random() * 24),
        delay: -Math.round(Math.random() * 28),
        direction: Math.random() > 0.5 ? 'cw' : 'ccw',
        initialRotation: Math.round(Math.random() * 360),
      }
    })
}
