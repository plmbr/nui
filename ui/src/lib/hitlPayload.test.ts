// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { describe, expect, it } from 'vitest'
import {
  formatHitlApprovalSummary,
  hitlApprovalCommand,
  isRedundantHitlApprovalMessage,
  normalizeHitlPayload,
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

  it('normalizes prompt-style questions and message-only payloads', () => {
    expect(
      normalizeHitlPayload({
        questions: [{ prompt: 'What kind of chart?' }],
      }).questions,
    ).toEqual([{ prompt: 'What kind of chart?', question: 'What kind of chart?' }])

    expect(normalizeHitlPayload({ message: 'Pick a color' }).questions).toEqual([
      { question: 'Pick a color' },
    ])
  })

  it('uses header as question text and extracts JSON questions from message', () => {
    expect(
      normalizeHitlPayload({
        questions: [
          {
            header: 'Choose an action',
            options: [{ label: 'Answer a question' }],
          },
        ],
      }).questions,
    ).toEqual([
      {
        header: 'Choose an action',
        question: 'Choose an action',
        options: [{ label: 'Answer a question' }],
      },
    ])

    expect(
      normalizeHitlPayload({
        message:
          'I can help. Which would you like? [{"header":"Choose an action","options":[{"label":"A"}]}]',
      }),
    ).toEqual({
      message: 'I can help. Which would you like?',
      questions: [
        {
          header: 'Choose an action',
          question: 'Choose an action',
          options: [{ label: 'A' }],
        },
      ],
    })
  })
})
