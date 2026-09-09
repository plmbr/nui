// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useEffect, useState } from 'react'
import { ExternalLink } from 'lucide-react'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { NuiLogo } from '@/components/NuiLogo'

const SHOW_ABOUT_EVENT = 'nui:show-about'
const DEFAULT_WEBSITE_URL = 'https://nui.plmbr.dev'

type AboutInfo = {
  appVersion?: string
  cliVersion?: string
  websiteURL?: string
}

function openExternal(url: string) {
  if (window.runtime?.BrowserOpenURL) {
    window.runtime.BrowserOpenURL(url)
    return
  }
  window.open(url, '_blank', 'noopener,noreferrer')
}

/** Desktop About dialog: macOS app menu (About nui) or Help → About nui. */
export function AboutDialog() {
  const [open, setOpen] = useState(false)
  const [info, setInfo] = useState<AboutInfo>({})

  useEffect(() => {
    if (!window.__NUI_DESKTOP__) return
    const off = window.runtime?.EventsOn?.(SHOW_ABOUT_EVENT, (...args: unknown[]) => {
      const payload = (args[0] ?? {}) as AboutInfo
      setInfo(payload)
      setOpen(true)
    })
    return () => {
      off?.()
    }
  }, [])

  if (!window.__NUI_DESKTOP__) return null

  const websiteURL = info.websiteURL?.trim() || DEFAULT_WEBSITE_URL
  const appVersion = info.appVersion?.trim() || 'dev'
  const cliVersion = info.cliVersion?.trim() || 'unavailable'
  const websiteLabel = websiteURL.replace(/^https?:\/\//, '')

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogContent
        className="about-dialog sm:max-w-sm"
        initialFocus={false}
      >
        <DialogHeader className="items-center text-center sm:text-center">
          <NuiLogo className="about-dialog__logo" decorative />
          <DialogTitle className="sr-only">About nui</DialogTitle>
          <DialogDescription>Self-hosted AI agent sessions</DialogDescription>
        </DialogHeader>

        <dl className="about-dialog__versions">
          <div className="about-dialog__row">
            <dt>App</dt>
            <dd>{appVersion}</dd>
          </div>
          <div className="about-dialog__row">
            <dt>CLI</dt>
            <dd>{cliVersion}</dd>
          </div>
        </dl>

        <a
          href={websiteURL}
          className="about-dialog__link"
          onClick={(event) => {
            event.preventDefault()
            openExternal(websiteURL)
          }}
        >
          {websiteLabel}
          <ExternalLink className="size-3.5 shrink-0 opacity-70" aria-hidden />
        </a>
      </DialogContent>
    </Dialog>
  )
}
