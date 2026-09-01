// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useState } from 'react'
import { Check, Copy } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { copyTextToClipboard } from '@/lib/clipboard'

interface Props {
  content: string
}

export function UserMessageBubble({ content }: Props) {
  const [copied, setCopied] = useState(false)

  const handleCopy = async () => {
    if (!content) return

    const copied = await copyTextToClipboard(content)
    if (!copied) return
    setCopied(true)
    window.setTimeout(() => setCopied(false), 2000)
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
