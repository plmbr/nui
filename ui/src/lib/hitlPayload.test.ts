// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { describe, expect, it } from 'vitest'
import {
  formatHitlApprovalSummary,
  hitlApprovalCommand,
  isRedundantHitlApprovalMessage,
  normalizeHitlQuestionOptions,
} from '@/lib/hitlPayload'

describe('hitlPayload', () => {
  it('normalizes string and object question options', () => {
    expect(normalizeHitlQuestionOptions(['Red', 'Blue'])).toEqual([
      { label: 'Red' },
      { label: 'Blue' },
    ])
    expect(
      normalizeHitlQuestionOptions([{ label: 'Go', description: 'Proceed' }]),
    ).toEqual([{ label: 'Go', description: 'Proceed' }])
  })

  it('formats approval summary from description or tool input', () => {
    expect(
      formatHitlApprovalSummary('Bash', { description: 'Run tests' }),
    ).toBe('Run tests')
    expect(formatHitlApprovalSummary('Bash', { command: 'npm test' })).toContain('npm test')
  })

  it('extracts approval command', () => {
    expect(hitlApprovalCommand({ command: 'echo hi' })).toBe('echo hi')
  })

  it('detects redundant approval messages', () => {
    expect(isRedundantHitlApprovalMessage('', { command: 'ls' })).toBe(true)
    expect(isRedundantHitlApprovalMessage('ls', { command: 'ls' })).toBe(true)
    expect(isRedundantHitlApprovalMessage('Please approve', { command: 'ls' })).toBe(false)
  })
})
