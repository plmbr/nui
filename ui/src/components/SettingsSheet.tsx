// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useEffect, useState } from 'react'
import { Settings, TriangleAlert } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from '@/components/ui/sheet'
import { useTheme } from '@/contexts/theme'
import { api } from '@/api'
import type { AgentType, Capabilities } from '@/types'

export function SettingsSheet() {
  const { theme, setTheme } = useTheme()
  const [capabilities, setCapabilities] = useState<Capabilities | null>(null)
  const [agentTypes, setAgentTypes] = useState<AgentType[]>([])
  const [defaultAgentType, setDefaultAgentType] = useState('')

  useEffect(() => {
    Promise.all([
      api.capabilities.get(),
      api.agentTypes.list(),
      api.settings.get().catch(() => ({ theme: 'light' as const })),
    ])
      .then(([caps, types, settings]) => {
        setCapabilities(caps)
        setAgentTypes(types)
        const preferred = settings.defaultAgentType
          ? types.find((t) => t.id === settings.defaultAgentType)
          : undefined
        const fallback = types.find((t) => t.available) ?? types[0]
        setDefaultAgentType(preferred?.id ?? fallback?.id ?? '')
      })
      .catch(() => {})
  }, [])

  const handleDefaultAgentChange = (id: string | null) => {
    if (!id) return
    setDefaultAgentType(id)
    api.settings.update({ defaultAgentType: id }).catch(() => {})
  }

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
            <p className="text-sm font-medium mb-3">Default agent</p>
            <p className="text-xs text-muted-foreground mb-2">
              Used when Loop creates a session on startup.
            </p>
            {agentTypes.length > 0 && (
              <Select value={defaultAgentType} onValueChange={handleDefaultAgentChange}>
                <SelectTrigger className="w-full">
                  <SelectValue placeholder="Select agent" />
                </SelectTrigger>
                <SelectContent>
                  {agentTypes.some((a) => a.isBuiltin) && (
                    <SelectGroup>
                      <SelectLabel>Built-in</SelectLabel>
                      {agentTypes.filter((a) => a.isBuiltin).map((a) => (
                        <SelectItem key={a.id} value={a.id} disabled={!a.available}>
                          {a.label}
                        </SelectItem>
                      ))}
                    </SelectGroup>
                  )}
                  {agentTypes.some((a) => !a.isBuiltin) && (
                    <SelectGroup>
                      <SelectLabel>Custom</SelectLabel>
                      {agentTypes.filter((a) => !a.isBuiltin).map((a) => (
                        <SelectItem key={a.id} value={a.id} disabled={!a.available}>
                          {a.label}
                        </SelectItem>
                      ))}
                    </SelectGroup>
                  )}
                </SelectContent>
              </Select>
            )}
          </div>
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
