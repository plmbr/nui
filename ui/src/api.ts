// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import type { AgentType, AppConfig, Capabilities, ChatMessage, CreateSessionRequest, DirectorySuggestions, Session, Settings } from './types'

const BASE = '/api'

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    headers: { 'Content-Type': 'application/json' },
    ...init,
  })
  if (!res.ok) {
    const text = await res.text()
    throw new Error(text || res.statusText)
  }
  if (res.status === 204) {
    return undefined as T
  }
  return res.json() as Promise<T>
}

export const api = {
  sessions: {
    list: (): Promise<Session[]> =>
      request('/sessions'),

    get: (id: string): Promise<Session> =>
      request(`/sessions/${id}`),

    create: (data: CreateSessionRequest): Promise<Session> =>
      request('/sessions', {
        method: 'POST',
        body: JSON.stringify(data),
      }),

    rename: (id: string, name: string): Promise<Session> =>
      request(`/sessions/${id}`, {
        method: 'PATCH',
        body: JSON.stringify({ name }),
      }),

    delete: (id: string): Promise<void> =>
      request(`/sessions/${id}`, { method: 'DELETE' }),
  },

  agentTypes: {
    list: (): Promise<AgentType[]> =>
      request('/agent-types'),
  },

  directories: {
    suggest: (path: string, signal?: AbortSignal): Promise<DirectorySuggestions> =>
      request(`/directories?path=${encodeURIComponent(path)}`, { signal }),
  },

  messages: {
    list: (sessionId: string): Promise<ChatMessage[]> =>
      request(`/sessions/${sessionId}/messages`),

    save: (sessionId: string, messages: ChatMessage[]): Promise<void> =>
      request(`/sessions/${sessionId}/messages`, {
        method: 'PUT',
        body: JSON.stringify(messages),
      }),
  },

  history: {
    get: (sessionId: string): Promise<ChatMessage[]> =>
      request(`/sessions/${sessionId}/history`),
  },

  config: {
    get: (): Promise<AppConfig> =>
      request('/config'),
  },

  settings: {
    get: (): Promise<Settings> =>
      request('/settings'),

    update: (patch: Partial<Settings>): Promise<Settings> =>
      request('/settings', {
        method: 'PUT',
        body: JSON.stringify(patch),
      }),
  },

  capabilities: {
    get: (): Promise<Capabilities> =>
      request('/capabilities'),
  },
}
