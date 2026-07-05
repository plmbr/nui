// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { beforeEach, describe, expect, it, vi } from 'vitest'

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
      subscribe: vi.fn(
        ({ complete }: { next?: (e: unknown) => void; complete?: () => void }) => {
          complete?.()
          return { unsubscribe: vi.fn() }
        },
      ),
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
} from '@/lib/sessionChatStore'

describe('sessionChatStore', () => {
  beforeEach(() => {
    clearSessionChat('sess-1')
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
})
