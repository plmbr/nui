// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useCallback, useEffect, useState } from 'react'
import { Plus, Trash2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { api } from '@/api'
import type { AgentFileInfo } from '@/types'

const NEW_AGENT_TEMPLATE = `adl: "1.0"
id: my-agent
name: My Agent
description: A custom agent
harness:
  type: claude-code
  sandbox: none
`

export function AgentsTab() {
  const [agents, setAgents] = useState<AgentFileInfo[]>([])
  const [selectedFile, setSelectedFile] = useState<string | null>(null)
  const [content, setContent] = useState('')
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [creating, setCreating] = useState(false)
  const [newFilename, setNewFilename] = useState('my-agent.yaml')
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const list = await api.agents.list()
      setAgents(list.sort((a, b) => a.name.localeCompare(b.name)))
    } catch {
      setAgents([])
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  const openAgent = async (file: string) => {
    setError(null)
    try {
      const res = await api.agents.get(file)
      setSelectedFile(file)
      setContent(res.content)
      setCreating(false)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load agent')
    }
  }

  const save = async () => {
    if (!selectedFile) return
    setSaving(true)
    setError(null)
    try {
      await api.agents.save(selectedFile, content)
      await load()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to save')
    } finally {
      setSaving(false)
    }
  }

  const create = async () => {
    setSaving(true)
    setError(null)
    try {
      const info = await api.agents.create(newFilename, NEW_AGENT_TEMPLATE)
      await load()
      setCreating(false)
      await openAgent(info.file)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to create agent')
    } finally {
      setSaving(false)
    }
  }

  const remove = async (file: string) => {
    if (!confirm(`Delete agent file "${file}"?`)) return
    setError(null)
    try {
      await api.agents.remove(file)
      if (selectedFile === file) {
        setSelectedFile(null)
        setContent('')
      }
      await load()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to delete')
    }
  }

  if (loading) {
    return <p className="text-sm text-muted-foreground">Loading agents…</p>
  }

  return (
    <div className="customize-tab-content flex flex-col gap-4 min-h-0">
      <p className="text-sm text-muted-foreground shrink-0">
        Agent definitions in <code className="text-xs">~/.loop/agents/</code> (ADL YAML).
      </p>

      <div className="flex flex-1 min-h-0 gap-4">
        <div className="w-56 shrink-0 flex flex-col gap-2">
          <Button
            variant="outline"
            size="sm"
            className="justify-start"
            onClick={() => {
              setCreating(true)
              setSelectedFile(null)
              setContent(NEW_AGENT_TEMPLATE)
            }}
          >
            <Plus className="size-3.5" />
            New agent
          </Button>
          <ul className="flex-1 overflow-y-auto rounded-lg border divide-y">
            {agents.map((agent) => (
              <li key={agent.file}>
                <button
                  type="button"
                  className="w-full text-left px-3 py-2 text-sm hover:bg-muted/60 data-active:bg-muted"
                  data-active={selectedFile === agent.file || undefined}
                  onClick={() => void openAgent(agent.file)}
                >
                  <span className="font-medium block truncate">{agent.name}</span>
                  <span className="text-xs text-muted-foreground block truncate">{agent.file}</span>
                </button>
              </li>
            ))}
          </ul>
        </div>

        <div className="flex-1 flex flex-col min-w-0 min-h-0 gap-3">
          {creating ? (
            <>
              <div className="space-y-1.5">
                <Label>Filename</Label>
                <Input value={newFilename} onChange={(e) => setNewFilename(e.target.value)} />
              </div>
              <Textarea
                className="flex-1 min-h-[320px] font-mono text-xs"
                value={content}
                onChange={(e) => setContent(e.target.value)}
              />
              <div className="flex gap-2">
                <Button size="sm" onClick={() => void create()} disabled={saving}>
                  Create
                </Button>
                <Button variant="outline" size="sm" onClick={() => setCreating(false)}>
                  Cancel
                </Button>
              </div>
            </>
          ) : selectedFile ? (
            <>
              <div className="flex items-center justify-between gap-2 shrink-0">
                <p className="text-sm font-medium truncate">{selectedFile}</p>
                <Button variant="ghost" size="sm" onClick={() => void remove(selectedFile)}>
                  <Trash2 className="size-3.5" />
                </Button>
              </div>
              <Textarea
                className="flex-1 min-h-[320px] font-mono text-xs"
                value={content}
                onChange={(e) => setContent(e.target.value)}
              />
              <Button size="sm" onClick={() => void save()} disabled={saving}>
                {saving ? 'Saving…' : 'Save changes'}
              </Button>
            </>
          ) : (
            <div className="flex flex-1 items-center justify-center text-sm text-muted-foreground">
              Select an agent to edit, or create a new one.
            </div>
          )}
        </div>
      </div>

      {error && <p className="text-sm text-destructive shrink-0">{error}</p>}
    </div>
  )
}
