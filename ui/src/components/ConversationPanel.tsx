// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { ChatPanel } from '@/components/ChatPanel'
import type { PromptSuggestion, Session } from '@/types'

interface Props {
  session: Session
  initialPrompt?: string
  hideInput?: boolean
  promptMode?: 'user' | 'auto'
  defaultPrompt?: string
  promptSuggestions?: PromptSuggestion[]
}

export function ConversationPanel({
  session,
  initialPrompt,
  hideInput,
  promptMode,
  defaultPrompt,
  promptSuggestions,
}: Props) {
  return (
    <ChatPanel
      session={session}
      initialPrompt={initialPrompt}
      hideInput={hideInput}
      promptMode={promptMode}
      defaultPrompt={defaultPrompt}
      promptSuggestions={promptSuggestions}
    />
  )
}
