// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { ChatPanel } from '@/components/ChatPanel'
import type { Session } from '@/types'

interface Props {
  session: Session
}

export function ConversationPanel({ session }: Props) {
  return (
    <div className="flex flex-col flex-1 min-h-0 overflow-hidden">
      <div className="conversation-header">
        <span className="text-sm font-semibold truncate">{session.name}</span>
      </div>
      <ChatPanel session={session} />
    </div>
  )
}
