// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { ChatPanel } from '@/components/ChatPanel'
import type { Session } from '@/types'

interface Props {
  session: Session
}

export function ConversationPanel({ session }: Props) {
  return (
    <div className="flex flex-col h-full overflow-hidden">
      <ChatPanel session={session} />
    </div>
  )
}
