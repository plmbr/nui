// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import type { AgentType, AppConfig, ChatMessage, CreateProjectRequest, Project, Settings } from './types'

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
  return res.json() as Promise<T>
}

export const api = {
  projects: {
    list: (): Promise<Project[]> =>
      request('/projects'),

    get: (id: string): Promise<Project> =>
      request(`/projects/${id}`),

    create: (data: CreateProjectRequest): Promise<Project> =>
      request('/projects', {
        method: 'POST',
        body: JSON.stringify(data),
      }),

    delete: (id: string): Promise<void> =>
      request(`/projects/${id}`, { method: 'DELETE' }),
  },

  agentTypes: {
    list: (): Promise<AgentType[]> =>
      request('/agent-types'),
  },

  messages: {
    list: (projectId: string): Promise<ChatMessage[]> =>
      request(`/projects/${projectId}/messages`),

    save: (projectId: string, messages: ChatMessage[]): Promise<void> =>
      request(`/projects/${projectId}/messages`, {
        method: 'PUT',
        body: JSON.stringify(messages),
      }),
  },

  history: {
    get: (projectId: string): Promise<ChatMessage[]> =>
      request(`/projects/${projectId}/history`),
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
}
