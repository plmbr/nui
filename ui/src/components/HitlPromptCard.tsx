// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useState } from 'react'
import { api } from '@/api'
import { dismissHitlRequest } from '@/lib/sessionChatStore'
import {
  formatHitlApprovalInline,
  formatHitlApprovalSummary,
  hitlApprovalCommand,
  isRedundantHitlApprovalMessage,
  normalizeHitlPayload,
  normalizeHitlQuestionOptions,
} from '@/lib/hitlPayload'
import type { HitlQuestion, HitlRequest } from '@/types'

interface Props {
  sessionId: string
  request: HitlRequest
}

function kindLabel(kind: string): string {
  switch (kind) {
    case 'question':
      return 'Question'
    case 'approval':
      return 'Tool approval'
    case 'review':
      return 'Review'
    case 'freeform':
      return 'Input'
    default:
      return 'Human input'
  }
}

function bareToolName(toolName: string | undefined): string {
  return toolName?.split(':').pop()?.split('__').pop()?.trim() || 'Tool'
}

function ApprovalToolPreview({
  toolName,
  toolInput,
  message,
}: {
  toolName: string
  toolInput?: Record<string, unknown>
  message?: string
}) {
  const [expanded, setExpanded] = useState(false)
  const inline = formatHitlApprovalInline(toolName, toolInput)
  const command = hitlApprovalCommand(toolInput)
  const hasDetails =
    Boolean(toolInput && Object.keys(toolInput).length > 0) ||
    (message?.trim() && !isRedundantHitlApprovalMessage(message, toolInput))

  return (
    <div className="hitl-prompt__approval">
      <button
        type="button"
        className="hitl-prompt__approval-summary"
        onClick={() => hasDetails && setExpanded((value) => !value)}
        disabled={!hasDetails}
        aria-expanded={hasDetails ? expanded : undefined}
        title={inline}
      >
        <span className="hitl-prompt__approval-inline">
          <span className="hitl-prompt__approval-tool">{bareToolName(toolName)}</span>
          {toolInput && Object.keys(toolInput).length > 0 && (
            <span className="hitl-prompt__approval-params">
              {' '}
              {JSON.stringify(toolInput)}
            </span>
          )}
        </span>
        {hasDetails && (
          <span className="hitl-prompt__approval-toggle">{expanded ? '▲' : '▼'}</span>
        )}
      </button>

      {expanded && hasDetails && (
        <div className="hitl-prompt__approval-details">
          {message?.trim() && !isRedundantHitlApprovalMessage(message, toolInput) && (
            <div className="hitl-prompt__approval-section">
              <div className="hitl-prompt__approval-label">Prompt</div>
              <pre className="hitl-prompt__approval-code">{message.trim()}</pre>
            </div>
          )}
          {command && (
            <div className="hitl-prompt__approval-section">
              <div className="hitl-prompt__approval-label">Command</div>
              <pre className="hitl-prompt__approval-code">{command}</pre>
            </div>
          )}
          {toolInput && Object.keys(toolInput).length > 0 && (
            <div className="hitl-prompt__approval-section">
              <div className="hitl-prompt__approval-label">Parameters</div>
              <pre className="hitl-prompt__approval-code">
                {JSON.stringify(toolInput, null, 2)}
              </pre>
            </div>
          )}
        </div>
      )}
    </div>
  )
}

export function HitlPromptCard({ sessionId, request }: Props) {
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')
  const [selectedOptions, setSelectedOptions] = useState<Record<number, string[]>>({})
  const [freeformText, setFreeformText] = useState('')

  const payload = normalizeHitlPayload(request.payload)
  const title = payload.title?.trim()
  const message = payload.message?.trim()
  const questions = payload.questions ?? []
  const actions = payload.actions ?? []

  async function submit(answers: Record<string, unknown>, status?: string) {
    setSubmitting(true)
    setError('')
    try {
      await api.hitl.respond(request.requestId, answers, status)
      dismissHitlRequest(sessionId, request.requestId)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to submit response.')
    } finally {
      setSubmitting(false)
    }
  }

  function toggleOption(questionIndex: number, label: string, multiSelect?: boolean) {
    setSelectedOptions((current) => {
      const prev = current[questionIndex] ?? []
      if (multiSelect) {
        const next = prev.includes(label)
          ? prev.filter((item) => item !== label)
          : [...prev, label]
        return { ...current, [questionIndex]: next }
      }
      return { ...current, [questionIndex]: [label] }
    })
  }

  function submitQuestionAnswers() {
    const answers: Record<string, unknown> = {}
    if (questions.length === 1 && !normalizeHitlQuestionOptions(questions[0]?.options).length) {
      answers.answer = freeformText.trim()
    } else {
      const questionAnswers = questions.map((q, index) => {
        const selected = selectedOptions[index] ?? []
        if (normalizeHitlQuestionOptions(q.options).length) {
          return q.multiSelect ? selected : selected[0] ?? ''
        }
        return freeformText.trim()
      })
      answers.answers = questionAnswers
      if (questionAnswers.length === 1) {
        answers.answer = questionAnswers[0]
      }
    }
    void submit(answers)
  }

  function renderQuestion(question: HitlQuestion, index: number) {
    const header = question.header?.trim()
    const text = question.question?.trim()
    const options = normalizeHitlQuestionOptions(question.options)

    return (
      <div key={index} className="hitl-prompt__question">
        {header && <div className="hitl-prompt__question-header">{header}</div>}
        {text && <p className="hitl-prompt__question-text">{text}</p>}
        {options.length > 0 ? (
          <div className="hitl-prompt__options" role={question.multiSelect ? 'group' : 'radiogroup'}>
            {options.map((option) => {
              const selected = (selectedOptions[index] ?? []).includes(option.label)
              return (
                <button
                  key={option.label}
                  type="button"
                  role={question.multiSelect ? 'checkbox' : 'radio'}
                  aria-checked={selected}
                  disabled={submitting}
                  className={[
                    'hitl-prompt__option',
                    selected ? 'hitl-prompt__option--selected' : '',
                  ].join(' ')}
                  onClick={() => toggleOption(index, option.label, question.multiSelect)}
                >
                  <span className="hitl-prompt__option-label">{option.label}</span>
                  {option.description && (
                    <span className="hitl-prompt__option-desc">{option.description}</span>
                  )}
                </button>
              )
            })}
          </div>
        ) : (
          <textarea
            className="hitl-prompt__textarea"
            value={freeformText}
            onChange={(e) => setFreeformText(e.target.value)}
            disabled={submitting}
            placeholder="Your answer…"
            rows={3}
            spellCheck={false}
            autoCorrect="off"
            autoCapitalize="off"
          />
        )}
      </div>
    )
  }

  const isQuestion = request.kind === 'question' || request.kind === 'freeform'
  const isApproval = request.kind === 'approval'
  const isReview = request.kind === 'review'
  const showApprovalPreview = isApproval && Boolean(payload.toolName)
  const showMessage =
    Boolean(message) &&
    !showApprovalPreview &&
    !(isApproval && isRedundantHitlApprovalMessage(message, payload.toolInput))
  const showDescription =
    Boolean(payload.description) &&
    !showApprovalPreview &&
    String(payload.description).trim() !== formatHitlApprovalSummary(
      payload.toolName,
      payload.toolInput,
      message,
    )
  const showApprovalTitle = Boolean(title) && !showApprovalPreview

  const questionReady = questions.length > 0
    ? questions.every((q, index) => {
        const options = normalizeHitlQuestionOptions(q.options)
        if (options.length) return (selectedOptions[index]?.length ?? 0) > 0
        return freeformText.trim().length > 0
      })
    : freeformText.trim().length > 0

  return (
    <div
      className={`hitl-prompt${showApprovalPreview ? ' hitl-prompt--approval' : ''}`}
      role="region"
      aria-label={title || kindLabel(request.kind)}
    >
      {!isApproval && (
        <div className="hitl-prompt__header">
          <span className="hitl-prompt__badge">{kindLabel(request.kind)}</span>
          {request.stepName && (
            <span className="hitl-prompt__step">{request.stepName}</span>
          )}
        </div>
      )}

      {!showApprovalTitle && title && isQuestion && questions.length > 0 && (
        <h3 className="hitl-prompt__title">{title}</h3>
      )}
      {showApprovalTitle && <h3 className="hitl-prompt__title">{title}</h3>}
      {showMessage && <p className="hitl-prompt__message">{message}</p>}

      {showDescription && (
        <p className="hitl-prompt__description">{String(payload.description)}</p>
      )}

      {showApprovalPreview && (
        <ApprovalToolPreview
          toolName={String(payload.toolName)}
          toolInput={payload.toolInput}
          message={message}
        />
      )}

      {isQuestion && questions.length > 0 && questions.map(renderQuestion)}
      {isQuestion && questions.length === 0 && (
        <textarea
          className="hitl-prompt__textarea"
          value={freeformText}
          onChange={(e) => setFreeformText(e.target.value)}
          disabled={submitting}
          placeholder="Your answer…"
          rows={3}
          spellCheck={false}
          autoCorrect="off"
          autoCapitalize="off"
        />
      )}

      {error && <p className="hitl-prompt__error">{error}</p>}

      <div className={`hitl-prompt__actions${isApproval ? ' hitl-prompt__actions--compact' : ''}`}>
        {isQuestion && (
          <button
            type="button"
            className="hitl-prompt__btn hitl-prompt__btn--primary"
            disabled={submitting || !questionReady}
            onClick={submitQuestionAnswers}
          >
            {submitting ? 'Submitting…' : 'Submit'}
          </button>
        )}

        {isApproval && (
          <>
            <button
              type="button"
              className="hitl-prompt__btn hitl-prompt__btn--primary"
              disabled={submitting}
              onClick={() => void submit({ action: 'approve', approved: true })}
            >
              Approve
            </button>
            <button
              type="button"
              className="hitl-prompt__btn hitl-prompt__btn--secondary"
              disabled={submitting}
              onClick={() => void submit({ action: 'reject', approved: false }, 'declined')}
            >
              Reject
            </button>
          </>
        )}

        {isReview && actions.length > 0 && actions.map((action) => (
          <button
            key={action.id}
            type="button"
            className="hitl-prompt__btn hitl-prompt__btn--primary"
            disabled={submitting}
            onClick={() => void submit({ action: action.id })}
          >
            {action.label}
          </button>
        ))}

        {isReview && actions.length === 0 && (
          <>
            <button
              type="button"
              className="hitl-prompt__btn hitl-prompt__btn--primary"
              disabled={submitting}
              onClick={() => void submit({ action: 'approve' })}
            >
              Approve
            </button>
            <button
              type="button"
              className="hitl-prompt__btn hitl-prompt__btn--secondary"
              disabled={submitting}
              onClick={() => void submit({ action: 'reject' }, 'declined')}
            >
              Reject
            </button>
          </>
        )}
      </div>
    </div>
  )
}
