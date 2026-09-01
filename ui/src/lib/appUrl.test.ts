// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { describe, expect, it } from 'vitest'
import {
  CUSTOMIZE_PATH,
  LAUNCH_PATH,
  SCHEDULES_PATH,
  agentGroupIdFromPath,
  customizePath,
  customizeTabFromSearch,
  isCustomizePath,
  isLaunchPath,
  isSchedulesPath,
  sessionIdFromPath,
  sessionListPath,
  sessionPath,
} from '@/lib/appUrl'

describe('appUrl', () => {
  it('builds session path', () => {
    expect(sessionPath('abc')).toBe('/sessions/abc')
  })

  it('parses session id from path', () => {
    expect(sessionIdFromPath('/sessions/550e8400-e29b-41d4-a716-446655440000')).toBe(
      '550e8400-e29b-41d4-a716-446655440000',
    )
    expect(sessionIdFromPath('/sessions/new')).toBeNull()
    expect(sessionIdFromPath('/sessions/create')).toBeNull()
    expect(sessionIdFromPath('/sessions/claude-code')).toBeNull()
  })

  it('parses agent group id from path', () => {
    expect(agentGroupIdFromPath('/sessions/claude-code')).toBe('claude-code')
    expect(agentGroupIdFromPath('/sessions/__builtin__')).toBe('__builtin__')
    expect(agentGroupIdFromPath('/sessions/new')).toBeNull()
    expect(agentGroupIdFromPath('/sessions/550e8400-e29b-41d4-a716-446655440000')).toBeNull()
  })

  it('builds session list path', () => {
    expect(sessionListPath('claude-code')).toBe('/sessions/claude-code')
    expect(sessionListPath('adl:my-agent')).toBe('/sessions/adl%3Amy-agent')
    expect(agentGroupIdFromPath('/sessions/adl%3Amy-agent')).toBe('adl:my-agent')
  })

  it('detects launch, customize, and schedules routes', () => {
    expect(isLaunchPath(LAUNCH_PATH)).toBe(true)
    expect(isCustomizePath(CUSTOMIZE_PATH)).toBe(true)
    expect(isSchedulesPath(SCHEDULES_PATH)).toBe(true)
    expect(isSchedulesPath('/other')).toBe(false)
  })

  it('parses customize tab from search and builds customize path', () => {
    expect(customizeTabFromSearch('?tab=mcp')).toBe('mcp')
    expect(customizeTabFromSearch('?tab=invalid')).toBeNull()
    expect(customizePath('env', 'vscode')).toBe('/customize?tab=env&embed=vscode')
  })
})
