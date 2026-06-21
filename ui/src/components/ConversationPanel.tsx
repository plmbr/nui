// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { ChatPanel } from '@/components/ChatPanel'
import type { Session } from '@/types'

interface Props {
  session: Session
  initialPrompt?: string
  hideInput?: boolean
  promptMode?: 'user' | 'auto'
  defaultPrompt?: string
  agentLabel?: string
  agentDescription?: string
}

export function ConversationPanel({
  session,
  initialPrompt,
  hideInput,
  promptMode,
  defaultPrompt,
  agentLabel,
  agentDescription,
}: Props) {
  return (
    <div className="flex flex-col flex-1 min-h-0 overflow-hidden">
      <div className="conversation-header">
        <div className="min-w-0">
          <span className="text-sm font-semibold truncate block">{session.name}</span>
          {agentLabel && (
            <div className="mt-0.5 min-w-0">
              <span className="text-xs font-medium text-muted-foreground truncate block">{agentLabel}</span>
              {agentDescription && (
                <span className="text-xs text-muted-foreground/80 truncate block">{agentDescription}</span>
              )}
            </div>
          )}
        </div>
      </div>
      <ChatPanel
        session={session}
        initialPrompt={initialPrompt}
        hideInput={hideInput}
        promptMode={promptMode}
        defaultPrompt={defaultPrompt}
      />
    </div>
  )
}
