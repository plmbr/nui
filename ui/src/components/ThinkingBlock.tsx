// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useState, type ReactNode } from 'react'
import { ChevronDown, ChevronRight } from 'lucide-react'

interface Props {
  children: ReactNode
  running?: boolean
}

export function ThinkingBlock({ children, running = false }: Props) {
  const [expanded, setExpanded] = useState(false)

  return (
    <div
      className={`thinking-block${expanded ? ' thinking-block--expanded' : ''}${running ? ' thinking-block--running' : ''}`}
    >
      <button
        type="button"
        className="thinking-block__header"
        onClick={() => setExpanded((value) => !value)}
        aria-expanded={expanded}
      >
        <span className="thinking-block__chevron" aria-hidden>
          {expanded ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
        </span>
        <span className="thinking-block__summary">Thinking</span>
        {running && <span className="thinking-block__status">Thinking…</span>}
      </button>

      {expanded && (
        <div className="thinking-block__body">
          <div className="thinking-block__content agui-message__text-part">{children}</div>
        </div>
      )}
    </div>
  )
}
