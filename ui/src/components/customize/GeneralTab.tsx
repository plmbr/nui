// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useEffect, useState } from 'react'
import { TriangleAlert } from 'lucide-react'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectTrigger,
  SelectValue,
  selectItemData,
} from '@/components/ui/select'
import { api } from '@/api'
import { pickDefaultAgentTypeId, selectableAgentTypes } from '@/lib/agentTypes'
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

  const handleDefaultAgentChange = (id: string | null) => {
    if (!id) return
    setDefaultAgentType(id)
    api.settings.update({ defaultAgentType: id }).catch(() => {})
  }

  const bwrapUnavailable = capabilities !== null && !capabilities.sandbox.bwrap.available

  const selectableAgentTypesList = selectableAgentTypes(agentTypes)

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
          Used when Loop creates a session on startup.
        </p>
        {selectableAgentTypesList.length > 0 && (
          <Select
            value={defaultAgentType}
            onValueChange={handleDefaultAgentChange}
            items={selectItemData(selectableAgentTypesList)}
          >
            <SelectTrigger className="w-full max-w-md">
              <SelectValue placeholder="Select agent" />
            </SelectTrigger>
            <SelectContent>
              {selectableAgentTypesList.some((a) => a.isBuiltin) && (
                <SelectGroup>
                  <SelectLabel>Built-in</SelectLabel>
                  {selectableAgentTypesList.filter((a) => a.isBuiltin).map((a) => (
                    <SelectItem key={a.id} value={a.id}>
                      {a.label}
                    </SelectItem>
                  ))}
                </SelectGroup>
              )}
              {selectableAgentTypesList.some((a) => !a.isBuiltin) && (
                <SelectGroup>
                  <SelectLabel>Custom</SelectLabel>
                  {selectableAgentTypesList.filter((a) => !a.isBuiltin).map((a) => (
                    <SelectItem key={a.id} value={a.id}>
                      {a.label}
                    </SelectItem>
                  ))}
                </SelectGroup>
              )}
            </SelectContent>
          </Select>
        )}
      </div>
    </div>
  )
}
