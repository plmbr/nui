// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useEffect, useMemo, useState } from 'react'
import { TriangleAlert } from 'lucide-react'
import { SearchableSelect } from '@/components/SearchableSelect'
import { api } from '@/api'
import { pickDefaultAgentTypeId, selectableAgentTypes } from '@/lib/agentTypes'
import { BUILTIN_AGENTS_LABEL, INSTALLED_AGENTS_LABEL } from '@/lib/sessionGroups'
import type { AgentType, Capabilities } from '@/types'

export function GeneralTab() {
  const [capabilities, setCapabilities] = useState<Capabilities | null>(null)
  const [agentTypes, setAgentTypes] = useState<AgentType[]>([])
  const [defaultAgentType, setDefaultAgentType] = useState('')

  useEffect(() => {
    Promise.all([
      api.capabilities.get(),
      api.agentTypes.list(),
      api.settings.get().catch(() => ({ theme: 'light' as const })),
    ])
      .then(([caps, types, settings]) => {
        setCapabilities(caps)
        setAgentTypes(types)
        setDefaultAgentType(pickDefaultAgentTypeId(types, settings.defaultAgentType))
      })
      .catch(() => {})
  }, [])

  const handleDefaultAgentChange = (id: string) => {
    setDefaultAgentType(id)
    api.settings.update({ defaultAgentType: id }).catch(() => {})
  }

  const bwrapUnavailable = capabilities !== null && !capabilities.sandbox.bwrap.available

  const selectableAgentTypesList = selectableAgentTypes(agentTypes)
  const agentSelectItems = useMemo(
    () =>
      selectableAgentTypesList.map((agent) => ({
        id: agent.id,
        label: agent.label,
        group: agent.isBuiltin ? BUILTIN_AGENTS_LABEL : INSTALLED_AGENTS_LABEL,
        description: agent.description,
      })),
    [selectableAgentTypesList],
  )

  return (
    <div className="customize-tab-content space-y-6">
      {bwrapUnavailable && (
        <div className="sandbox-warning">
          <TriangleAlert className="size-4 shrink-0 mt-0.5" />
          <div>
            <p className="font-medium">Sandboxing unavailable</p>
            <p className="text-xs mt-0.5 opacity-80">
              {capabilities!.sandbox.bwrap.error ?? 'bubblewrap (bwrap) not found'}
            </p>
          </div>
        </div>
      )}
      <div>
        <p className="text-sm font-medium mb-1">Default agent</p>
        <p className="text-xs text-muted-foreground mb-3">
          Used when nui creates a session on startup.
        </p>
        {selectableAgentTypesList.length > 0 && (
          <SearchableSelect
            value={defaultAgentType}
            onValueChange={handleDefaultAgentChange}
            items={agentSelectItems}
            placeholder="Select agent"
            searchPlaceholder="Search agents…"
            triggerClassName="max-w-md"
          />
        )}
      </div>
    </div>
  )
}
