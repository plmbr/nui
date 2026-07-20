// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { afterEach, describe, expect, it, vi } from 'vitest'
import { api } from '@/api'

function mockFetch(handler: (url: string, init?: RequestInit) => Response | Promise<Response>) {
  return vi.spyOn(globalThis, 'fetch').mockImplementation((input, init) => {
    const url = typeof input === 'string' ? input : input.url
    return Promise.resolve(handler(url, init))
  })
}

afterEach(() => {
  vi.restoreAllMocks()
})

describe('api request helper', () => {
  it('throws on non-ok responses with body text', async () => {
    mockFetch(() => new Response('bad request', { status: 400, statusText: 'Bad Request' }))
    await expect(api.settings.get()).rejects.toThrow('bad request')
  })

  it('returns undefined for 204 responses', async () => {
    mockFetch(() => new Response(null, { status: 204 }))
    await expect(api.sessions.delete('sess-1')).resolves.toBeUndefined()
  })
})

describe('api sessions', () => {
  it('creates a session via POST', async () => {
    mockFetch((url, init) => {
      expect(url).toBe('/api/sessions')
      expect(init?.method).toBe('POST')
      return new Response(JSON.stringify({ id: 's1', name: 'Test', agentType: 'claude-code', workingDir: '/tmp', createdAt: 'now' }), {
        status: 201,
        headers: { 'Content-Type': 'application/json' },
      })
    })
    const session = await api.sessions.create({ name: 'Test', agentType: 'claude-code', workingDir: '/tmp' })
    expect(session.id).toBe('s1')
  })

  it('lists sessions', async () => {
    mockFetch(() =>
      new Response(JSON.stringify([{ id: 's1', name: 'One', agentType: 'claude-code', workingDir: '/tmp', createdAt: 'now' }]), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
    const sessions = await api.sessions.list()
    expect(sessions).toHaveLength(1)
  })
})

describe('api settings and extensions', () => {
  it('updates settings', async () => {
    mockFetch((url, init) => {
      expect(url).toBe('/api/settings')
      expect(init?.method).toBe('PUT')
      return new Response(JSON.stringify({ theme: 'dark', defaultAgentType: 'anthropic' }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      })
    })
    const settings = await api.settings.update({ theme: 'dark', defaultAgentType: 'anthropic' })
    expect(settings.defaultAgentType).toBe('anthropic')
  })

  it('lists extensions', async () => {
    mockFetch(() =>
      new Response(JSON.stringify([{ name: 'corp-pack', displayName: 'Corp Pack', version: '1.0.0' }]), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
    const extensions = await api.extensions.list()
    expect(extensions[0].name).toBe('corp-pack')
  })

  it('reloads extensions', async () => {
    mockFetch((url, init) => {
      expect(url).toBe('/api/extensions/reload')
      expect(init?.method).toBe('POST')
      return new Response(JSON.stringify({ ok: true }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      })
    })
    const result = await api.extensions.reload()
    expect(result.ok).toBe(true)
  })

  it('fetches capabilities', async () => {
    mockFetch(() =>
      new Response(JSON.stringify({ sandbox: { bwrap: { available: false, error: 'not found' } } }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
    const caps = await api.capabilities.get()
    expect(caps.sandbox.bwrap.available).toBe(false)
  })
})

describe('api agent-types', () => {
  it('loads agent types', async () => {
    mockFetch(() =>
      new Response(
        JSON.stringify([
          { id: 'anthropic', label: 'Anthropic', harness: 'api', available: true, isBuiltin: true },
        ]),
        { status: 200, headers: { 'Content-Type': 'application/json' } },
      ),
    )
    const types = await api.agentTypes.list()
    expect(types[0].id).toBe('anthropic')
  })
})
