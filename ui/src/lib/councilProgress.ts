// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import type { CouncilMemberProgress, CouncilProgressState } from '@/types'

/** Incremental council_progress event payload (AG-UI custom value or run log). */
export interface CouncilProgressEvent {
  phase?: string
  round?: string
  roundIndex?: number
  roundsTotal?: number
  memberId?: string
  memberLabel?: string
  memberSessionId?: string
  runId?: string
  membersTotal?: number
  membersDone?: number
  quorum?: number
  elapsedMs?: number
  error?: string
  estimatedCost?: string
}

export function mergeCouncilProgress(
  prev: CouncilProgressState | undefined,
  value: CouncilProgressEvent,
): CouncilProgressState {
  const base = prev ?? { phase: '', members: [] as CouncilMemberProgress[] }
  let members = [...base.members]
  if (value.memberId) {
    const idx = members.findIndex((x) => x.id === value.memberId)
    const prevMember = idx >= 0 ? members[idx] : undefined
    const status =
      value.phase === 'member_failed'
        ? 'failed'
        : value.phase === 'member_completed'
          ? 'completed'
          : value.phase === 'member_started'
            ? 'running'
            : (prevMember?.status ?? 'queued')
    const entry: CouncilMemberProgress = {
      id: value.memberId,
      label: value.memberLabel || prevMember?.label || value.memberId,
      status,
      sessionId: value.memberSessionId || prevMember?.sessionId,
      runId: value.runId || prevMember?.runId,
      elapsedMs: value.elapsedMs ?? prevMember?.elapsedMs,
      error: value.error,
    }
    if (idx >= 0) members[idx] = { ...members[idx], ...entry }
    else members.push(entry)
  }
  return {
    phase: value.phase || base.phase,
    round: value.round ?? base.round,
    roundIndex: value.roundIndex ?? base.roundIndex,
    roundsTotal: value.roundsTotal ?? base.roundsTotal,
    membersTotal: value.membersTotal ?? base.membersTotal,
    membersDone: value.membersDone ?? base.membersDone,
    quorum: value.quorum ?? base.quorum,
    estimatedCost: value.estimatedCost ?? base.estimatedCost,
    members,
  }
}
