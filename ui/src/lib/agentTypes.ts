// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import type { AgentType } from '@/types'

/** Agent types that can be selected for a new session or as the default agent. */
export function selectableAgentTypes(types: AgentType[]): AgentType[] {
  return types.filter((t) => t.available)
}

export function pickDefaultAgentTypeId(
  types: AgentType[],
  preferredId?: string | null,
): string {
  const selectable = selectableAgentTypes(types)
  if (preferredId) {
    const preferred = selectable.find((t) => t.id === preferredId)
    if (preferred) return preferred.id
  }
  const builtin = selectable.find((t) => t.isBuiltin)
  return builtin?.id ?? selectable[0]?.id ?? ''
}
