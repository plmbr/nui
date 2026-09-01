// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { describe, expect, it } from 'vitest'
import { render } from '@testing-library/react'
import { HarnessIcon } from '@/components/HarnessIcon'

describe('HarnessIcon', () => {
  it('renders extension harness glyph', () => {
    const { container } = render(<HarnessIcon harness="extension" size="sm" />)
    expect(container.querySelector('svg')).toBeTruthy()
  })

  it('renders api harness glyph', () => {
    const { container } = render(<HarnessIcon harness="api" provider="openai" size="sm" />)
    expect(container.querySelector('img') ?? container.querySelector('svg')).toBeTruthy()
  })

  it('renders brand icon for anthropic api provider', () => {
    const { container } = render(<HarnessIcon harness="api" provider="anthropic" size="md" />)
    expect(container.querySelector('img')).toBeTruthy()
  })

  it('renders wordmark icon for claude-code harness', () => {
    const { container } = render(<HarnessIcon harness="claude-code" size="xl" />)
    const wrap = container.firstElementChild as HTMLElement
    const img = container.querySelector('img')
    expect(img).toBeTruthy()
    expect(wrap.className).toContain('w-28')
    expect(img?.getAttribute('src') ?? '').toContain('claude-code-text')
  })

  it('keeps square icon container for claude-code in list sizes', () => {
    const { container } = render(<HarnessIcon harness="claude-code" size="lg" />)
    const wrap = container.firstElementChild as HTMLElement
    expect(wrap.className).toContain('size-12')
    expect(wrap.className).not.toContain('w-24')
    expect(wrap.className).not.toContain('w-28')
    expect(container.querySelector('img')?.getAttribute('src') ?? '').toContain('claude-code-text')
  })

  it('renders gemini brand icon for api provider', () => {
    const { container } = render(<HarnessIcon harness="api" provider="gemini" size="xl" />)
    const src = container.querySelector('img')?.getAttribute('src') ?? ''
    expect(src).toMatch(/gemini|#3186FF/i)
  })

  it('renders antigravity brand icon with light/dark variants', () => {
    const { container } = render(<HarnessIcon harness="antigravity" size="xl" />)
    const imgs = container.querySelectorAll('img')
    expect(imgs.length).toBe(2)
    expect(imgs[0]?.getAttribute('src') ?? '').toContain('3e4b5d')
    expect(imgs[1]?.getAttribute('src') ?? '').toMatch(/e8eaed/i)
  })
})
