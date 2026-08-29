// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useCallback, useEffect, useState } from 'react'
import { Download, RefreshCw, X } from 'lucide-react'
import { api } from '@/api'
import type { UpdateStatus } from '@/types'

function isDesktopShell(): boolean {
  return typeof window !== 'undefined' && !!window.__NUI_DESKTOP__
}

function shouldShowBanner(st: UpdateStatus, skipped?: string): boolean {
  if (st.state !== 'available' && st.state !== 'ready' && st.state !== 'downloading' && st.state !== 'error') {
    return false
  }
  if (st.state === 'available' && skipped && st.availableVersion && skipped === st.availableVersion) {
    return false
  }
  return true
}

export function UpdateBanner() {
  const [status, setStatus] = useState<UpdateStatus | null>(null)
  const [skipped, setSkipped] = useState<string | undefined>()
  const [busy, setBusy] = useState(false)
  const desktop = isDesktopShell()

  const refresh = useCallback(async () => {
    try {
      const [st, settings] = await Promise.all([
        api.update.status(),
        api.settings.get().catch(() => null),
      ])
      setStatus(st)
      setSkipped(settings?.skippedUpdateVersion)
    } catch {
      /* ignore */
    }
  }, [])

  useEffect(() => {
    void refresh()
    const id = window.setInterval(() => void refresh(), 60_000)
    return () => window.clearInterval(id)
  }, [refresh])

  // Auto-check shortly after load (server also checks in background).
  useEffect(() => {
    const t = window.setTimeout(() => {
      api.settings
        .get()
        .then((s) => {
          if (s.autoCheckUpdates === false) return
          const opts = isDesktopShell() ? { target: 'pathCli' as const } : undefined
          return api.update.check(opts).then(setStatus)
        })
        .catch(() => {})
    }, 4_000)
    return () => window.clearTimeout(t)
  }, [])

  if (!status || !shouldShowBanner(status, skipped)) {
    return null
  }

  const applyTarget = desktop ? 'pathCli' : 'self'
  const label =
    status.state === 'ready'
      ? desktop
        ? 'Install CLI update'
        : 'Install and restart server'
      : status.state === 'downloading'
        ? `Downloading… ${Math.round((status.progress ?? 0) * 100)}%`
        : status.state === 'error'
          ? status.error || 'Update failed'
          : `Update available: v${status.availableVersion}`

  const onDownload = async () => {
    setBusy(true)
    try {
      const st = await api.update.download()
      setStatus(st)
    } catch (e) {
      setStatus((prev) =>
        prev
          ? { ...prev, state: 'error', error: e instanceof Error ? e.message : String(e) }
          : prev,
      )
    } finally {
      setBusy(false)
    }
  }

  const onApply = async () => {
    setBusy(true)
    try {
      const st = await api.update.apply(applyTarget)
      setStatus(st)
      if (st.state === 'idle' || !st.error) {
        // PATH CLI update doesn't require app restart; self does.
        if (!desktop) {
          window.setTimeout(() => window.location.reload(), 800)
        }
      }
    } catch (e) {
      setStatus((prev) =>
        prev
          ? { ...prev, state: 'error', error: e instanceof Error ? e.message : String(e) }
          : prev,
      )
    } finally {
      setBusy(false)
    }
  }

  const onDismiss = async () => {
    setBusy(true)
    try {
      const st = await api.update.dismiss()
      setStatus(st)
    } catch {
      setStatus((prev) => (prev ? { ...prev, state: 'idle', error: undefined } : prev))
    } finally {
      setBusy(false)
    }
  }

  const onLater = async () => {
    const ver = status.availableVersion
    if (ver) {
      try {
        await api.update.skip(ver)
        setSkipped(ver)
      } catch {
        /* ignore */
      }
    }
  }

  return (
    <div className="update-banner" role="status">
      <div className="update-banner__text">
        <RefreshCw className="size-3.5 shrink-0 opacity-70" aria-hidden />
        <span className="update-banner__message" title={status.error}>
          {desktop && status.state === 'available' ? `CLI ${label}` : label}
        </span>
      </div>
      <div className="update-banner__actions">
        {status.state === 'available' && (
          <button type="button" className="update-banner__btn" disabled={busy} onClick={() => void onDownload()}>
            <Download className="size-3.5" aria-hidden />
            Download
          </button>
        )}
        {status.state === 'ready' && (
          <button type="button" className="update-banner__btn" disabled={busy} onClick={() => void onApply()}>
            Install
          </button>
        )}
        {(status.state === 'available' || status.state === 'ready') && (
          <button type="button" className="update-banner__btn update-banner__btn--ghost" disabled={busy} onClick={() => void onLater()}>
            Later
          </button>
        )}
        {status.state === 'error' && (
          <button type="button" className="update-banner__btn" disabled={busy} onClick={() => void onDismiss()}>
            Dismiss
          </button>
        )}
        <button
          type="button"
          className="update-banner__icon-btn"
          aria-label="Dismiss"
          onClick={() => {
            if (status.state === 'error') {
              void onDismiss()
              return
            }
            void onLater()
          }}
        >
          <X className="size-3.5" />
        </button>
      </div>
    </div>
  )
}
