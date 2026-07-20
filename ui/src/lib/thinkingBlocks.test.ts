// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { describe, expect, it } from 'vitest'
import {
  hasIncompleteThinkingBlock,
  hasThinkingContent,
  segmentThinkingContent,
  stripThinkingBlocks,
} from '@/lib/thinkingBlocks'

describe('thinkingBlocks', () => {
  it('segments plain text without thinking blocks', () => {
    expect(segmentThinkingContent('Hello world')).toEqual([
      { type: 'text', content: 'Hello world' },
    ])
  })

  it('segments a complete thinking block', () => {
    expect(
      segmentThinkingContent('<think>step one</think>Answer'),
    ).toEqual([
      { type: 'thinking', content: 'step one', complete: true },
      { type: 'text', content: 'Answer' },
    ])
  })

  it('segments text before and after thinking', () => {
    expect(
      segmentThinkingContent('Intro <think>reasoning</think> Outro'),
    ).toEqual([
      { type: 'text', content: 'Intro ' },
      { type: 'thinking', content: 'reasoning', complete: true },
      { type: 'text', content: ' Outro' },
    ])
  })

  it('handles an incomplete thinking block while streaming', () => {
    expect(segmentThinkingContent('<think>partial')).toEqual([
      { type: 'thinking', content: 'partial', complete: false },
    ])
    expect(hasIncompleteThinkingBlock('<think>partial')).toBe(true)
  })

  it('strips thinking blocks from visible text', () => {
    expect(
      stripThinkingBlocks('<think>hidden</think>visible'),
    ).toBe('visible')
    expect(hasThinkingContent('<think>hidden</think>')).toBe(true)
  })
})
