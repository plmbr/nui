// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import type { HitlPayload, HitlQuestion, HitlQuestionOption } from '@/types'
import { formatToolCallSummary } from '@/lib/toolCallSummary'

function pickString(record: Record<string, unknown>, ...keys: string[]): string {
  for (const key of keys) {
    const value = record[key]
    if (typeof value === 'string' && value.trim()) {
      return value.trim()
    }
  }
  return ''
}

function normalizeHitlQuestion(question: HitlQuestion): HitlQuestion {
  const record = question as Record<string, unknown>
  const text = pickString(record, 'question', 'prompt', 'text')
  const header = pickString(record, 'header', 'id', 'name')
  return {
    ...question,
    ...(text ? { question: text } : {}),
    ...(header ? { header } : {}),
  }
}

export function normalizeHitlPayload(payload?: HitlPayload): HitlPayload {
  const base = payload ?? {}
  let questions = Array.isArray(base.questions)
    ? base.questions.map((question) => normalizeHitlQuestion(question))
    : []

  questions = questions.filter((question) => {
    const text = question.question?.trim()
    const options = normalizeHitlQuestionOptions(question.options)
    return Boolean(text) || options.length > 0
  })

  const message = pickString(base as Record<string, unknown>, 'message')
  const title = pickString(base as Record<string, unknown>, 'title')
  const topLevelQuestion = pickString(
    base as Record<string, unknown>,
    'question',
    'prompt',
    'text',
  )

  if (questions.length === 0) {
    const text = message || topLevelQuestion || title
    if (text) {
      questions = [
        {
          question: text,
          ...(title && title !== text ? { header: title } : {}),
        },
      ]
    }
  }

  return {
    ...base,
    ...(message ? { message } : {}),
    ...(title ? { title } : {}),
    questions,
  }
}

export function normalizeHitlQuestionOptions(
  options: HitlQuestion['options'],
): HitlQuestionOption[] {
  if (!options?.length) return []
  return options.map((option, index) => {
    if (typeof option === 'string') {
      const label = option.trim()
      return { label: label || `Option ${index + 1}` }
    }
    if (option && typeof option === 'object') {
      const record = option as Record<string, unknown>
      const label = pickString(record, 'label', 'name', 'value', 'text')
      const description = pickString(record, 'description')
      return {
        label: label || `Option ${index + 1}`,
        ...(description ? { description } : {}),
      }
    }
    return { label: `Option ${index + 1}` }
  })
}

export function formatHitlApprovalInline(
  toolName: string | undefined,
  toolInput: Record<string, unknown> | undefined,
): string {
  const bareName = toolName?.split(':').pop()?.split('__').pop()?.trim() || 'Tool'
  const input = toolInput ?? {}
  if (Object.keys(input).length === 0) return bareName
  return `${bareName} ${JSON.stringify(input)}`
}

export function formatHitlApprovalSummary(
  toolName: string | undefined,
  toolInput: Record<string, unknown> | undefined,
  message?: string,
): string {
  const input = toolInput ?? {}
  const description = pickString(input, 'description')
  if (description) return description

  const summary = formatToolCallSummary(toolName, input)
  if (summary) return summary

  const trimmedMessage = message?.trim()
  if (trimmedMessage) return trimmedMessage

  const bareName = toolName?.split(':').pop()?.split('__').pop()?.trim()
  return bareName ? `${bareName} action` : 'Tool action'
}

export function hitlApprovalCommand(
  toolInput: Record<string, unknown> | undefined,
): string {
  return pickString(toolInput ?? {}, 'command', 'cmd')
}

export function isRedundantHitlApprovalMessage(
  message: string | undefined,
  toolInput: Record<string, unknown> | undefined,
): boolean {
  const trimmed = message?.trim()
  if (!trimmed) return true

  const input = toolInput ?? {}
  const command = pickString(input, 'command', 'cmd')
  if (command && trimmed === command) return true

  const path = pickString(input, 'file_path', 'path', 'target_file')
  if (path && trimmed === path) return true

  return false
}
