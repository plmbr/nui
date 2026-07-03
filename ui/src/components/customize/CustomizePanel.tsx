// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { Settings, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import { GeneralTab } from '@/components/customize/GeneralTab'
import { ExtensionsTab } from '@/components/customize/ExtensionsTab'
import { MCPServersTab } from '@/components/customize/MCPServersTab'
import { SkillsTab } from '@/components/customize/SkillsTab'
import { AgentsTab } from '@/components/customize/AgentsTab'
import { SchedulesTab } from '@/components/customize/SchedulesTab'
import type { AgentType } from '@/types'

const TABS = [
  { id: 'general', label: 'General' },
  { id: 'extensions', label: 'Extensions' },
  { id: 'mcp', label: 'MCP servers' },
  { id: 'skills', label: 'Skills' },
  { id: 'agents', label: 'Agents' },
  { id: 'schedules', label: 'Schedules' },
] as const

type TabId = (typeof TABS)[number]['id']

export type CustomizeTabId = TabId

interface Props {
  onClose: () => void
  onAgentTypesChanged?: () => void
  agentTypes: AgentType[]
  tab: TabId
  onTabChange: (tab: TabId) => void
}

export function CustomizePanel({ onClose, onAgentTypesChanged, agentTypes, tab, onTabChange }: Props) {
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

      <div className="flex flex-1 min-h-0">
        <nav className="customize-tabs shrink-0 border-r bg-muted/20 p-2 w-44">
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

        <div className="flex-1 overflow-y-auto p-6">
          {tab === 'general' && <GeneralTab />}
          {tab === 'extensions' && <ExtensionsTab onChanged={onAgentTypesChanged} />}
          {tab === 'mcp' && <MCPServersTab />}
          {tab === 'skills' && <SkillsTab />}
          {tab === 'agents' && <AgentsTab onChanged={onAgentTypesChanged} />}
          {tab === 'schedules' && <SchedulesTab agentTypes={agentTypes} />}
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
