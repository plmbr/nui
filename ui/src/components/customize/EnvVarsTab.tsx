// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useCallback, useEffect, useMemo, useState } from 'react'
import { Eye, EyeOff, Plus, Trash2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { api } from '@/api'
import type { CredentialField, CustomEnvEntry } from '@/types'

interface Props {
  onChanged?: () => void
}

type KeyValue = { key: string; value: string }

function CustomEnvList({
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

export function EnvVarsTab({ onChanged }: Props) {
  const [fields, setFields] = useState<CredentialField[]>([])
  const [draft, setDraft] = useState<Record<string, string>>({})
  const [custom, setCustom] = useState<KeyValue[]>([])
  const [savedCustom, setSavedCustom] = useState<KeyValue[]>([])
  const [revealed, setRevealed] = useState<Record<string, boolean>>({})
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [savedAt, setSavedAt] = useState<number | null>(null)

  const applyResponse = useCallback((res: { fields: CredentialField[]; custom?: CustomEnvEntry[] }) => {
    setFields(res.fields)
    const next: Record<string, string> = {}
    for (const field of res.fields) {
      next[field.key] = field.value
    }
    setDraft(next)
    const customEntries = (res.custom ?? []).map((e) => ({ key: e.key, value: e.value }))
    setCustom(customEntries)
    setSavedCustom(customEntries)
    setRevealed({})
  }, [])

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const res = await api.env.get()
      applyResponse(res)
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
      setFields([])
      setCustom([])
      setSavedCustom([])
    } finally {
      setLoading(false)
    }
  }, [applyResponse])

  useEffect(() => {
    void load()
  }, [load])

  const credentialsDirty = useMemo(() => {
    for (const field of fields) {
      if ((draft[field.key] ?? '') !== field.value) return true
    }
    return false
  }, [draft, fields])

  const customDirty = useMemo(() => {
    const normalize = (entries: KeyValue[]) =>
      entries
        .map((e) => ({ key: e.key.trim(), value: e.value }))
        .filter((e) => e.key !== '' || e.value !== '')
    const a = normalize(custom)
    const b = normalize(savedCustom)
    if (a.length !== b.length) return true
    for (let i = 0; i < a.length; i++) {
      if (a[i].key !== b[i].key || a[i].value !== b[i].value) return true
    }
    return false
  }, [custom, savedCustom])

  const dirty = credentialsDirty || customDirty

  const groups = useMemo(() => {
    const order: string[] = []
    const map = new Map<string, CredentialField[]>()
    for (const field of fields) {
      if (!map.has(field.group)) {
        map.set(field.group, [])
        order.push(field.group)
      }
      map.get(field.group)!.push(field)
    }
    return order.map((group) => ({ group, fields: map.get(group)! }))
  }, [fields])

  const save = async () => {
    setSaving(true)
    setError(null)
    try {
      const env: Record<string, string> = {}
      for (const field of fields) {
        env[field.key] = draft[field.key] ?? ''
      }
      const customMap: Record<string, string> = {}
      for (const entry of custom) {
        const key = entry.key.trim()
        if (!key) continue
        customMap[key] = entry.value
      }
      const res = await api.env.update({ env, custom: customMap })
      applyResponse(res)
      setSavedAt(Date.now())
      onChanged?.()
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setSaving(false)
    }
  }

  if (loading) {
    return <p className="text-sm text-muted-foreground">Loading env vars…</p>
  }

  return (
    <div className="customize-tab-content space-y-6">
      <div className="space-y-1">
        <h2 className="text-sm font-semibold">Env vars</h2>
        <p className="text-xs text-muted-foreground">
          Applied to nui and processes it launches (extensions, harnesses, MCP). Values are saved to{' '}
          <code className="text-[11px]">~/.nui/secrets.json</code> (mode 0600). Process environment
          variables still take precedence when already set. Long-lived extension hosts reload after
          save.
        </p>
      </div>

      {error && (
        <p className="text-sm text-destructive" role="alert">
          {error}
        </p>
      )}

      <section className="space-y-4">
        <div className="space-y-1">
          <h3 className="text-sm font-semibold">Credentials</h3>
          <p className="text-xs text-muted-foreground">
            API provider keys and base URLs for desktop and other launches that do not inherit your
            shell environment.
          </p>
        </div>
        {groups.map(({ group, fields: groupFields }) => (
          <div key={group} className="space-y-3">
            <h4 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
              {group}
            </h4>
            <div className="space-y-3">
              {groupFields.map((field) => {
                const isSecret = field.secret
                const shown = !!revealed[field.key]
                const inputType = isSecret && !shown ? 'password' : 'text'
                return (
                  <div key={field.key} className="space-y-1.5">
                    <div className="flex items-center justify-between gap-2">
                      <Label htmlFor={`cred-${field.key}`}>{field.label}</Label>
                      <span className="text-[11px] text-muted-foreground font-mono">{field.key}</span>
                    </div>
                    <div className="flex gap-2">
                      <Input
                        id={`cred-${field.key}`}
                        type={inputType}
                        value={draft[field.key] ?? ''}
                        onChange={(e) =>
                          setDraft((prev) => ({ ...prev, [field.key]: e.target.value }))
                        }
                        placeholder={
                          field.fromEnv && !(draft[field.key] ?? '')
                            ? 'Using value from environment'
                            : undefined
                        }
                        autoComplete="off"
                        spellCheck={false}
                        className="font-mono"
                      />
                      {isSecret && (
                        <Button
                          type="button"
                          variant="outline"
                          size="icon-sm"
                          aria-label={shown ? `Hide ${field.label}` : `Show ${field.label}`}
                          aria-pressed={shown}
                          onClick={() =>
                            setRevealed((prev) => ({ ...prev, [field.key]: !prev[field.key] }))
                          }
                        >
                          {shown ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
                        </Button>
                      )}
                    </div>
                    {field.description && (
                      <p className="text-xs text-muted-foreground">{field.description}</p>
                    )}
                    {field.fromEnv && (
                      <p className="text-xs text-muted-foreground">
                        Also set in the process environment (takes precedence over the stored value).
                      </p>
                    )}
                  </div>
                )
              })}
            </div>
          </div>
        ))}
      </section>

      <section className="space-y-3">
        <div className="space-y-1">
          <h3 className="text-sm font-semibold">Custom</h3>
          <p className="text-xs text-muted-foreground">
            Free-form environment variables for extensions and other child processes. Core nui
            runtime keys (for example <code className="text-[11px]">NUI_API_URL</code>) are
            reserved; extension-owned <code className="text-[11px]">NUI_*</code> names are allowed.
          </p>
        </div>
        <CustomEnvList entries={custom} onChange={setCustom} />
      </section>

      <div className="flex items-center gap-3 pt-2">
        <Button type="button" onClick={() => void save()} disabled={!dirty || saving}>
          {saving ? 'Saving…' : 'Save env vars'}
        </Button>
        {savedAt && !dirty && <span className="text-xs text-muted-foreground">Saved</span>}
      </div>
    </div>
  )
}

/** @deprecated Use EnvVarsTab */
export const CredentialsTab = EnvVarsTab
