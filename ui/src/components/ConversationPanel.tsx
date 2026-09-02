// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { ChatPanel } from '@/components/ChatPanel'
import type { PromptSuggestion, Session } from '@/types'

interface Props {
  session: Session
  initialPrompt?: string
  skipBootstrap?: boolean
  hideInput?: boolean
  promptMode?: 'user' | 'auto'
  defaultPrompt?: string
  promptSuggestions?: PromptSuggestion[]
  slashCommands?: string[]
}

export function ConversationPanel({
  session,
  initialPrompt,
  skipBootstrap,
  hideInput,
  promptMode,
  defaultPrompt,
  promptSuggestions,
  slashCommands,
}: Props) {
  return (
    <ChatPanel
      session={session}
      initialPrompt={initialPrompt}
      skipBootstrap={skipBootstrap}
      hideInput={hideInput}
      promptMode={promptMode}
      defaultPrompt={defaultPrompt}
      promptSuggestions={promptSuggestions}
      slashCommands={slashCommands}
    />
  )
}
