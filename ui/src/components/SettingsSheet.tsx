// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { Settings } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from '@/components/ui/sheet'
import { useTheme } from '@/contexts/theme'

export function SettingsSheet() {
  const { theme, setTheme } = useTheme()

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
