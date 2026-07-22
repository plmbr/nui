// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useState } from 'react'
import { cn } from '@/lib/utils'
import type { AgentEvalResult } from '@/types'

const OUTPUT_PREVIEW_LEN = 200

function EvalResultRow({ res }: { res: AgentEvalResult }) {
  const [showOutput, setShowOutput] = useState(false)
  const output = res.output?.trim() ?? ''
  const truncated = output.length > OUTPUT_PREVIEW_LEN
  const preview = truncated ? `${output.slice(0, OUTPUT_PREVIEW_LEN)}…` : output

  return (
    <div
      className={cn(
        'rounded-md border px-3 py-2',
        res.status === 'pass' && 'border-green-500/40 bg-green-500/5',
        res.status === 'fail' && 'border-destructive/40 bg-destructive/5',
        (res.status === 'error' || res.status === 'skip') && 'border-amber-500/40 bg-amber-500/5',
      )}
    >
      <div className="flex items-center justify-between gap-2">
        <span className="font-medium">{res.name}</span>
        <span className="text-xs uppercase text-muted-foreground">{res.status}</span>
      </div>
      {(res.message || res.error) && (
        <p className="text-xs text-muted-foreground mt-1">{res.error || res.message}</p>
      )}
      {output && (
        <div className="mt-2 space-y-1">
          <p className="text-xs font-medium text-muted-foreground">Agent output</p>
          <pre className="text-xs whitespace-pre-wrap font-mono bg-muted/40 rounded px-2 py-1.5 max-h-40 overflow-y-auto">
            {showOutput || !truncated ? output : preview}
          </pre>
          {truncated && (
            <button
              type="button"
              className="text-xs text-primary hover:underline"
              onClick={() => setShowOutput((v) => !v)}
            >
              {showOutput ? 'Show less' : 'Show more'}
            </button>
          )}
        </div>
      )}
      <p className="text-xs text-muted-foreground mt-1">{res.duration}</p>
    </div>
  )
}

export { EvalResultRow }
