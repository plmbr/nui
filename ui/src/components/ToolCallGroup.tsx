// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useEffect, useState } from 'react'
import { ChevronDown, ChevronRight } from 'lucide-react'
import { ToolCallBubble } from '@/components/ToolCallBubble'
import type { ToolCallPart } from '@/hooks/useSessionChat'
import { buildToolGroupSummary, isToolGroupRunning } from '@/lib/toolCallDisplay'
import { isHarnessSubagentTool } from '@/lib/subagentTrace'

interface Props {
  parts: ToolCallPart[]
  sessionId?: string
}

function hasActiveSubagentTrace(parts: ToolCallPart[]): boolean {
  return parts.some(
    (part) =>
      isHarnessSubagentTool(part.toolName) &&
      part.toolResult === undefined &&
      (part.subagentTrace?.length ?? 0) > 0,
  )
}

export function ToolCallGroup({ parts, sessionId }: Props) {
  const running = isToolGroupRunning(parts)
  const [expanded, setExpanded] = useState(false)

  useEffect(() => {
    if (running && hasActiveSubagentTrace(parts)) {
      setExpanded(true)
    }
  }, [running, parts])

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
            <ToolCallBubble key={part.id} part={part} nested sessionId={sessionId} />
          ))}
        </div>
      )}
    </div>
  )
}
