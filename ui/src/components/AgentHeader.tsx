// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { MoreHorizontal, PanelLeft, Plus } from 'lucide-react'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { useSidebar } from '@/components/ui/sidebar'
import { harnessLabel } from '@/lib/agentDisplay'
import { scrollToSidebarSession } from '@/lib/scrollToSidebarSession'
import { cn } from '@/lib/utils'
import type { AgentType } from '@/types'

interface Props {
  name: string
  agent: AgentType
  sessionId: string
  onNewSession: () => void
}

export function AgentHeader({ name, agent, sessionId, onNewSession }: Props) {
  const typeLabel = harnessLabel(agent.harness, agent.sandbox)
  const { isMobile, setOpen, setOpenMobile } = useSidebar()
  const newSessionLabel = `New ${agent.label} session`

  function handleShowInSidebar() {
    if (isMobile) {
      setOpenMobile(true)
    } else {
      setOpen(true)
    }
    window.requestAnimationFrame(() => {
      scrollToSidebarSession(sessionId)
    })
  }

  return (
    <div className="group/agent-header flex min-w-0 items-center gap-1">
      <Tooltip>
        <TooltipTrigger
          className="app-agent-header inline-flex h-7 min-w-0 max-w-[min(12rem,50vw)] items-center truncate text-sm font-medium leading-none text-muted-foreground transition-colors hover:text-foreground md:max-w-[min(24rem,40vw)]"
          aria-label={`${name}, ${agent.label}, ${typeLabel}${agent.description ? `, ${agent.description}` : ''}`}
        >
          {name}
        </TooltipTrigger>
        <TooltipContent
          side="bottom"
          align="start"
          className="flex max-w-xs flex-col items-start gap-1 px-3 py-2"
        >
          <span className="font-medium">{typeLabel}</span>
          {agent.description && (
            <span className="text-background/80 leading-snug">{agent.description}</span>
          )}
        </TooltipContent>
      </Tooltip>
      <span
        className={cn(
          'app-agent-header-actions group-hover/agent-header:md:w-12 group-hover/agent-header:md:opacity-100',
        )}
      >
        <button
          type="button"
          className="inline-flex size-6 shrink-0 items-center justify-center rounded-md text-muted-foreground hover:bg-muted hover:text-foreground"
          aria-label={newSessionLabel}
          title={newSessionLabel}
          onClick={onNewSession}
        >
          <Plus className="size-3.5" />
        </button>
        <DropdownMenu>
          <DropdownMenuTrigger
            render={
              <button
                type="button"
                className="inline-flex size-6 shrink-0 items-center justify-center rounded-md text-muted-foreground hover:bg-muted hover:text-foreground"
                aria-label="Session options"
              />
            }
          >
            <MoreHorizontal className="size-3.5" />
          </DropdownMenuTrigger>
          <DropdownMenuContent align="start" className="w-48">
            <DropdownMenuItem onClick={handleShowInSidebar}>
              <PanelLeft className="size-3.5 text-muted-foreground" />
              Show in sidebar
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </span>
    </div>
  )
}
