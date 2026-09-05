// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { describe, expect, it } from 'vitest'
import { mergeCouncilProgress } from '@/lib/councilProgress'
import { apiMessagesToSessionMessages } from '@/lib/chatMessageUtils'
import type { ChatMessage } from '@/types'

describe('mergeCouncilProgress', () => {
  it('accumulates member status and session ids', () => {
    let state = mergeCouncilProgress(undefined, {
      phase: 'round_started',
      round: 'position',
      membersTotal: 2,
    })
    state = mergeCouncilProgress(state, {
      phase: 'member_started',
      memberId: 'a',
      memberLabel: 'Alpha',
      memberSessionId: 'sess-a',
      runId: 'run-a',
    })
    state = mergeCouncilProgress(state, {
      phase: 'member_completed',
      memberId: 'a',
      memberSessionId: 'sess-a',
      elapsedMs: 500,
    })
    expect(state.phase).toBe('member_completed')
    expect(state.members).toEqual([
      {
        id: 'a',
        label: 'Alpha',
        status: 'completed',
        sessionId: 'sess-a',
        runId: 'run-a',
        elapsedMs: 500,
        error: undefined,
      },
    ])
  })
})

describe('apiMessagesToSessionMessages councilProgress', () => {
  it('preserves councilProgress from stored messages', () => {
    const history: ChatMessage[] = [
      {
        id: '1',
        role: 'assistant',
        content: 'done',
        createdAt: '',
        councilProgress: {
          phase: 'complete',
          members: [{ id: 'a', label: 'Alpha', status: 'completed', sessionId: 'sess-a' }],
        },
      },
    ]
    const msgs = apiMessagesToSessionMessages(history)
    expect(msgs[0].councilProgress?.phase).toBe('complete')
    expect(msgs[0].councilProgress?.members[0].sessionId).toBe('sess-a')
  })
})
