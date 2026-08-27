// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import type { ColorMode, UIThemeId } from '@/types'

export interface UIThemeDefinition {
  id: UIThemeId
  label: string
  description: string
  /** Color modes this theme ships. Toggle reflects when only one is available. */
  modes: readonly ColorMode[]
  /** Plumeria backdrop and flower accents. */
  flowers: boolean
  /** Landing prompt placeholder copy. */
  landingPlaceholder: string
}

export const UI_THEMES: Record<UIThemeId, UIThemeDefinition> = {
  hawaiian: {
    id: 'hawaiian',
    label: 'Hawaiian',
    description: 'Warm lei accents with plumeria blooms',
    modes: ['light', 'dark'],
    flowers: true,
    landingPlaceholder:
      'Aloha! Drop a task here — nui will route the mahi so you can kick back and enjoy the breeze.',
  },
  standard: {
    id: 'standard',
    label: 'Standard',
    description: 'Clean look without floral accents',
    modes: ['light', 'dark'],
    flowers: false,
    landingPlaceholder: 'Drop a task here — nui will route it to the right agent.',
  },
}

export const DEFAULT_UI_THEME: UIThemeId = 'hawaiian'
export const UI_THEME_LIST: UIThemeDefinition[] = Object.values(UI_THEMES)

export function resolveUITheme(id: string | null | undefined): UIThemeDefinition {
  if (id && id in UI_THEMES) {
    return UI_THEMES[id as UIThemeId]
  }
  return UI_THEMES[DEFAULT_UI_THEME]
}

export function themeSupportsMode(def: UIThemeDefinition, mode: ColorMode): boolean {
  return def.modes.includes(mode)
}

export function preferredModeForTheme(def: UIThemeDefinition, preferred: ColorMode): ColorMode {
  if (themeSupportsMode(def, preferred)) return preferred
  return def.modes[0] ?? 'light'
}
