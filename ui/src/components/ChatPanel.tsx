// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useEffect, useRef, useState } from 'react'
import ReactMarkdown from 'react-markdown'
import rehypeHighlight from 'rehype-highlight'
import 'highlight.js/styles/github-dark.css'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { ScrollArea } from '@/components/ui/scroll-area'
import { api } from '@/api'
import type { ChatMessage, Project } from '@/types'

interface StreamEvent {
  type: 'text' | 'done' | 'error'
  content?: string
  sessionId?: string
  error?: string
}

interface Props {
  project: Project
}

export function ChatPanel({ project }: Props) {
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [input, setInput] = useState('')
  const [streaming, setStreaming] = useState(false)
  const streamingIdRef = useRef<string | null>(null)
  const bottomRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    setMessages([])
    api.messages.list(project.id).then(setMessages).catch(() => {})
  }, [project.id])

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  async function send() {
    const message = input.trim()
    if (!message || streaming) return
    setInput('')
    setStreaming(true)

    const userMsg: ChatMessage = {
      id: crypto.randomUUID(),
      role: 'user',
      content: message,
      createdAt: new Date().toISOString(),
    }
    const assistantId = crypto.randomUUID()
    streamingIdRef.current = assistantId
    setMessages((prev) => [
      ...prev,
      userMsg,
      { id: assistantId, role: 'assistant', content: '', createdAt: new Date().toISOString() },
    ])

    try {
      const res = await fetch(`/api/projects/${project.id}/chat`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ message }),
      })
      if (!res.ok || !res.body) {
        throw new Error(await res.text())
      }

      const reader = res.body.getReader()
      const decoder = new TextDecoder()
      let buffer = ''

      while (true) {
        const { done, value } = await reader.read()
        if (done) break
        buffer += decoder.decode(value, { stream: true })
        const lines = buffer.split('\n')
        buffer = lines.pop() ?? ''

        for (const line of lines) {
          if (!line.startsWith('data: ')) continue
          const payload: StreamEvent = JSON.parse(line.slice(6))
          if (payload.type === 'text' && payload.content) {
            setMessages((prev) =>
              prev.map((m) =>
                m.id === assistantId ? { ...m, content: m.content + payload.content } : m,
              ),
            )
          }
          if (payload.type === 'error') {
            setMessages((prev) =>
              prev.map((m) =>
                m.id === assistantId
                  ? { ...m, content: `Error: ${payload.error ?? 'unknown error'}` }
                  : m,
              ),
            )
          }
        }
      }
    } catch (err) {
      setMessages((prev) =>
        prev.map((m) =>
          m.id === assistantId
            ? { ...m, content: `Error: ${err instanceof Error ? err.message : 'Failed to connect'}` }
            : m,
        ),
      )
    } finally {
      streamingIdRef.current = null
      setStreaming(false)
    }
  }

  return (
    <div className="flex flex-col h-full">
      <ScrollArea className="flex-1">
        <div className="p-4 space-y-4">
          {messages.length === 0 && (
            <div className="text-center text-muted-foreground text-sm py-12">
              No messages yet. Start a conversation.
            </div>
          )}
          {messages.map((m) => (
            <div key={m.id} className={`flex ${m.role === 'user' ? 'justify-end' : 'justify-start'}`}>
              <div
                className={`max-w-[80%] rounded-lg px-4 py-2.5 text-sm break-words ${
                  m.role === 'user'
                    ? 'bg-primary text-primary-foreground whitespace-pre-wrap'
                    : 'bg-muted text-foreground prose prose-sm dark:prose-invert max-w-none'
                }`}
              >
                {m.role === 'user' ? (
                  m.content || (streaming && m.id === streamingIdRef.current ? '▋' : '')
                ) : (
                  <ReactMarkdown rehypePlugins={[rehypeHighlight]}>
                    {m.content || (streaming && m.id === streamingIdRef.current ? '▋' : '')}
                  </ReactMarkdown>
                )}
              </div>
            </div>
          ))}
          <div ref={bottomRef} />
        </div>
      </ScrollArea>
      <div className="border-t p-4 flex gap-2 items-end shrink-0">
        <Textarea
          className="flex-1 min-h-[60px] max-h-[160px] resize-none"
          placeholder="Ask your agent anything…"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && !e.shiftKey) {
              e.preventDefault()
              send()
            }
          }}
          disabled={streaming}
        />
        <Button onClick={send} disabled={streaming || !input.trim()}>
          {streaming ? 'Working…' : 'Send'}
        </Button>
      </div>
    </div>
  )
}
