// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { ChatPanel } from '@/components/ChatPanel'
import { SessionMenu } from '@/components/SessionMenu'
import type { Session } from '@/types'

interface Props {
  session: Session
  onRename: (newName: string) => Promise<void>
  onDelete: () => Promise<void>
}

export function ConversationPanel({ session, onRename, onDelete }: Props) {
  return (
    <div className="flex flex-col flex-1 min-h-0 overflow-hidden">
      <div className="conversation-header">
        <SessionMenu session={session} onRename={onRename} onDelete={onDelete} />
      </div>
      <ChatPanel session={session} />
    </div>
  )
}
