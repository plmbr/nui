// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useState } from 'react'
import { SlidersHorizontal, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import { GeneralTab } from '@/components/customize/GeneralTab'
import { ExtensionsTab } from '@/components/customize/ExtensionsTab'
import { MCPServersTab } from '@/components/customize/MCPServersTab'
import { SkillsTab } from '@/components/customize/SkillsTab'
import { AgentsTab } from '@/components/customize/AgentsTab'

const TABS = [
  { id: 'general', label: 'General' },
  { id: 'extensions', label: 'Extensions' },
  { id: 'mcp', label: 'MCP servers' },
  { id: 'skills', label: 'Skills' },
  { id: 'agents', label: 'Agents' },
] as const

type TabId = (typeof TABS)[number]['id']

interface Props {
  onClose: () => void
  onExtensionsChanged?: () => void
}

export function CustomizePanel({ onClose, onExtensionsChanged }: Props) {
  const [tab, setTab] = useState<TabId>('general')

  return (
    <div className="customize-panel flex flex-1 flex-col overflow-hidden">
      <div className="conversation-header justify-between">
        <div className="flex items-center gap-2 min-w-0">
          <SlidersHorizontal className="size-4 shrink-0 text-muted-foreground" />
          <h1 className="text-sm font-semibold truncate">Customize</h1>
        </div>
        <Button variant="ghost" size="sm" onClick={onClose} aria-label="Close customize panel">
          <X className="size-4" />
        </Button>
      </div>

      <div className="flex flex-1 min-h-0">
        <nav className="customize-tabs shrink-0 border-r bg-muted/20 p-2 w-44">
          {TABS.map((item) => (
            <button
              key={item.id}
              type="button"
              className={cn('customize-tab-btn', tab === item.id && 'customize-tab-btn--active')}
              onClick={() => setTab(item.id)}
            >
              {item.label}
            </button>
          ))}
        </nav>

        <div className="flex-1 overflow-y-auto p-6">
          {tab === 'general' && <GeneralTab />}
          {tab === 'extensions' && <ExtensionsTab onChanged={onExtensionsChanged} />}
          {tab === 'mcp' && <MCPServersTab />}
          {tab === 'skills' && <SkillsTab />}
          {tab === 'agents' && <AgentsTab />}
        </div>
      </div>
    </div>
  )
}

interface TriggerProps {
  active: boolean
  onOpen: () => void
}

export function CustomizeTrigger({ active, onOpen }: TriggerProps) {
  return (
    <Button
      variant={active ? 'secondary' : 'ghost'}
      size="sm"
      className="w-full justify-start gap-2"
      onClick={onOpen}
    >
      <SlidersHorizontal className="size-4 shrink-0" />
      <span className="group-data-[collapsible=icon]:hidden">Customize</span>
    </Button>
  )
}
