// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useMemo, useRef, useState } from 'react'
import { Check, Copy } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { looksLikeDiff, parseUnifiedDiff, type DiffLine } from '@/lib/diff'

interface Props {
  text: string
  className?: string
}

function formatLineNumber(value?: number): string {
  return value === undefined ? '' : String(value)
}

function DiffRow({ line }: { line: DiffLine }) {
  if (line.type === 'meta') {
    return (
      <div className="agui-diff__row agui-diff__row--meta">
        <span className="agui-diff__content">{line.content}</span>
      </div>
    )
  }

  return (
    <div className={`agui-diff__row agui-diff__row--${line.type}`}>
      <span className="agui-diff__gutter agui-diff__gutter--old" aria-hidden>
        {formatLineNumber(line.oldLine)}
      </span>
      <span className="agui-diff__gutter agui-diff__gutter--new" aria-hidden>
        {formatLineNumber(line.newLine)}
      </span>
      <span className="agui-diff__sign" aria-hidden>
        {line.type === 'add' ? '+' : line.type === 'delete' ? '-' : ' '}
      </span>
      <span className="agui-diff__content">{line.content}</span>
    </div>
  )
}

export function DiffBlock({ text, className }: Props) {
  const containerRef = useRef<HTMLDivElement>(null)
  const [copied, setCopied] = useState(false)
  const lines = useMemo(() => parseUnifiedDiff(text), [text])

  if (!looksLikeDiff(text, className)) return null

  const handleCopy = async () => {
    if (!text) return

    try {
      await navigator.clipboard.writeText(text)
      setCopied(true)
      window.setTimeout(() => setCopied(false), 2000)
    } catch {
      // Clipboard access may be denied in some contexts.
    }
  }

  return (
    <div ref={containerRef} className="agui-diff">
      <Button
        type="button"
        variant="ghost"
        size="icon"
        className="agui-diff__copy"
        onClick={handleCopy}
        aria-label={copied ? 'Copied' : 'Copy diff'}
        title={copied ? 'Copied!' : 'Copy'}
      >
        {copied ? <Check aria-hidden /> : <Copy aria-hidden />}
      </Button>
      <div className="agui-diff__table" role="group" aria-label="Code diff">
        {lines.map((line, index) => (
          <DiffRow key={`${line.type}-${index}`} line={line} />
        ))}
      </div>
    </div>
  )
}
