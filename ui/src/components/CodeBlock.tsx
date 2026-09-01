// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useRef, useState } from 'react'
import { Check, Copy } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { copyTextToClipboard } from '@/lib/clipboard'

interface Props extends React.HTMLAttributes<HTMLPreElement> {
  children?: React.ReactNode
}

export function CodeBlock({ children, className, ...props }: Props) {
  const preRef = useRef<HTMLPreElement>(null)
  const [copied, setCopied] = useState(false)

  const handleCopy = async () => {
    const text = preRef.current?.textContent ?? ''
    if (!text) return

    const copied = await copyTextToClipboard(text)
    if (!copied) return
    setCopied(true)
    window.setTimeout(() => setCopied(false), 2000)
  }

  return (
    <div className="agui-code-block not-prose">
      <Button
        type="button"
        variant="ghost"
        size="icon-sm"
        className="agui-code-block__copy"
        onClick={handleCopy}
        aria-label={copied ? 'Copied' : 'Copy code'}
        title={copied ? 'Copied!' : 'Copy'}
      >
        {copied ? <Check aria-hidden /> : <Copy aria-hidden />}
      </Button>
      <pre ref={preRef} className={className} {...props}>
        {children}
      </pre>
    </div>
  )
}
