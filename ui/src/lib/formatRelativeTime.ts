// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

const HAS_TZ = /(?:Z|[+-]\d{2}:?\d{2})$/i
const DATE_ONLY = /^\d{4}-\d{2}-\d{2}$/

/** Parse an ISO timestamp; timezone-less datetimes are treated as UTC. */
export function parseISOTimestamp(iso: string): number | null {
  const trimmed = iso.trim()
  if (!trimmed) return null

  if (HAS_TZ.test(trimmed)) {
    const ts = Date.parse(trimmed)
    return Number.isNaN(ts) ? null : ts
  }

  if (DATE_ONLY.test(trimmed)) {
    const ts = Date.parse(`${trimmed}T00:00:00Z`)
    return Number.isNaN(ts) ? null : ts
  }

  const ts = Date.parse(`${trimmed}Z`)
  return Number.isNaN(ts) ? null : ts
}

export function formatRelativeTime(iso: string, now = Date.now(), includeAgo = true): string {
  const ts = parseISOTimestamp(iso)
  if (ts == null) return ''
  const diffMs = Math.max(0, now - ts)
  const minute = 60_000
  const hour = 60 * minute
  const day = 24 * hour
  const week = 7 * day
  const suffix = includeAgo ? ' ago' : ''

  if (diffMs < minute) return 'now'
  if (diffMs < hour) return `${Math.floor(diffMs / minute)}m${suffix}`
  if (diffMs < day) return `${Math.floor(diffMs / hour)}h${suffix}`
  if (diffMs < week) return `${Math.floor(diffMs / day)}d${suffix}`
  return `${Math.floor(diffMs / week)}w${suffix}`
}

export function formatExactTime(iso: string): string {
  const ts = parseISOTimestamp(iso)
  if (ts == null) return ''
  return new Date(ts).toLocaleString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
  })
}

/** Compact local timestamp for session titles, e.g. 2026-07-02 18:09 */
export function formatShortLocalTime(iso: string): string {
  const ts = parseISOTimestamp(iso)
  if (ts == null) return ''
  const d = new Date(ts)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}
