// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { describe, expect, it } from 'vitest'
import { parseOpenSessionCustomValue } from '@/lib/openSession'

describe('parseOpenSessionCustomValue', () => {
  it('parses a valid open_session payload', () => {
    expect(
      parseOpenSessionCustomValue(
        {
          sessionId: 's1',
          prompt: 'fix the flaky test',
          agentType: 'claude-code',
          toolCallId: 'tc-1',
        },
        'source-1',
      ),
    ).toEqual({
      sessionId: 's1',
      prompt: 'fix the flaky test',
      agentType: 'claude-code',
      toolCallId: 'tc-1',
      sourceSessionId: 'source-1',
    })
  })

  it('rejects missing sessionId', () => {
    expect(parseOpenSessionCustomValue({ prompt: 'x' }, 'source-1')).toBeNull()
    expect(parseOpenSessionCustomValue(undefined, 'source-1')).toBeNull()
  })

  it('omits blank optional fields', () => {
    expect(
      parseOpenSessionCustomValue({ sessionId: '  s2  ', prompt: '  ', agentType: '' }, 'src'),
    ).toEqual({
      sessionId: 's2',
      prompt: undefined,
      agentType: undefined,
      toolCallId: undefined,
      sourceSessionId: 'src',
    })
  })
})
