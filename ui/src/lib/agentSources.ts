// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import type { AgentType, ExtensionInfo } from '@/types'

export const LOCAL_CUSTOM_AGENT_SOURCE = 'local'

/** Extension name from an agent id like ext:loop-extension/agent-id, or null for local agents. */
export function parseExtensionNameFromAgentId(id: string): string | null {
  if (!id.startsWith('ext:')) return null
  const rest = id.slice(4)
  const slash = rest.indexOf('/')
  if (slash <= 0) return null
  return rest.slice(0, slash)
}

/** Stable filter key for a custom agent's source (local or ext:<name>). */
export function customAgentSourceKey(agent: AgentType): string {
  if (agent.source === 'extension') {
    const extName = parseExtensionNameFromAgentId(agent.id)
    if (extName) return `ext:${extName}`
  }
  const extName = parseExtensionNameFromAgentId(agent.id)
  if (extName) return `ext:${extName}`
  return LOCAL_CUSTOM_AGENT_SOURCE
}

export function customAgentSourceLabel(
  sourceKey: string,
  extensions: ExtensionInfo[],
): string {
  if (sourceKey === LOCAL_CUSTOM_AGENT_SOURCE) return 'Local'
  if (sourceKey.startsWith('ext:')) {
    const extName = sourceKey.slice(4)
    const ext = extensions.find((e) => e.name === extName)
    return ext?.displayName || extName
  }
  return sourceKey
}

export interface CustomAgentSourceOption {
  key: string
  label: string
}

/** Build sorted source filter pills from custom agents and installed extensions. */
export function buildCustomAgentSourceOptions(
  agents: AgentType[],
  extensions: ExtensionInfo[],
): CustomAgentSourceOption[] {
  const keys = new Set<string>()
  for (const agent of agents) {
    keys.add(customAgentSourceKey(agent))
  }

  const options: CustomAgentSourceOption[] = []

  if (keys.has(LOCAL_CUSTOM_AGENT_SOURCE)) {
    options.push({
      key: LOCAL_CUSTOM_AGENT_SOURCE,
      label: customAgentSourceLabel(LOCAL_CUSTOM_AGENT_SOURCE, extensions),
    })
  }

  const extensionKeys = [...keys]
    .filter((key) => key.startsWith('ext:'))
    .sort((a, b) =>
      customAgentSourceLabel(a, extensions).localeCompare(
        customAgentSourceLabel(b, extensions),
        undefined,
        { sensitivity: 'base' },
      ),
    )

  for (const key of extensionKeys) {
    options.push({ key, label: customAgentSourceLabel(key, extensions) })
  }

  return options
}

export function sortCustomAgentsByName(agents: AgentType[]): AgentType[] {
  return [...agents].sort((a, b) =>
    a.label.localeCompare(b.label, undefined, { sensitivity: 'base' }),
  )
}

export function filterCustomAgentsBySources(
  agents: AgentType[],
  selectedSourceKeys: ReadonlySet<string>,
): AgentType[] {
  if (selectedSourceKeys.size === 0) return agents
  return agents.filter((agent) => selectedSourceKeys.has(customAgentSourceKey(agent)))
}
