// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { describe, expect, it } from 'vitest'
import {
  DEFAULT_UI_THEME,
  preferredModeForTheme,
  resolveUITheme,
  themeSupportsMode,
  UI_THEME_LIST,
} from '@/lib/uiThemes'

describe('uiThemes', () => {
  it('defaults to standard', () => {
    expect(resolveUITheme(undefined).id).toBe(DEFAULT_UI_THEME)
    expect(resolveUITheme('unknown').id).toBe('standard')
    expect(DEFAULT_UI_THEME).toBe('standard')
  })

  it('lists Standard before Hawaiian in settings order', () => {
    expect(UI_THEME_LIST.map((t) => t.id)).toEqual(['standard', 'hawaiian'])
  })

  it('hawaiian shows flowers and supports both modes', () => {
    const def = resolveUITheme('hawaiian')
    expect(def.flowers).toBe(true)
    expect(themeSupportsMode(def, 'light')).toBe(true)
    expect(themeSupportsMode(def, 'dark')).toBe(true)
  })

  it('standard hides flowers and supports both modes', () => {
    const def = resolveUITheme('standard')
    expect(def.flowers).toBe(false)
    expect(themeSupportsMode(def, 'light')).toBe(true)
    expect(themeSupportsMode(def, 'dark')).toBe(true)
  })

  it('preferredModeForTheme falls back when mode is unsupported', () => {
    const lightOnly = { ...resolveUITheme('hawaiian'), modes: ['light'] as const }
    expect(preferredModeForTheme(lightOnly, 'dark')).toBe('light')
    expect(preferredModeForTheme(resolveUITheme('hawaiian'), 'dark')).toBe('dark')
  })
})
