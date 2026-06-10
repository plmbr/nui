// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { ChatPanel } from '@/components/ChatPanel'
import type { Project } from '@/types'

interface Props {
  project: Project
}

export function ConversationPanel({ project }: Props) {
  return (
    <div className="flex flex-col h-full overflow-hidden">
      <ChatPanel project={project} />
    </div>
  )
}
