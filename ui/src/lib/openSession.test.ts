// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { describe, expect, it } from 'vitest'
import {
  isLaunchSessionToolName,
  parseLaunchSessionToolResult,
  parseOpenSessionCustomValue,
  unwrapToolResultJSON,
} from '@/lib/openSession'

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

describe('isLaunchSessionToolName', () => {
  it('matches bare and qualified names', () => {
    expect(isLaunchSessionToolName('launch_session')).toBe(true)
    expect(isLaunchSessionToolName('nui-orchestrator__launch_session')).toBe(true)
    expect(isLaunchSessionToolName('mcp__nui-orchestrator__launch_session')).toBe(true)
    expect(isLaunchSessionToolName('list_agents')).toBe(false)
    expect(isLaunchSessionToolName(undefined)).toBe(false)
  })
})

describe('parseLaunchSessionToolResult', () => {
  it('parses bare JSON payloads', () => {
    expect(
      parseLaunchSessionToolResult(
        JSON.stringify({
          session: { id: 's1', agentType: 'claude-code' },
          prompt: 'do it',
        }),
        'src',
        'tc-1',
      ),
    ).toEqual({
      sessionId: 's1',
      prompt: 'do it',
      agentType: 'claude-code',
      toolCallId: 'tc-1',
      sourceSessionId: 'src',
    })
  })

  it('parses Claude Code content-block arrays', () => {
    const wrapped = JSON.stringify([
      {
        type: 'text',
        text: JSON.stringify({
          prompt: 'summarize the repo',
          session: {
            id: 'target-sess-1',
            agentType: 'claude-code',
          },
        }),
      },
    ])
    expect(parseLaunchSessionToolResult(wrapped, 'orch', 'tc-launch')).toEqual({
      sessionId: 'target-sess-1',
      prompt: 'summarize the repo',
      agentType: 'claude-code',
      toolCallId: 'tc-launch',
      sourceSessionId: 'orch',
    })
  })

  it('rejects errors and invalid payloads', () => {
    expect(parseLaunchSessionToolResult('error: nope', 'src')).toBeNull()
    expect(parseLaunchSessionToolResult('{"session":{}}', 'src')).toBeNull()
    expect(parseLaunchSessionToolResult(undefined, 'src')).toBeNull()
  })
})

describe('unwrapToolResultJSON', () => {
  it('leaves object JSON unchanged', () => {
    expect(unwrapToolResultJSON('{"a":1}')).toBe('{"a":1}')
  })
})
