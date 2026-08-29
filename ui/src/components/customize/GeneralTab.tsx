// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useEffect, useMemo, useState } from 'react'
import { TriangleAlert } from 'lucide-react'
import { SearchableSelect } from '@/components/SearchableSelect'
import { api } from '@/api'
import { useTheme } from '@/contexts/theme'
import {
  pickDefaultAgentTypeId,
  pickDefaultHarnessRef,
  selectableAgentTypes,
  selectableHarnessRefs,
} from '@/lib/agentTypes'
import { BUILTIN_AGENTS_LABEL, INSTALLED_AGENTS_LABEL } from '@/lib/sessionGroups'
import { UI_THEME_LIST } from '@/lib/uiThemes'
import type { AgentType, Capabilities, UIThemeId, UpdateStatus } from '@/types'
import {
  checkDesktopAppUpdate,
  downloadDesktopAppUpdate,
  hasDesktopAppUpdater,
  quitAndInstallDesktopApp,
} from '@/lib/desktopUpdate'

function isDesktopShell(): boolean {
  return typeof window !== 'undefined' && !!window.__NUI_DESKTOP__
}

function DesktopAppUpdateControls() {
  const [busy, setBusy] = useState(false)
  const [msg, setMsg] = useState('')
  const [st, setSt] = useState<UpdateStatus | null>(null)
  if (!hasDesktopAppUpdater()) return null

  return (
    <div className="mt-4 space-y-2">
      <p className="text-xs text-muted-foreground">Desktop app</p>
      <div className="flex flex-wrap items-center gap-2">
        <button
          type="button"
          className="rounded-md border border-border px-3 py-1.5 text-sm hover:bg-muted disabled:opacity-50"
          disabled={busy}
          onClick={() => {
            setBusy(true)
            setMsg('')
            void checkDesktopAppUpdate(false)
              .then((s) => {
                setSt(s)
                if (s?.state === 'upToDate') setMsg(`App up to date (${s.currentVersion})`)
                else if (s?.state === 'available') setMsg(`App update: ${s.availableVersion}`)
                else if (s?.error) setMsg(s.error)
              })
              .finally(() => setBusy(false))
          }}
        >
          Check for app updates
        </button>
        {st?.state === 'available' && (
          <button
            type="button"
            className="rounded-md border border-border px-3 py-1.5 text-sm hover:bg-muted disabled:opacity-50"
            disabled={busy}
            onClick={() => {
              setBusy(true)
              void downloadDesktopAppUpdate()
                .then(setSt)
                .finally(() => setBusy(false))
            }}
          >
            Download app
          </button>
        )}
        {st?.state === 'ready' && (
          <button
            type="button"
            className="rounded-md border border-border px-3 py-1.5 text-sm hover:bg-muted disabled:opacity-50"
            disabled={busy}
            onClick={() => {
              setBusy(true)
              void quitAndInstallDesktopApp()
                .then(setSt)
                .finally(() => setBusy(false))
            }}
          >
            Install & restart
          </button>
        )}
      </div>
      {msg && <p className="text-xs text-muted-foreground">{msg}</p>}
    </div>
  )
}

export function GeneralTab() {
  const { uiTheme, setUITheme } = useTheme()
  const [capabilities, setCapabilities] = useState<Capabilities | null>(null)
  const [agentTypes, setAgentTypes] = useState<AgentType[]>([])
  const [defaultAgentType, setDefaultAgentType] = useState('')
  const [defaultHarness, setDefaultHarness] = useState('')
  const [disableSloganAnimation, setDisableSloganAnimation] = useState(false)
  const [autoCheckUpdates, setAutoCheckUpdates] = useState(true)
  const [updateStatus, setUpdateStatus] = useState<UpdateStatus | null>(null)
  const [updateBusy, setUpdateBusy] = useState(false)
  const [updateMessage, setUpdateMessage] = useState('')
  const desktop = isDesktopShell()

  useEffect(() => {
    Promise.all([
      api.capabilities.get(),
      api.agentTypes.list(),
      api.settings.get().catch(() => ({ theme: 'light' as const })),
      api.update.status().catch(() => null),
    ])
      .then(([caps, types, settings, st]) => {
        setCapabilities(caps)
        setAgentTypes(types)
        setDefaultAgentType(pickDefaultAgentTypeId(types, settings.defaultAgentType))
        setDefaultHarness(pickDefaultHarnessRef(types, settings.defaultHarness))
        setDisableSloganAnimation(settings.disableSloganAnimation ?? false)
        setAutoCheckUpdates(settings.autoCheckUpdates !== false)
        if (st) setUpdateStatus(st)
      })
      .catch(() => {})
  }, [])

  const handleDefaultAgentChange = (id: string) => {
    setDefaultAgentType(id)
    api.settings.update({ defaultAgentType: id }).catch(() => {})
  }

  const handleDefaultHarnessChange = (ref: string) => {
    setDefaultHarness(ref)
    api.settings.update({ defaultHarness: ref }).catch(() => {})
  }

  const handleUIThemeChange = (id: UIThemeId) => {
    setUITheme(id)
  }

  const handleDisableSloganAnimationChange = (disabled: boolean) => {
    setDisableSloganAnimation(disabled)
    api.settings.update({ disableSloganAnimation: disabled }).catch(() => {})
  }

  const handleAutoCheckChange = (enabled: boolean) => {
    setAutoCheckUpdates(enabled)
    api.settings.update({ autoCheckUpdates: enabled }).catch(() => {})
  }

  const handleCheckUpdates = async () => {
    setUpdateBusy(true)
    setUpdateMessage('')
    try {
      const st = await api.update.check(desktop ? { target: 'pathCli' } : undefined)
      setUpdateStatus(st)
      if (st.state === 'upToDate') {
        setUpdateMessage(`Up to date (${st.currentVersion})`)
      } else if (st.state === 'available') {
        setUpdateMessage(`Update available: ${st.availableVersion}`)
      } else if (st.error) {
        setUpdateMessage(st.error)
      }
    } catch (e) {
      setUpdateMessage(e instanceof Error ? e.message : String(e))
    } finally {
      setUpdateBusy(false)
    }
  }

  const handleDownload = async () => {
    setUpdateBusy(true)
    try {
      const st = await api.update.download()
      setUpdateStatus(st)
      if (st.error) setUpdateMessage(st.error)
    } catch (e) {
      setUpdateMessage(e instanceof Error ? e.message : String(e))
    } finally {
      setUpdateBusy(false)
    }
  }

  const handleInstall = async () => {
    setUpdateBusy(true)
    try {
      const st = await api.update.apply(desktop ? 'pathCli' : 'self')
      setUpdateStatus(st)
      if (st.error) {
        setUpdateMessage(st.error)
      } else {
        setUpdateMessage(
          desktop
            ? `CLI installed (${st.currentVersion})`
            : `Installed ${st.currentVersion} — reload the page`,
        )
      }
    } catch (e) {
      setUpdateMessage(e instanceof Error ? e.message : String(e))
    } finally {
      setUpdateBusy(false)
    }
  }

  const bwrapUnavailable = capabilities !== null && !capabilities.sandbox.bwrap.available

  const selectableAgentTypesList = selectableAgentTypes(agentTypes)
  const agentSelectItems = useMemo(
    () =>
      selectableAgentTypesList.map((agent) => ({
        id: agent.id,
        label: agent.label,
        group: agent.isBuiltin ? BUILTIN_AGENTS_LABEL : INSTALLED_AGENTS_LABEL,
        description: agent.description,
      })),
    [selectableAgentTypesList],
  )

  const harnessSelectItems = useMemo(
    () =>
      selectableHarnessRefs(agentTypes).map((h) => ({
        id: h.ref,
        label: h.label,
        group: h.group,
      })),
    [agentTypes],
  )

  return (
    <div className="customize-tab-content space-y-6">
      {bwrapUnavailable && (
        <div className="sandbox-warning">
          <TriangleAlert className="size-4 shrink-0 mt-0.5" />
          <div>
            <p className="font-medium">Sandboxing unavailable</p>
            <p className="text-xs mt-0.5 opacity-80">
              {capabilities!.sandbox.bwrap.error ?? 'bubblewrap (bwrap) not found'}
            </p>
          </div>
        </div>
      )}
      <div>
        <p className="text-sm font-medium mb-1">Appearance</p>
        <div className="mt-4">
          <p className="text-sm font-medium mb-2">Theme</p>
          <p className="text-xs text-muted-foreground mb-3">
            Use the header toggle for light and dark when the theme supports both.
          </p>
          <div className="flex flex-wrap gap-4">
            {UI_THEME_LIST.map((def) => {
              const modeLabel =
                def.modes.length === 2
                  ? 'Light & dark'
                  : def.modes[0] === 'dark'
                    ? 'Dark only'
                    : 'Light only'
              return (
                <label key={def.id} className="flex items-center gap-2 text-sm cursor-pointer">
                  <input
                    type="checkbox"
                    className="size-4 rounded border-input"
                    checked={uiTheme === def.id}
                    onChange={() => handleUIThemeChange(def.id)}
                  />
                  <span>
                    {def.label}
                    <span className="text-muted-foreground"> · {modeLabel}</span>
                  </span>
                </label>
              )
            })}
          </div>
        </div>
        <label className="flex items-center gap-2 text-sm cursor-pointer mt-4">
          <input
            type="checkbox"
            className="size-4 rounded border-input"
            checked={disableSloganAnimation}
            onChange={(e) => handleDisableSloganAnimationChange(e.target.checked)}
          />
          Disable slogan animation
        </label>
      </div>
      <div>
        <p className="text-sm font-medium mb-1">Updates</p>
        <p className="text-xs text-muted-foreground mb-3">
          {desktop
            ? 'Checks GitHub Releases for a newer CLI on your PATH. App updates are separate.'
            : 'Checks GitHub Releases for a newer nui CLI/server. Downloads require confirmation.'}
        </p>
        <label className="flex items-center gap-2 text-sm cursor-pointer mb-3">
          <input
            type="checkbox"
            className="size-4 rounded border-input"
            checked={autoCheckUpdates}
            onChange={(e) => handleAutoCheckChange(e.target.checked)}
          />
          Automatically check for updates
        </label>
        <div className="flex flex-wrap items-center gap-2">
          <button
            type="button"
            className="rounded-md border border-border px-3 py-1.5 text-sm hover:bg-muted disabled:opacity-50"
            disabled={updateBusy}
            onClick={() => void handleCheckUpdates()}
          >
            Check for updates
          </button>
          {updateStatus?.state === 'available' && (
            <button
              type="button"
              className="rounded-md border border-border px-3 py-1.5 text-sm hover:bg-muted disabled:opacity-50"
              disabled={updateBusy}
              onClick={() => void handleDownload()}
            >
              Download {updateStatus.availableVersion}
            </button>
          )}
          {updateStatus?.state === 'ready' && (
            <button
              type="button"
              className="rounded-md border border-border px-3 py-1.5 text-sm hover:bg-muted disabled:opacity-50"
              disabled={updateBusy}
              onClick={() => void handleInstall()}
            >
              {desktop ? 'Install CLI' : 'Install'}
            </button>
          )}
        </div>
        {updateMessage && (
          <p className="text-xs text-muted-foreground mt-2">{updateMessage}</p>
        )}
        {updateStatus?.state === 'downloading' && (
          <p className="text-xs text-muted-foreground mt-2">
            Downloading… {Math.round((updateStatus.progress ?? 0) * 100)}%
          </p>
        )}
        {desktop && (
          <DesktopAppUpdateControls />
        )}
      </div>
      <div>
        <p className="text-sm font-medium mb-1">Default harness</p>
        <p className="text-xs text-muted-foreground mb-3">
          Used by the nui master agent (launcher orchestrator).
        </p>
        {harnessSelectItems.length > 0 && (
          <SearchableSelect
            value={defaultHarness}
            onValueChange={handleDefaultHarnessChange}
            items={harnessSelectItems}
            placeholder="Select harness"
            searchPlaceholder="Search harnesses…"
            triggerClassName="max-w-md"
          />
        )}
      </div>
      <div>
        <p className="text-sm font-medium mb-1">Default agent</p>
        <p className="text-xs text-muted-foreground mb-3">
          Used when nui creates a session on startup.
        </p>
        {selectableAgentTypesList.length > 0 && (
          <SearchableSelect
            value={defaultAgentType}
            onValueChange={handleDefaultAgentChange}
            items={agentSelectItems}
            placeholder="Select agent"
            searchPlaceholder="Search agents…"
            triggerClassName="max-w-md"
          />
        )}
      </div>
    </div>
  )
}
