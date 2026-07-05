// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useMemo } from 'react'
import hljs from 'highlight.js'
import { formatToolCallDisplay } from '@/lib/formatToolCallDisplay'

function detectLanguage(text: string): string {
  const trimmed = text.trim()
  if (!trimmed) return 'plaintext'
  if (trimmed.startsWith('{') || trimmed.startsWith('[')) {
    try {
      JSON.parse(trimmed)
      return 'json'
    } catch {
      return 'plaintext'
    }
  }
  return 'plaintext'
}

function highlightText(text: string): string {
  const language = detectLanguage(text)
  if (language === 'plaintext') return hljs.highlight(text, { language: 'plaintext' }).value
  try {
    return hljs.highlight(text, { language }).value
  } catch {
    return hljs.highlight(text, { language: 'plaintext' }).value
  }
}

interface Props {
  value: unknown
}

export function ToolCallCode({ value }: Props) {
  const text = formatToolCallDisplay(value)
  const html = useMemo(() => (text ? highlightText(text) : ''), [text])

  if (!text) return null

  return (
    <pre className="tool-call__code">
      <code className="hljs" dangerouslySetInnerHTML={{ __html: html }} />
    </pre>
  )
}
