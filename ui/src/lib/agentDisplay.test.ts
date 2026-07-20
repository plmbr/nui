// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { describe, expect, it } from 'vitest'
import { harnessLabel } from '@/lib/agentDisplay'

describe('harnessLabel', () => {
  it('labels extension harness', () => {
    expect(harnessLabel('extension')).toBe('extension harness')
  })

  it('appends docker sandbox suffix', () => {
    expect(harnessLabel('claude-code', 'docker')).toBe('claude-code · docker')
  })

  it('appends bubblewrap suffix', () => {
    expect(harnessLabel('pi', 'bubblewrap')).toBe('pi · bwrap')
  })
})
