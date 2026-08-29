// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import type { UpdateStatus } from '@/types'

type DesktopAppBridge = {
  CheckForAppUpdate: (force: boolean) => Promise<UpdateStatus>
  DownloadAppUpdate: () => Promise<UpdateStatus>
  AppUpdateStatus: () => Promise<UpdateStatus>
  QuitAndInstall: () => Promise<UpdateStatus>
}

function bridge(): DesktopAppBridge | null {
  if (typeof window === 'undefined' || !window.__NUI_DESKTOP__) return null
  const app = window.go?.main?.App
  if (!app?.CheckForAppUpdate) return null
  return app
}

export function hasDesktopAppUpdater(): boolean {
  return bridge() != null
}

export async function checkDesktopAppUpdate(force = false): Promise<UpdateStatus | null> {
  const b = bridge()
  if (!b) return null
  return b.CheckForAppUpdate(force)
}

export async function downloadDesktopAppUpdate(): Promise<UpdateStatus | null> {
  const b = bridge()
  if (!b) return null
  return b.DownloadAppUpdate()
}

export async function desktopAppUpdateStatus(): Promise<UpdateStatus | null> {
  const b = bridge()
  if (!b) return null
  return b.AppUpdateStatus()
}

export async function quitAndInstallDesktopApp(): Promise<UpdateStatus | null> {
  const b = bridge()
  if (!b) return null
  return b.QuitAndInstall()
}

declare global {
  interface Window {
    go?: {
      main?: {
        App?: DesktopAppBridge
      }
    }
    runtime?: {
      EventsOn?: (event: string, callback: (...args: unknown[]) => void) => () => void
      EventsOff?: (event: string) => void
    }
  }
}
