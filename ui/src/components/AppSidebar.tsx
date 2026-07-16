// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useEffect, useMemo, useState, useSyncExternalStore } from 'react'
import { ChevronRight, CalendarClock, Filter, List, Loader2, MoreHorizontal, Pencil, Plus, Square, Trash2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { cn } from '@/lib/utils'
import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuAction,
  SidebarMenuButton,
  SidebarMenuItem,
  useSidebar,
} from '@/components/ui/sidebar'
import { groupSessionsByAgentType, type SessionGroup } from '@/lib/sessionGroups'
import {
  getRunningProgressSnapshot,
  getRunningSessionsSnapshot,
  getSessionProgress,
  stopSessionRun,
  subscribeRunningProgress,
  subscribeSessionRuns,
} from '@/lib/sessionChatStore'
import { decodeSessionProgress, type SessionProgress } from '@/lib/sessionProgress'
import { formatExactTime, formatRelativeTime } from '@/lib/formatRelativeTime'
import { sessionDisplayName } from '@/lib/sessionDisplay'
import { ConfirmDeleteDialog } from '@/components/ConfirmDeleteDialog'
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
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import type { AgentType, Session } from '@/types'
import { SidebarResizeHandle } from '@/components/SidebarResizeHandle'

interface DisplaySessionGroup extends SessionGroup {
  totalSessions: number
}

interface Props {
  sessions: Session[]
  agentTypes: AgentType[]
  selectedId: string | null
  newSessionOpen: boolean
  schedulesPanelOpen: boolean
  sessionListGroupId: string | null
  sidebarWidth: number
  onSidebarWidthChange: (width: number) => void
  onSidebarWidthCommit: (width: number) => void
  onSelect: (id: string) => void
  onOpenNewSession: () => void
  onOpenSchedules: () => void
  onOpenNewSessionForGroup: (groupId: string) => void
  onOpenSessionList: (groupId: string) => void
  onRename: (id: string, newName: string) => Promise<void>
  onDelete: (id: string) => Promise<void>
}

interface SessionListItemProps {
  session: Session
  isActive: boolean
  onSelect: () => void
  onRename: (newName: string) => Promise<void>
  onDelete: () => Promise<void>
}

function useRunningSessions(): Set<string> {
  const snapshot = useSyncExternalStore(
    subscribeSessionRuns,
    getRunningSessionsSnapshot,
    getRunningSessionsSnapshot,
  )
  return useMemo(
    () => new Set(snapshot ? snapshot.split(',').filter(Boolean) : []),
    [snapshot],
  )
}

function parseProgressSnapshot(snapshot: string): Map<string, SessionProgress> {
  const map = new Map<string, SessionProgress>()
  if (!snapshot) return map
  for (const part of snapshot.split('|')) {
    const eq = part.indexOf('=')
    if (eq <= 0) continue
    const id = part.slice(0, eq)
    const progress = decodeSessionProgress(part.slice(eq + 1))
    if (progress) map.set(id, progress)
  }
  return map
}

function useSessionProgressMap(): Map<string, SessionProgress> {
  const snapshot = useSyncExternalStore(
    subscribeRunningProgress,
    getRunningProgressSnapshot,
    getRunningProgressSnapshot,
  )
  return useMemo(() => parseProgressSnapshot(snapshot), [snapshot])
}

function SessionListItem({ session, isActive, onSelect, onRename, onDelete }: SessionListItemProps) {
  const [renameOpen, setRenameOpen] = useState(false)
  const [nameValue, setNameValue] = useState(session.name)
  const [deleteOpen, setDeleteOpen] = useState(false)
  const runningSessions = useRunningSessions()
  const progressMap = useSessionProgressMap()
  const sessionRunning = runningSessions.has(session.id)
  const progress = progressMap.get(session.id) ?? getSessionProgress(session.id)
  const displayName = sessionDisplayName(session)
  const lastActivityAt = session.lastRunAt ?? session.createdAt
  const timeLabel = lastActivityAt
    ? (sessionRunning ? 'now' : formatRelativeTime(lastActivityAt, Date.now(), false))
    : ''
  const timeTitle = lastActivityAt ? formatExactTime(lastActivityAt) : ''
  const scheduledTitle = session.scheduleId
    ? `Scheduled · ${session.scheduleName || session.scheduleId}`
    : undefined

  useEffect(() => {
    setNameValue(session.name)
  }, [session.name])

  async function saveRename() {
    const trimmed = nameValue.trim()
    setRenameOpen(false)
    if (trimmed && trimmed !== session.name) {
      await onRename(trimmed)
    }
  }

  return (
    <>
      <SidebarMenuItem data-sidebar-session-id={session.id}>
        <SidebarMenuButton
          isActive={isActive}
          onClick={onSelect}
          title={
            timeTitle
              ? `${displayName} · ${timeTitle}${progress ? ` — ${progress.label}` : ''}`
              : progress
                ? `${displayName} — ${progress.label}`
                : displayName
          }
          className="sidebar-session__button"
        >
          {session.scheduleId && (
            <span className="sidebar-session__scheduled shrink-0" title={scheduledTitle}>
              <CalendarClock className="sidebar-session__scheduled-icon" aria-hidden />
            </span>
          )}
          <span className="sidebar-session__name truncate" title={displayName}>
            {displayName}
          </span>
          {timeLabel && (
            <span className="sidebar-meta-rail sidebar-session__time" title={timeTitle}>
              {timeLabel}
            </span>
          )}
          {sessionRunning && (
            <span className="sidebar-session__status" role="status" aria-label={progress?.label ?? 'Running'}>
              <Loader2 className="sidebar-session__status-icon" aria-hidden />
            </span>
          )}
        </SidebarMenuButton>
        <DropdownMenu>
          <DropdownMenuTrigger
            render={<SidebarMenuAction showOnHover aria-label="Session options" />}
          >
            <MoreHorizontal className="size-4" />
          </DropdownMenuTrigger>
          <DropdownMenuContent align="start" className="w-64">
            {sessionRunning && (
              <>
                <DropdownMenuItem onClick={() => void stopSessionRun(session.id)}>
                  <Square className="size-3.5 text-muted-foreground" />
                  Stop Agent
                </DropdownMenuItem>
                <DropdownMenuSeparator />
              </>
            )}
            <DropdownMenuItem onClick={() => { setNameValue(session.name); setRenameOpen(true) }}>
              <Pencil className="size-3.5 text-muted-foreground" />
              Rename
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <div className="px-2 py-1.5 space-y-2.5">
              <div>
                <p className="text-[10px] uppercase tracking-wide text-muted-foreground font-medium mb-0.5">Working Directory</p>
                <p className="text-xs break-all leading-snug">{session.workingDir || '(server working directory)'}</p>
              </div>
              <div>
                <p className="text-[10px] uppercase tracking-wide text-muted-foreground font-medium mb-0.5">Agent</p>
                <p className="text-xs">{session.agentType}</p>
              </div>
              <div>
                <p className="text-[10px] uppercase tracking-wide text-muted-foreground font-medium mb-0.5">Created</p>
                <p className="text-xs">{new Date(session.createdAt).toLocaleString()}</p>
              </div>
              <div>
                <p className="text-[10px] uppercase tracking-wide text-muted-foreground font-medium mb-0.5">Last run</p>
                <p className="text-xs">
                  {sessionRunning
                    ? 'Running now'
                    : session.lastRunAt
                      ? new Date(session.lastRunAt).toLocaleString()
                      : 'Never'}
                </p>
              </div>
              {session.scheduleId && (
                <div>
                  <p className="text-[10px] uppercase tracking-wide text-muted-foreground font-medium mb-0.5">Schedule</p>
                  <p className="text-xs">{session.scheduleName || session.scheduleId}</p>
                </div>
              )}
            </div>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              className="text-destructive data-highlighted:bg-destructive/10 data-highlighted:text-destructive"
              onClick={() => setDeleteOpen(true)}
            >
              <Trash2 className="size-3.5" />
              Delete Session
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </SidebarMenuItem>

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
              if (e.key === 'Enter') saveRename()
              if (e.key === 'Escape') setRenameOpen(false)
            }}
          />
          <DialogFooter>
            <Button variant="outline" onClick={() => setRenameOpen(false)}>Cancel</Button>
            <Button onClick={saveRename}>Rename</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <ConfirmDeleteDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title="Delete session?"
        description={
          <>
            This will permanently delete <strong>{session.name}</strong> and its associated chat
            history. This action cannot be undone.
          </>
        }
        onConfirm={onDelete}
      />
    </>
  )
}

interface CollapsibleSessionGroupProps {
  group: DisplaySessionGroup
  runningOnly: boolean
  selectedId: string | null
  listViewOpen: boolean
  onSelect: (id: string) => void
  onOpenNewSessionForGroup: (groupId: string) => void
  onOpenSessionList: (groupId: string) => void
  onRename: (id: string, newName: string) => Promise<void>
  onDelete: (id: string) => Promise<void>
}

function CollapsibleSessionGroup({
  group,
  runningOnly,
  selectedId,
  listViewOpen,
  onSelect,
  onOpenNewSessionForGroup,
  onOpenSessionList,
  onRename,
  onDelete,
}: CollapsibleSessionGroupProps) {
  const { closeMobileSidebar } = useSidebar()
  const hasSelected = group.sessions.some((s) => s.id === selectedId)
  const [open, setOpen] = useState(true)
  const sessionCount = group.sessions.length
  const countTitle = runningOnly && sessionCount !== group.totalSessions
    ? `${sessionCount} running (${group.totalSessions} total)`
    : `${sessionCount} session${sessionCount === 1 ? '' : 's'}`

  useEffect(() => {
    if (hasSelected) setOpen(true)
  }, [hasSelected])

  return (
    <SidebarGroup className="px-2 pb-2 pt-0">
      <SidebarGroupLabel
        render={<button type="button" />}
        className={cn(
          'group/label relative h-9 cursor-pointer gap-1.5 pr-8 text-sm font-semibold text-sidebar-foreground hover:bg-sidebar-accent/60 hover:text-sidebar-accent-foreground',
          listViewOpen && 'bg-sidebar-accent/60 text-sidebar-accent-foreground',
        )}
        onClick={() => setOpen((value) => !value)}
        aria-expanded={open}
        title={group.label}
      >
        <ChevronRight
          className={cn(
            'size-4 shrink-0 text-muted-foreground transition-transform duration-200',
            open && 'rotate-90',
          )}
        />
        <span className="flex-1 truncate text-left">{group.label}</span>
        <span className="flex shrink-0 items-center">
          <span
            className={cn(
              'sidebar-group-actions group-hover/label:w-12 group-hover/label:opacity-100',
              listViewOpen && 'sidebar-group-actions--visible',
            )}
          >
            <button
              type="button"
              className="inline-flex size-6 shrink-0 items-center justify-center rounded-md text-muted-foreground hover:bg-sidebar-accent hover:text-sidebar-accent-foreground"
              aria-label={`New ${group.label} session`}
              title="New session"
              onClick={(event) => {
                event.stopPropagation()
                onOpenNewSessionForGroup(group.id)
                closeMobileSidebar()
              }}
            >
              <Plus className="size-3.5" />
            </button>
            <button
              type="button"
              className={cn(
                'inline-flex size-6 shrink-0 items-center justify-center rounded-md text-muted-foreground hover:bg-sidebar-accent hover:text-sidebar-accent-foreground',
                listViewOpen && 'text-sidebar-accent-foreground',
              )}
              aria-label={`List ${group.label} sessions`}
              title="List sessions"
              onClick={(event) => {
                event.stopPropagation()
                onOpenSessionList(group.id)
                closeMobileSidebar()
              }}
            >
              <List className="size-3.5" />
            </button>
          </span>
          <span className="sidebar-meta-rail">
            <span className="sidebar-group-count" title={countTitle}>
              {sessionCount}
            </span>
          </span>
        </span>
      </SidebarGroupLabel>
      {open && (
        <SidebarGroupContent>
          <SidebarMenu className="pl-6">
            {group.sessions.map((s) => (
              <SessionListItem
                key={s.id}
                session={s}
                isActive={s.id === selectedId}
                onSelect={() => {
                  onSelect(s.id)
                  closeMobileSidebar()
                }}
                onRename={(newName) => onRename(s.id, newName)}
                onDelete={() => onDelete(s.id)}
              />
            ))}
          </SidebarMenu>
        </SidebarGroupContent>
      )}
    </SidebarGroup>
  )
}

export function AppSidebar({
  sessions,
  agentTypes,
  selectedId,
  newSessionOpen,
  schedulesPanelOpen,
  sessionListGroupId,
  sidebarWidth,
  onSidebarWidthChange,
  onSidebarWidthCommit,
  onSelect,
  onOpenNewSession,
  onOpenSchedules,
  onOpenNewSessionForGroup,
  onOpenSessionList,
  onRename,
  onDelete,
}: Props) {
  const { closeMobileSidebar } = useSidebar()
  const runningSessions = useRunningSessions()
  const [runningOnly, setRunningOnly] = useState(false)

  const groups = useMemo((): DisplaySessionGroup[] => {
    const allGroups = groupSessionsByAgentType(sessions, agentTypes)
    if (!runningOnly) {
      return allGroups.map((group) => ({
        ...group,
        totalSessions: group.sessions.length,
      }))
    }
    return allGroups.map((group) => ({
      ...group,
      totalSessions: group.sessions.length,
      sessions: group.sessions.filter((session) => runningSessions.has(session.id)),
    }))
  }, [sessions, agentTypes, runningOnly, runningSessions])

  const visibleGroups = useMemo(
    () => groups.filter((group) => group.sessions.length > 0),
    [groups],
  )

  return (
    <Sidebar
      collapsible="offcanvas"
      className="app-sidebar top-12 bottom-auto h-[calc(100svh-3rem)] max-h-[calc(100svh-3rem)]"
    >
      <SidebarResizeHandle
        width={sidebarWidth}
        onWidthChange={onSidebarWidthChange}
        onWidthCommit={onSidebarWidthCommit}
      />
      <SidebarHeader className="sidebar-actions shrink-0 px-3 py-3">
        <div className="flex flex-wrap items-center gap-2">
          <Button
            size="sm"
            className={cn(
              'sidebar-header__btn-primary w-auto gap-2 border-0 shadow-none md:min-h-8 min-h-11 min-w-11 md:min-w-0',
              newSessionOpen && 'ring-1 ring-primary/60',
            )}
            onClick={() => {
              onOpenNewSession()
              closeMobileSidebar()
            }}
            aria-pressed={newSessionOpen}
            aria-label="New Session"
          >
            <Plus className="size-4 shrink-0" />
            <span className="hidden md:inline">New Session</span>
          </Button>
          <Button
            size="sm"
            variant="outline"
            className={cn(
              'w-auto gap-2 bg-background/40 md:min-h-8 min-h-11 min-w-11 md:min-w-0',
              schedulesPanelOpen && 'ring-1 ring-primary/40 bg-primary/10',
            )}
            onClick={() => {
              onOpenSchedules()
              closeMobileSidebar()
            }}
            aria-pressed={schedulesPanelOpen}
            aria-label="Schedule"
          >
            <CalendarClock className="size-4 shrink-0" />
            <span className="hidden md:inline">Schedule</span>
          </Button>
          <Button
            size="sm"
            variant="outline"
            className={cn(
              'ml-auto size-11 shrink-0 bg-background/40 p-0 md:size-8',
              runningOnly && 'ring-1 ring-primary/40 bg-primary/10 text-primary',
            )}
            onClick={() => setRunningOnly((value) => !value)}
            aria-pressed={runningOnly}
            aria-label="Show running sessions only"
            title="Running only"
          >
            <Filter className="size-4 shrink-0" />
          </Button>
        </div>
      </SidebarHeader>
        <SidebarContent className="min-h-0 flex-1 overflow-x-hidden overflow-y-auto overscroll-y-contain pb-4">
          {groups.length === 0 ? (
            <p className="px-4 py-4 text-xs text-muted-foreground group-data-[collapsible=icon]:hidden">
              No sessions yet.
            </p>
          ) : (
            <>
              {runningOnly && visibleGroups.length === 0 && (
                <p className="px-4 py-3 text-xs text-muted-foreground group-data-[collapsible=icon]:hidden">
                  No running sessions.
                </p>
              )}
              {visibleGroups.map((group) => (
                <CollapsibleSessionGroup
                  key={group.id}
                  group={group}
                  runningOnly={runningOnly}
                  selectedId={selectedId}
                  listViewOpen={sessionListGroupId === group.id}
                  onSelect={onSelect}
                  onOpenNewSessionForGroup={onOpenNewSessionForGroup}
                  onOpenSessionList={onOpenSessionList}
                  onRename={onRename}
                  onDelete={onDelete}
                />
              ))}
            </>
          )}
        </SidebarContent>
      </Sidebar>
  )
}
