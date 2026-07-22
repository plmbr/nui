// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { TriangleAlert } from 'lucide-react'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { appVersion } from '@/lib/version'

const developmentWarning =
  "nui is under active development and doesn't have all the guardrails in place yet. Use caution when you configure agents with MCP servers, skills and plugins. Always use AI tools from trusted resources. Pay attention to the prompts you input especially when working with CLI based harnesses."

export function AppVersion() {
  return (
    <div className="app-header__version-group">
      <span className="app-header__version" aria-label={`nui version ${appVersion}`}>
        v{appVersion}
      </span>
      <Tooltip>
        <TooltipTrigger
          type="button"
          className="app-header__version-warning"
          aria-label="Development warning"
        >
          <TriangleAlert className="size-3.5" aria-hidden="true" />
        </TooltipTrigger>
        <TooltipContent side="bottom" align="end" className="app-header__version-warning-content">
          {developmentWarning}
        </TooltipContent>
      </Tooltip>
    </div>
  )
}
