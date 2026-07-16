// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useEffect, useMemo, useState } from 'react'
import { List, Trash2, X, CalendarClock } from 'lucide-react'
import { sessionDisplayName } from '@/lib/sessionDisplay'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import { ConfirmDeleteDialog } from '@/components/ConfirmDeleteDialog'
import { BUILTIN_GROUP_ID, type SessionGroup } from '@/lib/sessionGroups'

interface Props {
  group: SessionGroup
  selectedId: string | null
  onSelect: (id: string) => void
  onClose: () => void
  onBulkDelete: (ids: string[]) => Promise<void>
}

export function SessionsListPanel({
  group,
  selectedId,
  onSelect,
  onClose,
  onBulkDelete,
}: Props) {
  const sessionIds = useMemo(() => group.sessions.map((s) => s.id), [group.sessions])
  const [checkedIds, setCheckedIds] = useState<Set<string>>(new Set())
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [deleting, setDeleting] = useState(false)

  useEffect(() => {
    setCheckedIds(new Set())
  }, [group.id])

  const allChecked = sessionIds.length > 0 && sessionIds.every((id) => checkedIds.has(id))
  const someChecked = checkedIds.size > 0
  const headerChecked = allChecked ? true : someChecked ? 'indeterminate' : false

  function toggleAll() {
    if (allChecked) {
      setCheckedIds(new Set())
    } else {
      setCheckedIds(new Set(sessionIds))
    }
  }

  function toggleOne(id: string) {
    setCheckedIds((prev) => {
      const next = new Set(prev)
      if (next.has(id)) {
        next.delete(id)
      } else {
        next.add(id)
      }
      return next
    })
  }

  async function confirmDelete() {
    const ids = Array.from(checkedIds)
    if (ids.length === 0) return
    setDeleting(true)
    try {
      await onBulkDelete(ids)
      setCheckedIds(new Set())
    } finally {
      setDeleting(false)
    }
  }

  return (
    <div className="customize-panel flex flex-1 min-h-0 flex-col overflow-hidden">
      <div className="conversation-header justify-between">
        <div className="flex items-center gap-2 min-w-0">
          <List className="size-4 shrink-0 text-muted-foreground" />
          <h1 className="text-sm font-semibold truncate">
            {group.id === BUILTIN_GROUP_ID ? group.label : `${group.label} sessions`}
          </h1>
          <span className="text-xs text-muted-foreground tabular-nums">{group.sessions.length}</span>
        </div>
        <Button variant="ghost" size="sm" onClick={onClose} aria-label="Close session list">
          <X className="size-4" />
        </Button>
      </div>

      <div className="flex flex-1 flex-col min-h-0">
        {group.sessions.length === 0 ? (
          <div className="empty-state">No sessions in this category.</div>
        ) : (
          <>
            <div className="flex shrink-0 items-center gap-2 border-b px-4 py-2 md:px-6">
              <Button
                variant="destructive"
                size="sm"
                disabled={!someChecked || deleting}
                onClick={() => setDeleteOpen(true)}
              >
                <Trash2 className="size-3.5" />
                <span className="hidden sm:inline">Delete selected</span>
                {someChecked ? ` (${checkedIds.size})` : ''}
              </Button>
            </div>

            <ul className="md:hidden flex-1 divide-y overflow-auto">
              {group.sessions.map((session) => {
                const isActive = session.id === selectedId
                const isChecked = checkedIds.has(session.id)
                return (
                  <li
                    key={session.id}
                    className={cn(
                      'sessions-card flex cursor-pointer gap-3 px-4 py-3',
                      isActive && 'sessions-card--active',
                      isChecked && 'sessions-card--checked',
                    )}
                    onClick={() => onSelect(session.id)}
                  >
                    <div className="pt-0.5" onClick={(e) => e.stopPropagation()}>
                      <input
                        type="checkbox"
                        className="size-4 cursor-pointer accent-primary"
                        aria-label={`Select ${session.name}`}
                        checked={isChecked}
                        onChange={() => toggleOne(session.id)}
                      />
                    </div>
                    <div className="min-w-0 flex-1">
                      <div className="flex items-start justify-between gap-2">
                        <span className="inline-flex min-w-0 items-center gap-1.5 font-medium">
                          {session.scheduleId && (
                            <CalendarClock
                              className="size-3.5 shrink-0 text-muted-foreground"
                              title={`Scheduled · ${session.scheduleName || session.scheduleId}`}
                            />
                          )}
                          <span className="truncate">{sessionDisplayName(session)}</span>
                        </span>
                        <span className="shrink-0 text-xs tabular-nums text-muted-foreground">
                          {new Date(session.createdAt).toLocaleDateString()}
                        </span>
                      </div>
                      <p className="mt-0.5 truncate text-xs text-muted-foreground" title={session.workingDir}>
                        {session.workingDir || '(server working directory)'}
                      </p>
                    </div>
                  </li>
                )
              })}
            </ul>

            <div className="hidden flex-1 overflow-auto md:block">
              <table className="sessions-table w-full text-sm">
                <thead className="sticky top-0 z-10 bg-muted/80 backdrop-blur-sm">
                  <tr>
                    <th className="sessions-table__checkbox">
                      <input
                        type="checkbox"
                        aria-label="Select all sessions"
                        checked={headerChecked === true}
                        ref={(el) => {
                          if (el) el.indeterminate = headerChecked === 'indeterminate'
                        }}
                        onChange={toggleAll}
                      />
                    </th>
                    <th>Name</th>
                    <th>Working directory</th>
                    <th>Created</th>
                    <th>Last run</th>
                  </tr>
                </thead>
                <tbody>
                  {group.sessions.map((session) => {
                    const isActive = session.id === selectedId
                    const isChecked = checkedIds.has(session.id)
                    return (
                      <tr
                        key={session.id}
                        className={cn(
                          'sessions-table__row cursor-pointer',
                          isActive && 'sessions-table__row--active',
                          isChecked && 'sessions-table__row--checked',
                        )}
                        onClick={() => onSelect(session.id)}
                      >
                        <td className="sessions-table__checkbox" onClick={(e) => e.stopPropagation()}>
                          <input
                            type="checkbox"
                            aria-label={`Select ${session.name}`}
                            checked={isChecked}
                            onChange={() => toggleOne(session.id)}
                          />
                        </td>
                        <td className="font-medium">
                          <span className="inline-flex items-center gap-1.5 min-w-0">
                            {session.scheduleId && (
                              <CalendarClock
                                className="size-3.5 shrink-0 text-muted-foreground"
                                title={`Scheduled · ${session.scheduleName || session.scheduleId}`}
                              />
                            )}
                            <span className="truncate">{sessionDisplayName(session)}</span>
                          </span>
                        </td>
                        <td className="text-muted-foreground max-w-xs truncate" title={session.workingDir}>
                          {session.workingDir || '(server working directory)'}
                        </td>
                        <td className="text-muted-foreground whitespace-nowrap tabular-nums">
                          {new Date(session.createdAt).toLocaleString()}
                        </td>
                        <td className="text-muted-foreground whitespace-nowrap tabular-nums">
                          {session.lastRunAt
                            ? new Date(session.lastRunAt).toLocaleString()
                            : '—'}
                        </td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>
          </>
        )}
      </div>

      <ConfirmDeleteDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title={`Delete ${checkedIds.size} session${checkedIds.size === 1 ? '' : 's'}?`}
        description={
          <>
            This will permanently delete the selected session
            {checkedIds.size === 1 ? '' : 's'} and associated chat history. This action cannot be
            undone.
          </>
        }
        confirming={deleting}
        onConfirm={confirmDelete}
      />
    </div>
  )
}
