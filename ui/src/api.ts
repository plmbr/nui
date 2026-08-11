// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import type { AgentType, AgentFileContent, AgentFileInfo, AgentDeployerInfo, AgentDeployResult, AgentEvalSummary, Bootstrap, Capabilities, ChatMessage, CreateScheduleRequest, CreateSessionRequest, Credentials, DirectorySuggestions, ExtensionInfo, HitlRequest, HitlResponse, MCPOAuthStatus, MCPServer, MentionListResponse, MemorySummary, Schedule, Session, Settings, SkillEntry, UploadedImage } from './types'

export interface RunRecord {
  runId: string
  sessionId: string
  status: 'running' | 'completed' | 'failed' | 'cancelled' | 'awaiting_user'
  message?: string
  output?: string
  error?: string
  startedAt: string
  finishedAt?: string
}

export interface AgentRunEvent {
  type: string
  content?: string
  sessionId?: string
  error?: string
  toolCallId?: string
  toolName?: string
  toolArgs?: string
  imageData?: string
  imageMediaType?: string
}

export interface RunFinishedPayload {
  type: 'run_finished'
  status: RunRecord['status']
  output?: string
  error?: string
}

/** API prefix: `/api` in browser, absolute localhost URL in the Wails desktop shell. */
export function apiBase(): string {
  const configured = typeof window !== 'undefined' ? window.__NUI_API_BASE__ : undefined
  if (typeof configured === 'string' && configured.trim()) {
    return configured.replace(/\/$/, '')
  }
  return '/api'
}

/** Origin for non-/api routes (e.g. /mcp-call-tool). Empty string = same origin. */
export function serverOrigin(): string {
  const base = apiBase()
  if (base === '/api') return ''
  return base.replace(/\/api\/?$/, '')
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${apiBase()}${path}`, {
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

async function uploadSessionFile(
  sessionId: string,
  file: File | Blob,
  filename?: string,
): Promise<UploadedImage> {
  const form = new FormData()
  const name = filename?.trim() || (file instanceof File ? file.name : 'upload')
  form.append('file', file, name)
  const res = await fetch(`${apiBase()}/sessions/${sessionId}/uploads`, {
    method: 'POST',
    body: form,
  })
  if (!res.ok) {
    const text = await res.text()
    throw new Error(text || res.statusText)
  }
  return res.json() as Promise<UploadedImage>
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

    createFromUrl: (opts?: { agent?: string; cwd?: string }): Promise<Session> => {
      const params = new URLSearchParams()
      if (opts?.agent?.trim()) params.set('agent', opts.agent.trim())
      if (opts?.cwd?.trim()) params.set('cwd', opts.cwd.trim())
      const qs = params.toString()
      return request(`/sessions/create${qs ? `?${qs}` : ''}`, { method: 'POST' })
    },

    ensureDefault: (): Promise<Session> =>
      request('/sessions/ensure-default', { method: 'POST' }),

    rename: (id: string, name: string): Promise<Session> =>
      request(`/sessions/${id}`, {
        method: 'PATCH',
        body: JSON.stringify({ name }),
      }),

    delete: (id: string): Promise<void> =>
      request(`/sessions/${id}`, { method: 'DELETE' }),

    stop: (id: string): Promise<{ ok: boolean }> =>
      request(`/sessions/${id}/stop`, { method: 'POST' }),

    bulkDelete: (ids: string[]): Promise<{ deleted: string[]; notFound: string[] }> =>
      request('/sessions/bulk-delete', {
        method: 'POST',
        body: JSON.stringify({ ids }),
      }),

    subscribeChanges: (onChange: () => void, onError?: (err: Error) => void): () => void => {
      const controller = new AbortController()
      let retryMs = 1000

      void (async () => {
        while (!controller.signal.aborted) {
          try {
            const res = await fetch(`${apiBase()}/sessions/events`, { signal: controller.signal })
            if (!res.ok || !res.body) {
              throw new Error(res.statusText || 'Failed to connect to session events')
            }
            retryMs = 1000
            const reader = res.body.getReader()
            const decoder = new TextDecoder()
            let buffer = ''

            while (!controller.signal.aborted) {
              const { done, value } = await reader.read()
              if (done) break
              buffer += decoder.decode(value, { stream: true })
              const chunks = buffer.split('\n\n')
              buffer = chunks.pop() ?? ''
              for (const chunk of chunks) {
                if (!chunk.includes('data: ')) continue
                onChange()
              }
            }
          } catch (err) {
            if (controller.signal.aborted) return
            onError?.(err instanceof Error ? err : new Error(String(err)))
          }
          if (controller.signal.aborted) return
          await new Promise((resolve) => setTimeout(resolve, retryMs))
          retryMs = Math.min(retryMs * 2, 30_000)
        }
      })()

      return () => controller.abort()
    },
  },

  agentTypes: {
    list: (): Promise<AgentType[]> =>
      request('/agent-types'),
  },

  directories: {
    suggest: (path: string, signal?: AbortSignal): Promise<DirectorySuggestions> =>
      request(`/directories?path=${encodeURIComponent(path)}`, { signal }),
  },

  mentions: {
    list: (
      sessionId: string,
      opts: { parent?: string; query?: string },
      signal?: AbortSignal,
    ): Promise<MentionListResponse> => {
      const params = new URLSearchParams()
      if (opts.parent?.trim()) params.set('parent', opts.parent.trim())
      if (opts.query?.trim()) params.set('query', opts.query.trim())
      const qs = params.toString()
      return request(`/sessions/${sessionId}/mentions${qs ? `?${qs}` : ''}`, { signal })
    },
  },

  uploads: {
    upload: uploadSessionFile,
    image: uploadSessionFile,
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

  runs: {
    list: (sessionId: string): Promise<RunRecord[]> =>
      request(`/sessions/${sessionId}/runs`),

    subscribeEvents: (
      sessionId: string,
      runId: string,
      handlers: {
        onEvent: (event: AgentRunEvent) => void
        onFinished: (payload: RunFinishedPayload) => void
        onError?: (err: Error) => void
      },
    ): () => void => {
      const controller = new AbortController()
      let lastEventId = 0

      void (async () => {
        try {
          const res = await fetch(`${apiBase()}/sessions/${sessionId}/runs/${runId}/events`, {
            signal: controller.signal,
            headers: lastEventId > 0 ? { 'Last-Event-ID': String(lastEventId) } : {},
          })
          if (!res.ok || !res.body) {
            throw new Error(res.statusText || 'Failed to connect to run events')
          }
          const reader = res.body.getReader()
          const decoder = new TextDecoder()
          let buffer = ''

          while (true) {
            const { done, value } = await reader.read()
            if (done) break
            buffer += decoder.decode(value, { stream: true })
            const chunks = buffer.split('\n\n')
            buffer = chunks.pop() ?? ''
            for (const chunk of chunks) {
              let eventId = 0
              let data = ''
              for (const line of chunk.split('\n')) {
                if (line.startsWith('id: ')) {
                  eventId = Number.parseInt(line.slice(4), 10) || 0
                } else if (line.startsWith('data: ')) {
                  data = line.slice(6)
                }
              }
              if (!data) continue
              if (eventId > lastEventId) lastEventId = eventId
              const parsed = JSON.parse(data) as AgentRunEvent | RunFinishedPayload
              if (parsed && typeof parsed === 'object' && 'type' in parsed && parsed.type === 'run_finished') {
                handlers.onFinished(parsed as RunFinishedPayload)
                return
              }
              handlers.onEvent(parsed as AgentRunEvent)
            }
          }
        } catch (err) {
          if (!controller.signal.aborted) {
            handlers.onError?.(err instanceof Error ? err : new Error(String(err)))
          }
        }
      })()

      return () => controller.abort()
    },
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

  credentials: {
    get: (): Promise<Credentials> =>
      request('/credentials'),

    update: (env: Record<string, string>): Promise<Credentials> =>
      request('/credentials', {
        method: 'PUT',
        body: JSON.stringify({ env }),
      }),
  },

  bootstrap: {
    get: (): Promise<Bootstrap> =>
      request('/bootstrap'),
  },

  orchestrate: (data: { prompt: string; workingDir?: string }): Promise<{
    session: Session
    prompt: string
    selectedAgentType: string
  }> =>
    request('/orchestrate', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

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

  mcpOAuth: {
    redirectUri: (): Promise<{ redirectUri: string }> =>
      request('/mcp-oauth/redirect-uri'),

    status: (): Promise<{ servers: { serverKey: string; name: string; status: MCPOAuthStatus }[] }> =>
      request('/mcp-oauth/status'),

    start: (body: { serverKey: string; server?: MCPServer }): Promise<{ flowId: string; authUrl: string; redirectUri: string; serverKey: string }> =>
      request('/mcp-oauth/start', { method: 'POST', body: JSON.stringify(body) }),

    flow: (flowId: string): Promise<{ flowId: string; serverKey: string; status: 'pending' | 'completed' | 'failed'; error?: string }> =>
      request(`/mcp-oauth/flow?flowId=${encodeURIComponent(flowId)}`),

    complete: (body: { flowId: string; callbackUrl: string }): Promise<{ ok: boolean }> =>
      request('/mcp-oauth/complete', { method: 'POST', body: JSON.stringify(body) }),

    disconnect: (serverKey: string): Promise<void> =>
      request(`/mcp-oauth/disconnect?serverKey=${encodeURIComponent(serverKey)}`, { method: 'DELETE' }),
  },

  skills: {
    list: (): Promise<SkillEntry[]> =>
      request('/skills'),

    remove: (name: string): Promise<void> =>
      request(`/skills/${encodeURIComponent(name)}`, { method: 'DELETE' }),
  },

  memory: {
    list: (): Promise<MemorySummary> =>
      request('/memory'),

    getUser: (): Promise<{ content: string }> =>
      request('/memory/user'),

    saveUser: (content: string): Promise<void> =>
      request('/memory/user', {
        method: 'PUT',
        body: JSON.stringify({ content }),
      }),

    deleteUser: (): Promise<void> =>
      request('/memory/user', { method: 'DELETE' }),

    getAgent: (agentId: string): Promise<{ agentId: string; content: string }> =>
      request(`/memory/agents/${encodeURIComponent(agentId)}`),

    saveAgent: (agentId: string, content: string): Promise<void> =>
      request(`/memory/agents/${encodeURIComponent(agentId)}`, {
        method: 'PUT',
        body: JSON.stringify({ content }),
      }),

    deleteAgent: (agentId: string): Promise<void> =>
      request(`/memory/agents/${encodeURIComponent(agentId)}`, { method: 'DELETE' }),
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

    listDeployers: (): Promise<AgentDeployerInfo[]> =>
      request<{ deployers: AgentDeployerInfo[] }>('/agent-deployers').then((r) => r.deployers ?? []),

    deploy: (agentId: string, deployerId: string): Promise<AgentDeployResult> =>
      request(`/agents/${encodeURIComponent(agentId)}/deploy`, {
        method: 'POST',
        body: JSON.stringify({ deployerId }),
      }),

    runEvals: (
      agentId: string,
      options?: { workingDir?: string; cases?: string[]; parallel?: number },
    ): Promise<AgentEvalSummary> =>
      request(`/agents/${encodeURIComponent(agentId)}/evals/run`, {
        method: 'POST',
        body: JSON.stringify(options ?? {}),
      }),
  },

  hitl: {
    respond: (
      requestId: string,
      answers: Record<string, unknown>,
      status?: string,
    ): Promise<HitlResponse> =>
      request(`/hitl/requests/${encodeURIComponent(requestId)}/respond`, {
        method: 'POST',
        body: JSON.stringify({
          answers,
          ...(status ? { status } : {}),
        }),
      }),

    listPending: (sessionId: string): Promise<HitlRequest[]> => {
      const params = new URLSearchParams({ pending: 'true' })
      if (sessionId.trim()) params.set('sessionId', sessionId.trim())
      return request(`/hitl/requests?${params.toString()}`)
    },

    get: (requestId: string): Promise<HitlRequest> =>
      request(`/hitl/requests/${encodeURIComponent(requestId)}`),
  },

  schedules: {
    list: (): Promise<Schedule[]> =>
      request('/schedules'),

    create: (data: CreateScheduleRequest): Promise<Schedule> =>
      request('/schedules', {
        method: 'POST',
        body: JSON.stringify(data),
      }),

    patch: (id: string, patch: Partial<Pick<Schedule, 'name' | 'agentType' | 'prompt' | 'workingDir' | 'interval' | 'cron' | 'runAt' | 'enabled'>>): Promise<Schedule> =>
      request(`/schedules/${id}`, {
        method: 'PATCH',
        body: JSON.stringify(patch),
      }),

    delete: (id: string): Promise<{ ok: boolean }> =>
      request(`/schedules/${id}`, { method: 'DELETE' }),

    runNow: (id: string): Promise<Schedule> =>
      request(`/schedules/${id}/run-now`, { method: 'POST' }),
  },
}
