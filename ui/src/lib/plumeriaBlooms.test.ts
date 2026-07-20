// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { describe, expect, it } from 'vitest'
import { generateEvenRandomBlooms } from './plumeriaBlooms'

describe('generateEvenRandomBlooms', () => {
  it('returns the requested number of blooms', () => {
    expect(generateEvenRandomBlooms(14)).toHaveLength(14)
  })

  it('keeps blooms inside the panel bounds', () => {
    for (const bloom of generateEvenRandomBlooms(12)) {
      expect(bloom.left).toBeGreaterThanOrEqual(0)
      expect(bloom.left).toBeLessThanOrEqual(100)
      expect(bloom.top).toBeGreaterThanOrEqual(0)
      expect(bloom.top).toBeLessThanOrEqual(100)
    }
  })

  it('produces different layouts across runs', () => {
    const first = generateEvenRandomBlooms(12).map((bloom) => `${bloom.left.toFixed(1)},${bloom.top.toFixed(1)}`)
    let different = false
    for (let i = 0; i < 8; i++) {
      const next = generateEvenRandomBlooms(12).map((bloom) => `${bloom.left.toFixed(1)},${bloom.top.toFixed(1)}`)
      if (next.join('|') !== first.join('|')) {
        different = true
        break
      }
    }
    expect(different).toBe(true)
  })
})
