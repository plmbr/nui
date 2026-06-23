// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useCallback, useEffect, useState } from 'react'
import { Plus, Trash2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { api } from '@/api'
import type { MCPServer } from '@/types'

function emptyServer(): MCPServer {
  return { name: '', type: 'stdio', command: '' }
}

export function MCPServersTab() {
  const [servers, setServers] = useState<MCPServer[]>([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await api.mcpServers.list()
      setServers(res.mcpServers.length > 0 ? res.mcpServers : [])
    } catch {
      setServers([])
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

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
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to save')
    } finally {
      setSaving(false)
    }
  }

  if (loading) {
    return <p className="text-sm text-muted-foreground">Loading MCP servers…</p>
  }

  return (
    <div className="customize-tab-content space-y-4">
      <div>
        <p className="text-sm text-muted-foreground">
          User MCP servers stored in <code className="text-xs">~/.loop/mcp-servers.json</code>.
          Extension-provided servers are managed separately.
        </p>
      </div>

      {servers.length === 0 ? (
        <p className="text-sm text-muted-foreground">No MCP servers configured yet.</p>
      ) : (
        <div className="space-y-4">
          {servers.map((server, index) => (
            <div key={index} className="rounded-lg border p-4 space-y-3">
              <div className="flex items-center justify-between gap-2">
                <p className="text-sm font-medium">Server {index + 1}</p>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setServers((prev) => prev.filter((_, i) => i !== index))}
                >
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
                  <Input
                    value={server.type ?? 'stdio'}
                    onChange={(e) => updateServer(index, { type: e.target.value })}
                    placeholder="stdio | http | sse"
                  />
                </div>
              </div>
              {(server.type === 'http' || server.type === 'sse') ? (
                <div className="space-y-1.5">
                  <Label>URL</Label>
                  <Input
                    value={server.url ?? ''}
                    onChange={(e) => updateServer(index, { url: e.target.value })}
                    placeholder="http://localhost:3000/mcp"
                  />
                </div>
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
          ))}
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
    </div>
  )
}
