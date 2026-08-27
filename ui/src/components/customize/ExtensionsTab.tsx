// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useCallback, useEffect, useMemo, useState } from 'react'
import { Eye, EyeOff, Plus, RefreshCw, Trash2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { SearchInput } from '@/components/SearchInput'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { api } from '@/api'
import { filterBySearchQuery } from '@/lib/searchFilter'
import type { ExtensionInfo } from '@/types'

interface Props {
  onChanged?: () => void
}

type KeyValue = { key: string; value: string }

function EnvKeyValueList({
  entries,
  onChange,
}: {
  entries: KeyValue[]
  onChange: (entries: KeyValue[]) => void
}) {
  const [revealed, setRevealed] = useState<Record<number, boolean>>({})

  return (
    <div className="space-y-2">
      {entries.map((entry, index) => {
        const shown = !!revealed[index]
        return (
          <div
            key={index}
            className="grid grid-cols-1 gap-2 items-center sm:grid-cols-[minmax(7rem,10rem)_minmax(0,1fr)_auto_auto]"
          >
            <Input
              placeholder="KEY"
              value={entry.key}
              onChange={(e) => {
                const next = [...entries]
                next[index] = { ...entry, key: e.target.value }
                onChange(next)
              }}
              className="font-mono text-xs"
              autoComplete="off"
              spellCheck={false}
            />
            <Input
              type={shown ? 'text' : 'password'}
              placeholder="value"
              value={entry.value}
              onChange={(e) => {
                const next = [...entries]
                next[index] = { ...entry, value: e.target.value }
                onChange(next)
              }}
              className="font-mono text-xs"
              autoComplete="off"
              spellCheck={false}
            />
            <Button
              type="button"
              variant="outline"
              size="icon-sm"
              aria-label={shown ? `Hide value for ${entry.key || 'variable'}` : `Show value for ${entry.key || 'variable'}`}
              aria-pressed={shown}
              onClick={() => setRevealed((prev) => ({ ...prev, [index]: !prev[index] }))}
            >
              {shown ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
            </Button>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              className="shrink-0"
              onClick={() => onChange(entries.filter((_, i) => i !== index))}
            >
              <Trash2 className="size-3.5" />
            </Button>
          </div>
        )
      })}
      <Button
        type="button"
        variant="outline"
        size="sm"
        onClick={() => onChange([...entries, { key: '', value: '' }])}
      >
        <Plus className="size-3.5" />
        Add variable
      </Button>
    </div>
  )
}

export function ExtensionsTab({ onChanged }: Props) {
  const [extensions, setExtensions] = useState<ExtensionInfo[]>([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState<string | null>(null)
  const [searchQuery, setSearchQuery] = useState('')
  const [envTarget, setEnvTarget] = useState<ExtensionInfo | null>(null)
  const [envEntries, setEnvEntries] = useState<KeyValue[]>([])
  const [envLoading, setEnvLoading] = useState(false)
  const [envSaving, setEnvSaving] = useState(false)
  const [envError, setEnvError] = useState<string | null>(null)

  const filteredExtensions = useMemo(
    () =>
      filterBySearchQuery(extensions, searchQuery, (ext) =>
        [ext.displayName, ext.name, ext.description, ext.version].filter(Boolean).join(' '),
      ),
    [extensions, searchQuery],
  )

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const list = await api.extensions.list()
      setExtensions(list.sort((a, b) => (a.displayName || a.name).localeCompare(b.displayName || b.name)))
    } catch {
      setExtensions([])
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  const toggleExtension = async (ext: ExtensionInfo) => {
    setSaving(ext.name)
    try {
      const settings = await api.settings.get()
      const disabled = new Set(settings.disabledExtensions ?? [])
      if (ext.disabled) {
        disabled.delete(ext.name)
      } else {
        disabled.add(ext.name)
      }
      await api.settings.update({ disabledExtensions: [...disabled] })
      await api.extensions.reload()
      await load()
      onChanged?.()
    } finally {
      setSaving(null)
    }
  }

  const openEnv = async (ext: ExtensionInfo) => {
    setEnvTarget(ext)
    setEnvError(null)
    setEnvLoading(true)
    setEnvEntries([])
    try {
      const res = await api.extensions.getEnv(ext.name)
      const entries = Object.entries(res.env ?? {}).map(([key, value]) => ({ key, value }))
      setEnvEntries(entries.length > 0 ? entries : [])
    } catch (err) {
      setEnvError(err instanceof Error ? err.message : String(err))
    } finally {
      setEnvLoading(false)
    }
  }

  const saveEnv = async () => {
    if (!envTarget) return
    setEnvSaving(true)
    setEnvError(null)
    try {
      const env: Record<string, string> = {}
      for (const entry of envEntries) {
        const key = entry.key.trim()
        if (!key) continue
        env[key] = entry.value
      }
      await api.extensions.updateEnv(envTarget.name, env)
      setEnvTarget(null)
      await load()
      onChanged?.()
    } catch (err) {
      setEnvError(err instanceof Error ? err.message : String(err))
    } finally {
      setEnvSaving(false)
    }
  }

  if (loading) {
    return <p className="text-sm text-muted-foreground">Loading extensions…</p>
  }

  if (extensions.length === 0) {
    return (
      <div className="customize-tab-content">
        <p className="text-sm text-muted-foreground">
          No extensions installed. Add packages to <code className="text-xs">~/.nui/extensions/</code>.
        </p>
      </div>
    )
  }

  return (
    <div className="customize-tab-content space-y-4">
      <div className="flex items-center justify-between gap-4">
        <p className="text-sm text-muted-foreground">
          Installed extensions from <code className="text-xs">~/.nui/extensions/</code>
        </p>
        <Button variant="outline" size="sm" onClick={() => void load()}>
          <RefreshCw className="size-3.5" />
          Refresh
        </Button>
      </div>
      <SearchInput
        value={searchQuery}
        onChange={setSearchQuery}
        placeholder="Search extensions…"
        aria-label="Search installed extensions"
      />
      {filteredExtensions.length === 0 ? (
        <p className="text-sm text-muted-foreground">No extensions match your search.</p>
      ) : (
        <ul className="divide-y rounded-lg border">
          {filteredExtensions.map((ext) => (
            <li key={ext.name} className="flex items-start justify-between gap-4 p-4">
              <div className="min-w-0 flex-1 space-y-3">
                <div>
                  <p className="font-medium text-sm">{ext.displayName || ext.name}</p>
                  {ext.description && (
                    <p className="text-xs text-muted-foreground mt-1">{ext.description}</p>
                  )}
                  <div className="mt-2 flex flex-wrap gap-2 text-xs text-muted-foreground">
                    {ext.version && <span>v{ext.version}</span>}
                    {ext.harnesses && ext.harnesses.length > 0 && (
                      <span>
                        {ext.harnesses.length} harness{ext.harnesses.length === 1 ? '' : 'es'}
                      </span>
                    )}
                    {ext.mcpServers && ext.mcpServers.length > 0 && (
                      <span>{ext.mcpServers.length} MCP</span>
                    )}
                    {ext.skills && ext.skills.length > 0 && (
                      <span>
                        {ext.skills.length} skill{ext.skills.length === 1 ? '' : 's'}
                      </span>
                    )}
                    {ext.rules && ext.rules.length > 0 && (
                      <span>
                        {ext.rules.length} rule{ext.rules.length === 1 ? '' : 's'}
                      </span>
                    )}
                    {ext.mentionProviders && ext.mentionProviders.length > 0 && (
                      <span>
                        {ext.mentionProviders.length} mention
                        {ext.mentionProviders.length === 1 ? '' : 's'}
                      </span>
                    )}
                    {ext.agents && ext.agents.length > 0 && (
                      <span>
                        {ext.agents.length} agent{ext.agents.length === 1 ? '' : 's'}
                      </span>
                    )}
                  </div>
                </div>
                <div className="flex flex-wrap items-center gap-2">
                  <span className="text-xs font-semibold shrink-0">Env vars</span>
                  <span className="min-w-0 text-[11px] font-mono text-muted-foreground">
                    {ext.envKeys && ext.envKeys.length > 0
                      ? ext.envKeys.join(', ')
                      : 'None'}
                  </span>
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    className="shrink-0"
                    onClick={() => void openEnv(ext)}
                  >
                    Edit
                  </Button>
                </div>
              </div>
              <label className="flex shrink-0 items-center gap-2 text-sm cursor-pointer pt-0.5">
                <input
                  type="checkbox"
                  className="size-4 rounded border-input"
                  checked={!ext.disabled}
                  disabled={saving === ext.name}
                  onChange={() => void toggleExtension(ext)}
                />
                Enabled
              </label>
            </li>
          ))}
        </ul>
      )}

      <Dialog
        open={!!envTarget}
        onOpenChange={(open) => {
          if (!open) setEnvTarget(null)
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Env vars — {envTarget?.displayName || envTarget?.name}</DialogTitle>
            <DialogDescription>
              Per-extension environment variables (stored in{' '}
              <code className="text-[11px]">~/.nui/extension-env.json</code>). Applied when this
              extension&apos;s processes start. Keys only are shown on the extensions list.
            </DialogDescription>
          </DialogHeader>
          {envLoading ? (
            <p className="text-sm text-muted-foreground">Loading…</p>
          ) : (
            <EnvKeyValueList entries={envEntries} onChange={setEnvEntries} />
          )}
          {envError && (
            <p className="text-sm text-destructive" role="alert">
              {envError}
            </p>
          )}
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setEnvTarget(null)}>
              Cancel
            </Button>
            <Button type="button" onClick={() => void saveEnv()} disabled={envLoading || envSaving}>
              {envSaving ? 'Saving…' : 'Save'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
