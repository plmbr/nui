// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useEffect, useLayoutEffect, useRef, useState } from 'react'
import ReactMarkdown from 'react-markdown'
import rehypeHighlight from 'rehype-highlight'
import remarkGfm from 'remark-gfm'
import { ThinkingIndicator } from '@/components/ThinkingIndicator'
import { ToolCallBubble } from '@/components/ToolCallBubble'
import { imageSrc, useSessionChat } from '@/hooks/useSessionChat'
import { normalizeMarkdown, stripInlineCodeDelimiters } from '@/lib/markdown'
import type { Session } from '@/types'

const AUTO_PROMPT_FALLBACK = 'Follow your system instructions and run.'

function getContentHeightBelow(anchor: HTMLElement, endBefore: HTMLElement): number {
  if (anchor.nextElementSibling === endBefore) return 0
  return Math.max(0, endBefore.offsetTop - (anchor.offsetTop + anchor.offsetHeight))
}

function updateScrollSpacer(
  container: HTMLElement,
  anchor: HTMLElement,
  spacer: HTMLElement,
) {
  const paddingTop = Number.parseFloat(getComputedStyle(container).paddingTop) || 0
  const contentBelow = getContentHeightBelow(anchor, spacer)
  const spacerHeight = Math.max(
    0,
    container.clientHeight - anchor.offsetHeight - contentBelow - paddingTop,
  )
  spacer.style.height = `${spacerHeight}px`
}

function scrollMessageToTop(container: HTMLElement, message: HTMLElement) {
  const offset =
    message.getBoundingClientRect().top -
    container.getBoundingClientRect().top +
    container.scrollTop
  container.scrollTo({ top: offset, behavior: 'auto' })
}

interface Props {
  session: Session
  initialPrompt?: string
  hideInput?: boolean
  promptMode?: 'user' | 'auto'
  defaultPrompt?: string
}

export function ChatPanel({
  session,
  initialPrompt,
  hideInput,
  promptMode = 'user',
  defaultPrompt,
}: Props) {
  const { messages, sendMessage, isRunning, isLoading } = useSessionChat(session.id)
  const [input, setInput] = useState('')
  const messagesContainerRef = useRef<HTMLDivElement>(null)
  const scrollSpacerRef = useRef<HTMLDivElement>(null)
  const messageRefs = useRef<Map<string, HTMLDivElement>>(new Map())
  const scrollPendingRef = useRef(false)
  const inputRef = useRef<HTMLTextAreaElement>(null)
  const initialPromptSentRef = useRef(false)

  const setMessageRef = (id: string) => (el: HTMLDivElement | null) => {
    if (el) messageRefs.current.set(id, el)
    else messageRefs.current.delete(id)
  }

  const markScrollAnchor = () => {
    scrollPendingRef.current = true
  }

  useEffect(() => {
    if (hideInput) return
    inputRef.current?.focus()
  }, [session.id, hideInput])

  useEffect(() => {
    if (hideInput || isRunning) return
    inputRef.current?.focus()
  }, [isRunning, hideInput])

  useEffect(() => {
    if (initialPromptSentRef.current || isLoading || isRunning) return
    if (messages.length > 0) return

    if (promptMode === 'auto') {
      const prompt =
        initialPrompt?.trim() || defaultPrompt?.trim() || AUTO_PROMPT_FALLBACK
      initialPromptSentRef.current = true
      markScrollAnchor()
      sendMessage(prompt)
      return
    }

    const bootstrapPrompt = initialPrompt?.trim()
    if (!bootstrapPrompt) return
    initialPromptSentRef.current = true
    markScrollAnchor()
    sendMessage(bootstrapPrompt)
  }, [
    initialPrompt,
    defaultPrompt,
    promptMode,
    isLoading,
    isRunning,
    messages.length,
    sendMessage,
  ])

  useEffect(() => {
    if (isLoading) return
    if (messages.length > 0) {
      setInput('')
      return
    }
    if (promptMode !== 'user' || hideInput) return
    if (initialPrompt?.trim()) return
    setInput(defaultPrompt?.trim() ?? '')
  }, [
    session.id,
    defaultPrompt,
    initialPrompt,
    promptMode,
    hideInput,
    isLoading,
    messages.length,
  ])

  useLayoutEffect(() => {
    if (!scrollPendingRef.current) return

    const container = messagesContainerRef.current
    const spacer = scrollSpacerRef.current
    if (!container || !spacer) return

    const lastUserMsg = [...messages].reverse().find((m) => m.role === 'user')
    if (!lastUserMsg) return

    const anchorEl = messageRefs.current.get(lastUserMsg.id)
    if (!anchorEl) return

    updateScrollSpacer(container, anchorEl, spacer)
    scrollPendingRef.current = false
    scrollMessageToTop(container, anchorEl)
  }, [messages.length])

  const submit = () => {
    const text = input.trim()
    if (!text || isRunning) return
    setInput('')
    markScrollAnchor()
    sendMessage(text)
  }

  const onKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      submit()
    }
  }

  const streamingAssistantId = isRunning
    ? [...messages].reverse().find((m) => m.role === 'assistant')?.id
    : undefined

  return (
    <div className="agui-chat flex flex-col flex-1 min-h-0">
      <div ref={messagesContainerRef} className="agui-chat__messages">
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

          const isStreaming = isRunning && msg.id === streamingAssistantId

          return (
            <div
              key={msg.id}
              ref={setMessageRef(msg.id)}
              className={`agui-message agui-message--${msg.role}`}
            >
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
                    {msg.content ? (
                      <>
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
                          {normalizeMarkdown(msg.content)}
                        </ReactMarkdown>
                        {isStreaming && <ThinkingIndicator variant="streaming" />}
                      </>
                    ) : isStreaming ? (
                      <ThinkingIndicator />
                    ) : null}
                  </>
                ) : (
                  <p>{msg.content}</p>
                )}
              </div>
            </div>
          )
        })}
        {messages.length > 0 && (
          <div ref={scrollSpacerRef} className="agui-chat__scroll-spacer" aria-hidden="true" />
        )}
      </div>

      {!hideInput && (
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
      )}
    </div>
  )
}
