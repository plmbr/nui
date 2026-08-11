// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useCallback, useEffect, useMemo, useState } from 'react'
import { Eye, EyeOff } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { api } from '@/api'
import type { CredentialField } from '@/types'

interface Props {
  onChanged?: () => void
}

export function CredentialsTab({ onChanged }: Props) {
  const [fields, setFields] = useState<CredentialField[]>([])
  const [draft, setDraft] = useState<Record<string, string>>({})
  const [revealed, setRevealed] = useState<Record<string, boolean>>({})
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [savedAt, setSavedAt] = useState<number | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const res = await api.credentials.get()
      setFields(res.fields)
      const next: Record<string, string> = {}
      for (const field of res.fields) {
        next[field.key] = field.value
      }
      setDraft(next)
      setRevealed({})
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
      setFields([])
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  const dirty = useMemo(() => {
    for (const field of fields) {
      if ((draft[field.key] ?? '') !== field.value) return true
    }
    return false
  }, [draft, fields])

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
      const res = await api.credentials.update(env)
      setFields(res.fields)
      const next: Record<string, string> = {}
      for (const field of res.fields) {
        next[field.key] = field.value
      }
      setDraft(next)
      setSavedAt(Date.now())
      onChanged?.()
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setSaving(false)
    }
  }

  if (loading) {
    return <p className="text-sm text-muted-foreground">Loading credentials…</p>
  }

  return (
    <div className="customize-tab-content space-y-6">
      <div className="space-y-1">
        <h2 className="text-sm font-semibold">Credentials</h2>
        <p className="text-xs text-muted-foreground">
          Store API keys for desktop and other launches that do not inherit your shell environment.
          Values are saved to <code className="text-[11px]">~/.nui/secrets.json</code> (mode 0600).
          Process environment variables still take precedence when set.
        </p>
      </div>

      {error && (
        <p className="text-sm text-destructive" role="alert">
          {error}
        </p>
      )}

      {groups.map(({ group, fields: groupFields }) => (
        <section key={group} className="space-y-3">
          <h3 className="text-sm font-semibold">{group}</h3>
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
        </section>
      ))}

      <div className="flex items-center gap-3 pt-2">
        <Button type="button" onClick={() => void save()} disabled={!dirty || saving}>
          {saving ? 'Saving…' : 'Save credentials'}
        </Button>
        {savedAt && !dirty && (
          <span className="text-xs text-muted-foreground">Saved</span>
        )}
      </div>
    </div>
  )
}
