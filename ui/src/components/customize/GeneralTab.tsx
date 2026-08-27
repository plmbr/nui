// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useEffect, useMemo, useState } from 'react'
import { TriangleAlert } from 'lucide-react'
import { SearchableSelect } from '@/components/SearchableSelect'
import { api } from '@/api'
import { useTheme } from '@/contexts/theme'
import {
  pickDefaultAgentTypeId,
  pickDefaultHarnessRef,
  selectableAgentTypes,
  selectableHarnessRefs,
} from '@/lib/agentTypes'
import { BUILTIN_AGENTS_LABEL, INSTALLED_AGENTS_LABEL } from '@/lib/sessionGroups'
import { UI_THEME_LIST } from '@/lib/uiThemes'
import type { AgentType, Capabilities, UIThemeId } from '@/types'

export function GeneralTab() {
  const { uiTheme, setUITheme } = useTheme()
  const [capabilities, setCapabilities] = useState<Capabilities | null>(null)
  const [agentTypes, setAgentTypes] = useState<AgentType[]>([])
  const [defaultAgentType, setDefaultAgentType] = useState('')
  const [defaultHarness, setDefaultHarness] = useState('')

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
        setDefaultHarness(pickDefaultHarnessRef(types, settings.defaultHarness))
      })
      .catch(() => {})
  }, [])

  const handleDefaultAgentChange = (id: string) => {
    setDefaultAgentType(id)
    api.settings.update({ defaultAgentType: id }).catch(() => {})
  }

  const handleDefaultHarnessChange = (ref: string) => {
    setDefaultHarness(ref)
    api.settings.update({ defaultHarness: ref }).catch(() => {})
  }

  const handleUIThemeChange = (id: UIThemeId) => {
    setUITheme(id)
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

  const harnessSelectItems = useMemo(
    () =>
      selectableHarnessRefs(agentTypes).map((h) => ({
        id: h.ref,
        label: h.label,
        group: h.group,
      })),
    [agentTypes],
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
        <p className="text-sm font-medium mb-1">Appearance</p>
        <p className="text-xs text-muted-foreground mb-3">
          Visual theme for the app. Use the header toggle for light and dark when the theme supports both.
        </p>
        <div className="grid grid-cols-2 gap-3 max-w-md">
          {UI_THEME_LIST.map((def) => {
            const active = uiTheme === def.id
            const modeLabel =
              def.modes.length === 2
                ? 'Light & dark'
                : def.modes[0] === 'dark'
                  ? 'Dark only'
                  : 'Light only'
            return (
              <button
                key={def.id}
                type="button"
                className="theme-card"
                data-active={active}
                aria-pressed={active}
                onClick={() => handleUIThemeChange(def.id)}
              >
                <div
                  className={`theme-card__preview ${
                    def.flowers ? 'theme-card__preview--hawaiian' : 'theme-card__preview--standard'
                  }`}
                >
                  <div className="theme-card__sidebar" />
                  <div className="theme-card__main">
                    {def.flowers && <span className="theme-card__bloom" aria-hidden />}
                    <div className="theme-card__bubble theme-card__bubble--user" />
                    <div className="theme-card__bubble theme-card__bubble--agent" />
                  </div>
                </div>
                <div className="theme-card__label">
                  <span>{def.label}</span>
                  <span className="theme-card__meta">{modeLabel}</span>
                </div>
              </button>
            )
          })}
        </div>
      </div>
      <div>
        <p className="text-sm font-medium mb-1">Default harness</p>
        <p className="text-xs text-muted-foreground mb-3">
          Used by the nui master agent (launcher orchestrator).
        </p>
        {harnessSelectItems.length > 0 && (
          <SearchableSelect
            value={defaultHarness}
            onValueChange={handleDefaultHarnessChange}
            items={harnessSelectItems}
            placeholder="Select harness"
            searchPlaceholder="Search harnesses…"
            triggerClassName="max-w-md"
          />
        )}
      </div>
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
