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
})
