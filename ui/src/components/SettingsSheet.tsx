// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useEffect, useState } from 'react'
import { Settings, TriangleAlert } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from '@/components/ui/sheet'
import { useTheme } from '@/contexts/theme'
import { api } from '@/api'
import type { Capabilities } from '@/types'

export function SettingsSheet() {
  const { theme, setTheme } = useTheme()
  const [capabilities, setCapabilities] = useState<Capabilities | null>(null)

  useEffect(() => {
    api.capabilities.get()
      .then(setCapabilities)
      .catch(() => {})
  }, [])

  const bwrapUnavailable = capabilities !== null && !capabilities.sandbox.bwrap.available

  return (
    <Sheet>
      <SheetTrigger asChild>
        <Button variant="ghost" size="sm" className="w-full justify-start gap-2">
          <Settings className="size-4 shrink-0" />
          <span className="group-data-[collapsible=icon]:hidden">Settings</span>
        </Button>
      </SheetTrigger>
      <SheetContent side="right" className="w-72">
        <SheetHeader>
          <SheetTitle>Settings</SheetTitle>
        </SheetHeader>
        <div className="mt-6 space-y-4 px-1">
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
            <p className="text-sm font-medium mb-3">Theme</p>
            <div className="flex gap-2">
              <button
                onClick={() => setTheme('light')}
                className="theme-btn"
                data-active={theme === 'light'}
              >
                Light
              </button>
              <button
                onClick={() => setTheme('dark')}
                className="theme-btn"
                data-active={theme === 'dark'}
              >
                Dark
              </button>
            </div>
          </div>
        </div>
      </SheetContent>
    </Sheet>
  )
}
