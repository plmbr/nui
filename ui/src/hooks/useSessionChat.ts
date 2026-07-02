// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useCallback, useEffect, useSyncExternalStore } from 'react'
import {
  ensureSessionChatLoaded,
  getSessionChatSnapshot,
  sendMessage as storeSendMessage,
  stopRun as storeStopRun,
  subscribeSessionChat,
} from '@/lib/sessionChatStore'

export type {
  AssistantPart,
  SessionChatMessage,
  TextPart,
  ToolCallPart,
} from '@/lib/chatMessageUtils'
export type { ChatImage } from '@/types'
export { imageSrc } from '@/lib/chatMessageUtils'

export function useSessionChat(sessionId: string) {
  useEffect(() => {
    void ensureSessionChatLoaded(sessionId)
  }, [sessionId])

  const snapshot = useSyncExternalStore(
    (listener) => subscribeSessionChat(sessionId, listener),
    () => getSessionChatSnapshot(sessionId),
    () => getSessionChatSnapshot(sessionId),
  )

  const sendMessage = useCallback(
    (text: string) => {
      storeSendMessage(sessionId, text)
    },
    [sessionId],
  )

  const stopRun = useCallback(() => {
    void storeStopRun(sessionId)
  }, [sessionId])

  return {
    messages: snapshot.messages,
    sendMessage,
    stopRun,
    isRunning: snapshot.isRunning,
    isLoading: snapshot.isLoading,
  }
}
