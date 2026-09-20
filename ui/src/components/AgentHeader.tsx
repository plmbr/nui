// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useEffect, useState, useSyncExternalStore } from 'react'
import { Loader2, MoreHorizontal, PanelLeft, Pencil, Plus } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Input } from '@/components/ui/input'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { useSidebar } from '@/components/ui/sidebar'
import { harnessLabel } from '@/lib/agentDisplay'
import { scrollToSidebarSession } from '@/lib/scrollToSidebarSession'
import {
  getRunningSessionsSnapshot,
  subscribeSessionRuns,
} from '@/lib/sessionChatStore'
import { cn } from '@/lib/utils'
import type { AgentType } from '@/types'

interface Props {
  name: string
  sessionName: string
  agent: AgentType
  sessionId: string
  onNewSession: () => void
  onRename: (newName: string) => Promise<void>
}

export function AgentHeader({ name, sessionName, agent, sessionId, onNewSession, onRename }: Props) {
  const typeLabel = harnessLabel(agent.harness, agent.sandbox)
  const { isMobile, setOpen, setOpenMobile } = useSidebar()
  const newSessionLabel = `New ${agent.label} session`
  const runningSnapshot = useSyncExternalStore(
    subscribeSessionRuns,
    getRunningSessionsSnapshot,
    getRunningSessionsSnapshot,
  )
  const isRunning = runningSnapshot.split(',').includes(sessionId)
  const [renameOpen, setRenameOpen] = useState(false)
  const [nameValue, setNameValue] = useState(sessionName)

  useEffect(() => {
    setNameValue(sessionName)
  }, [sessionName])

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

  async function saveRename() {
    const trimmed = nameValue.trim()
    setRenameOpen(false)
    if (trimmed && trimmed !== sessionName) {
      await onRename(trimmed)
    }
  }

  return (
    <>
      <div className="group/agent-header flex min-w-0 items-center gap-1">
        <Tooltip>
          <TooltipTrigger
            className="app-agent-header inline-flex h-7 min-w-0 max-w-[min(18rem,70vw)] items-center gap-2 truncate text-sm leading-none transition-colors hover:text-foreground md:max-w-[min(32rem,50vw)]"
            aria-label={`${agent.label}, ${name}, ${typeLabel}${agent.description ? `, ${agent.description}` : ''}`}
          >
            <span className="shrink-0 font-medium text-muted-foreground">{agent.label}</span>
            <span className="shrink-0 select-none text-muted-foreground/35" aria-hidden="true">/</span>
            <span className="min-w-0 truncate font-medium text-muted-foreground">{name}</span>
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
        {isRunning && (
          <span className="sidebar-session__status" role="status" aria-label="Running">
            <span className="flex size-3.5 items-center justify-center animate-spin" aria-hidden>
              <Loader2 className="size-3.5 text-muted-foreground" />
            </span>
          </span>
        )}
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
              <DropdownMenuItem onClick={() => { setNameValue(sessionName); setRenameOpen(true) }}>
                <Pencil className="size-3.5 text-muted-foreground" />
                Rename
              </DropdownMenuItem>
              <DropdownMenuItem onClick={handleShowInSidebar}>
                <PanelLeft className="size-3.5 text-muted-foreground" />
                Show in sidebar
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </span>
      </div>

      <Dialog open={renameOpen} onOpenChange={setRenameOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Rename session</DialogTitle>
          </DialogHeader>
          <Input
            value={nameValue}
            autoFocus
            onChange={(e) => setNameValue(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') void saveRename()
              if (e.key === 'Escape') setRenameOpen(false)
            }}
          />
          <DialogFooter>
            <Button variant="outline" onClick={() => setRenameOpen(false)}>Cancel</Button>
            <Button onClick={() => void saveRename()}>Rename</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
