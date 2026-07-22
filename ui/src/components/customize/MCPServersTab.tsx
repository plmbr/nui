// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link2, Plus, Trash2 } from 'lucide-react'
import { ConfirmDeleteDialog } from '@/components/ConfirmDeleteDialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { SearchInput } from '@/components/SearchInput'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  selectItemData,
} from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import { api } from '@/api'
import type { MCPOAuthStatus, MCPServer } from '@/types'

function emptyServer(): MCPServer {
  return { name: '', type: 'stdio', command: '' }
}

const MCP_SERVER_TYPES = [
  { id: 'stdio', label: 'stdio (local command)' },
  { id: 'http', label: 'http (remote)' },
  { id: 'sse', label: 'sse (remote)' },
] as const

function normalizeServerType(type: string | undefined): string {
  const t = (type ?? 'stdio').toLowerCase()
  if (t === 'http' || t === 'sse') return t
  return 'stdio'
}

function isRemote(server: MCPServer): boolean {
  const t = normalizeServerType(server.type)
  return t === 'http' || t === 'sse' || Boolean(server.url?.trim())
}

function serverKey(server: MCPServer): string {
  const url = server.url?.trim()
  if (url) return url.replace(/\/$/, '')
  return server.name.trim()
}

function statusLabel(status: MCPOAuthStatus | undefined): string {
  switch (status) {
    case 'connected':
      return 'Connected'
    case 'needs_auth':
      return 'Needs authentication'
    case 'expired':
      return 'Expired'
    default:
      return 'N/A'
  }
}

function oauthSuccessMessage(data: unknown): boolean {
  return typeof data === 'object' && data !== null && (data as { type?: string }).type === 'nui-mcp-oauth-success'
}

async function pollOAuthFlow(
  flowId: string,
  timeoutMs = 3 * 60 * 1000,
): Promise<{ status: 'completed' | 'failed' | 'timeout'; serverKey?: string; error?: string }> {
  const deadline = Date.now() + timeoutMs
  while (Date.now() < deadline) {
    await new Promise((resolve) => setTimeout(resolve, 1000))
    try {
      const res = await api.mcpOAuth.flow(flowId)
      if (res.status === 'completed') {
        return { status: 'completed', serverKey: res.serverKey }
      }
      if (res.status === 'failed') {
        return { status: 'failed', serverKey: res.serverKey, error: res.error }
      }
    } catch {
      // flow record may not exist yet; keep polling
    }
  }
  return { status: 'timeout' }
}

export function MCPServersTab() {
  const [servers, setServers] = useState<MCPServer[]>([])
  const [authStatus, setAuthStatus] = useState<Record<string, MCPOAuthStatus>>({})
  const [redirectUri, setRedirectUri] = useState('')
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [connectingKey, setConnectingKey] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [deleteIndex, setDeleteIndex] = useState<number | null>(null)
  const [searchQuery, setSearchQuery] = useState('')

  const filteredServerEntries = useMemo(() => {
    const query = searchQuery.trim().toLowerCase()
    return servers
      .map((server, index) => ({ server, index }))
      .filter(({ server }) => {
        if (!query) return true
        const haystack = [
          server.name,
          server.type,
          server.command,
          server.url,
          server.args?.join(' '),
        ]
          .filter(Boolean)
          .join(' ')
          .toLowerCase()
        return haystack.includes(query)
      })
  }, [servers, searchQuery])

  const refreshAuthStatus = useCallback(async () => {
    try {
      const res = await api.mcpOAuth.status()
      const map: Record<string, MCPOAuthStatus> = {}
      for (const entry of res.servers) {
        map[entry.serverKey] = entry.status
        if (entry.name) map[entry.name] = entry.status
      }
      setAuthStatus(map)
    } catch {
      setAuthStatus({})
    }
  }, [])

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const [listRes, redirectRes] = await Promise.all([
        api.mcpServers.list(),
        api.mcpOAuth.redirectUri(),
      ])
      setServers(listRes.mcpServers.length > 0 ? listRes.mcpServers : [])
      setRedirectUri(redirectRes.redirectUri)
      await refreshAuthStatus()
    } catch {
      setServers([])
    } finally {
      setLoading(false)
    }
  }, [refreshAuthStatus])

  useEffect(() => {
    void load()
  }, [load])

  useEffect(() => {
    const onOAuthSuccess = () => {
      void refreshAuthStatus()
      setConnectingKey(null)
    }
    const onMessage = (event: MessageEvent) => {
      if (oauthSuccessMessage(event.data)) {
        onOAuthSuccess()
      }
    }
    let channel: BroadcastChannel | null = null
    try {
      channel = new BroadcastChannel('nui-mcp-oauth')
      channel.onmessage = (event) => {
        if (oauthSuccessMessage(event.data)) {
          onOAuthSuccess()
        }
      }
    } catch {
      // BroadcastChannel unavailable
    }
    window.addEventListener('message', onMessage)
    return () => {
      window.removeEventListener('message', onMessage)
      channel?.close()
    }
  }, [refreshAuthStatus])

  const updateServer = (index: number, patch: Partial<MCPServer>) => {
    setServers((prev) => prev.map((s, i) => (i === index ? { ...s, ...patch } : s)))
  }

  const save = async () => {
    setSaving(true)
    setError(null)
    try {
      const cleaned = servers.filter((s) => s.name.trim())
      const res = await api.mcpServers.save(cleaned)
      setServers(res.mcpServers)
      await refreshAuthStatus()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to save')
    } finally {
      setSaving(false)
    }
  }

  const connectServer = async (server: MCPServer) => {
    const key = serverKey(server)
    if (!key) return
    setConnectingKey(key)
    setError(null)
    try {
      const res = await api.mcpOAuth.start({ serverKey: key, server })
      window.open(res.authUrl, '_blank', 'noopener,noreferrer')
      const outcome = await pollOAuthFlow(res.flowId)
      if (outcome.status === 'completed') {
        const connectedKey = outcome.serverKey ?? res.serverKey ?? key
        setAuthStatus((prev) => ({ ...prev, [connectedKey]: 'connected', [server.name.trim()]: 'connected' }))
        await refreshAuthStatus()
      } else if (outcome.status === 'failed') {
        setError(outcome.error ?? 'OAuth failed')
      } else {
        setError('OAuth timed out. If you completed sign-in, refresh this page.')
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to start OAuth')
    } finally {
      setConnectingKey(null)
    }
  }

  const disconnectServer = async (server: MCPServer) => {
    const key = serverKey(server)
    if (!key) return
    setError(null)
    try {
      await api.mcpOAuth.disconnect(key)
      await refreshAuthStatus()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to disconnect')
    }
  }

  if (loading) {
    return <p className="text-sm text-muted-foreground">Loading MCP servers…</p>
  }

  return (
    <div className="customize-tab-content space-y-4">
      <div className="space-y-2">
        <p className="text-sm text-muted-foreground">
          User MCP servers stored in <code className="text-xs">~/.nui/mcp-servers.json</code>.
          Remote servers can use OAuth — register this redirect URI with your provider:
        </p>
        {redirectUri && (
          <code className="block text-xs break-all rounded border bg-muted/40 p-2">{redirectUri}</code>
        )}
      </div>

      {servers.length === 0 ? (
        <p className="text-sm text-muted-foreground">No MCP servers configured yet.</p>
      ) : (
        <div className="space-y-4">
          <SearchInput
            value={searchQuery}
            onChange={setSearchQuery}
            placeholder="Search MCP servers…"
            aria-label="Search configured MCP servers"
          />
          {filteredServerEntries.length === 0 ? (
            <p className="text-sm text-muted-foreground">No MCP servers match your search.</p>
          ) : (
          filteredServerEntries.map(({ server, index }) => {
            const key = serverKey(server)
            const status = authStatus[key] ?? authStatus[server.name]
            const remote = isRemote(server)
            return (
              <div key={index} className="rounded-lg border p-4 space-y-3">
                <div className="flex items-center justify-between gap-2">
                  <div className="flex items-center gap-2">
                    <p className="text-sm font-medium">Server {index + 1}</p>
                    {remote && (
                      <span className="text-xs rounded-full border px-2 py-0.5 text-muted-foreground">
                        {statusLabel(status)}
                      </span>
                    )}
                  </div>
                  <Button variant="ghost" size="sm" onClick={() => setDeleteIndex(index)}>
                    <Trash2 className="size-3.5" />
                  </Button>
                </div>
                <div className="grid gap-3 sm:grid-cols-2">
                  <div className="space-y-1.5">
                    <Label>Name</Label>
                    <Input
                      value={server.name}
                      onChange={(e) => updateServer(index, { name: e.target.value })}
                      placeholder="my-server"
                    />
                  </div>
                  <div className="space-y-1.5">
                    <Label>Type</Label>
                    <Select
                      value={normalizeServerType(server.type)}
                      onValueChange={(value) => {
                        if (!value) return
                        updateServer(index, { type: value })
                      }}
                      items={selectItemData([...MCP_SERVER_TYPES])}
                    >
                      <SelectTrigger className="w-full">
                        <SelectValue placeholder="Select type" />
                      </SelectTrigger>
                      <SelectContent>
                        {MCP_SERVER_TYPES.map((item) => (
                          <SelectItem key={item.id} value={item.id}>
                            {item.label}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                </div>
                {remote ? (
                  <>
                    <div className="space-y-1.5">
                      <Label>URL</Label>
                      <Input
                        value={server.url ?? ''}
                        onChange={(e) => updateServer(index, { url: e.target.value })}
                        placeholder="https://mcp.example.com/mcp"
                      />
                    </div>
                    <div className="grid gap-3 sm:grid-cols-2">
                      <div className="space-y-1.5">
                        <Label>OAuth client ID</Label>
                        <Input
                          value={server.auth?.clientId ?? ''}
                          onChange={(e) =>
                            updateServer(index, {
                              auth: { ...server.auth, clientId: e.target.value },
                            })
                          }
                          placeholder="required for GitHub, Slack, etc."
                        />
                      </div>
                      <div className="space-y-1.5">
                        <Label>OAuth client secret</Label>
                        <Input
                          type="password"
                          value={server.auth?.clientSecret ?? ''}
                          onChange={(e) =>
                            updateServer(index, {
                              auth: { ...server.auth, clientSecret: e.target.value },
                            })
                          }
                          placeholder="required for GitHub OAuth apps"
                          autoComplete="off"
                        />
                      </div>
                    </div>
                    <div className="space-y-1.5">
                      <Label>OAuth scopes (comma-separated)</Label>
                      <Input
                        value={(server.auth?.scopes ?? []).join(', ')}
                        onChange={(e) =>
                          updateServer(index, {
                            auth: {
                              ...server.auth,
                              scopes: e.target.value
                                .split(',')
                                .map((s) => s.trim())
                                .filter(Boolean),
                            },
                          })
                        }
                        placeholder="read, write"
                      />
                      <p className="text-xs text-muted-foreground">
                        Required when the provider does not support automatic registration (e.g. GitHub Copilot MCP).
                        Create an OAuth app, register the redirect URI shown above, and enter both client ID and client secret.
                      </p>
                    </div>
                    <div className="flex flex-wrap gap-2">
                      <Button
                        size="sm"
                        variant="outline"
                        disabled={connectingKey === key}
                        onClick={() => void connectServer(server)}
                      >
                        <Link2 className="size-3.5" />
                        {connectingKey === key ? 'Connecting…' : 'Connect'}
                      </Button>
                      {(status === 'connected' || status === 'expired') && (
                        <Button size="sm" variant="ghost" onClick={() => void disconnectServer(server)}>
                          Disconnect
                        </Button>
                      )}
                    </div>
                  </>
                ) : (
                  <>
                    <div className="space-y-1.5">
                      <Label>Command</Label>
                      <Input
                        value={server.command ?? ''}
                        onChange={(e) => updateServer(index, { command: e.target.value })}
                        placeholder="npx"
                      />
                    </div>
                    <div className="space-y-1.5">
                      <Label>Args (one per line)</Label>
                      <Textarea
                        value={(server.args ?? []).join('\n')}
                        onChange={(e) =>
                          updateServer(index, {
                            args: e.target.value.split('\n').filter((line) => line.length > 0),
                          })
                        }
                        rows={3}
                        placeholder="-y&#10;@modelcontextprotocol/server-filesystem&#10;/path"
                      />
                    </div>
                  </>
                )}
              </div>
            )
          })
          )}
        </div>
      )}

      <div className="flex flex-wrap items-center gap-2">
        <Button variant="outline" size="sm" onClick={() => setServers((prev) => [...prev, emptyServer()])}>
          <Plus className="size-3.5" />
          Add server
        </Button>
        <Button size="sm" onClick={() => void save()} disabled={saving}>
          {saving ? 'Saving…' : 'Save changes'}
        </Button>
      </div>
      {error && <p className="text-sm text-destructive">{error}</p>}

      <ConfirmDeleteDialog
        open={deleteIndex != null}
        onOpenChange={(open) => { if (!open) setDeleteIndex(null) }}
        title="Remove MCP server?"
        description={
          deleteIndex != null ? (
            <>
              This will remove{' '}
              <strong>{servers[deleteIndex]?.name.trim() || `Server ${deleteIndex + 1}`}</strong>{' '}
              from the list. Save changes to persist the removal.
            </>
          ) : null
        }
        confirmLabel="Remove"
        onConfirm={() => {
          if (deleteIndex == null) return
          setServers((prev) => prev.filter((_, i) => i !== deleteIndex))
        }}
      />
    </div>
  )
}
