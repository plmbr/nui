// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useState } from 'react'
import { Check, Copy } from 'lucide-react'
import { Button } from '@/components/ui/button'

interface Props {
  content: string
}

export function UserMessageBubble({ content }: Props) {
  const [copied, setCopied] = useState(false)

  const handleCopy = async () => {
    if (!content) return

    try {
      await navigator.clipboard.writeText(content)
      setCopied(true)
      window.setTimeout(() => setCopied(false), 2000)
    } catch {
      // Clipboard access may be denied in some contexts.
    }
  }

  return (
    <div className="agui-message__user-wrap">
      <div className="agui-message__bubble">
        <Button
          type="button"
          variant="ghost"
          size="icon-sm"
          className="agui-code-block__copy"
          onClick={handleCopy}
          aria-label={copied ? 'Copied' : 'Copy message'}
          title={copied ? 'Copied!' : 'Copy'}
        >
          {copied ? <Check aria-hidden /> : <Copy aria-hidden />}
        </Button>
        <p>{content}</p>
      </div>
    </div>
  )
}
