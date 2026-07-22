// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { AppBridge, PostMessageTransport } from '@modelcontextprotocol/ext-apps/app-bridge'
import { useEffect, useRef, useState } from 'react'

interface McpAppFrameProps {
  serverName: string
  resourceUri: string
  sessionId?: string
  toolName?: string
  toolInput?: Record<string, unknown>
  toolResult?: unknown
}

function buildCallToolResult(toolResult: unknown): {
  content: unknown[]
  structuredContent?: unknown
  isError?: boolean
} {
  if (!toolResult || typeof toolResult !== 'object') {
    return { content: [{ type: 'text', text: String(toolResult ?? '') }], isError: false }
  }
  const r = toolResult as Record<string, unknown>

  let content: unknown[]
  if (typeof r.content === 'string') {
    content = [{ type: 'text', text: r.content }]
  } else if (Array.isArray(r.content)) {
    content = r.content
  } else {
    content = [{ type: 'text', text: JSON.stringify(r.content ?? '') }]
  }

  return {
    content,
    structuredContent: r.structuredContent,
    isError: !!r.isError,
  }
}

export function McpAppFrame({
  serverName,
  resourceUri,
  sessionId,
  toolName,
  toolInput,
  toolResult,
}: McpAppFrameProps) {
  const iframeRef = useRef<HTMLIFrameElement>(null)
  const bridgeRef = useRef<AppBridge | null>(null)
  const [height, setHeight] = useState(400)
  const toolInputSentRef = useRef(false)
  const toolResultSentRef = useRef(false)

  const resourceUrl = `/mcp-resource?server=${encodeURIComponent(serverName)}&uri=${encodeURIComponent(resourceUri)}${
    sessionId ? `&sessionId=${encodeURIComponent(sessionId)}` : ''
  }`
  const bareToolName = toolName?.split('__').pop() ?? toolName ?? ''

  useEffect(() => {
    const iframe = iframeRef.current
    if (!iframe) return

    const bridge = new AppBridge(
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      null as any,
      { name: 'nui-chat', version: '1.0.0' },
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      { serverTools: {} } as any,
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      { hostContext: { toolName: bareToolName } } as any,
    )
    bridgeRef.current = bridge

    bridge.oncalltool = async (params) => {
      const res = await fetch('/mcp-call-tool', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          server: serverName,
          name: params.name,
          arguments: params.arguments ?? {},
        }),
      })
      if (!res.ok) {
        const text = await res.text().catch(() => '')
        throw new Error(`tools/call failed: ${res.status} ${text}`)
      }
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      return (await res.json()) as any
    }

    bridge.addEventListener('sizechange', ({ height: h }) => {
      if (h) setHeight(h)
    })

    bridge.addEventListener('initialized', async () => {
      if (!toolInputSentRef.current) {
        await bridge.sendToolInput({ arguments: toolInput ?? {} })
        toolInputSentRef.current = true
      }
      if (!toolResultSentRef.current && toolResult !== undefined) {
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        await bridge.sendToolResult(buildCallToolResult(toolResult) as any)
        toolResultSentRef.current = true
      }
    })

    let connected = false
    const doConnect = () => {
      if (connected) return
      const win = iframe.contentWindow
      if (!win) return
      connected = true
      const transport = new PostMessageTransport(win, win)
      bridge.connect(transport).catch(console.error)
    }

    iframe.addEventListener('load', doConnect)
    if (iframe.contentDocument?.readyState === 'complete') {
      doConnect()
    }

    return () => {
      iframe.removeEventListener('load', doConnect)
      bridge.close().catch(() => {})
      bridgeRef.current = null
      toolInputSentRef.current = false
      toolResultSentRef.current = false
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  useEffect(() => {
    if (toolResult === undefined || toolResultSentRef.current) return
    const bridge = bridgeRef.current
    if (!bridge || !toolInputSentRef.current) return
    toolResultSentRef.current = true
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    bridge.sendToolResult(buildCallToolResult(toolResult) as any).catch(console.error)
  }, [toolResult])

  return (
    <div className="mcp-app-frame">
      <iframe
        ref={iframeRef}
        src={resourceUrl}
        title={`MCP App: ${bareToolName}`}
        sandbox="allow-scripts allow-same-origin allow-forms allow-popups"
        style={{ height: `${height}px` }}
      />
    </div>
  )
}
