// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useEffect, useRef, useState } from 'react'
import ReactMarkdown from 'react-markdown'
import rehypeHighlight from 'rehype-highlight'
import remarkGfm from 'remark-gfm'
import 'highlight.js/styles/github-dark.css'
import { ToolCallBubble } from '@/components/ToolCallBubble'
import { imageSrc, useSessionChat } from '@/hooks/useSessionChat'
import { normalizeMarkdown, stripInlineCodeDelimiters } from '@/lib/markdown'
import type { Session } from '@/types'

interface Props {
  session: Session
}

export function ChatPanel({ session }: Props) {
  const { messages, sendMessage, isRunning } = useSessionChat(session.id)
  const [input, setInput] = useState('')
  const bottomRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLTextAreaElement>(null)

  useEffect(() => {
    inputRef.current?.focus()
  }, [session.id])

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, isRunning])

  const submit = () => {
    const text = input.trim()
    if (!text || isRunning) return
    setInput('')
    sendMessage(text)
  }

  const onKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      submit()
    }
  }

  return (
    <div className="agui-chat flex flex-col flex-1 min-h-0">
      <div className="agui-chat__messages">
        {messages.length === 0 && (
          <div className="agui-chat__empty">
            <div className="agui-chat__empty-icon">✦</div>
            <p>Start a conversation with your agent</p>
          </div>
        )}

        {messages.map((msg) => {
          if (msg.role === 'tool') {
            return <ToolCallBubble key={msg.id} msg={msg} />
          }

          return (
            <div key={msg.id} className={`agui-message agui-message--${msg.role}`}>
              <div className="agui-message__role">
                {msg.role === 'user' ? 'You' : 'Agent'}
              </div>
              <div
                className={`agui-message__bubble${msg.error ? ' agui-message__bubble--error' : ''}`}
              >
                {msg.role === 'assistant' ? (
                  <>
                    {msg.images?.map((img) => (
                      <img
                        key={img.id}
                        src={imageSrc(img)}
                        alt="Agent image"
                        className="agui-message__image"
                        loading="lazy"
                      />
                    ))}
                    <ReactMarkdown
                      remarkPlugins={[remarkGfm]}
                      rehypePlugins={[rehypeHighlight]}
                      components={{
                        code: ({ className, children, ...props }) => {
                          const isBlock = className?.includes('language-')
                          if (isBlock) {
                            return (
                              <code className={className} {...props}>
                                {children}
                              </code>
                            )
                          }
                          const text = stripInlineCodeDelimiters(String(children ?? ''))
                          return (
                            <code className="agui-inline-code" {...props}>
                              {text}
                            </code>
                          )
                        },
                        img: ({ src, alt }) => (
                          <img
                            src={src}
                            alt={alt ?? 'image'}
                            className="agui-message__image"
                            loading="lazy"
                          />
                        ),
                      }}
                    >
                      {normalizeMarkdown(msg.content) || (isRunning ? '▋' : '')}
                    </ReactMarkdown>
                  </>
                ) : (
                  <p>{msg.content}</p>
                )}
              </div>
            </div>
          )
        })}

        {isRunning && messages[messages.length - 1]?.role === 'user' && (
          <div className="agui-message agui-message--assistant">
            <div className="agui-message__role">Agent</div>
            <div className="agui-message__bubble">
              <span className="agui-thinking">▋</span>
            </div>
          </div>
        )}

        <div ref={bottomRef} />
      </div>

      <div className="agui-chat__input-area">
        <textarea
          ref={inputRef}
          className="agui-chat__input"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={onKeyDown}
          placeholder="Message your agent… (Enter to send, Shift+Enter for newline)"
          rows={1}
          disabled={isRunning}
        />
        <button
          type="button"
          className="agui-chat__send"
          onClick={submit}
          disabled={isRunning || !input.trim()}
          aria-label="Send message"
        >
          {isRunning ? (
            <span className="agui-chat__spinner" />
          ) : (
            <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor" aria-hidden>
              <path d="M2.01 21L23 12 2.01 3 2 10l15 2-15 2z" />
            </svg>
          )}
        </button>
      </div>
    </div>
  )
}
