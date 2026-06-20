// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useState } from 'react'
import { McpAppFrame } from '@/components/McpAppFrame'
import type { SessionChatMessage } from '@/hooks/useSessionChat'

interface Props {
  msg: SessionChatMessage
}

export function ToolCallBubble({ msg }: Props) {
  const [expanded, setExpanded] = useState(false)
  const baseName = msg.toolName?.split(':').pop() ?? msg.toolName ?? 'tool'

  return (
    <div className="tool-call">
      <button
        type="button"
        className="tool-call__header"
        onClick={() => setExpanded((e) => !e)}
      >
        <span className="tool-call__icon">⚙</span>
        <span className="tool-call__name">{baseName}</span>
        <span className="tool-call__status">
          {msg.toolResult !== undefined ? '✓ done' : '⋯ running'}
        </span>
        <span className="tool-call__toggle">{expanded ? '▲' : '▼'}</span>
      </button>

      {expanded && (
        <div className="tool-call__body">
          {msg.toolArgs && Object.keys(msg.toolArgs).length > 0 && (
            <div className="tool-call__section">
              <div className="tool-call__label">Input</div>
              <pre className="tool-call__code">{JSON.stringify(msg.toolArgs, null, 2)}</pre>
            </div>
          )}
          {msg.toolResult !== undefined && (
            <div className="tool-call__section">
              <div className="tool-call__label">Output</div>
              <pre className="tool-call__code">
                {typeof msg.toolResult === 'string'
                  ? msg.toolResult
                  : JSON.stringify(msg.toolResult, null, 2)}
              </pre>
            </div>
          )}
        </div>
      )}

      {msg.mcpAppResourceUri && msg.mcpAppServerName && (
        <McpAppFrame
          serverName={msg.mcpAppServerName}
          resourceUri={msg.mcpAppResourceUri}
          toolName={msg.toolName}
          toolInput={msg.mcpAppToolInput}
          toolResult={msg.toolResult}
        />
      )}
    </div>
  )
}
