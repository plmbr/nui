// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { ChatPanel } from '@/components/ChatPanel'
import type { Session } from '@/types'

interface Props {
  session: Session
  initialPrompt?: string
  hideInput?: boolean
  promptMode?: 'user' | 'auto'
  defaultPrompt?: string
}

export function ConversationPanel({
  session,
  initialPrompt,
  hideInput,
  promptMode,
  defaultPrompt,
}: Props) {
  return (
    <ChatPanel
      session={session}
      initialPrompt={initialPrompt}
      hideInput={hideInput}
      promptMode={promptMode}
      defaultPrompt={defaultPrompt}
    />
  )
}
