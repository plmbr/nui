// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useCallback, useEffect, useState } from 'react'
import { Download, RefreshCw, X } from 'lucide-react'
import type { UpdateStatus } from '@/types'
import {
  checkDesktopAppUpdate,
  desktopAppUpdateStatus,
  downloadDesktopAppUpdate,
  hasDesktopAppUpdater,
  quitAndInstallDesktopApp,
} from '@/lib/desktopUpdate'
import { api } from '@/api'

function shouldShow(st: UpdateStatus | null, skipped?: string): boolean {
  if (!st) return false
  if (st.state !== 'available' && st.state !== 'ready' && st.state !== 'downloading' && st.state !== 'error') {
    return false
  }
  if (st.state === 'available' && skipped && st.availableVersion && skipped === `app:${st.availableVersion}`) {
    return false
  }
  return true
}

/** Electron-style banner for updating the Wails desktop app itself. */
export function DesktopAppUpdateBanner() {
  const [status, setStatus] = useState<UpdateStatus | null>(null)
  const [skipped, setSkipped] = useState<string | undefined>()
  const [busy, setBusy] = useState(false)
  const enabled = hasDesktopAppUpdater()

  const refresh = useCallback(async () => {
    if (!enabled) return
    try {
      const [st, settings] = await Promise.all([
        desktopAppUpdateStatus(),
        api.settings.get().catch(() => null),
      ])
      setStatus(st)
      setSkipped(settings?.skippedUpdateVersion)
    } catch {
      /* ignore */
    }
  }, [enabled])

  useEffect(() => {
    if (!enabled) return
    void refresh()
    const off = window.runtime?.EventsOn?.('update:app:status', (payload) => {
      setStatus(payload as UpdateStatus)
    })
    const t = window.setTimeout(() => {
      api.settings
        .get()
        .then((s) => {
          if (s.autoCheckUpdates === false) return
          return checkDesktopAppUpdate(false).then(setStatus)
        })
        .catch(() => {})
    }, 6_000)
    return () => {
      window.clearTimeout(t)
      off?.()
    }
  }, [enabled, refresh])

  if (!enabled || !shouldShow(status, skipped)) {
    return null
  }

  const label =
    status!.state === 'ready'
      ? 'Install app update and restart'
      : status!.state === 'downloading'
        ? `Downloading app… ${Math.round((status!.progress ?? 0) * 100)}%`
        : status!.state === 'error'
          ? status!.error || 'App update failed'
          : `App update available: v${status!.availableVersion}`

  return (
    <div className="update-banner update-banner--app" role="status">
      <div className="update-banner__text">
        <RefreshCw className="size-3.5 shrink-0 opacity-70" aria-hidden />
        <span>{label}</span>
      </div>
      <div className="update-banner__actions">
        {status!.state === 'available' && (
          <button
            type="button"
            className="update-banner__btn"
            disabled={busy}
            onClick={() => {
              setBusy(true)
              void downloadDesktopAppUpdate()
                .then(setStatus)
                .finally(() => setBusy(false))
            }}
          >
            <Download className="size-3.5" aria-hidden />
            Download
          </button>
        )}
        {status!.state === 'ready' && (
          <button
            type="button"
            className="update-banner__btn"
            disabled={busy}
            onClick={() => {
              setBusy(true)
              void quitAndInstallDesktopApp()
                .then(setStatus)
                .finally(() => setBusy(false))
            }}
          >
            Install & restart
          </button>
        )}
        {(status!.state === 'available' || status!.state === 'ready') && (
          <button
            type="button"
            className="update-banner__btn update-banner__btn--ghost"
            disabled={busy}
            onClick={() => {
              const ver = status!.availableVersion
              if (ver) {
                void api.update.skip(`app:${ver}`).then(() => setSkipped(`app:${ver}`))
              }
            }}
          >
            Later
          </button>
        )}
        <button
          type="button"
          className="update-banner__icon-btn"
          aria-label="Dismiss"
          onClick={() => {
            const ver = status!.availableVersion
            if (ver) void api.update.skip(`app:${ver}`).then(() => setSkipped(`app:${ver}`))
          }}
        >
          <X className="size-3.5" />
        </button>
      </div>
    </div>
  )
}
