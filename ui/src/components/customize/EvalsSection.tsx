// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useState } from 'react'
import type { ReactNode } from 'react'
import { ChevronDown, ChevronRight, FlaskConical, Plus, Trash2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import {
  type FormEval,
  type EvalExpectType,
  defaultFormEval,
  isConversationEval,
  usesSimpleGrader,
} from '@/lib/adlAgentForm'

interface Props {
  evals: FormEval[]
  onChange: (evals: FormEval[]) => void
  onRunCase?: (name: string) => void
  runningCase?: string | null
}

function RequiredMark() {
  return <span className="text-destructive" aria-hidden="true">*</span>
}

function FieldLabel({
  required,
  children,
}: {
  required?: boolean
  children: ReactNode
}) {
  return (
    <Label>
      {children}
      {required && <> <RequiredMark /></>}
    </Label>
  )
}

function EvalAdvanced({
  ev,
  index,
  updateEval,
}: {
  ev: FormEval
  index: number
  updateEval: (index: number, partial: Partial<FormEval>) => void
}) {
  const conversation = isConversationEval(ev)
  const simpleGrader = usesSimpleGrader(ev)
  const [open, setOpen] = useState(
    () =>
      conversation ||
      !simpleGrader ||
      Boolean(ev.description.trim()) ||
      Boolean(ev.timeout.trim()) ||
      Boolean(ev.tags.trim()) ||
      Boolean(ev.workingDir.trim()),
  )

  return (
    <div className="space-y-2">
      <button
        type="button"
        className="inline-flex items-center gap-1 text-xs font-medium text-muted-foreground hover:text-foreground"
        onClick={() => setOpen((v) => !v)}
        aria-expanded={open}
      >
        {open ? <ChevronDown className="size-3.5" /> : <ChevronRight className="size-3.5" />}
        Advanced
      </button>
      {open && (
        <div className="space-y-3 rounded-md border border-dashed p-3">
          {conversation && (
            <p className="text-xs text-amber-700 dark:text-amber-300">
              Conversation eval — edit message turns in YAML mode. Assistant turns are not injected
              into the session; only user messages are sent at run time.
            </p>
          )}
          <div className="space-y-1.5">
            <Label>Description</Label>
            <Input
              value={ev.description}
              onChange={(e) => updateEval(index, { description: e.target.value })}
              placeholder="What this eval verifies"
              disabled={conversation}
            />
          </div>
          {!conversation && (
            <div className="space-y-1.5">
              <Label>Grader</Label>
              <Select
                value={ev.expectType || 'contains'}
                onValueChange={(v) =>
                  updateEval(index, {
                    expectType: (v ?? 'contains') as EvalExpectType,
                  })
                }
                items={[
                  { value: 'contains', label: 'Contains' },
                  { value: 'exact', label: 'Exact match' },
                  { value: 'regex', label: 'Regex' },
                  { value: 'llm', label: 'LLM judge' },
                  { value: 'none', label: 'Manual (none)' },
                ]}
              >
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="contains">Contains</SelectItem>
                  <SelectItem value="exact">Exact match</SelectItem>
                  <SelectItem value="regex">Regex</SelectItem>
                  <SelectItem value="llm">LLM judge</SelectItem>
                  <SelectItem value="none">Manual (none)</SelectItem>
                </SelectContent>
              </Select>
            </div>
          )}
          {!conversation && !simpleGrader && (
            <>
              {(ev.expectType === 'exact' || ev.expectType === 'regex') && (
                <div className="space-y-1.5">
                  <FieldLabel required>Expected value</FieldLabel>
                  <Input
                    value={ev.expectValue}
                    onChange={(e) => updateEval(index, { expectValue: e.target.value })}
                    placeholder={ev.expectType === 'regex' ? 'pattern' : 'expected text'}
                  />
                </div>
              )}
              {ev.expectType === 'llm' && (
                <div className="space-y-1.5">
                  <FieldLabel required>Criteria</FieldLabel>
                  <Textarea
                    value={ev.expectCriteria}
                    onChange={(e) => updateEval(index, { expectCriteria: e.target.value })}
                    rows={2}
                    placeholder="Rubric for the LLM judge"
                  />
                </div>
              )}
            </>
          )}
          {conversation && (
            <div className="space-y-2">
              <Label>Messages (read-only)</Label>
              {ev.messages.map((msg, msgIndex) => (
                <div
                  key={msgIndex}
                  className="grid grid-cols-1 gap-1 text-xs sm:grid-cols-[5rem_minmax(0,1fr)]"
                >
                  <span className="font-medium capitalize text-muted-foreground">{msg.role}</span>
                  <span className="font-mono whitespace-pre-wrap">{msg.content || '—'}</span>
                </div>
              ))}
            </div>
          )}
          <div className="grid gap-3 sm:grid-cols-2">
            <div className="space-y-1.5">
              <Label>Timeout (seconds)</Label>
              <Input
                value={ev.timeout}
                onChange={(e) => updateEval(index, { timeout: e.target.value })}
                placeholder="120"
              />
            </div>
            <div className="space-y-1.5">
              <Label>Tags</Label>
              <Input
                value={ev.tags}
                onChange={(e) => updateEval(index, { tags: e.target.value })}
                placeholder="smoke, regression"
              />
              <p className="text-xs text-muted-foreground">CLI filtering only for now.</p>
            </div>
            <div className="space-y-1.5 sm:col-span-2">
              <Label>Working dir override</Label>
              <Input
                value={ev.workingDir}
                onChange={(e) => updateEval(index, { workingDir: e.target.value })}
                placeholder="Optional path for this eval case"
              />
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

export function EvalsSection({ evals, onChange, onRunCase, runningCase }: Props) {
  const [expanded, setExpanded] = useState<Set<number>>(() => {
    if (evals.length <= 1) return new Set(evals.map((_, i) => i))
    return new Set<number>()
  })

  const updateEval = (index: number, partial: Partial<FormEval>) => {
    const next = [...evals]
    next[index] = { ...next[index], ...partial }
    onChange(next)
  }

  const toggleExpanded = (index: number) => {
    setExpanded((prev) => {
      const next = new Set(prev)
      if (next.has(index)) next.delete(index)
      else next.add(index)
      return next
    })
  }

  const addEval = () => {
    const base = defaultFormEval(`eval-${evals.length + 1}`)
    onChange([...evals, base])
    setExpanded((prev) => new Set([...prev, evals.length]))
  }

  return (
    <section className="space-y-3">
      <h3 className="text-sm font-semibold">Evals</h3>
      <p className="text-xs text-muted-foreground">
        Smoke tests for agent behavior. Name, prompt, and expected text are enough for most cases.
      </p>
      {evals.length > 0 && (
        <ul className="space-y-3">
          {evals.map((ev, index) => {
            const conversation = isConversationEval(ev)
            const simpleGrader = usesSimpleGrader(ev)
            const isExpanded = expanded.has(index)
            const canRun = onRunCase && !ev.disabled && ev.name.trim()

            return (
              <li key={`eval-${index}`} className="rounded-md border">
                <div className="flex items-center gap-2 p-3">
                  {evals.length > 1 && (
                    <button
                      type="button"
                      className="shrink-0 text-muted-foreground hover:text-foreground"
                      onClick={() => toggleExpanded(index)}
                      aria-expanded={isExpanded}
                      aria-label={isExpanded ? 'Collapse eval' : 'Expand eval'}
                    >
                      {isExpanded ? (
                        <ChevronDown className="size-4" />
                      ) : (
                        <ChevronRight className="size-4" />
                      )}
                    </button>
                  )}
                  <Input
                    value={ev.name}
                    onChange={(e) => updateEval(index, { name: e.target.value })}
                    placeholder="polite-greeting"
                    className="h-8 flex-1"
                    aria-label="Eval name"
                  />
                  <label className="flex shrink-0 items-center gap-1.5 text-xs">
                    <input
                      type="checkbox"
                      checked={!ev.disabled}
                      onChange={(e) => updateEval(index, { disabled: !e.target.checked })}
                      className="rounded border"
                    />
                    On
                  </label>
                  {canRun && (
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      className="shrink-0"
                      disabled={runningCase != null}
                      onClick={() => onRunCase(ev.name.trim())}
                    >
                      <FlaskConical className="size-3.5" />
                      {runningCase === ev.name.trim() ? 'Running…' : 'Run'}
                    </Button>
                  )}
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    className="shrink-0"
                    onClick={() => {
                      onChange(evals.filter((_, i) => i !== index))
                      setExpanded((prev) => {
                        const next = new Set<number>()
                        for (const i of prev) {
                          if (i < index) next.add(i)
                          else if (i > index) next.add(i - 1)
                        }
                        return next
                      })
                    }}
                  >
                    <Trash2 className="size-3.5" />
                  </Button>
                </div>

                {(evals.length === 1 || isExpanded) && (
                  <div className="space-y-3 border-t px-3 pb-3 pt-3">
                    {conversation ? (
                      <EvalAdvanced ev={ev} index={index} updateEval={updateEval} />
                    ) : (
                      <>
                        <div className="space-y-1.5">
                          <FieldLabel required>Prompt</FieldLabel>
                          <Textarea
                            value={ev.input}
                            onChange={(e) => updateEval(index, { input: e.target.value })}
                            rows={3}
                            placeholder="User message sent to the agent"
                          />
                        </div>
                        {simpleGrader && (
                          <div className="space-y-1.5">
                            <FieldLabel required>Expected text</FieldLabel>
                            <Input
                              value={ev.expectValue}
                              onChange={(e) =>
                                updateEval(index, {
                                  expectValue: e.target.value,
                                  expectType: 'contains',
                                })
                              }
                              placeholder="Substring the response should include"
                            />
                          </div>
                        )}
                        {!simpleGrader && (
                          <p className="text-xs text-muted-foreground">
                            Grader: <span className="font-medium">{ev.expectType}</span> — set
                            expected values in Advanced.
                          </p>
                        )}
                        <EvalAdvanced ev={ev} index={index} updateEval={updateEval} />
                      </>
                    )}
                  </div>
                )}
              </li>
            )
          })}
        </ul>
      )}
      <Button type="button" variant="outline" size="sm" onClick={addEval}>
        <Plus className="size-3.5" />
        Add eval
      </Button>
    </section>
  )
}
