// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { Settings, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import { GeneralTab } from '@/components/customize/GeneralTab'
import { ExtensionsTab } from '@/components/customize/ExtensionsTab'
import { MCPServersTab } from '@/components/customize/MCPServersTab'
import { SkillsTab } from '@/components/customize/SkillsTab'
import { MemoryTab } from '@/components/customize/MemoryTab'
import { AgentsTab } from '@/components/customize/AgentsTab'

const TABS = [
  { id: 'general', label: 'General' },
  { id: 'extensions', label: 'Extensions' },
  { id: 'mcp', label: 'MCP servers' },
  { id: 'skills', label: 'Skills' },
  { id: 'memory', label: 'Memory' },
  { id: 'agents', label: 'Agents' },
] as const

type TabId = (typeof TABS)[number]['id']

export type CustomizeTabId = TabId

interface Props {
  onClose: () => void
  onAgentTypesChanged?: () => void
  tab: TabId
  onTabChange: (tab: TabId) => void
}

export function CustomizePanel({ onClose, onAgentTypesChanged, tab, onTabChange }: Props) {
  return (
    <div className="customize-panel flex flex-1 flex-col overflow-hidden">
      <div className="conversation-header justify-between">
        <div className="flex items-center gap-2 min-w-0">
          <Settings className="size-4 shrink-0 text-muted-foreground" />
          <h1 className="text-sm font-semibold truncate">Customize</h1>
        </div>
        <Button variant="ghost" size="sm" onClick={onClose} aria-label="Close customize panel">
          <X className="size-4" />
        </Button>
      </div>

      <div className="flex flex-1 min-h-0 flex-col md:flex-row">
        <nav className="customize-tabs shrink-0 flex flex-row gap-1 overflow-x-auto border-b bg-muted/20 p-2 md:w-44 md:flex-col md:overflow-x-visible md:border-b-0 md:border-r">
          {TABS.map((item) => (
            <button
              key={item.id}
              type="button"
              className={cn('customize-tab-btn', tab === item.id && 'customize-tab-btn--active')}
              onClick={() => onTabChange(item.id)}
            >
              {item.label}
            </button>
          ))}
        </nav>

        <div className="flex-1 overflow-y-auto p-4 md:p-6">
          {tab === 'general' && <GeneralTab />}
          {tab === 'extensions' && <ExtensionsTab onChanged={onAgentTypesChanged} />}
          {tab === 'mcp' && <MCPServersTab />}
          {tab === 'skills' && <SkillsTab />}
          {tab === 'memory' && <MemoryTab />}
          {tab === 'agents' && <AgentsTab onChanged={onAgentTypesChanged} />}
        </div>
      </div>
    </div>
  )
}

interface TriggerProps {
  active: boolean
  onOpen: () => void
  compact?: boolean
}

export function CustomizeTrigger({ active, onOpen, compact = false }: TriggerProps) {
  if (compact) {
    return (
      <Button
        type="button"
        variant={active ? 'secondary' : 'ghost'}
        size="icon-sm"
        onClick={onOpen}
        aria-pressed={active}
        aria-label="Customize"
      >
        <Settings className="size-4" />
      </Button>
    )
  }

  return (
    <Button
      variant={active ? 'secondary' : 'ghost'}
      size="sm"
      className="w-full justify-start gap-2"
      onClick={onOpen}
    >
      <Settings className="size-4 shrink-0" />
      <span className="group-data-[collapsible=icon]:hidden">Customize</span>
    </Button>
  )
}
