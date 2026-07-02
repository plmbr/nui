// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useEffect, useLayoutEffect, useRef, useState } from 'react'
import {
  Flame,
  Gamepad2,
  Sparkles,
  type LucideIcon,
} from 'lucide-react'
import ReactMarkdown from 'react-markdown'
import rehypeHighlight from 'rehype-highlight'
import remarkGfm from 'remark-gfm'
import { CodeBlock } from '@/components/CodeBlock'
import { DiffBlock } from '@/components/DiffBlock'
import { ThinkingIndicator } from '@/components/ThinkingIndicator'
import { MentionMenu } from '@/components/MentionMenu'
import { ToolCallBubble } from '@/components/ToolCallBubble'
import { imageSrc, useSessionChat, type AssistantPart } from '@/hooks/useSessionChat'
import { useMentionMenu } from '@/hooks/useMentionMenu'
import { looksLikeDiff } from '@/lib/diff'
import { normalizeMarkdown, stripInlineCodeDelimiters } from '@/lib/markdown'
import { getCodeBlockInfo } from '@/lib/reactNodeText'
import type { PromptSuggestion, Session } from '@/types'

const AUTO_PROMPT_FALLBACK = 'Follow your system instructions and run.'
const SCROLL_ANCHOR_TOP_GAP = 12

const SUGGESTION_ICONS: Record<string, LucideIcon> = {
  sparkles: Sparkles,
  flame: Flame,
  'gamepad-2': Gamepad2,
  gamepad2: Gamepad2,
}

function SuggestionPillIcon({ icon }: { icon?: string }) {
  const Icon = (icon && SUGGESTION_ICONS[icon.toLowerCase()]) || Sparkles
  return <Icon className="agui-chat__suggestion-pill-icon" aria-hidden />
}

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
    container.clientHeight - anchor.offsetHeight - contentBelow - paddingTop - SCROLL_ANCHOR_TOP_GAP,
  )
  spacer.style.height = `${spacerHeight}px`
}

function scrollMessageToTop(container: HTMLElement, message: HTMLElement) {
  const offset =
    message.getBoundingClientRect().top -
    container.getBoundingClientRect().top +
    container.scrollTop
  container.scrollTo({
    top: Math.max(0, offset - SCROLL_ANCHOR_TOP_GAP),
    behavior: 'auto',
  })
}

interface Props {
  session: Session
  initialPrompt?: string
  hideInput?: boolean
  promptMode?: 'user' | 'auto'
  defaultPrompt?: string
  promptSuggestions?: PromptSuggestion[]
}

export function ChatPanel({
  session,
  initialPrompt,
  hideInput,
  promptMode = 'user',
  defaultPrompt,
  promptSuggestions,
}: Props) {
  const { messages, sendMessage, stopRun, isRunning, isLoading } = useSessionChat(session.id)
  const [input, setInput] = useState('')
  const messagesContainerRef = useRef<HTMLDivElement>(null)
  const scrollSpacerRef = useRef<HTMLDivElement>(null)
  const messageRefs = useRef<Map<string, HTMLDivElement>>(new Map())
  const scrollPendingRef = useRef(false)
  const anchoredUserMsgIdRef = useRef<string | null>(null)
  const inputRef = useRef<HTMLTextAreaElement>(null)
  const initialPromptSentRef = useRef(false)

  useEffect(() => {
    initialPromptSentRef.current = false
  }, [session.id])

  const mention = useMentionMenu({
    sessionId: session.id,
    input,
    setInput,
    inputRef,
    disabled: isRunning || hideInput,
  })

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
    const container = messagesContainerRef.current
    const spacer = scrollSpacerRef.current
    if (!container || !spacer) return

    const lastUserMsg = [...messages].reverse().find((m) => m.role === 'user')
    if (!lastUserMsg) return

    const anchorEl = messageRefs.current.get(lastUserMsg.id)
    if (!anchorEl) return

    if (scrollPendingRef.current) {
      updateScrollSpacer(container, anchorEl, spacer)
      scrollPendingRef.current = false
      anchoredUserMsgIdRef.current = lastUserMsg.id
      scrollMessageToTop(container, anchorEl)
      return
    }

    if (anchoredUserMsgIdRef.current !== lastUserMsg.id) return

    updateScrollSpacer(container, anchorEl, spacer)
  }, [messages, isRunning])

  const submit = () => {
    const text = input.trim()
    if (!text || isRunning) return
    setInput('')
    markScrollAnchor()
    sendMessage(text)
  }

  const submitPrompt = (text: string) => {
    const trimmed = text.trim()
    if (!trimmed || isRunning) return
    setInput('')
    markScrollAnchor()
    sendMessage(trimmed)
  }

  const showPromptSuggestions =
    !hideInput &&
    promptMode === 'user' &&
    messages.length === 0 &&
    !initialPrompt?.trim() &&
    (promptSuggestions?.length ?? 0) > 0

  const onKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (mention.handleKeyDown(e)) return
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      submit()
    }
  }

  const streamingAssistantId = isRunning
    ? [...messages].reverse().find((m) => m.role === 'assistant')?.id
    : undefined

  const renderAssistantText = (content: string) => (
    <ReactMarkdown
      remarkPlugins={[remarkGfm]}
      rehypePlugins={[rehypeHighlight]}
      components={{
        pre: ({ children, ...props }) => {
          const { text, className } = getCodeBlockInfo(children)
          if (looksLikeDiff(text, className)) {
            return <DiffBlock text={text} className={className} />
          }
          return <CodeBlock {...props}>{children}</CodeBlock>
        },
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
      {normalizeMarkdown(content)}
    </ReactMarkdown>
  )

  const renderAssistantPart = (part: AssistantPart, partIndex: number, msgId: string) => {
    if (part.type === 'tool') {
      return <ToolCallBubble key={part.id} part={part} />
    }

    return (
      <div key={`${msgId}-text-${partIndex}`} className="agui-message__text-part">
        {renderAssistantText(part.content)}
      </div>
    )
  }

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
          const isStreaming = isRunning && msg.id === streamingAssistantId
          const parts = msg.parts
          const hasParts = msg.role === 'assistant' && parts && parts.length > 0

          return (
            <div
              key={msg.id}
              ref={setMessageRef(msg.id)}
              className={`agui-message agui-message--${msg.role}`}
            >
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
                    {hasParts ? (
                      parts!.map((part, index) => renderAssistantPart(part, index, msg.id))
                    ) : msg.content ? (
                      renderAssistantText(msg.content)
                    ) : null}
                    {isStreaming && <ThinkingIndicator variant="streaming" />}
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
        {showPromptSuggestions && (
          <div className="agui-chat__suggestions" role="list">
            {promptSuggestions!.map((suggestion) => (
              <button
                key={suggestion.title}
                type="button"
                role="listitem"
                className="agui-chat__suggestion-pill"
                disabled={isRunning}
                onClick={() => submitPrompt(suggestion.prompt)}
              >
                <SuggestionPillIcon icon={suggestion.icon} />
                {suggestion.title}
              </button>
            ))}
          </div>
        )}
        <div className="agui-chat__input-row">
        <div className="agui-chat__input-wrap">
          <MentionMenu
            open={mention.open}
            items={mention.items}
            breadcrumb={mention.breadcrumb}
            activeIndex={mention.activeIndex}
            loading={mention.loading}
            parent={mention.parent}
            onSelect={mention.applySelection}
            onBack={mention.goBack}
            onHover={mention.setActiveIndex}
          />
          <textarea
            ref={inputRef}
            className="agui-chat__input"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={onKeyDown}
            placeholder="Message your agent… (@ to mention, Enter to send)"
            rows={1}
            disabled={isRunning}
            aria-autocomplete={mention.open ? 'list' : undefined}
            aria-expanded={mention.open}
            aria-controls={mention.open ? 'mention-menu' : undefined}
            aria-activedescendant={
              mention.open && mention.items.length > 0
                ? `mention-option-${mention.activeIndex}`
                : undefined
            }
          />
        </div>
        <button
          type="button"
          className={isRunning ? 'agui-chat__stop' : 'agui-chat__send'}
          onClick={isRunning ? () => void stopRun() : submit}
          disabled={!isRunning && !input.trim()}
          aria-label={isRunning ? 'Stop agent' : 'Send message'}
        >
          {isRunning ? (
            <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor" aria-hidden>
              <rect x="6" y="6" width="12" height="12" rx="1" />
            </svg>
          ) : (
            <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor" aria-hidden>
              <path d="M2.01 21L23 12 2.01 3 2 10l15 2-15 2z" />
            </svg>
          )}
        </button>
        </div>
      </div>
      )}
    </div>
  )
}
