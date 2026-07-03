// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { formatShortLocalTime } from '@/lib/formatRelativeTime'
import type { Session } from '@/types'

type SessionNameInput = Pick<Session, 'name' | 'scheduleId' | 'scheduleName' | 'lastRunAt'>

/** Display name for a session; scheduled runs use local time from lastRunAt, not UTC in stored name. */
export function sessionDisplayName(session: SessionNameInput): string {
  if (session.scheduleId && session.lastRunAt) {
    const base =
      session.scheduleName?.trim() ||
      session.name.split(' · ')[0]?.trim() ||
      session.name
    return `${base} · ${formatShortLocalTime(session.lastRunAt)}`
  }
  return session.name
}
