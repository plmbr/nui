// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useState } from 'react'
import { ChevronDown, ChevronRight } from 'lucide-react'
import { ToolCallBubble } from '@/components/ToolCallBubble'
import type { ToolCallPart } from '@/hooks/useSessionChat'
import { buildToolGroupSummary, isToolGroupRunning } from '@/lib/toolCallDisplay'

interface Props {
  parts: ToolCallPart[]
}

export function ToolCallGroup({ parts }: Props) {
  const running = isToolGroupRunning(parts)
  const [expanded, setExpanded] = useState(false)

  if (parts.length === 0) return null

  const summary = buildToolGroupSummary(parts)

  return (
    <div className={`tool-group${expanded ? ' tool-group--expanded' : ''}${running ? ' tool-group--running' : ''}`}>
      <button
        type="button"
        className="tool-group__header"
        onClick={() => setExpanded((value) => !value)}
        aria-expanded={expanded}
      >
        <span className="tool-group__chevron" aria-hidden>
          {expanded ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
        </span>
        <span className="tool-group__summary">{summary}</span>
        {running && <span className="tool-group__status">Running…</span>}
      </button>

      {expanded && (
        <div className="tool-group__timeline">
          {parts.map((part) => (
            <ToolCallBubble key={part.id} part={part} nested />
          ))}
        </div>
      )}
    </div>
  )
}
