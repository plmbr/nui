// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { createContext, useContext, useEffect, useState } from 'react'
import { api } from '@/api'
import { isEmbedHost } from '@/lib/embedHost'
import { isExternalThemeActive, requestEmbedHostTheme } from '@/lib/externalTheme'
import {
  DEFAULT_UI_THEME,
  preferredModeForTheme,
  resolveUITheme,
  themeSupportsMode,
  type UIThemeDefinition,
} from '@/lib/uiThemes'
import type { ColorMode, UIThemeId } from '@/types'

interface ThemeContextValue {
  /** Color mode (light / dark). */
  theme: ColorMode
  setTheme: (theme: ColorMode) => void
  /** Visual theme (Hawaiian, Standard, …). */
  uiTheme: UIThemeId
  uiThemeDef: UIThemeDefinition
  setUITheme: (id: UIThemeId) => void
  /** Whether the active visual theme supports dark mode. */
  supportsDark: boolean
  /** Whether the active visual theme supports light mode. */
  supportsLight: boolean
  /** Whether light/dark can be toggled for the active visual theme. */
  canToggleMode: boolean
}

const ThemeContext = createContext<ThemeContextValue>({
  theme: 'light',
  setTheme: () => {},
  uiTheme: DEFAULT_UI_THEME,
  uiThemeDef: resolveUITheme(DEFAULT_UI_THEME),
  setUITheme: () => {},
  supportsDark: true,
  supportsLight: true,
  canToggleMode: true,
})

function readCachedUITheme(): UIThemeId {
  return resolveUITheme(localStorage.getItem('uiTheme')).id
}

function readCachedColorMode(uiThemeId: UIThemeId): ColorMode {
  const def = resolveUITheme(uiThemeId)
  const cached = localStorage.getItem('theme') as ColorMode | null
  return preferredModeForTheme(def, cached === 'dark' || cached === 'light' ? cached : 'light')
}

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const [uiTheme, setUIThemeState] = useState<UIThemeId>(() => readCachedUITheme())
  const [theme, setThemeState] = useState<ColorMode>(() => readCachedColorMode(readCachedUITheme()))

  const uiThemeDef = resolveUITheme(uiTheme)
  const supportsDark = themeSupportsMode(uiThemeDef, 'dark')
  const supportsLight = themeSupportsMode(uiThemeDef, 'light')
  const canToggleMode = supportsDark && supportsLight

  const embedHost = isEmbedHost()

  // Apply color mode + visual theme to DOM; keep localStorage as fast-path cache
  useEffect(() => {
    document.documentElement.dataset.uiTheme = uiTheme
    if (embedHost) {
      if (!isExternalThemeActive()) {
        document.documentElement.classList.toggle('dark', theme === 'dark')
        document.documentElement.style.colorScheme = theme
      }
      requestEmbedHostTheme()
      return
    }
    document.documentElement.classList.toggle('dark', theme === 'dark')
    document.documentElement.style.colorScheme = theme
    localStorage.setItem('theme', theme)
    localStorage.setItem('uiTheme', uiTheme)
  }, [embedHost, theme, uiTheme])

  // On mount: reconcile with server (source of truth)
  useEffect(() => {
    api.settings
      .get()
      .then((s) => {
        const nextUI = resolveUITheme(s.uiTheme).id
        const nextMode = preferredModeForTheme(
          resolveUITheme(nextUI),
          s.theme === 'dark' || s.theme === 'light' ? s.theme : 'light',
        )
        setUIThemeState(nextUI)
        setThemeState(nextMode)
        if (embedHost) {
          requestEmbedHostTheme()
        }
      })
      .catch(() => {})
  }, [embedHost])

  function setTheme(t: ColorMode) {
    if (!themeSupportsMode(uiThemeDef, t)) return
    setThemeState(t)
    api.settings.update({ theme: t }).catch(() => {})
  }

  function setUITheme(id: UIThemeId) {
    const def = resolveUITheme(id)
    const nextMode = preferredModeForTheme(def, theme)
    setUIThemeState(def.id)
    setThemeState(nextMode)
    api.settings
      .update({
        uiTheme: def.id,
        ...(nextMode !== theme ? { theme: nextMode } : {}),
      })
      .catch(() => {})
  }

  return (
    <ThemeContext.Provider
      value={{
        theme,
        setTheme,
        uiTheme,
        uiThemeDef,
        setUITheme,
        supportsDark,
        supportsLight,
        canToggleMode,
      }}
    >
      {children}
    </ThemeContext.Provider>
  )
}

export function useTheme() {
  return useContext(ThemeContext)
}
