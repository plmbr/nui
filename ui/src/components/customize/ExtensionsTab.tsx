// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useCallback, useEffect, useState } from 'react'
import { RefreshCw } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { api } from '@/api'
import type { ExtensionInfo } from '@/types'

interface Props {
  onChanged?: () => void
}

export function ExtensionsTab({ onChanged }: Props) {
  const [extensions, setExtensions] = useState<ExtensionInfo[]>([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState<string | null>(null)

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
      <ul className="divide-y rounded-lg border">
        {extensions.map((ext) => (
          <li key={ext.name} className="flex items-start justify-between gap-4 p-4">
            <div className="min-w-0 flex-1">
              <p className="font-medium text-sm">{ext.displayName || ext.name}</p>
              {ext.description && (
                <p className="text-xs text-muted-foreground mt-1">{ext.description}</p>
              )}
              <div className="mt-2 flex flex-wrap gap-2 text-xs text-muted-foreground">
                {ext.version && <span>v{ext.version}</span>}
                {ext.harnesses && ext.harnesses.length > 0 && (
                  <span>{ext.harnesses.length} harness{ext.harnesses.length === 1 ? '' : 'es'}</span>
                )}
                {ext.mcpServers && ext.mcpServers.length > 0 && (
                  <span>{ext.mcpServers.length} MCP</span>
                )}
                {ext.skills && ext.skills.length > 0 && (
                  <span>{ext.skills.length} skill{ext.skills.length === 1 ? '' : 's'}</span>
                )}
                {ext.rules && ext.rules.length > 0 && (
                  <span>{ext.rules.length} rule{ext.rules.length === 1 ? '' : 's'}</span>
                )}
                {ext.mentionProviders && ext.mentionProviders.length > 0 && (
                  <span>{ext.mentionProviders.length} mention{ext.mentionProviders.length === 1 ? '' : 's'}</span>
                )}
                {ext.agents && ext.agents.length > 0 && (
                  <span>{ext.agents.length} agent{ext.agents.length === 1 ? '' : 's'}</span>
                )}
              </div>
            </div>
            <label className="flex shrink-0 items-center gap-2 text-sm cursor-pointer">
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
    </div>
  )
}
