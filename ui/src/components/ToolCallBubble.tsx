// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useState } from 'react'
import { McpAppFrame } from '@/components/McpAppFrame'
import { ToolCallGroup } from '@/components/ToolCallGroup'
import { imageSrc, type ToolCallPart } from '@/hooks/useSessionChat'
import { ToolCallCode } from '@/components/ToolCallCode'
import { formatMcpServerLabel, formatToolDisplayName, isMcpToolName } from '@/lib/toolCallDisplay'
import { formatToolCallSummary } from '@/lib/toolCallSummary'
import { extractImagesFromValue } from '@/lib/images'
import { isHarnessSubagentTool } from '@/lib/subagentTrace'

interface Props {
  part: ToolCallPart
  nested?: boolean
}

export function ToolCallBubble({ part: msg, nested = false }: Props) {
  const [expanded, setExpanded] = useState(false)
  const displayName = formatToolDisplayName(msg.toolName)
  const mcpServer = formatMcpServerLabel(msg.toolName)
  const detail = formatToolCallSummary(msg.toolName, msg.toolArgs)
  const isDone = msg.toolResult !== undefined
  const hasSubagentTrace = isHarnessSubagentTool(msg.toolName) && (msg.subagentTrace?.length ?? 0) > 0
  const showSubagentTrace = hasSubagentTrace && (!isDone || expanded)
  const hasDetails =
    (msg.toolArgs && Object.keys(msg.toolArgs).length > 0) || msg.toolResult !== undefined
  const toolImages =
    msg.toolResult !== undefined ? extractImagesFromValue(msg.toolResult) : []

  const mcpToolId = msg.toolName && isMcpToolName(msg.toolName) ? msg.toolName : undefined

  const expandedBody = (
    <>
      {mcpToolId && (
        <div className="tool-call__section">
          <div className="tool-call__label">Tool</div>
          <code className="tool-call__identifier">{mcpToolId}</code>
        </div>
      )}
      {msg.toolArgs && Object.keys(msg.toolArgs).length > 0 && (
        <div className="tool-call__section">
          <div className="tool-call__label">Input</div>
          <ToolCallCode value={msg.toolArgs} />
        </div>
      )}
      {msg.toolResult !== undefined && (
        <div className="tool-call__section">
          <div className="tool-call__label">Output</div>
          <ToolCallCode value={msg.toolResult} />
        </div>
      )}
    </>
  )

  if (nested) {
    return (
      <div className={`tool-call-item${isDone ? '' : ' tool-call-item--running'}`}>
        <div className="tool-call-item__row">
          {mcpServer && <span className="tool-call-item__server">{mcpServer}</span>}
          <span className="tool-call-item__name">{displayName}</span>
          {detail && <span className="tool-call-item__detail">{detail}</span>}
          {!isDone && <span className="tool-call-item__status">Running…</span>}
          {hasDetails && isDone && (
            <button
              type="button"
              className="tool-call-item__result"
              onClick={() => setExpanded((value) => !value)}
              aria-expanded={expanded}
            >
              {expanded ? 'Hide' : 'Result'}
            </button>
          )}
        </div>

        {expanded && <div className="tool-call-item__body not-prose">{expandedBody}</div>}

        {showSubagentTrace && msg.subagentTrace && (
          <div className="tool-call__subagent-trace">
            <ToolCallGroup parts={msg.subagentTrace.filter((part) => part.type === 'tool')} />
            {msg.subagentTrace
              .filter((part) => part.type === 'text')
              .map((part, index) => (
                <div key={`subagent-text-${index}`} className="tool-call__subagent-text">
                  {part.content}
                </div>
              ))}
          </div>
        )}

        {toolImages.map((img) => (
          <img
            key={img.id}
            src={imageSrc(img)}
            alt="Tool result"
            className="agui-message__image"
            loading="lazy"
          />
        ))}

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

  return (
    <div className="tool-call">
      <button
        type="button"
        className="tool-call__header"
        onClick={() => setExpanded((value) => !value)}
      >
        <span className="tool-call__icon">⚙</span>
        <div className="tool-call__summary">
          {mcpServer && <span className="tool-call__server">{mcpServer}</span>}
          <span className="tool-call__name">{displayName}</span>
          {detail && <span className="tool-call__detail">{detail}</span>}
        </div>
        <span className="tool-call__status">
          {isDone ? '✓ done' : '⋯ running'}
        </span>
        <span className="tool-call__toggle">{expanded ? '▲' : '▼'}</span>
      </button>

      {expanded && <div className="tool-call__body not-prose">{expandedBody}</div>}

      {showSubagentTrace && msg.subagentTrace && (
        <div className="tool-call__subagent-trace">
          <ToolCallGroup parts={msg.subagentTrace.filter((part) => part.type === 'tool')} />
          {msg.subagentTrace
            .filter((part) => part.type === 'text')
            .map((part, index) => (
              <div key={`subagent-text-${index}`} className="tool-call__subagent-text">
                {part.content}
              </div>
            ))}
        </div>
      )}

      {toolImages.map((img) => (
        <img
          key={img.id}
          src={imageSrc(img)}
          alt="Tool result"
          className="agui-message__image"
          loading="lazy"
        />
      ))}

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
