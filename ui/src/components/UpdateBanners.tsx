// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useCallback, useEffect, useState } from 'react'
import { Download, X } from 'lucide-react'
import { api } from '@/api'
import type { UpdateStatus } from '@/types'
import {
  checkDesktopAppUpdate,
  desktopAppUpdateStatus,
  dismissDesktopAppUpdate,
  downloadDesktopAppUpdate,
  hasDesktopAppUpdater,
  quitAndInstallDesktopApp,
} from '@/lib/desktopUpdate'

type Track = 'cli' | 'app'

function isDesktopShell(): boolean {
  return typeof window !== 'undefined' && !!window.__NUI_DESKTOP__
}

function shouldShow(st: UpdateStatus, skipped: string | undefined, skipKey: string): boolean {
  if (st.state !== 'available' && st.state !== 'ready' && st.state !== 'downloading' && st.state !== 'error') {
    return false
  }
  if (st.state === 'available' && skipped && st.availableVersion && skipped === skipKey) {
    return false
  }
  return true
}

function bannerLabel(track: Track, status: UpdateStatus): string {
  const version = status.availableVersion ? `v${status.availableVersion}` : ''
  switch (status.state) {
    case 'ready':
      return track === 'app' ? 'Ready to install and restart' : 'Ready to install'
    case 'downloading':
      return `Downloading… ${Math.round((status.progress ?? 0) * 100)}%`
    case 'error':
      return status.error || 'Update failed'
    default:
      return version ? `${version} available` : 'Update available'
  }
}

interface BannerRowProps {
  track: Track
  status: UpdateStatus
  busy: boolean
  onDownload: () => void
  onApply: () => void
  onLater: () => void
  onDismiss: () => void
}

function BannerRow({ track, status, busy, onDownload, onApply, onLater, onDismiss }: BannerRowProps) {
  const label = bannerLabel(track, status)
  const trackLabel = track === 'cli' ? 'CLI' : 'App'

  return (
    <div className="update-banner" data-track={track} role="status">
      <div className="update-banner__main">
        <span className="update-banner__badge">{trackLabel}</span>
        <span className="update-banner__message" title={status.error ?? label}>
          {label}
        </span>
      </div>
      <div className="update-banner__actions">
        {status.state === 'available' && (
          <button
            type="button"
            className="update-banner__btn"
            disabled={busy}
            aria-label={`Download ${trackLabel} update`}
            onClick={onDownload}
          >
            <Download className="size-3.5" aria-hidden />
            <span className="update-banner__btn-label">Download</span>
          </button>
        )}
        {status.state === 'ready' && (
          <button type="button" className="update-banner__btn" disabled={busy} onClick={onApply}>
            {track === 'app' ? 'Install & restart' : 'Install'}
          </button>
        )}
        {(status.state === 'available' || status.state === 'ready') && (
          <button
            type="button"
            className="update-banner__btn update-banner__btn--ghost"
            disabled={busy}
            onClick={onLater}
          >
            Later
          </button>
        )}
        {status.state === 'error' && (
          <button type="button" className="update-banner__btn" disabled={busy} onClick={onDismiss}>
            Dismiss
          </button>
        )}
        <button
          type="button"
          className="update-banner__icon-btn"
          aria-label="Dismiss"
          disabled={busy}
          onClick={() => {
            if (status.state === 'error') {
              onDismiss()
              return
            }
            onLater()
          }}
        >
          <X className="size-3.5" />
        </button>
      </div>
    </div>
  )
}

export function UpdateBanners() {
  const desktop = isDesktopShell()
  const appUpdaterEnabled = hasDesktopAppUpdater()

  const [cliStatus, setCliStatus] = useState<UpdateStatus | null>(null)
  const [appStatus, setAppStatus] = useState<UpdateStatus | null>(null)
  const [skipped, setSkipped] = useState<string | undefined>()
  const [cliBusy, setCliBusy] = useState(false)
  const [appBusy, setAppBusy] = useState(false)

  const refreshCli = useCallback(async () => {
    try {
      const [st, settings] = await Promise.all([
        api.update.status(),
        api.settings.get().catch(() => null),
      ])
      setCliStatus(st)
      setSkipped(settings?.skippedUpdateVersion)
    } catch {
      /* ignore */
    }
  }, [])

  const refreshApp = useCallback(async () => {
    if (!appUpdaterEnabled) return
    try {
      const [st, settings] = await Promise.all([
        desktopAppUpdateStatus(),
        api.settings.get().catch(() => null),
      ])
      setAppStatus(st)
      setSkipped(settings?.skippedUpdateVersion)
    } catch {
      /* ignore */
    }
  }, [appUpdaterEnabled])

  useEffect(() => {
    void refreshCli()
    const id = window.setInterval(() => void refreshCli(), 60_000)
    return () => window.clearInterval(id)
  }, [refreshCli])

  useEffect(() => {
    if (!appUpdaterEnabled) return
    void refreshApp()
    const off = window.runtime?.EventsOn?.('update:app:status', (payload) => {
      setAppStatus(payload as UpdateStatus)
    })
    return () => off?.()
  }, [appUpdaterEnabled, refreshApp])

  useEffect(() => {
    const t = window.setTimeout(() => {
      api.settings
        .get()
        .then((s) => {
          if (s.autoCheckUpdates === false) return
          const opts = desktop ? { target: 'pathCli' as const } : undefined
          return api.update.check(opts).then(setCliStatus)
        })
        .catch(() => {})
    }, 4_000)
    return () => window.clearTimeout(t)
  }, [desktop])

  useEffect(() => {
    if (!appUpdaterEnabled) return
    const t = window.setTimeout(() => {
      api.settings
        .get()
        .then((s) => {
          if (s.autoCheckUpdates === false) return
          return checkDesktopAppUpdate(false).then(setAppStatus)
        })
        .catch(() => {})
    }, 6_000)
    return () => window.clearTimeout(t)
  }, [appUpdaterEnabled])

  const showCli = cliStatus && shouldShow(cliStatus, skipped, cliStatus.availableVersion ?? '')
  const showApp =
    appStatus &&
    shouldShow(appStatus, skipped, appStatus.availableVersion ? `app:${appStatus.availableVersion}` : '')

  if (!showCli && !showApp) {
    return null
  }

  const cliApplyTarget = desktop ? 'pathCli' : 'self'

  return (
    <div className="update-banners">
      {showCli && cliStatus && (
        <BannerRow
          track="cli"
          status={cliStatus}
          busy={cliBusy}
          onDownload={() => {
            setCliBusy(true)
            void api.update
              .download()
              .then(setCliStatus)
              .catch((e) =>
                setCliStatus((prev) =>
                  prev
                    ? { ...prev, state: 'error', error: e instanceof Error ? e.message : String(e) }
                    : prev,
                ),
              )
              .finally(() => setCliBusy(false))
          }}
          onApply={() => {
            setCliBusy(true)
            void api.update
              .apply(cliApplyTarget)
              .then((st) => {
                setCliStatus(st)
                if ((st.state === 'idle' || !st.error) && !desktop) {
                  window.setTimeout(() => window.location.reload(), 800)
                }
              })
              .catch((e) =>
                setCliStatus((prev) =>
                  prev
                    ? { ...prev, state: 'error', error: e instanceof Error ? e.message : String(e) }
                    : prev,
                ),
              )
              .finally(() => setCliBusy(false))
          }}
          onLater={() => {
            const ver = cliStatus.availableVersion
            if (!ver) return
            void api.update.skip(ver).then(() => setSkipped(ver))
          }}
          onDismiss={() => {
            setCliBusy(true)
            void api.update
              .dismiss()
              .then(setCliStatus)
              .catch(() =>
                setCliStatus((prev) => (prev ? { ...prev, state: 'idle', error: undefined } : prev)),
              )
              .finally(() => setCliBusy(false))
          }}
        />
      )}
      {showApp && appStatus && (
        <BannerRow
          track="app"
          status={appStatus}
          busy={appBusy}
          onDownload={() => {
            setAppBusy(true)
            void downloadDesktopAppUpdate()
              .then(setAppStatus)
              .finally(() => setAppBusy(false))
          }}
          onApply={() => {
            setAppBusy(true)
            void quitAndInstallDesktopApp()
              .then(setAppStatus)
              .finally(() => setAppBusy(false))
          }}
          onLater={() => {
            const ver = appStatus.availableVersion
            if (!ver) return
            void api.update.skip(`app:${ver}`).then(() => setSkipped(`app:${ver}`))
          }}
          onDismiss={() => {
            setAppBusy(true)
            void (async () => {
              try {
                const st = (await dismissDesktopAppUpdate()) ?? (await api.update.dismiss())
                setAppStatus(st)
              } catch {
                setAppStatus((prev) => (prev ? { ...prev, state: 'idle', error: undefined } : prev))
              } finally {
                setAppBusy(false)
              }
            })()
          }}
        />
      )}
    </div>
  )
}
