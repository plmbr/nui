// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { beforeEach, describe, expect, it, vi } from 'vitest'

type SubHandlers = {
  next?: (e: unknown) => void
  complete?: () => void
}

let lastSubscriber: SubHandlers | undefined

vi.mock('@/api', () => ({
  api: {
    messages: {
      list: vi.fn().mockResolvedValue([]),
      save: vi.fn(),
    },
    history: {
      get: vi.fn().mockResolvedValue([]),
    },
    hitl: {
      listPending: vi.fn().mockResolvedValue([]),
      get: vi.fn(),
      respond: vi.fn(),
    },
    sessions: {
      stop: vi.fn().mockResolvedValue(undefined),
      runs: {
        list: vi.fn().mockResolvedValue([]),
      },
    },
  },
}))

vi.mock('@ag-ui/client', () => ({
  HttpAgent: class MockHttpAgent {
    run = vi.fn(() => ({
      subscribe: vi.fn((handlers: SubHandlers) => {
        lastSubscriber = handlers
        return { unsubscribe: vi.fn() }
      }),
    }))
    constructor(_opts: unknown) {}
  },
}))

import { api } from '@/api'
import {
  clearSessionChat,
  ensureSessionChatLoaded,
  getSessionChatSnapshot,
  sendMessage,
  subscribeOpenSession,
} from '@/lib/sessionChatStore'

describe('sessionChatStore', () => {
  beforeEach(() => {
    clearSessionChat('sess-1')
    clearSessionChat('sess-launch')
    lastSubscriber = undefined
    vi.clearAllMocks()
  })

  it('loads messages from API on ensureSessionChatLoaded', async () => {
    vi.mocked(api.messages.list).mockResolvedValue([
      { id: 'm1', role: 'user', content: 'hello', createdAt: '2026-01-01T00:00:00Z' },
    ])
    await ensureSessionChatLoaded('sess-1')
    const snap = getSessionChatSnapshot('sess-1')
    expect(snap.isLoading).toBe(false)
    expect(snap.messages).toHaveLength(1)
    expect(snap.messages[0].content).toBe('hello')
  })

  it('queues user message when sendMessage is called', () => {
    sendMessage('sess-1', 'test message')
    const snap = getSessionChatSnapshot('sess-1')
    expect(snap.messages.some((m) => m.role === 'user' && m.content === 'test message')).toBe(true)
    expect(snap.messages.some((m) => m.role === 'assistant')).toBe(true)
  })

  it('notifies open_session listeners from CUSTOM events', () => {
    const seen: Array<{ sessionId: string; prompt?: string }> = []
    const unsub = subscribeOpenSession((event) => {
      seen.push({ sessionId: event.sessionId, prompt: event.prompt })
    })

    sendMessage('sess-launch', 'delegate this')
    expect(lastSubscriber?.next).toBeTypeOf('function')

    lastSubscriber!.next!({
      type: 'CUSTOM',
      name: 'open_session',
      value: { sessionId: 'target-1', prompt: 'do the work' },
    })

    expect(seen).toEqual([{ sessionId: 'target-1', prompt: 'do the work' }])
    unsub()
  })
})
