// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { describe, expect, it } from 'vitest'
import {
  appendTextPart,
  applyAssistantError,
  assistantTextContent,
  dedupeChatMessages,
  updateToolPart,
  type SessionChatMessage,
} from '@/lib/chatMessageUtils'

describe('chatMessageUtils', () => {
  it('appends text to the last text part', () => {
    const parts = appendTextPart([{ type: 'text', content: 'hi' }], ' there')
    expect(parts).toEqual([{ type: 'text', content: 'hi there' }])
    const withTool = appendTextPart(
      [{ type: 'tool', id: 't1', toolCallId: 'tc1' }],
      'next',
    )
    expect(withTool).toHaveLength(2)
    expect(withTool[1]).toEqual({ type: 'text', content: 'next' })
  })

  it('updates tool parts by id', () => {
    const parts = updateToolPart(
      [{ type: 'tool', id: 'p1', toolCallId: 'tc1', toolName: 'Read' }],
      'p1',
      { toolResult: 'ok' },
    )
    expect(parts[0]).toMatchObject({ toolResult: 'ok' })
  })

  it('computes assistant text from parts', () => {
    const msg: SessionChatMessage = {
      id: '1',
      role: 'assistant',
      content: '',
      parts: [
        { type: 'text', content: 'Hello' },
        { type: 'tool', id: 't1' },
        { type: 'text', content: ' world' },
      ],
    }
    expect(assistantTextContent(msg)).toBe('Hello world')
  })

  it('marks assistant errors', () => {
    const msg: SessionChatMessage = { id: '1', role: 'assistant', content: '' }
    const errored = applyAssistantError(msg, 'failed')
    expect(errored.error).toBe(true)
    expect(errored.content).toBe('failed')
  })

  it('dedupes consecutive duplicate messages by id', () => {
    const msgs: SessionChatMessage[] = [
      { id: 'a', role: 'user', content: 'one' },
      { id: 'a', role: 'user', content: 'one' },
      { id: 'b', role: 'assistant', content: 'reply' },
    ]
    const deduped = dedupeChatMessages(msgs)
    expect(deduped).toHaveLength(2)
  })
})
