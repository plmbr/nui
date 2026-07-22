// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { EvalsSection } from '@/components/customize/EvalsSection'
import { defaultFormEval } from '@/lib/adlAgentForm'

describe('EvalsSection', () => {
  it('renders smoke-test fields for a new eval', () => {
    const onChange = vi.fn()
    render(
      <EvalsSection
        evals={[defaultFormEval('smoke')]}
        onChange={onChange}
      />,
    )

    expect(screen.getByLabelText('Eval name')).toHaveValue('smoke')
    expect(screen.getByText('Prompt')).toBeInTheDocument()
    expect(screen.getByText('Expected text')).toBeInTheDocument()
    expect(screen.queryByText('Input mode')).not.toBeInTheDocument()
  })

  it('reveals grader options in Advanced', () => {
    render(
      <EvalsSection
        evals={[defaultFormEval('smoke')]}
        onChange={vi.fn()}
      />,
    )

    fireEvent.click(screen.getByRole('button', { name: /advanced/i }))
    expect(screen.getByText('Grader')).toBeInTheDocument()
  })

  it('shows read-only notice for conversation evals', () => {
    const conversationEval = {
      ...defaultFormEval('follow-up'),
      inputMode: 'conversation' as const,
      input: '',
      messages: [
        { role: 'user' as const, content: 'Hi' },
        { role: 'assistant' as const, content: 'Hello' },
        { role: 'user' as const, content: 'Bye' },
      ],
    }

    render(
      <EvalsSection
        evals={[conversationEval]}
        onChange={vi.fn()}
      />,
    )

    expect(screen.getByText(/conversation eval/i)).toBeInTheDocument()
    expect(screen.getByText('Hi')).toBeInTheDocument()
    expect(screen.queryByText('Prompt')).not.toBeInTheDocument()
  })

  it('calls onRunCase for enabled evals', () => {
    const onRunCase = vi.fn()
    const evalCase = {
      ...defaultFormEval('smoke'),
      input: 'say hello',
      expectValue: 'hello',
    }

    render(
      <EvalsSection
        evals={[evalCase]}
        onChange={vi.fn()}
        onRunCase={onRunCase}
      />,
    )

    fireEvent.click(screen.getByRole('button', { name: /^run$/i }))
    expect(onRunCase).toHaveBeenCalledWith('smoke')
  })
})
