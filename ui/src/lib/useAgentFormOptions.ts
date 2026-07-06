// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useCallback, useEffect, useState } from 'react'
import { api } from '@/api'
import {
  buildHarnessOptions,
  type AgentFormOptions,
  type MCPOption,
  type SkillOption,
} from '@/lib/adlAgentForm'

export function useAgentFormOptions() {
  const [options, setOptions] = useState<AgentFormOptions>({
    harnesses: buildHarnessOptions([]),
    skills: [],
    mcpServers: [],
  })
  const [loading, setLoading] = useState(true)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const [agentTypes, skills, mcpRes, extensions] = await Promise.all([
        api.agentTypes.list(),
        api.skills.list().catch(() => []),
        api.mcpServers.list().catch(() => ({ mcpServers: [] })),
        api.extensions.list().catch(() => []),
      ])

      const harnesses = buildHarnessOptions(agentTypes)

      const skillOptions: SkillOption[] = skills.map((s) => ({
        id: `skill:${s.name}`,
        label: s.name,
        group: 'Installed skills',
        name: s.name,
        ref: s.name,
      }))

      for (const ext of extensions) {
        if (ext.disabled) continue
        const extLabel = ext.displayName || ext.name
        for (const skill of ext.skills ?? []) {
          const ref = `ext:${ext.name}/${skill}`
          skillOptions.push({
            id: `ext-skill:${ref}`,
            label: `${skill} (${extLabel})`,
            group: 'Extension skills',
            name: skill,
            ref,
          })
        }
      }

      const mcpOptions: MCPOption[] = mcpRes.mcpServers.map((s) => ({
        id: `user:${s.name}`,
        label: s.name,
        group: 'User MCP servers',
        name: s.name,
        server: s,
      }))

      for (const ext of extensions) {
        if (ext.disabled) continue
        const extLabel = ext.displayName || ext.name
        const configByName = new Map(
          (ext.mcpServerConfigs ?? []).map((s) => [s.name, s]),
        )
        for (const server of ext.mcpServers ?? []) {
          const ref = `ext:${ext.name}/${server}`
          const config = configByName.get(server)
          mcpOptions.push({
            id: `ext-mcp:${ref}`,
            label: `${server} (${extLabel})`,
            group: 'Extension MCP servers',
            name: server,
            ref: config ? undefined : ref,
            server: config,
          })
        }
      }

      setOptions({ harnesses, skills: skillOptions, mcpServers: mcpOptions })
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  return { options, loading, reload: load }
}
