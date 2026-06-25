// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { harnessLabel } from '@/lib/agentDisplay'
import type { AgentType } from '@/types'

interface Props {
  name: string
  agent: AgentType
}

export function AgentHeader({ name, agent }: Props) {
  const typeLabel = harnessLabel(agent.harness, agent.sandbox)

  return (
    <Tooltip>
      <TooltipTrigger
        className="app-agent-header min-w-0 max-w-[min(24rem,40vw)] truncate text-sm font-medium text-muted-foreground transition-colors hover:text-foreground"
        aria-label={`${name}, ${agent.label}, ${typeLabel}${agent.description ? `, ${agent.description}` : ''}`}
      >
        {name}
      </TooltipTrigger>
      <TooltipContent
        side="bottom"
        align="start"
        className="flex max-w-xs flex-col items-start gap-1 px-3 py-2"
      >
        <span className="font-medium">{typeLabel}</span>
        {agent.description && (
          <span className="text-background/80 leading-snug">{agent.description}</span>
        )}
      </TooltipContent>
    </Tooltip>
  )
}
