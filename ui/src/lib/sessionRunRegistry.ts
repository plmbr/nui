// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { api } from '@/api'

type StopFn = () => void | Promise<void>

const running = new Map<string, StopFn>()
const listeners = new Set<() => void>()
let snapshot = ''

function emit() {
  snapshot = [...running.keys()].sort().join(',')
  for (const listener of listeners) {
    listener()
  }
}

export function getRunningSessionsSnapshot(): string {
  return snapshot
}

export function isSessionRunning(sessionId: string): boolean {
  return running.has(sessionId)
}

export function registerSessionRun(sessionId: string, stop: StopFn) {
  running.set(sessionId, stop)
  emit()
}

export function unregisterSessionRun(sessionId: string) {
  if (running.delete(sessionId)) {
    emit()
  }
}

export async function stopSessionRun(sessionId: string) {
  const stop = running.get(sessionId)
  if (stop) {
    await stop()
    return
  }
  await api.sessions.stop(sessionId)
}

export function subscribeSessionRuns(listener: () => void): () => void {
  listeners.add(listener)
  return () => listeners.delete(listener)
}
