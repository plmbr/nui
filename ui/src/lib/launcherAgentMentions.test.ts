// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { describe, expect, it } from 'vitest'
import {
  detectLauncherMentionTrigger,
  formatLauncherAgentMentionToken,
  hasCompleteLauncherAgentMention,
  isLauncherAgentOnlyMention,
  launchableAgentsForMention,
  listLauncherMentionItems,
  parseLauncherAgentMention,
} from '@/lib/launcherAgentMentions'
import type { AgentType } from '@/types'

const agents: AgentType[] = [
  {
    id: 'nui',
    label: 'nui',
    harness: 'api',
    isBuiltin: true,
    available: true,
  },
  {
    id: 'claude-code',
    label: 'Claude Code',
    harness: 'claude-code',
    isBuiltin: true,
    available: true,
  },
  {
    id: 'ext:pack/demo',
    label: 'Demo Agent',
    description: 'demo helper',
    harness: 'extension',
    isBuiltin: false,
    available: true,
  },
  {
    id: 'offline-agent',
    label: 'Offline',
    harness: 'claude-code',
    isBuiltin: false,
    available: false,
  },
]

describe('launchableAgentsForMention', () => {
  it('excludes nui and unavailable agents', () => {
    expect(launchableAgentsForMention(agents).map((agent) => agent.id)).toEqual([
      'claude-code',
      'ext:pack/demo',
    ])
  })
})

describe('listLauncherMentionItems', () => {
  it('shows agents directly at root', () => {
    const { items } = listLauncherMentionItems(agents, '')
    expect(items.map((item) => item.value)).toEqual(['claude-code', 'ext:pack/demo'])
  })

  it('filters agents by query', () => {
    const { items } = listLauncherMentionItems(agents, 'demo')
    expect(items.map((item) => item.value)).toEqual(['ext:pack/demo'])
  })

  it('filters agents when typing partial names', () => {
    const { items } = listLauncherMentionItems(agents, 'claude')
    expect(items.map((item) => item.value)).toEqual(['claude-code'])
  })

  it('matches substrings in labels case-insensitively', () => {
    const withGe: AgentType[] = [
      ...agents,
      { id: 'antigravity', label: 'Antigravity', harness: 'antigravity', isBuiltin: true, available: true },
      { id: 'baseten', label: 'BaseTen Agent', harness: 'api', isBuiltin: false, available: true },
    ]
    const { items } = listLauncherMentionItems(withGe, 'ge')
    expect(items.map((item) => item.value).sort()).toEqual(['baseten', 'ext:pack/demo'])
  })
})

describe('formatLauncherAgentMentionToken', () => {
  it('adds a readable label suffix', () => {
    expect(
      formatLauncherAgentMentionToken(
        'ext:pack/suite-updater',
        'Suite Updater',
      ),
    ).toBe('@ext:pack/suite-updater:[Suite Updater]')
  })
})

describe('hasCompleteLauncherAgentMention', () => {
  it('detects readable mentions', () => {
    expect(hasCompleteLauncherAgentMention('@ext:pack/demo:[Demo Agent] fix')).toBe(true)
  })

  it('does not treat spaces as completing an in-progress mention', () => {
    expect(hasCompleteLauncherAgentMention('@claude-code fix tests')).toBe(false)
    expect(hasCompleteLauncherAgentMention('@Claude Code')).toBe(false)
  })

  it('treats in-progress mentions as incomplete', () => {
    expect(hasCompleteLauncherAgentMention('@ge')).toBe(false)
  })
})

describe('detectLauncherMentionTrigger', () => {
  it('does not open when an agent is already selected', () => {
    const prompt = '@ext:pack/demo:[Demo Agent] fix tests @'
    expect(detectLauncherMentionTrigger(prompt, prompt.length)).toBeNull()
  })

  it('keeps the trigger open when the query contains spaces', () => {
    const prompt = '@Claude Code'
    expect(detectLauncherMentionTrigger(prompt, prompt.length)).toEqual({
      triggerStart: 0,
      query: 'Claude Code',
    })
  })

  it('ends the trigger on newlines', () => {
    const prompt = '@Claude\n'
    expect(detectLauncherMentionTrigger(prompt, prompt.length)).toBeNull()
  })
})

describe('listLauncherMentionItems with spaced queries', () => {
  it('filters multi-word labels while typing a space', () => {
    const withSpaces: AgentType[] = [
      ...agents,
      {
        id: 'suite-updater',
        label: 'Suite Updater',
        harness: 'extension',
        isBuiltin: false,
        available: true,
      },
    ]
    const { items } = listLauncherMentionItems(withSpaces, 'Suite Up')
    expect(items.map((item) => item.value)).toEqual(['suite-updater'])
  })
})

describe('parseLauncherAgentMention', () => {
  it('parses a leading agent mention', () => {
    expect(parseLauncherAgentMention('@claude-code fix tests')).toEqual({
      agentId: 'claude-code',
      delegated: 'fix tests',
    })
  })

  it('parses readable mention tokens', () => {
    expect(
      parseLauncherAgentMention(
        '@ext:pack/suite-updater:[Suite Updater] update suite',
      ),
    ).toEqual({
      agentId: 'ext:pack/suite-updater',
      delegated: 'update suite',
    })
  })

  it('parses mention-only readable tokens', () => {
    expect(
      parseLauncherAgentMention('@claude-code:[Claude Code]'),
    ).toEqual({
      agentId: 'claude-code',
      delegated: '',
    })
  })

  it('matches multi-word labels when agents are provided', () => {
    expect(parseLauncherAgentMention('@claude code what can you do', agents)).toEqual({
      agentId: 'claude-code',
      delegated: 'what can you do',
    })
  })

  it('rejects unknown spaced mentions when agents are provided', () => {
    expect(parseLauncherAgentMention('@claude what can you do', agents)).toBeNull()
  })

  it('rejects non-leading mentions', () => {
    expect(parseLauncherAgentMention('ask @claude-code to help')).toBeNull()
  })

  it('detects mention-only launcher prompts', () => {
    expect(isLauncherAgentOnlyMention('@claude-code:[Claude Code]')).toBe(true)
    expect(isLauncherAgentOnlyMention('@claude-code fix tests')).toBe(false)
    expect(isLauncherAgentOnlyMention('@Claude Code', agents)).toBe(true)
  })
})
