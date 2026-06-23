// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import type { AgentType, AgentFileContent, AgentFileInfo, Bootstrap, Capabilities, ChatMessage, CreateSessionRequest, DirectorySuggestions, ExtensionInfo, MCPServer, Session, Settings, SkillEntry } from './types'

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

    ensureDefault: (): Promise<Session> =>
      request('/sessions/ensure-default', { method: 'POST' }),

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

  settings: {
    get: (): Promise<Settings> =>
      request('/settings'),

    update: (patch: Partial<Settings>): Promise<Settings> =>
      request('/settings', {
        method: 'PUT',
        body: JSON.stringify(patch),
      }),
  },

  bootstrap: {
    get: (): Promise<Bootstrap> =>
      request('/bootstrap'),
  },

  capabilities: {
    get: (): Promise<Capabilities> =>
      request('/capabilities'),
  },

  extensions: {
    list: (): Promise<ExtensionInfo[]> =>
      request('/extensions'),

    reload: (): Promise<{ ok: boolean }> =>
      request('/extensions/reload', { method: 'POST' }),
  },

  mcpServers: {
    list: (): Promise<{ mcpServers: MCPServer[] }> =>
      request('/mcp-servers'),

    save: (mcpServers: MCPServer[]): Promise<{ mcpServers: MCPServer[] }> =>
      request('/mcp-servers', {
        method: 'PUT',
        body: JSON.stringify({ mcpServers }),
      }),
  },

  skills: {
    list: (): Promise<SkillEntry[]> =>
      request('/skills'),

    remove: (name: string): Promise<void> =>
      request(`/skills/${encodeURIComponent(name)}`, { method: 'DELETE' }),
  },

  agents: {
    list: (): Promise<AgentFileInfo[]> =>
      request('/agents'),

    get: (file: string): Promise<AgentFileContent> =>
      request(`/agents/${encodeURIComponent(file)}`),

    create: (file: string, content: string): Promise<AgentFileInfo> =>
      request('/agents', {
        method: 'POST',
        body: JSON.stringify({ file, content }),
      }),

    save: (file: string, content: string): Promise<AgentFileInfo> =>
      request(`/agents/${encodeURIComponent(file)}`, {
        method: 'PUT',
        body: JSON.stringify({ content }),
      }),

    remove: (file: string): Promise<void> =>
      request(`/agents/${encodeURIComponent(file)}`, { method: 'DELETE' }),
  },
}
