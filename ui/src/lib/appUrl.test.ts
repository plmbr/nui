// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { describe, expect, it } from 'vitest'
import {
  CUSTOMIZE_PATH,
  LAUNCH_PATH,
  SCHEDULES_PATH,
  isCustomizePath,
  isLaunchPath,
  isSchedulesPath,
  sessionIdFromPath,
  sessionPath,
} from '@/lib/appUrl'

describe('appUrl', () => {
  it('builds session path', () => {
    expect(sessionPath('abc')).toBe('/sessions/abc')
  })

  it('parses session id from path', () => {
    expect(sessionIdFromPath('/sessions/abc')).toBe('abc')
    expect(sessionIdFromPath('/sessions/new')).toBeNull()
    expect(sessionIdFromPath('/sessions/create')).toBeNull()
  })

  it('detects launch, customize, and schedules routes', () => {
    expect(isLaunchPath(LAUNCH_PATH)).toBe(true)
    expect(isCustomizePath(CUSTOMIZE_PATH)).toBe(true)
    expect(isSchedulesPath(SCHEDULES_PATH)).toBe(true)
    expect(isSchedulesPath('/other')).toBe(false)
  })
})
